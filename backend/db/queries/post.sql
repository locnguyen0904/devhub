-- name: CreatePostTx :one
INSERT INTO posts (
    id, author_id, slug, title, subtitle, body_markdown, body_html,
    cover_image_url, status, reading_minutes, canonical_url
) VALUES (
    @id, @author_id, @slug, @title, @subtitle, @body_markdown, @body_html,
    @cover_image_url, 'draft', @reading_minutes, @canonical_url
)
RETURNING *;

-- name: GetPostByID :one
SELECT * FROM posts WHERE id = @id AND deleted_at IS NULL;

-- name: GetPublishedPostBySlug :one
SELECT p.* FROM posts p
JOIN users u ON u.id = p.author_id
WHERE u.username = @username AND p.slug = @slug
  AND p.status = 'published' AND p.deleted_at IS NULL;

-- name: SlugExistsForAuthor :one
SELECT EXISTS (
    SELECT 1 FROM posts
    WHERE author_id = @author_id AND slug = @slug AND deleted_at IS NULL
);

-- name: UpdatePost :one
-- COALESCE keeps every unset field: a PATCH that changes only the title leaves
-- the body untouched. Only the author can update, enforced in the WHERE clause.
UPDATE posts SET
    title           = COALESCE(sqlc.narg('title'), title),
    subtitle        = COALESCE(sqlc.narg('subtitle'), subtitle),
    body_markdown   = COALESCE(sqlc.narg('body_markdown'), body_markdown),
    body_html       = COALESCE(sqlc.narg('body_html'), body_html),
    cover_image_url = COALESCE(sqlc.narg('cover_image_url'), cover_image_url),
    reading_minutes = COALESCE(sqlc.narg('reading_minutes'), reading_minutes),
    updated_at      = now()
WHERE id = @id AND author_id = @author_id AND deleted_at IS NULL
RETURNING *;

-- name: PublishPost :one
UPDATE posts SET status = 'published', slug = @slug, published_at = now(), updated_at = now()
WHERE id = @id AND author_id = @author_id AND deleted_at IS NULL
RETURNING *;

-- name: UnpublishPost :one
UPDATE posts SET status = 'draft', updated_at = now()
WHERE id = @id AND author_id = @author_id AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeletePost :exec
UPDATE posts SET deleted_at = now()
WHERE id = @id AND author_id = @author_id AND deleted_at IS NULL;

-- name: ListPublishedFeed :many
-- Cursor pagination matching idx_posts_feed. The tuple comparison walks the
-- index directly; offset pagination is forbidden (docs/03-api.md §1).
SELECT * FROM posts
WHERE status = 'published' AND deleted_at IS NULL
  AND (@use_cursor::bool = false OR (published_at, id) < (@cursor_at::timestamptz, @cursor_id::uuid))
ORDER BY published_at DESC, id DESC
LIMIT @lim;

-- name: ListMyPosts :many
SELECT * FROM posts
WHERE author_id = @author_id AND deleted_at IS NULL
  AND (@status_filter::text = 'all' OR status = @status_filter::text)
ORDER BY updated_at DESC
LIMIT @lim;
