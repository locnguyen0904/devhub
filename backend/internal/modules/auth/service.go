package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
	"github.com/locnguyen0904/devhub/backend/internal/platform/cache"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
	"github.com/locnguyen0904/devhub/backend/internal/platform/random"
)

const (
	stateKeyPrefix = "oauth:state:"
	stateTTL       = 10 * time.Minute
	refreshBytes   = 32
	stateBytes     = 32
)

// SessionMeta is the request context stored with a refresh token, so a user can
// later see and revoke individual sessions.
type SessionMeta struct {
	UserAgent *string
	IP        *netip.Addr
}

// Tokens is everything a successful login or refresh produces.
type Tokens struct {
	Access          string
	AccessExpiresIn int // seconds
	Refresh         string
	RefreshMaxAge   int // seconds
	User            user.User
}

// accessIssuer is the slice of token.Issuer the service uses. A local interface
// keeps the service unit-testable without a real signing key.
type accessIssuer interface {
	Issue(userID uuid.UUID, username string) (string, time.Time, error)
}

// tokenStore is the persistence the service needs, behind an interface so the
// rotation and reuse-detection logic can be unit-tested without a database.
type tokenStore interface {
	userIDByGitHub(ctx context.Context, githubID string) (uuid.UUID, error)
	linkGitHubTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, githubID string) error
	createRefreshToken(ctx context.Context, p sqlcgen.CreateRefreshTokenParams) error
	refreshTokenByHash(ctx context.Context, hash []byte) (storedToken, error)
	revokeToken(ctx context.Context, id uuid.UUID) error
	revokeFamily(ctx context.Context, familyID uuid.UUID) error
	revokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type service struct {
	db      *database.DB
	repo    tokenStore
	users   userService
	github  *githubClient
	issuer  accessIssuer
	cache   *cache.Client
	refresh time.Duration
	now     func() time.Time
}

func newService(
	db *database.DB,
	repo tokenStore,
	users userService,
	gh *githubClient,
	issuer accessIssuer,
	c *cache.Client,
	refreshTTL time.Duration,
) *service {
	return &service{
		db: db, repo: repo, users: users, github: gh,
		issuer: issuer, cache: c, refresh: refreshTTL, now: time.Now,
	}
}

// AuthorizeURL creates a single-use state, stores it in Redis, and returns the
// GitHub URL to send the browser to. The state defends against login CSRF: an
// attacker cannot forge a callback for a state they never planted.
func (s *service) AuthorizeURL(ctx context.Context) (string, error) {
	state, err := random.Token(stateBytes)
	if err != nil {
		return "", err
	}
	if err := s.cache.Set(ctx, stateKeyPrefix+state, "1", stateTTL); err != nil {
		return "", err
	}
	return s.github.authCodeURL(state), nil
}

// Complete finishes the OAuth dance: verify state, exchange the code, resolve or
// provision the user, and mint a token pair.
func (s *service) Complete(ctx context.Context, code, state string, meta SessionMeta) (Tokens, error) {
	_, found, err := s.cache.GetDel(ctx, stateKeyPrefix+state)
	if err != nil {
		return Tokens{}, err
	}
	if !found {
		return Tokens{}, httpx.New(httpx.CodeUnauthenticated, "Invalid or expired login state", nil)
	}

	identity, err := s.github.exchange(ctx, code)
	if err != nil {
		return Tokens{}, err
	}

	u, err := s.resolveUser(ctx, identity)
	if err != nil {
		return Tokens{}, err
	}

	return s.issueTokens(ctx, u, uuid.Must(uuid.NewV7()), meta)
}

