-- name: ListRoles :many
SELECT
    r.id_role,
    r.nama_role,
    r.created_at,
    COUNT(*) OVER() AS total_count
FROM ROLES r
WHERE (
    sqlc.arg(search_term)::text = '' OR
    r.nama_role ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN r.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN r.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_role' AND NOT sqlc.arg(sort_desc)::bool THEN r.id_role END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_role' AND sqlc.arg(sort_desc)::bool THEN r.id_role END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_role' AND NOT sqlc.arg(sort_desc)::bool THEN r.nama_role END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_role' AND sqlc.arg(sort_desc)::bool THEN r.nama_role END DESC,
    r.id_role ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetRoleByID :one
SELECT id_role, nama_role, created_at
FROM ROLES
WHERE id_role = $1 LIMIT 1;

-- name: GetRoleByName :one
SELECT id_role, nama_role, created_at
FROM ROLES
WHERE nama_role = $1 LIMIT 1;

-- name: CreateRole :one
INSERT INTO ROLES (
    nama_role
) VALUES (
    $1
)
RETURNING id_role, nama_role, created_at;

-- name: UpdateRole :one
UPDATE ROLES
SET nama_role = $2
WHERE id_role = $1
RETURNING id_role, nama_role, created_at;

-- name: DeleteRole :execrows
DELETE FROM ROLES
WHERE id_role = $1;

-- name: ListRolePermissionIDs :many
SELECT rha.id_hak_akses
FROM ROLE_HAK_AKSES rha
WHERE rha.id_role = $1
ORDER BY rha.id_hak_akses ASC;

-- name: ListRolePermissions :many
SELECT h.kode_permission
FROM ROLE_HAK_AKSES rha
JOIN HAK_AKSES h ON h.id_hak_akses = rha.id_hak_akses
WHERE rha.id_role = $1
ORDER BY h.kode_permission ASC;

-- name: CreateRoleHakAkses :exec
INSERT INTO ROLE_HAK_AKSES (
    id_role,
    id_hak_akses
) VALUES (
    $1,
    $2
)
ON CONFLICT (id_role, id_hak_akses) DO NOTHING;

-- name: DeleteRoleHakAksesByRoleID :execrows
DELETE FROM ROLE_HAK_AKSES
WHERE id_role = $1;

-- name: UpdateUserRole :one
UPDATE USERS
SET id_role = $2
WHERE id_user = $1
RETURNING id_user, username, status, id_role, id_departemen, id_mitra, created_at;
