package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

type repository struct {
	db *database.DB
}

func newRepository(db *database.DB) *repository {
	return &repository{db: db}
}

func (r *repository) getByID(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := sqlcgen.New(r.db.Pool).GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, httpx.NotFound("User not found")
	}
	if err != nil {
		return User{}, fmt.Errorf("get user by id: %w", err)
	}
	return toDomain(row), nil
}

// createTx inserts a user inside the caller's transaction so it can commit
// together with the oauth account the auth module writes.
func (r *repository) createTx(ctx context.Context, tx pgx.Tx, p sqlcgen.CreateUserParams) (User, error) {
	row, err := sqlcgen.New(r.db.Pool).WithTx(tx).CreateUser(ctx, p)
	if err != nil {
		return User{}, err
	}
	return toDomain(row), nil
}

func (r *repository) findBrief(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Brief, error) {
	rows, err := sqlcgen.New(r.db.Pool).GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get users by ids: %w", err)
	}
	briefs := make(map[uuid.UUID]Brief, len(rows))
	for _, row := range rows {
		briefs[row.ID] = Brief{
			ID:          row.ID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
			AvatarURL:   row.AvatarUrl,
		}
	}
	return briefs, nil
}

func toDomain(row sqlcgen.User) User {
	return User{
		ID:             row.ID,
		Username:       row.Username,
		Email:          row.Email,
		DisplayName:    row.DisplayName,
		AvatarURL:      row.AvatarUrl,
		GitHubUsername: row.GithubUsername,
		Role:           row.Role,
		CreatedAt:      row.CreatedAt,
	}
}
