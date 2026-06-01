-- name: GetHakAksesByID :one
SELECT * FROM HAK_AKSES
WHERE id_hak_akses = $1 LIMIT 1;

-- name: ListHakAkses :many
SELECT *
FROM HAK_AKSES
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_halaman ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_hak_akses' AND NOT sqlc.arg(sort_desc)::bool THEN id_hak_akses END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_hak_akses' AND sqlc.arg(sort_desc)::bool THEN id_hak_akses END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_halaman' AND NOT sqlc.arg(sort_desc)::bool THEN nama_halaman END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_halaman' AND sqlc.arg(sort_desc)::bool THEN nama_halaman END DESC,
    nama_halaman ASC,
    id_hak_akses ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountHakAkses :one
SELECT COUNT(*)
FROM HAK_AKSES
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_halaman ILIKE '%' || sqlc.arg(search_term)::text || '%'
);

-- name: CreateHakAkses :one
INSERT INTO HAK_AKSES (nama_halaman)
VALUES ($1)
RETURNING *;

-- name: UpdateHakAkses :one
UPDATE HAK_AKSES
SET nama_halaman = $2
WHERE id_hak_akses = $1
RETURNING *;

-- name: DeleteHakAkses :execrows
DELETE FROM HAK_AKSES
WHERE id_hak_akses = $1;
