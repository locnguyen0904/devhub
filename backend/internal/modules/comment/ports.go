package comment

import (
	"context"

	"github.com/google/uuid"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
)

// authorFinder resolves comment authors for display. Declared here so comment
// depends on this narrow contract, not on user's repository.
type authorFinder interface {
	FindBrief(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]user.Brief, error)
}

// markdownRenderer renders and sanitizes a comment body. Comments allow markdown
// too, so the same render-then-sanitize rule as posts applies.
type markdownRenderer interface {
	Render(source string) (string, error)
}
