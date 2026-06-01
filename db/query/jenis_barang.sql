-- name: GetJenisBarangByID :one
SELECT * FROM JENIS_BARANG
WHERE id_jenis_barang = $1 LIMIT 1;

-- name: ListJenisBarang :many
SELECT *
FROM JENIS_BARANG
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_jenis_barang ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    kode ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_jenis_barang' AND NOT sqlc.arg(sort_desc)::bool THEN id_jenis_barang END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_jenis_barang' AND sqlc.arg(sort_desc)::bool THEN id_jenis_barang END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'kode' AND NOT sqlc.arg(sort_desc)::bool THEN kode END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'kode' AND sqlc.arg(sort_desc)::bool THEN kode END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_jenis_barang' AND NOT sqlc.arg(sort_desc)::bool THEN nama_jenis_barang END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_jenis_barang' AND sqlc.arg(sort_desc)::bool THEN nama_jenis_barang END DESC,
    nama_jenis_barang ASC,
    id_jenis_barang ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountJenisBarang :one
SELECT COUNT(*)
FROM JENIS_BARANG
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_jenis_barang ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    kode ILIKE '%' || sqlc.arg(search_term)::text || '%'
);

-- name: CreateJenisBarang :one
INSERT INTO JENIS_BARANG (nama_jenis_barang, kode)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateJenisBarang :one
UPDATE JENIS_BARANG
SET nama_jenis_barang = $2, kode = $3
WHERE id_jenis_barang = $1
RETURNING *;

-- name: DeleteJenisBarang :execrows
DELETE FROM JENIS_BARANG
WHERE id_jenis_barang = $1;
