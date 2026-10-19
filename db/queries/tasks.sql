-- name: CreateTask :one
INSERT INTO tasks (user_id, title, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTasksByUser :many
SELECT * FROM tasks
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetTask :one
SELECT * FROM tasks
WHERE id = $1 AND user_id = $2;

-- name: UpdateTask :one
UPDATE tasks
SET title       = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    done        = COALESCE(sqlc.narg('done'), done),
    updated_at  = now()
WHERE id = sqlc.arg('id') AND user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks
WHERE id = $1 AND user_id = $2;
