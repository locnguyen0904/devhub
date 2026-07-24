-- name: UpsertTag :one
-- Create the tag or return the existing one. ON CONFLICT DO UPDATE (rather than
-- DO NOTHING) so RETURNING always yields a row even when the tag already exists.
INSERT INTO tags (id, name)
VALUES (@id, @name)
ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: SearchTags :many
-- Prefix/fuzzy autocomplete over tag names, most-used first.
SELECT * FROM tags
WHERE name ILIKE @query || '%'
ORDER BY post_count DESC
LIMIT @lim;

-- name: PopularTags :many
SELECT * FROM tags ORDER BY post_count DESC LIMIT @lim;

-- name: AttachTagTx :exec
INSERT INTO post_tags (post_id, tag_id, position)
VALUES (@post_id, @tag_id, @position)
ON CONFLICT DO NOTHING;

-- name: TagsForPost :many
SELECT t.* FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
WHERE pt.post_id = @post_id
ORDER BY pt.position;

-- name: IncrementTagCounts :exec
UPDATE tags SET post_count = post_count + 1 WHERE id = ANY(@ids::uuid[]);

-- name: TagsForPosts :many
-- Batch variant of TagsForPost so a feed resolves tags for every post at once.
SELECT pt.post_id, t.id, t.name, t.color_key
FROM tags t
JOIN post_tags pt ON pt.tag_id = t.id
WHERE pt.post_id = ANY(@post_ids::uuid[])
ORDER BY pt.position;
