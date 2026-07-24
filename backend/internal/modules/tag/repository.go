package tag

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

// upsertTx creates the tag or returns the existing one, within the caller's tx.
func (r *repository) upsertTx(ctx context.Context, tx pgx.Tx, name string) (Tag, error) {
	row, err := r.queries().WithTx(tx).UpsertTag(ctx, sqlcgen.UpsertTagParams{
		ID:   uuid.Must(uuid.NewV7()),
		Name: name,
	})
	if err != nil {
		return Tag{}, fmt.Errorf("upsert tag %q: %w", name, err)
	}
	return Tag{ID: row.ID, Name: row.Name, ColorKey: row.ColorKey}, nil
}

func (r *repository) attachTx(ctx context.Context, tx pgx.Tx, postID, tagID uuid.UUID, position int) error {
	err := r.queries().WithTx(tx).AttachTagTx(ctx, sqlcgen.AttachTagTxParams{
		PostID:   postID,
		TagID:    tagID,
		Position: int16(position),
	})
	if err != nil {
		return fmt.Errorf("attach tag: %w", err)
	}
	return nil
}

func (r *repository) incrementCountsTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) error {
	if err := r.queries().WithTx(tx).IncrementTagCounts(ctx, ids); err != nil {
		return fmt.Errorf("increment tag counts: %w", err)
	}
	return nil
}

func (r *repository) search(ctx context.Context, query string, limit int32) ([]Tag, error) {
	// sqlc infers the concatenated param as nullable, hence the pointer.
	rows, err := r.queries().SearchTags(ctx, sqlcgen.SearchTagsParams{Query: &query, Lim: limit})
	if err != nil {
		return nil, fmt.Errorf("search tags: %w", err)
	}
	return toTagsFromModels(rows), nil
}

func (r *repository) popular(ctx context.Context, limit int32) ([]Tag, error) {
	rows, err := r.queries().PopularTags(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("popular tags: %w", err)
	}
	return toTagsFromModels(rows), nil
}

// forPosts returns tags grouped by post id, preserving each post's tag order.
func (r *repository) forPosts(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]Tag, error) {
	rows, err := r.queries().TagsForPosts(ctx, postIDs)
	if err != nil {
		return nil, fmt.Errorf("tags for posts: %w", err)
	}
	byPost := make(map[uuid.UUID][]Tag, len(postIDs))
	for _, row := range rows {
		byPost[row.PostID] = append(byPost[row.PostID], Tag{
			ID: row.ID, Name: row.Name, ColorKey: row.ColorKey,
		})
	}
	return byPost, nil
}

func toTagsFromModels(rows []sqlcgen.Tag) []Tag {
	tags := make([]Tag, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, Tag{ID: row.ID, Name: row.Name, ColorKey: row.ColorKey})
	}
	return tags
}
