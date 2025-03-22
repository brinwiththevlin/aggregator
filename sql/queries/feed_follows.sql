-- CreateFeedFollow :one
WITH inserted AS (
    INSERT INTO feeds (id, created_at, updated_at, user_id, feed_id)
        VALUES ($1, $2, $3, $4, $5)
    RETURNING
        *
)
SELECT
    inserted.*,
    f.name,
    u.name
FROM
    inserted
    INNER JOIN feeds f ON f.id = inserted.feed_id
    INNER JOIN users u ON u.id = inserted.user_id;




