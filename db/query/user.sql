-- name: GetUserByUsername :one
SELECT u.id_user, u.username, u.password, u.id_role, r.nama_role, u.id_departemen, u.id_mitra, u.status, u.must_change_password, u.password_changed_at, u.created_at
FROM USERS u
JOIN ROLES r ON r.id_role = u.id_role
WHERE u.username = $1 LIMIT 1;

-- name: GetUserPermissions :many
SELECT h.KODE_PERMISSION
FROM HAK_AKSES h
JOIN ROLE_HAK_AKSES rha ON h.ID_HAK_AKSES = rha.ID_HAK_AKSES
JOIN USERS u ON u.ID_ROLE = rha.ID_ROLE
WHERE u.ID_USER = $1
UNION
SELECT h.KODE_PERMISSION
FROM HAK_AKSES h
JOIN USER_AKSES ua ON h.ID_HAK_AKSES = ua.ID_HAK_AKSES
WHERE ua.ID_USER = $1;

-- name: GetUserPermissionIDs :many
SELECT id_hak_akses
FROM USER_AKSES
WHERE id_user = $1;

-- name: CreateUser :one
INSERT INTO USERS (
    username, password, id_role, id_departemen, id_mitra, status, must_change_password, created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING id_user, username, id_role, id_departemen, id_mitra, status, must_change_password, created_at;

-- name: CreateUserAkses :exec
INSERT INTO USER_AKSES (
    id_user, id_hak_akses
) VALUES (
    $1, $2
);

-- name: DeleteUserAksesByUserID :execrows
DELETE FROM USER_AKSES
WHERE id_user = $1;

-- name: ListUsers :many
SELECT u.id_user, u.username, u.status, u.id_role, r.nama_role, u.id_departemen, u.id_mitra, d.nama_departemen, m.nama_perusahaan, u.must_change_password, u.created_at
FROM USERS u
JOIN ROLES r ON r.id_role = u.id_role
LEFT JOIN DEPARTEMEN d ON u.id_departemen = d.id_departemen
LEFT JOIN MITRA m ON u.id_mitra = m.id_mitra
WHERE (
    sqlc.arg(search_term)::text = '' OR
    u.username ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    u.status ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    r.nama_role ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    COALESCE(d.nama_departemen, '') ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    COALESCE(m.nama_perusahaan, '') ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN u.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN u.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_user' AND NOT sqlc.arg(sort_desc)::bool THEN u.id_user END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_user' AND sqlc.arg(sort_desc)::bool THEN u.id_user END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'username' AND NOT sqlc.arg(sort_desc)::bool THEN u.username END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'username' AND sqlc.arg(sort_desc)::bool THEN u.username END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'status' AND NOT sqlc.arg(sort_desc)::bool THEN u.status END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'status' AND sqlc.arg(sort_desc)::bool THEN u.status END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_role' AND NOT sqlc.arg(sort_desc)::bool THEN u.id_role END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_role' AND sqlc.arg(sort_desc)::bool THEN u.id_role END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_role' AND NOT sqlc.arg(sort_desc)::bool THEN r.nama_role END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_role' AND sqlc.arg(sort_desc)::bool THEN r.nama_role END DESC,
    u.id_user ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsers :one
SELECT COUNT(*)
FROM USERS u
JOIN ROLES r ON r.id_role = u.id_role
LEFT JOIN DEPARTEMEN d ON u.id_departemen = d.id_departemen
LEFT JOIN MITRA m ON u.id_mitra = m.id_mitra
WHERE (
    sqlc.arg(search_term)::text = '' OR
    u.username ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    u.status ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    r.nama_role ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    COALESCE(d.nama_departemen, '') ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    COALESCE(m.nama_perusahaan, '') ILIKE '%' || sqlc.arg(search_term)::text || '%'
);

-- name: GetUserByID :one
SELECT u.id_user, u.username, u.status, u.id_role, r.nama_role, u.id_departemen, u.id_mitra, d.nama_departemen, m.nama_perusahaan, u.must_change_password, u.password_changed_at, u.created_at
FROM USERS u
JOIN ROLES r ON r.id_role = u.id_role
LEFT JOIN DEPARTEMEN d ON u.id_departemen = d.id_departemen
LEFT JOIN MITRA m ON u.id_mitra = m.id_mitra
WHERE u.id_user = $1 LIMIT 1;

-- name: UpdateUser :one
UPDATE USERS
SET username = $2,
    password = $3,
    id_role = $4,
    id_departemen = $5,
    id_mitra = $6,
    status = $7,
    must_change_password = $8,
    password_changed_at = $9,
    updated_by = $10
WHERE id_user = $1
RETURNING id_user, username, id_role, id_departemen, id_mitra, status, must_change_password, created_at;

-- name: UpdateUserStatus :one
UPDATE USERS
SET status = $2
WHERE id_user = $1
RETURNING id_user, username, status;

-- name: UpdateUserPasswordForChange :execrows
UPDATE USERS
SET password = $2,
    must_change_password = FALSE,
    password_changed_at = NOW(),
    updated_by = $3
WHERE id_user = $1;

-- name: ResetUserPasswordTemporary :execrows
UPDATE USERS
SET password = $2,
    must_change_password = TRUE,
    password_changed_at = NULL,
    updated_by = $3
WHERE id_user = $1;

-- name: DeleteUser :execrows
DELETE FROM USERS
WHERE id_user = $1;