// resolveUser returns the existing user for a GitHub identity, or provisions a
// new one. Provisioning writes the user and the oauth account in one
// transaction so a crash cannot leave a user with no way to log in again.
func (s *service) resolveUser(ctx context.Context, identity user.GitHubIdentity) (user.User, error) {
	userID, err := s.repo.userIDByGitHub(ctx, identity.GitHubID)
	if err == nil {
		return s.users.GetByID(ctx, userID)
	}
	if !errors.Is(err, errNoAccount) {
		return user.User{}, err
	}

	var created user.User
	txErr := s.db.InTx(ctx, func(tx pgx.Tx) error {
		created, err = s.users.CreateFromGitHubTx(ctx, tx, identity)
		if err != nil {
			return err
		}
		return s.repo.linkGitHubTx(ctx, tx, created.ID, identity.GitHubID)
	})
	if txErr != nil {
		return user.User{}, fmt.Errorf("provision user: %w", txErr)
	}
	return created, nil
}

// Refresh rotates a refresh token: the presented token is revoked and a new one
// issued in the same family. Replaying an already-revoked token is treated as
// theft and burns the whole family.
func (s *service) Refresh(ctx context.Context, raw string, meta SessionMeta) (Tokens, error) {
	stored, err := s.repo.refreshTokenByHash(ctx, hashToken(raw))
	if errors.Is(err, errNoToken) {
		return Tokens{}, errUnauthenticatedSession
	}
	if err != nil {
		return Tokens{}, err
	}

	if stored.Revoked {
		// A revoked token being presented again means it leaked: whoever holds
		// it and whoever rotated it are different parties. Burn the family.
		if rerr := s.repo.revokeFamily(ctx, stored.FamilyID); rerr != nil {
			return Tokens{}, rerr
		}
		return Tokens{}, errUnauthenticatedSession
	}

	if s.now().After(stored.ExpiresAt) {
		if rerr := s.repo.revokeToken(ctx, stored.ID); rerr != nil {
			return Tokens{}, rerr
		}
		return Tokens{}, errUnauthenticatedSession
	}

	u, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return Tokens{}, err
	}

	if err := s.repo.revokeToken(ctx, stored.ID); err != nil {
		return Tokens{}, err
	}
	return s.issueTokens(ctx, u, stored.FamilyID, meta)
}

// Logout revokes the presented refresh token. An unknown token is not an error:
// the caller's intent — no valid session afterward — is already satisfied.
func (s *service) Logout(ctx context.Context, raw string) error {
	stored, err := s.repo.refreshTokenByHash(ctx, hashToken(raw))
	if errors.Is(err, errNoToken) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.repo.revokeToken(ctx, stored.ID)
}

// LogoutAll revokes every live session for a user.
func (s *service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	return s.repo.revokeAllForUser(ctx, userID)
}

func (s *service) issueTokens(ctx context.Context, u user.User, familyID uuid.UUID, meta SessionMeta) (Tokens, error) {
	access, accessExp, err := s.issuer.Issue(u.ID, u.Username)
	if err != nil {
		return Tokens{}, err
	}

	raw, err := random.Token(refreshBytes)
	if err != nil {
		return Tokens{}, err
	}
	refreshExp := s.now().Add(s.refresh)

	err = s.repo.createRefreshToken(ctx, sqlcgen.CreateRefreshTokenParams{
		ID:        uuid.Must(uuid.NewV7()),
		UserID:    u.ID,
		TokenHash: hashToken(raw),
		FamilyID:  familyID,
		ExpiresAt: refreshExp,
		UserAgent: meta.UserAgent,
		Ip:        meta.IP,
	})
	if err != nil {
		return Tokens{}, err
	}

	return Tokens{
		Access:          access,
		AccessExpiresIn: int(time.Until(accessExp).Seconds()),
		Refresh:         raw,
		RefreshMaxAge:   int(s.refresh.Seconds()),
		User:            u,
	}, nil
}

// hashToken is what lands in the database. Storing the hash, never the raw
// token, means a database leak cannot be used to impersonate anyone.
func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

// errUnauthenticatedSession is the single response to every refresh failure, so
// a caller cannot tell an unknown token from a revoked or expired one.
var errUnauthenticatedSession = httpx.New(httpx.CodeUnauthenticated, "Session expired, please sign in again", nil)
