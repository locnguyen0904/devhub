package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
	"github.com/locnguyen0904/devhub/backend/internal/platform/random"
)

// maxUsernameAttempts caps how many times we retry on a colliding username
// before giving up. Each retry adds fresh entropy, so exhausting this is
// effectively impossible; the cap only stops a runaway loop.
const maxUsernameAttempts = 5

// errUsernameExhausted means every generated username collided, which given the
// random suffix is effectively impossible and points at a deeper problem.
var errUsernameExhausted = errors.New("exhausted username attempts")

// Service is the identity API other modules depend on.
type Service interface {
	// GetByID returns a user or an httpx not-found error.
	GetByID(ctx context.Context, id uuid.UUID) (User, error)
	// CreateFromGitHubTx inserts a new user from a GitHub identity within the
	// caller's transaction. It runs in the caller's tx — not its own — so the
	// oauth account the auth module writes commits atomically with the user.
	CreateFromGitHubTx(ctx context.Context, tx pgx.Tx, gh GitHubIdentity) (User, error)
}

type service struct {
	repo *repository
}

func newService(repo *repository) *service {
	return &service{repo: repo}
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (User, error) {
	return s.repo.getByID(ctx, id)
}

func (s *service) CreateFromGitHubTx(ctx context.Context, tx pgx.Tx, gh GitHubIdentity) (User, error) {
	displayName := gh.Name
	if strings.TrimSpace(displayName) == "" {
		displayName = gh.Login
	}

	base := sanitizeUsername(gh.Login)

	// Retry on username collision rather than pre-checking: the UNIQUE index is
	// the only race-free source of truth, so we insert and react to conflicts.
	for attempt := 0; attempt < maxUsernameAttempts; attempt++ {
		username := base
		if attempt > 0 {
			suffix, err := random.Token(4)
			if err != nil {
				return User{}, err
			}
			username = clampUsername(base + "-" + suffix)
		}

		u, err := s.repo.createTx(ctx, tx, sqlcgen.CreateUserParams{
			ID:             uuid.Must(uuid.NewV7()),
			Username:       username,
			Email:          gh.Email,
			DisplayName:    displayName,
			AvatarUrl:      gh.AvatarURL,
			GithubUsername: &gh.Login,
		})
		if err == nil {
			return u, nil
		}
		if isUsernameConflict(err) {
			continue
		}
		return User{}, fmt.Errorf("create user from github: %w", err)
	}
	return User{}, fmt.Errorf("create user from github: %w", errUsernameExhausted)
}

// isUsernameConflict reports whether err is a unique-violation on the username.
// An email conflict is a genuinely different situation (two accounts, one
// email) and must not be silently retried into a new username.
func isUsernameConflict(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "username")
}

// sanitizeUsername reduces a GitHub login to our allowed alphabet and length.
// The username CHECK is ^[a-zA-Z0-9_-]{3,30}$.
func sanitizeUsername(login string) string {
	var b strings.Builder
	for _, r := range login {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		}
	}
	s := b.String()
	if len(s) < 3 {
		// Logins are rarely this short after sanitising; "dev" keeps it valid
		// and the collision retry gives it a unique suffix.
		s += "dev"
	}
	return clampUsername(s)
}

func clampUsername(s string) string {
	if len(s) > 30 {
		return s[:30]
	}
	return s
}
