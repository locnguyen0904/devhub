-- name: GetOAuthAccount :one
SELECT * FROM oauth_accounts
WHERE provider = @provider AND provider_user_id = @provider_user_id;

-- name: CreateOAuthAccount :one
INSERT INTO oauth_accounts (id, user_id, provider, provider_user_id, scopes)
VALUES (@id, @user_id, @provider, @provider_user_id, @scopes)
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (id, user_id, token_hash, family_id, expires_at, user_agent, ip)
VALUES (@id, @user_id, @token_hash, @family_id, @expires_at, @user_agent, @ip)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = @token_hash;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE id = @id AND revoked_at IS NULL;

-- name: RevokeRefreshTokenFamily :exec
-- Revokes every live token in a family at once. Used both on normal logout-all
-- and, critically, when a revoked token is replayed — the whole family is
-- burned because replay means the token was stolen.
UPDATE refresh_tokens SET revoked_at = now()
WHERE family_id = @family_id AND revoked_at IS NULL;

-- name: RevokeAllUserTokens :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE user_id = @user_id AND revoked_at IS NULL;
