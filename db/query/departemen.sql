-- name: GetDepartemenByID :one
SELECT * FROM DEPARTEMEN
WHERE id_departemen = $1 LIMIT 1;

-- name: ListDepartemen :many
SELECT * FROM DEPARTEMEN
ORDER BY nama_departemen ASC;

-- name: CreateDepartemen :one
INSERT INTO DEPARTEMEN (nama_departemen)
VALUES ($1)
RETURNING *;

-- name: UpdateDepartemen :one
UPDATE DEPARTEMEN
SET nama_departemen = $2
WHERE id_departemen = $1
RETURNING *;

-- name: DeleteDepartemen :execrows
DELETE FROM DEPARTEMEN
WHERE id_departemen = $1;
