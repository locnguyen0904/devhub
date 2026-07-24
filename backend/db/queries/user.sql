-- name: GetUserByID :one
SELECT * FROM users WHERE id = @id;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = @username;

-- name: CreateUser :one
INSERT INTO users (id, username, email, display_name, avatar_url, github_username)
VALUES (@id, @username, @email, @display_name, @avatar_url, @github_username)
RETURNING *;

-- name: UsernameExists :one
SELECT EXISTS (SELECT 1 FROM users WHERE username = @username);
