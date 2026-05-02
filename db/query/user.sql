-- name: GetUserByUsername :one
SELECT id_user, username, password, is_manager, id_departemen, id_mitra, created_at
FROM USERS
WHERE username = $1 LIMIT 1;

-- name: GetUserPermissions :many
SELECT h.NAMA_HALAMAN
FROM HAK_AKSES h
JOIN USER_AKSES ua ON h.ID_HAK_AKSES = ua.ID_HAK_AKSES
WHERE ua.ID_USER = $1;
