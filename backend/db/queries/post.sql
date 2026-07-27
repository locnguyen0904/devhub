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

-- name: ListHotFeed :many
-- Hacker News style ranking (docs/02-data-model.md §8): comments weigh 2x
-- reactions, and the age penalty pushes older posts down. Bounded to 7 days so
-- the scan stays small; the result is cached in Redis, so this runs rarely.
-- The score is only an ORDER BY key, never selected, to avoid a float-to-int
-- scan mismatch.
SELECT sqlc.embed(p) FROM posts p
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND p.published_at > now() - INTERVAL '7 days'
ORDER BY (p.reaction_count + 2 * p.comment_count + 1)
    / POWER(EXTRACT(EPOCH FROM (now() - p.published_at)) / 3600 + 2, 1.5) DESC
LIMIT @lim;

-- name: ListFeedByTag :many
SELECT sqlc.embed(p) FROM posts p
JOIN post_tags pt ON pt.post_id = p.id
JOIN tags t ON t.id = pt.tag_id
WHERE t.name = @tag AND p.status = 'published' AND p.deleted_at IS NULL
  AND (@use_cursor::bool = false OR (p.published_at, p.id) < (@cursor_at::timestamptz, @cursor_id::uuid))
ORDER BY p.published_at DESC, p.id DESC
LIMIT @lim;

-- name: SearchPosts :many
-- Full-text search over the generated search_vector. 'simple' config matches
-- how the vector is built (no English stemmer to mangle Vietnamese). The
-- headline is a snippet of the body with the matched terms marked.
SELECT sqlc.embed(p),
    ts_headline('simple', p.body_markdown, query, 'MaxWords=30, MinWords=15, MaxFragments=1') AS headline
FROM posts p, websearch_to_tsquery('simple', @q) query
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND p.search_vector @@ query
ORDER BY ts_rank(p.search_vector, query) DESC
LIMIT @lim;

-- name: AddViewCounts :exec
-- Apply many buffered view deltas in one statement. Rows for posts that no
-- longer exist simply match nothing.
UPDATE posts SET view_count = view_count + d.delta
FROM (SELECT unnest(@ids::uuid[]) AS id, unnest(@deltas::bigint[]) AS delta) d
WHERE posts.id = d.id;

-- name: ListMyPosts :many
SELECT * FROM posts
WHERE author_id = @author_id AND deleted_at IS NULL
  AND (@status_filter::text = 'all' OR status = @status_filter::text)
ORDER BY updated_at DESC
LIMIT @lim;
