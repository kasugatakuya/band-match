-- name: CreateUser :one
INSERT INTO users (email, password_hash, google_sub, display_name, avatar_url, email_verified_at, birth_date, gender)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByGoogleSub :one
SELECT * FROM users WHERE google_sub = $1;

-- name: UpdateProfile :one
UPDATE users
SET display_name = $2, bio = $3, prefecture = $4, parts = $5, email_notifications = $6,
    birth_date = $7, gender = $8
WHERE id = $1
RETURNING *;

-- name: MarkEmailVerified :exec
UPDATE users SET email_verified_at = COALESCE(email_verified_at, now()) WHERE id = $1;

-- name: SetPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: ClearPassword :exec
UPDATE users SET password_hash = NULL WHERE id = $1;

-- name: LinkGoogle :one
UPDATE users
SET google_sub = $2,
    avatar_url = CASE WHEN avatar_url = '' THEN sqlc.arg(avatar_url)::text ELSE avatar_url END,
    email_verified_at = COALESCE(email_verified_at, now())
WHERE id = $1
RETURNING *;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3);

-- name: GetSessionUser :one
SELECT u.* FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < now();

-- name: CreateEmailToken :exec
INSERT INTO email_tokens (token_hash, user_id, kind, expires_at) VALUES ($1, $2, $3, $4);

-- name: ConsumeEmailToken :one
UPDATE email_tokens SET used_at = now()
WHERE token_hash = $1 AND kind = $2 AND used_at IS NULL AND expires_at > now()
RETURNING user_id;

-- name: DeleteUserEmailTokens :exec
DELETE FROM email_tokens WHERE user_id = $1 AND kind = $2;

-- name: DeleteAllUserEmailTokens :exec
DELETE FROM email_tokens WHERE user_id = $1;
