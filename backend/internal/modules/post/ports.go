package post

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/modules/tag"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
)

// authorFinder is what post needs from the user module to show authorship. It
// resolves many authors at once so a feed does not issue one query per post.
type authorFinder interface {
	FindBrief(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]user.Brief, error)
}

// tagLinker is what post needs from the tag module. EnsureAndAttachTx runs
// inside post's create transaction so a post and its tags commit together.
type tagLinker interface {
	EnsureAndAttachTx(ctx context.Context, tx pgx.Tx, postID uuid.UUID, names []string) ([]tag.Tag, error)
	TagsForPosts(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]tag.Tag, error)
}

// markdownRenderer renders and sanitizes post bodies. Declared here so the
// service depends on the behaviour, not the concrete platform package.
type markdownRenderer interface {
	Render(source string) (string, error)
}
