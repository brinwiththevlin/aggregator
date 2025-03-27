-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING
    *;

-- name: GetPostsForUser :many
SELECT
    p.title,
    p.url,
    p.description,
    p.published_at
FROM
    posts p
    LEFT JOIN feeds f ON p.feed_id = f.id
    LEFT JOIN users u ON f.user_id = u.id
WHERE
    u.id = $1
ORDER BY
    p.created_at DESC
LIMIT $2;

