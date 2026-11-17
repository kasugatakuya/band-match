-- name: CreatePost :one
INSERT INTO posts (user_id, kind, title, body, prefecture, parts, genres, artists)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPost :one
SELECT sqlc.embed(p), u.display_name AS author_name, u.avatar_url AS author_avatar,
       u.birth_date AS author_birth_date, u.gender AS author_gender
FROM posts p JOIN users u ON u.id = p.user_id
WHERE p.id = $1;

-- name: ListPosts :many
SELECT sqlc.embed(p), u.display_name AS author_name, u.avatar_url AS author_avatar,
       u.birth_date AS author_birth_date, u.gender AS author_gender
FROM posts p JOIN users u ON u.id = p.user_id
WHERE p.status = 'open'
  AND (sqlc.narg(kind)::text IS NULL OR p.kind = sqlc.narg(kind))
  AND (sqlc.narg(prefecture)::text IS NULL OR p.prefecture = sqlc.narg(prefecture))
  AND (sqlc.narg(part)::text IS NULL OR sqlc.narg(part) = ANY (p.parts))
  AND (sqlc.narg(genre)::text IS NULL OR sqlc.narg(genre) = ANY (p.genres))
  AND (sqlc.narg(artist)::text IS NULL OR EXISTS (
        SELECT 1 FROM unnest(p.artists) AS a WHERE a ILIKE '%' || sqlc.narg(artist) || '%'))
  AND (sqlc.narg(q)::text IS NULL
        OR p.title ILIKE '%' || sqlc.narg(q) || '%'
        OR p.body ILIKE '%' || sqlc.narg(q) || '%')
  AND (sqlc.narg(before)::timestamptz IS NULL OR p.created_at < sqlc.narg(before))
ORDER BY p.created_at DESC
LIMIT sqlc.arg(lim);

-- name: ListPostsByUser :many
SELECT sqlc.embed(p), u.display_name AS author_name, u.avatar_url AS author_avatar,
       u.birth_date AS author_birth_date, u.gender AS author_gender
FROM posts p JOIN users u ON u.id = p.user_id
WHERE p.user_id = $1
  AND (sqlc.arg(include_closed)::boolean OR p.status = 'open')
ORDER BY p.created_at DESC
LIMIT 200;

-- name: UpdatePost :one
UPDATE posts
SET kind = $3, title = $4, body = $5, prefecture = $6, parts = $7, genres = $8, artists = $9,
    status = $10, updated_at = now()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: DeletePost :execrows
DELETE FROM posts WHERE id = $1 AND user_id = $2;

-- name: CountRecentPostsByUser :one
SELECT count(*) FROM posts WHERE user_id = $1 AND created_at > now() - interval '1 day';
