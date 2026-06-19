-- name: GetWarnaByID :one
SELECT * FROM WARNA
WHERE id_warna = $1
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'WARNA' AND mdd.id_record = WARNA.id_warna
  )
LIMIT 1;

-- name: ListWarna :many
SELECT *
FROM WARNA
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_warna ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'WARNA' AND mdd.id_record = WARNA.id_warna
  )
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_warna' AND NOT sqlc.arg(sort_desc)::bool THEN id_warna END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_warna' AND sqlc.arg(sort_desc)::bool THEN id_warna END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_warna' AND NOT sqlc.arg(sort_desc)::bool THEN nama_warna END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_warna' AND sqlc.arg(sort_desc)::bool THEN nama_warna END DESC,
    nama_warna ASC,
    id_warna ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountWarna :one
SELECT COUNT(*)
FROM WARNA
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_warna ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'WARNA' AND mdd.id_record = WARNA.id_warna
  );

-- name: CreateWarna :one
INSERT INTO WARNA (nama_warna, kode_hex)
VALUES ($1, $2)
RETURNING *;

-- name: UpdateWarna :one
UPDATE WARNA
SET nama_warna = $2, kode_hex = $3
WHERE id_warna = $1
RETURNING *;

-- name: DeleteWarna :execrows
DELETE FROM WARNA
WHERE id_warna = $1;
