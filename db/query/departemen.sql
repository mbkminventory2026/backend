-- name: GetDepartemenByID :one
SELECT * FROM DEPARTEMEN
WHERE id_departemen = $1
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'DEPARTEMEN' AND mdd.id_record = DEPARTEMEN.id_departemen
  )
LIMIT 1;

-- name: ListDepartemen :many
SELECT *
FROM DEPARTEMEN
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_departemen ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'DEPARTEMEN' AND mdd.id_record = DEPARTEMEN.id_departemen
  )
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_departemen' AND NOT sqlc.arg(sort_desc)::bool THEN id_departemen END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_departemen' AND sqlc.arg(sort_desc)::bool THEN id_departemen END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_departemen' AND NOT sqlc.arg(sort_desc)::bool THEN nama_departemen END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_departemen' AND sqlc.arg(sort_desc)::bool THEN nama_departemen END DESC,
    nama_departemen ASC,
    id_departemen ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountDepartemen :one
SELECT COUNT(*)
FROM DEPARTEMEN
WHERE (
    sqlc.arg(search_term)::text = '' OR
    nama_departemen ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'DEPARTEMEN' AND mdd.id_record = DEPARTEMEN.id_departemen
  );

-- name: CreateDepartemen :one
INSERT INTO DEPARTEMEN (nama_departemen)
VALUES ($1)
RETURNING *;

-- name: UpdateDepartemen :one
UPDATE DEPARTEMEN
SET nama_departemen = $2
WHERE id_departemen = $1
RETURNING *;

-- name: DeleteDepartemen :execrows
DELETE FROM DEPARTEMEN
WHERE id_departemen = $1;
