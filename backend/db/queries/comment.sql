-- name: CreateCommentTx :one
INSERT INTO comments (id, post_id, author_id, parent_id, body_markdown, body_html, depth)
VALUES (@id, @post_id, @author_id, @parent_id, @body_markdown, @body_html, @depth)
RETURNING *;

-- name: GetCommentByID :one
SELECT * FROM comments WHERE id = @id;

-- name: ListCommentsForPost :many
-- The whole tree in one query: two levels only, so the service groups replies
-- under their parents in Go. Soft-deleted rows are kept so a deleted comment
-- with replies still renders as a placeholder.
SELECT * FROM comments
WHERE post_id = @post_id
ORDER BY created_at ASC;

-- name: UpdateCommentTx :one
UPDATE comments SET body_markdown = @body_markdown, body_html = @body_html, updated_at = now()
WHERE id = @id AND author_id = @author_id AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCommentTx :one
UPDATE comments SET deleted_at = now()
WHERE id = @id AND author_id = @author_id AND deleted_at IS NULL
RETURNING id;

-- name: IncrementPostCommentCount :exec
UPDATE posts SET comment_count = comment_count + @delta WHERE id = @post_id;
