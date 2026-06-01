-- name: GetMitraByID :one
SELECT * FROM MITRA
WHERE id_mitra = $1 LIMIT 1;

-- name: ListMitra :many
SELECT *
FROM MITRA
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_perusahaan ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    tipe_perusahaan ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    email ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    no_telp ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_mitra' AND NOT sqlc.arg(sort_desc)::bool THEN id_mitra END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_mitra' AND sqlc.arg(sort_desc)::bool THEN id_mitra END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_perusahaan' AND NOT sqlc.arg(sort_desc)::bool THEN nama_perusahaan END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_perusahaan' AND sqlc.arg(sort_desc)::bool THEN nama_perusahaan END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'email' AND NOT sqlc.arg(sort_desc)::bool THEN email END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'email' AND sqlc.arg(sort_desc)::bool THEN email END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'no_telp' AND NOT sqlc.arg(sort_desc)::bool THEN no_telp END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'no_telp' AND sqlc.arg(sort_desc)::bool THEN no_telp END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tipe_perusahaan' AND NOT sqlc.arg(sort_desc)::bool THEN tipe_perusahaan END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tipe_perusahaan' AND sqlc.arg(sort_desc)::bool THEN tipe_perusahaan END DESC,
    nama_perusahaan ASC,
    id_mitra ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountMitra :one
SELECT COUNT(*)
FROM MITRA
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_perusahaan ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    tipe_perusahaan ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    email ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    no_telp ILIKE '%' || sqlc.arg(search_term)::text || '%'
);

-- name: CreateMitra :one
INSERT INTO MITRA (
    nama_perusahaan, tipe_perusahaan, email, no_telp, alamat, kota, kode_pos
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: UpdateMitra :one
UPDATE MITRA
SET nama_perusahaan = $2, tipe_perusahaan = $3, email = $4, no_telp = $5, alamat = $6, kota = $7, kode_pos = $8
WHERE id_mitra = $1
RETURNING *;

-- name: DeleteMitra :execrows
DELETE FROM MITRA
WHERE id_mitra = $1;
