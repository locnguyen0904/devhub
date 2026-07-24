package post

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *repository) queries() *sqlcgen.Queries { return sqlcgen.New(r.db.Pool) }

func (r *repository) createTx(ctx context.Context, tx pgx.Tx, p sqlcgen.CreatePostTxParams) (Post, error) {
	row, err := r.queries().WithTx(tx).CreatePostTx(ctx, p)
	if err != nil {
		return Post{}, fmt.Errorf("create post: %w", err)
	}
	return toDomain(row), nil
}

func (r *repository) getByID(ctx context.Context, id uuid.UUID) (Post, error) {
	row, err := r.queries().GetPostByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, httpx.NotFound("Post not found")
	}
	if err != nil {
		return Post{}, fmt.Errorf("get post by id: %w", err)
	}
	return toDomain(row), nil
}

func (r *repository) getPublishedBySlug(ctx context.Context, username, slug string) (Post, error) {
	row, err := r.queries().GetPublishedPostBySlug(ctx, sqlcgen.GetPublishedPostBySlugParams{
		Username: username,
		Slug:     slug,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, httpx.NotFound("Post not found")
	}
	if err != nil {
		return Post{}, fmt.Errorf("get post by slug: %w", err)
	}
	return toDomain(row), nil
}

func (r *repository) slugTaken(ctx context.Context, authorID uuid.UUID, slug string) (bool, error) {
	taken, err := r.queries().SlugExistsForAuthor(ctx, sqlcgen.SlugExistsForAuthorParams{
		AuthorID: authorID,
		Slug:     slug,
	})
	if err != nil {
		return false, fmt.Errorf("check slug: %w", err)
	}
	return taken, nil
}

func (r *repository) update(ctx context.Context, p sqlcgen.UpdatePostParams) (Post, error) {
	row, err := r.queries().UpdatePost(ctx, p)
	if errors.Is(err, pgx.ErrNoRows) {
		// No row means the post is missing or not owned by this author. Both are
		// reported as not-found so a caller cannot probe which posts exist.
		return Post{}, httpx.NotFound("Post not found")
	}
	if err != nil {
		return Post{}, fmt.Errorf("update post: %w", err)
	}
	return toDomain(row), nil
}

func (r *repository) publish(ctx context.Context, id, authorID uuid.UUID, slug string) (Post, error) {
	row, err := r.queries().PublishPost(ctx, sqlcgen.PublishPostParams{ID: id, AuthorID: authorID, Slug: slug})
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, httpx.NotFound("Post not found")
	}
	if err != nil {
		return Post{}, fmt.Errorf("publish post: %w", err)
	}
	return toDomain(row), nil
}

func (r *repository) unpublish(ctx context.Context, id, authorID uuid.UUID) (Post, error) {
	row, err := r.queries().UnpublishPost(ctx, sqlcgen.UnpublishPostParams{ID: id, AuthorID: authorID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Post{}, httpx.NotFound("Post not found")
	}
	if err != nil {
		return Post{}, fmt.Errorf("unpublish post: %w", err)
	}
	return toDomain(row), nil
}

func (r *repository) softDelete(ctx context.Context, id, authorID uuid.UUID) error {
	if err := r.queries().SoftDeletePost(ctx, sqlcgen.SoftDeletePostParams{ID: id, AuthorID: authorID}); err != nil {
		return fmt.Errorf("delete post: %w", err)
	}
	return nil
}

func (r *repository) feed(ctx context.Context, cursor *feedCursor, limit int32) ([]Post, error) {
	params := sqlcgen.ListPublishedFeedParams{Lim: limit}
	if cursor != nil {
		params.UseCursor = true
		params.CursorAt = cursor.publishedAt
		params.CursorID = cursor.id
	}
	rows, err := r.queries().ListPublishedFeed(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list feed: %w", err)
	}
	return toDomains(rows), nil
}

func (r *repository) myPosts(ctx context.Context, authorID uuid.UUID, statusFilter string, limit int32) ([]Post, error) {
	rows, err := r.queries().ListMyPosts(ctx, sqlcgen.ListMyPostsParams{
		AuthorID:     authorID,
		StatusFilter: statusFilter,
		Lim:          limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list my posts: %w", err)
	}
	return toDomains(rows), nil
}

func toDomains(rows []sqlcgen.Post) []Post {
	posts := make([]Post, 0, len(rows))
	for _, row := range rows {
		posts = append(posts, toDomain(row))
	}
	return posts
}

func toDomain(row sqlcgen.Post) Post {
	p := Post{
		ID:             row.ID,
		AuthorID:       row.AuthorID,
		Slug:           row.Slug,
		Title:          row.Title,
		Subtitle:       row.Subtitle,
		BodyMarkdown:   row.BodyMarkdown,
		BodyHTML:       row.BodyHtml,
		CoverImageURL:  row.CoverImageUrl,
		Status:         row.Status,
		ReadingMinutes: int(row.ReadingMinutes),
		CommentCount:   int(row.CommentCount),
		ReactionCount:  int(row.ReactionCount),
		ViewCount:      row.ViewCount,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
	if row.PublishedAt.Valid {
		t := row.PublishedAt.Time
		p.PublishedAt = &t
	}
	return p
}

// feedCursor is the decoded position in the feed, matching idx_posts_feed.
type feedCursor struct {
	publishedAt time.Time
	id          uuid.UUID
}
