-- name: PingDatabase :one
-- A real round-trip through the sqlc path, so the status page proves the whole
-- Postgres -> repository -> service chain rather than just that the pool opened.
SELECT 1::int AS ok;
