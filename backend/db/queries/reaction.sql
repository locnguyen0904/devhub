-- name: AddReactionTx :execrows
-- Idempotent: a repeated reaction of the same kind changes nothing. The rows
-- affected tells the service whether to bump the post's reaction count.
INSERT INTO reactions (user_id, post_id, kind)
VALUES (@user_id, @post_id, @kind)
ON CONFLICT DO NOTHING;

-- name: RemoveReactionTx :execrows
DELETE FROM reactions
WHERE user_id = @user_id AND post_id = @post_id AND kind = @kind;

-- name: AdjustPostReactionCount :exec
UPDATE posts SET reaction_count = reaction_count + @delta WHERE id = @post_id;

-- name: GetPostReactionCount :one
-- The reaction module maintains this denormalized counter, so it also reads it
-- back to return the fresh total after a toggle.
SELECT reaction_count FROM posts WHERE id = @id;

-- name: ViewerReactions :many
SELECT kind FROM reactions WHERE user_id = @user_id AND post_id = @post_id;

-- name: AddBookmark :exec
INSERT INTO bookmarks (user_id, post_id)
VALUES (@user_id, @post_id)
ON CONFLICT DO NOTHING;

-- name: RemoveBookmark :exec
DELETE FROM bookmarks WHERE user_id = @user_id AND post_id = @post_id;

-- name: ViewerBookmarked :one
SELECT EXISTS (SELECT 1 FROM bookmarks WHERE user_id = @user_id AND post_id = @post_id);

-- name: ListBookmarkedPostIDs :many
-- Ids of the user's saved posts, most-recently saved first. The post module
-- loads and enriches them, reusing its card logic. Bounded by limit; a per-user
-- bookmark list is small enough that a single page suffices for now.
SELECT p.id FROM posts p
JOIN bookmarks b ON b.post_id = p.id
WHERE b.user_id = @user_id AND p.deleted_at IS NULL
ORDER BY b.created_at DESC
LIMIT @lim;
