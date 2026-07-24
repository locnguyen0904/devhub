package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// fakeStore records revocations and serves a single configurable token.
type fakeStore struct {
	token         storedToken
	tokenErr      error
	created       []sqlcgen.CreateRefreshTokenParams
	revokedTokens []uuid.UUID
	revokedFams   []uuid.UUID
}

func (f *fakeStore) userIDByGitHub(context.Context, string) (uuid.UUID, error) {
	return uuid.Nil, errNoAccount
}
func (f *fakeStore) linkGitHubTx(context.Context, pgx.Tx, uuid.UUID, string) error { return nil }
func (f *fakeStore) createRefreshToken(_ context.Context, p sqlcgen.CreateRefreshTokenParams) error {
	f.created = append(f.created, p)
	return nil
}

func (f *fakeStore) refreshTokenByHash(context.Context, []byte) (storedToken, error) {
	return f.token, f.tokenErr
}

func (f *fakeStore) revokeToken(_ context.Context, id uuid.UUID) error {
	f.revokedTokens = append(f.revokedTokens, id)
	return nil
}

func (f *fakeStore) revokeFamily(_ context.Context, id uuid.UUID) error {
	f.revokedFams = append(f.revokedFams, id)
	return nil
}
func (f *fakeStore) revokeAllForUser(context.Context, uuid.UUID) error { return nil }

type fakeUsers struct{ u user.User }

func (f fakeUsers) GetByID(context.Context, uuid.UUID) (user.User, error) { return f.u, nil }
func (f fakeUsers) CreateFromGitHubTx(context.Context, pgx.Tx, user.GitHubIdentity) (user.User, error) {
	return f.u, nil
}

type fakeIssuer struct{}

func (fakeIssuer) Issue(uuid.UUID, string) (string, time.Time, error) {
	return "access-token", time.Now().Add(15 * time.Minute), nil
}

func newTestService(store *fakeStore, u user.User) *service {
	return &service{
		repo:    store,
		users:   fakeUsers{u: u},
		issuer:  fakeIssuer{},
		refresh: 720 * time.Hour,
		now:     time.Now,
	}
}

func TestRefreshRotatesValidToken(t *testing.T) {
	id := uuid.New()
	family := uuid.New()
	store := &fakeStore{token: storedToken{
		ID: id, UserID: uuid.New(), FamilyID: family,
		ExpiresAt: time.Now().Add(time.Hour), Revoked: false,
	}}
	svc := newTestService(store, user.User{ID: store.token.UserID, Username: "loc"})

	tokens, err := svc.Refresh(context.Background(), "raw-token", SessionMeta{})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if tokens.Access == "" || tokens.Refresh == "" {
		t.Errorf("Refresh() returned empty tokens: %+v", tokens)
	}

	// The presented token must be revoked, and the new one must stay in the
	// same family so the session chain is traceable.
	if len(store.revokedTokens) != 1 || store.revokedTokens[0] != id {
		t.Errorf("revoked tokens = %v, want [%v]", store.revokedTokens, id)
	}
	if len(store.created) != 1 || store.created[0].FamilyID != family {
		t.Errorf("new token family = %v, want %v", store.created, family)
	}
}

func TestRefreshBurnsFamilyOnReuse(t *testing.T) {
	family := uuid.New()
	store := &fakeStore{token: storedToken{
		ID: uuid.New(), UserID: uuid.New(), FamilyID: family,
		ExpiresAt: time.Now().Add(time.Hour), Revoked: true, // already revoked → replay
	}}
	svc := newTestService(store, user.User{})

	_, err := svc.Refresh(context.Background(), "stolen-token", SessionMeta{})

	if !isUnauthenticated(err) {
		t.Errorf("Refresh(reused) error = %v, want unauthenticated", err)
	}
	// Replaying a revoked token means it leaked: the whole family must burn.
	if len(store.revokedFams) != 1 || store.revokedFams[0] != family {
		t.Errorf("revoked families = %v, want [%v]", store.revokedFams, family)
	}
	if len(store.created) != 0 {
		t.Errorf("reuse must not mint a new token, got %d", len(store.created))
	}
}

func TestRefreshRejectsExpiredToken(t *testing.T) {
	id := uuid.New()
	store := &fakeStore{token: storedToken{
		ID: id, UserID: uuid.New(), FamilyID: uuid.New(),
		ExpiresAt: time.Now().Add(-time.Minute), Revoked: false, // expired
	}}
	svc := newTestService(store, user.User{})

	_, err := svc.Refresh(context.Background(), "expired-token", SessionMeta{})

	if !isUnauthenticated(err) {
		t.Errorf("Refresh(expired) error = %v, want unauthenticated", err)
	}
	if len(store.revokedTokens) != 1 || store.revokedTokens[0] != id {
		t.Errorf("expired token should be revoked, got %v", store.revokedTokens)
	}
}

func TestRefreshRejectsUnknownToken(t *testing.T) {
	store := &fakeStore{tokenErr: errNoToken}
	svc := newTestService(store, user.User{})

	_, err := svc.Refresh(context.Background(), "never-issued", SessionMeta{})

	if !isUnauthenticated(err) {
		t.Errorf("Refresh(unknown) error = %v, want unauthenticated", err)
	}
}

func isUnauthenticated(err error) bool {
	var h *httpx.Error
	return errors.As(err, &h) && h.Code == httpx.CodeUnauthenticated
}
