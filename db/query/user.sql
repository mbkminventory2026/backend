-- name: GetUserByUsername :one
SELECT id_user, username, password, is_manager, id_departemen, id_mitra, created_at
FROM USERS
WHERE username = $1 LIMIT 1;

-- name: GetUserPermissions :many
SELECT h.NAMA_HALAMAN
FROM HAK_AKSES h
JOIN USER_AKSES ua ON h.ID_HAK_AKSES = ua.ID_HAK_AKSES
WHERE ua.ID_USER = $1;

-- name: CreateUser :one
INSERT INTO USERS (
    username, password, is_manager, id_departemen, id_mitra
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING id_user, username, is_manager, id_departemen, id_mitra, created_at;

-- name: CreateUserAkses :exec
INSERT INTO USER_AKSES (
    id_user, id_hak_akses
) VALUES (
    $1, $2
);

-- name: ListUsers :many
SELECT u.id_user, u.username, u.is_manager, d.nama_departemen, m.nama_perusahaan, u.created_at
FROM USERS u
LEFT JOIN DEPARTEMEN d ON u.id_departemen = d.id_departemen
LEFT JOIN MITRA m ON u.id_mitra = m.id_mitra
ORDER BY u.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetUserByID :one
SELECT u.id_user, u.username, u.is_manager, u.id_departemen, u.id_mitra, d.nama_departemen, m.nama_perusahaan, u.created_at
FROM USERS u
LEFT JOIN DEPARTEMEN d ON u.id_departemen = d.id_departemen
LEFT JOIN MITRA m ON u.id_mitra = m.id_mitra
WHERE u.id_user = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE USERS
SET username = $2,
    password = $3,
    is_manager = $4,
    id_departemen = $5,
    id_mitra = $6
WHERE id_user = $1
RETURNING id_user, username, is_manager, id_departemen, id_mitra, created_at;

-- name: DeleteUser :execrows
DELETE FROM USERS
WHERE id_user = $1;
