package reaction

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
)

type repository struct {
	db *database.DB
}

func newRepository(db *database.DB) *repository {
	return &repository{db: db}
}

func (r *repository) queries() *sqlcgen.Queries { return sqlcgen.New(r.db.Pool) }

// addReactionTx inserts a reaction and reports whether a new row was created, so
// the caller only bumps the count on a real change.
func (r *repository) addReactionTx(ctx context.Context, tx pgx.Tx, userID, postID uuid.UUID, kind string) (bool, error) {
	n, err := r.queries().WithTx(tx).AddReactionTx(ctx, sqlcgen.AddReactionTxParams{
		UserID: userID, PostID: postID, Kind: kind,
	})
	if err != nil {
		return false, fmt.Errorf("add reaction: %w", err)
	}
	return n > 0, nil
}

func (r *repository) removeReactionTx(ctx context.Context, tx pgx.Tx, userID, postID uuid.UUID, kind string) (bool, error) {
	n, err := r.queries().WithTx(tx).RemoveReactionTx(ctx, sqlcgen.RemoveReactionTxParams{
		UserID: userID, PostID: postID, Kind: kind,
	})
	if err != nil {
		return false, fmt.Errorf("remove reaction: %w", err)
	}
	return n > 0, nil
}

func (r *repository) adjustCountTx(ctx context.Context, tx pgx.Tx, postID uuid.UUID, delta int32) error {
	err := r.queries().WithTx(tx).AdjustPostReactionCount(ctx, sqlcgen.AdjustPostReactionCountParams{
		PostID: postID, Delta: delta,
	})
	if err != nil {
		return fmt.Errorf("adjust reaction count: %w", err)
	}
	return nil
}

func (r *repository) reactionCount(ctx context.Context, postID uuid.UUID) (int, error) {
	n, err := r.queries().GetPostReactionCount(ctx, postID)
	if err != nil {
		return 0, fmt.Errorf("get reaction count: %w", err)
	}
	return int(n), nil
}

func (r *repository) viewerReactions(ctx context.Context, userID, postID uuid.UUID) ([]string, error) {
	kinds, err := r.queries().ViewerReactions(ctx, sqlcgen.ViewerReactionsParams{UserID: userID, PostID: postID})
	if err != nil {
		return nil, fmt.Errorf("viewer reactions: %w", err)
	}
	return kinds, nil
}

func (r *repository) addBookmark(ctx context.Context, userID, postID uuid.UUID) error {
	if err := r.queries().AddBookmark(ctx, sqlcgen.AddBookmarkParams{UserID: userID, PostID: postID}); err != nil {
		return fmt.Errorf("add bookmark: %w", err)
	}
	return nil
}

func (r *repository) removeBookmark(ctx context.Context, userID, postID uuid.UUID) error {
	if err := r.queries().RemoveBookmark(ctx, sqlcgen.RemoveBookmarkParams{UserID: userID, PostID: postID}); err != nil {
		return fmt.Errorf("remove bookmark: %w", err)
	}
	return nil
}

func (r *repository) viewerBookmarked(ctx context.Context, userID, postID uuid.UUID) (bool, error) {
	ok, err := r.queries().ViewerBookmarked(ctx, sqlcgen.ViewerBookmarkedParams{UserID: userID, PostID: postID})
	if err != nil {
		return false, fmt.Errorf("viewer bookmarked: %w", err)
	}
	return ok, nil
}

func (r *repository) bookmarkedPostIDs(ctx context.Context, userID uuid.UUID, limit int32) ([]uuid.UUID, error) {
	ids, err := r.queries().ListBookmarkedPostIDs(ctx, sqlcgen.ListBookmarkedPostIDsParams{UserID: userID, Lim: limit})
	if err != nil {
		return nil, fmt.Errorf("list bookmarked ids: %w", err)
	}
	return ids, nil
}
