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