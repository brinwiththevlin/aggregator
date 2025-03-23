-- name: CreateFeedFollow :one
WITH inserted AS (
INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
        VALUES ($1, $2, $3, $4, $5)
    RETURNING
        *)
    SELECT
        inserted.*,
        f.name AS feed_name,
        u.name AS user_name
    FROM
        inserted
        INNER JOIN feeds f ON f.id = inserted.feed_id
        INNER JOIN users u ON u.id = inserted.user_id;

-- name: GetFeedFollowForUser :many
SELECT
    f.name AS feed_name,
    u.name AS useer_name
FROM
    feed_follows o
    LEFT JOIN users u ON o.user_id = u.id
    LEFT JOIN feeds f ON o.feed_id = f.id
WHERE
    u.id = $1;

-- name: DeleteFeedFollow :exec
Delete from feed_follows
where user_id = $1 and feed_id = $2;

