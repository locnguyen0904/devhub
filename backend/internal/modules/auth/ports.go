package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
)

// userService is what auth needs from the user module. Declared here, in the
// consumer, so auth depends on this narrow contract rather than user's
// repository. CreateFromGitHubTx takes a transaction because provisioning must
// write the user and the oauth account atomically, across both modules' tables.
type userService interface {
	GetByID(ctx context.Context, id uuid.UUID) (user.User, error)
	CreateFromGitHubTx(ctx context.Context, tx pgx.Tx, gh user.GitHubIdentity) (user.User, error)
}
