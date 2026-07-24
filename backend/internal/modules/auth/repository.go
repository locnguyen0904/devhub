package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
)

// errNoAccount signals that no oauth account exists for a provider identity.
// It never leaves the module: the service turns "no account" into a provision.
var errNoAccount = errors.New("oauth account not found")

// storedToken is the subset of a refresh_tokens row the rotation logic needs.
type storedToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FamilyID  uuid.UUID
	ExpiresAt time.Time
	Revoked   bool
}

type repository struct {
	db *database.DB
}

func newRepository(db *database.DB) *repository {
	return &repository{db: db}
}

func (r *repository) queries() *sqlcgen.Queries { return sqlcgen.New(r.db.Pool) }

// userIDByGitHub returns the user linked to a GitHub account id, or errNoAccount.
func (r *repository) userIDByGitHub(ctx context.Context, githubID string) (uuid.UUID, error) {
	acc, err := r.queries().GetOAuthAccount(ctx, sqlcgen.GetOAuthAccountParams{
		Provider:       "github",
		ProviderUserID: githubID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, errNoAccount
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("get oauth account: %w", err)
	}
	return acc.UserID, nil
}

// linkGitHubTx records the oauth account inside the caller's transaction, so it
// commits together with the user row.
func (r *repository) linkGitHubTx(ctx context.Context, tx pgx.Tx, userID uuid.UUID, githubID string) error {
	_, err := r.queries().WithTx(tx).CreateOAuthAccount(ctx, sqlcgen.CreateOAuthAccountParams{
		ID:             uuid.Must(uuid.NewV7()),
		UserID:         userID,
		Provider:       "github",
		ProviderUserID: githubID,
	})
	if err != nil {
		return fmt.Errorf("create oauth account: %w", err)
	}
	return nil
}

func (r *repository) createRefreshToken(ctx context.Context, p sqlcgen.CreateRefreshTokenParams) error {
	if _, err := r.queries().CreateRefreshToken(ctx, p); err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *repository) refreshTokenByHash(ctx context.Context, hash []byte) (storedToken, error) {
	row, err := r.queries().GetRefreshTokenByHash(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return storedToken{}, errNoToken
	}
	if err != nil {
		return storedToken{}, fmt.Errorf("get refresh token: %w", err)
	}
	return storedToken{
		ID:        row.ID,
		UserID:    row.UserID,
		FamilyID:  row.FamilyID,
		ExpiresAt: row.ExpiresAt,
		Revoked:   row.RevokedAt.Valid,
	}, nil
}

func (r *repository) revokeToken(ctx context.Context, id uuid.UUID) error {
	if err := r.queries().RevokeRefreshToken(ctx, id); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *repository) revokeFamily(ctx context.Context, familyID uuid.UUID) error {
	if err := r.queries().RevokeRefreshTokenFamily(ctx, familyID); err != nil {
		return fmt.Errorf("revoke token family: %w", err)
	}
	return nil
}

func (r *repository) revokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if err := r.queries().RevokeAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	return nil
}

// errNoToken signals an unknown refresh token hash.
var errNoToken = errors.New("refresh token not found")
