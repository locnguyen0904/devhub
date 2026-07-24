package health

import (
	"context"
	"fmt"

	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
)

type repository struct {
	q *sqlcgen.Queries
}

func newRepository(db *database.DB) *repository {
	return &repository{q: sqlcgen.New(db.Pool)}
}

// Ping runs a real query rather than checking the pool, so a database that
// accepts connections but cannot serve queries still reports as unhealthy.
func (r *repository) Ping(ctx context.Context) error {
	if _, err := r.q.PingDatabase(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}
