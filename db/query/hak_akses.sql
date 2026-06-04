-- name: GetHakAksesByID :one
SELECT * FROM HAK_AKSES
WHERE id_hak_akses = $1 LIMIT 1;

-- name: ListHakAkses :many
SELECT *
FROM HAK_AKSES
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_halaman ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    kode_permission ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    domain_permission ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    aksi_permission ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_hak_akses' AND NOT sqlc.arg(sort_desc)::bool THEN id_hak_akses END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_hak_akses' AND sqlc.arg(sort_desc)::bool THEN id_hak_akses END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'kode_permission' AND NOT sqlc.arg(sort_desc)::bool THEN kode_permission END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'kode_permission' AND sqlc.arg(sort_desc)::bool THEN kode_permission END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_halaman' AND NOT sqlc.arg(sort_desc)::bool THEN nama_halaman END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_halaman' AND sqlc.arg(sort_desc)::bool THEN nama_halaman END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'domain_permission' AND NOT sqlc.arg(sort_desc)::bool THEN domain_permission END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'domain_permission' AND sqlc.arg(sort_desc)::bool THEN domain_permission END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'aksi_permission' AND NOT sqlc.arg(sort_desc)::bool THEN aksi_permission END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'aksi_permission' AND sqlc.arg(sort_desc)::bool THEN aksi_permission END DESC,
    kode_permission ASC,
    id_hak_akses ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountHakAkses :one
SELECT COUNT(*)
FROM HAK_AKSES
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_halaman ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    kode_permission ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    domain_permission ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    aksi_permission ILIKE '%' || sqlc.arg(search_term)::text || '%'
);

-- name: CreateHakAkses :one
INSERT INTO HAK_AKSES (kode_permission, nama_halaman, deskripsi, domain_permission, aksi_permission)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateHakAkses :one
UPDATE HAK_AKSES
SET kode_permission = $2,
    nama_halaman = $3,
    deskripsi = $4,
    domain_permission = $5,
    aksi_permission = $6
WHERE id_hak_akses = $1
RETURNING *;

-- name: DeleteHakAkses :execrows
DELETE FROM HAK_AKSES
WHERE id_hak_akses = $1;
