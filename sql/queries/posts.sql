-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING *;

-- name: GetPostsForUser :many
SELECT 
p.*,
f.name as feed_name,
u.id as user_id
FROM posts p 
inner join feeds f on f.id = p.feed_id
inner join users u on u.id = f.user_id
WHERE u.id = $1
ORDER BY published_at DESC
LIMIT $2
;