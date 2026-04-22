-- name: GetUserByUsername :one
SELECT id_user, username, password, karyawan, created_at
FROM "USER"
WHERE username = $1 LIMIT 1;
