-- name: CreateFeedFollow :one
WITH insert_op AS (

    INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
    VALUES (
        $1,
        $2,
        $3,
        $4,
        $5
    )
    RETURNING *

)
SELECT
    insert_op.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM insert_op
JOIN feeds ON insert_op.feed_id = feeds.id
JOIN users ON insert_op.user_id = users.id;

-- name: GetFeedFollowsForUser :many
SELECT
    ff.id,
    ff.created_at,
    ff.updated_at,
    ff.user_id,
    ff.feed_id,
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows ff
JOIN feeds ON ff.feed_id = feeds.id
JOIN users ON ff.user_id = users.id
WHERE ff.user_id = $1
ORDER BY ff.created_at DESC;

-- name: DeleteFeedFollow :exec
DELETE FROM feed_follows ff
WHERE ff.user_id = $1
  AND ff.feed_id = (SELECT feed_id FROM feeds WHERE url = $2);

