-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: ListFeeds :many
SELECT 
    f.name,
    f.url,
    u.name AS user_name
FROM feeds f
LEFT JOIN users u ON f.user_id = u.id;

-- name: ListFeed :one
SELECT 
    f.id,
    f.name,
    f.url,
    f.user_id
FROM feeds f
WHERE f.url = $1;

-- name: UpdateFetchedFeed :exec
UPDATE feeds
SET updated_at = NOW(), last_fetched_at = NOW()
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;