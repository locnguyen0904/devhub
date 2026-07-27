package comment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *repository) queries() *sqlcgen.Queries { return sqlcgen.New(r.db.Pool) }

func (r *repository) createTx(ctx context.Context, tx pgx.Tx, c Comment) (Comment, error) {
	row, err := r.queries().WithTx(tx).CreateCommentTx(ctx, sqlcgen.CreateCommentTxParams{
		ID:           c.ID,
		PostID:       c.PostID,
		AuthorID:     c.AuthorID,
		ParentID:     toPgUUID(c.ParentID),
		BodyMarkdown: c.BodyMarkdown,
		BodyHtml:     c.BodyHTML,
		Depth:        int16(c.Depth),
	})
	if err != nil {
		return Comment{}, fmt.Errorf("create comment: %w", err)
	}
	return toDomain(row), nil
}

func (r *repository) getByID(ctx context.Context, id uuid.UUID) (Comment, error) {
	row, err := r.queries().GetCommentByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Comment{}, httpx.NotFound("Comment not found")
	}
	if err != nil {
		return Comment{}, fmt.Errorf("get comment: %w", err)
	}
	return toDomain(row), nil
}

func (r *repository) listForPost(ctx context.Context, postID uuid.UUID) ([]Comment, error) {
	rows, err := r.queries().ListCommentsForPost(ctx, postID)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	out := make([]Comment, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

func (r *repository) update(ctx context.Context, id, authorID uuid.UUID, markdown, html string) (Comment, error) {
	row, err := r.queries().UpdateCommentTx(ctx, sqlcgen.UpdateCommentTxParams{
		ID:           id,
		AuthorID:     authorID,
		BodyMarkdown: markdown,
		BodyHtml:     html,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Comment{}, httpx.NotFound("Comment not found")
	}
	if err != nil {
		return Comment{}, fmt.Errorf("update comment: %w", err)
	}
	return toDomain(row), nil
}

// softDeleteTx marks a comment deleted within the caller's transaction. It
// reports whether a row actually changed, so the caller only adjusts the post's
// comment count when a live comment was really removed.
func (r *repository) softDeleteTx(ctx context.Context, tx pgx.Tx, id, authorID uuid.UUID) (bool, error) {
	_, err := r.queries().WithTx(tx).SoftDeleteCommentTx(ctx, sqlcgen.SoftDeleteCommentTxParams{
		ID:       id,
		AuthorID: authorID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("soft delete comment: %w", err)
	}
	return true, nil
}

func (r *repository) adjustCountTx(ctx context.Context, tx pgx.Tx, postID uuid.UUID, delta int32) error {
	err := r.queries().WithTx(tx).IncrementPostCommentCount(ctx, sqlcgen.IncrementPostCommentCountParams{
		PostID: postID,
		Delta:  delta,
	})
	if err != nil {
		return fmt.Errorf("adjust comment count: %w", err)
	}
	return nil
}

func toDomain(row sqlcgen.Comment) Comment {
	c := Comment{
		ID:           row.ID,
		PostID:       row.PostID,
		AuthorID:     row.AuthorID,
		BodyMarkdown: row.BodyMarkdown,
		BodyHTML:     row.BodyHtml,
		Depth:        int(row.Depth),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		Deleted:      row.DeletedAt.Valid,
	}
	if row.ParentID.Valid {
		id := row.ParentID.Bytes
		pid := uuid.UUID(id)
		c.ParentID = &pid
	}
	return c
}

func toPgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}
