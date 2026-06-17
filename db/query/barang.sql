-- name: GetBarangByID :one
SELECT b.*, m.nama_perusahaan, j.nama_jenis_barang
FROM BARANG b
JOIN MITRA m ON b.id_mitra = m.id_mitra
JOIN JENIS_BARANG j ON b.id_jenis_barang = j.id_jenis_barang
WHERE b.id_barang = $1
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'BARANG' AND mdd.id_record = b.id_barang
  )
LIMIT 1;

-- name: ListBarang :many
SELECT b.*, m.nama_perusahaan, j.nama_jenis_barang
FROM BARANG b
JOIN MITRA m ON b.id_mitra = m.id_mitra
JOIN JENIS_BARANG j ON b.id_jenis_barang = j.id_jenis_barang
WHERE (
    sqlc.arg(search_term)::text = '' OR
    b.nama_barang ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    b.kode ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    j.nama_jenis_barang ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    m.nama_perusahaan ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    b.satuan ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    b.lokasi_rak ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'BARANG' AND mdd.id_record = b.id_barang
  )
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN b.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN b.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_barang' AND NOT sqlc.arg(sort_desc)::bool THEN b.id_barang END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_barang' AND sqlc.arg(sort_desc)::bool THEN b.id_barang END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'kode' AND NOT sqlc.arg(sort_desc)::bool THEN b.kode END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'kode' AND sqlc.arg(sort_desc)::bool THEN b.kode END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_barang' AND NOT sqlc.arg(sort_desc)::bool THEN b.nama_barang END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_barang' AND sqlc.arg(sort_desc)::bool THEN b.nama_barang END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_jenis_barang' AND NOT sqlc.arg(sort_desc)::bool THEN j.nama_jenis_barang END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_jenis_barang' AND sqlc.arg(sort_desc)::bool THEN j.nama_jenis_barang END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_perusahaan' AND NOT sqlc.arg(sort_desc)::bool THEN m.nama_perusahaan END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_perusahaan' AND sqlc.arg(sort_desc)::bool THEN m.nama_perusahaan END DESC,
    b.created_at DESC,
    b.id_barang DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountBarang :one
SELECT COUNT(*)
FROM BARANG b
JOIN MITRA m ON b.id_mitra = m.id_mitra
JOIN JENIS_BARANG j ON b.id_jenis_barang = j.id_jenis_barang
WHERE (
    sqlc.arg(search_term)::text = '' OR
    b.nama_barang ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    b.kode ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    j.nama_jenis_barang ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    m.nama_perusahaan ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    b.satuan ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    b.lokasi_rak ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
  AND NOT EXISTS (
      SELECT 1 FROM MASTER_DATA_DELETED mdd
      WHERE mdd.nama_tabel = 'BARANG' AND mdd.id_record = b.id_barang
  );

-- name: CreateBarang :one
INSERT INTO BARANG (
    nama_barang, kode, id_jenis_barang, id_mitra, satuan, lokasi_rak, stok_minimum
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: UpdateBarang :one
UPDATE BARANG
SET nama_barang = $2, kode = $3, id_jenis_barang = $4, id_mitra = $5, satuan = $6, lokasi_rak = $7, stok_minimum = $8
WHERE id_barang = $1
RETURNING *;

-- name: DeleteBarang :execrows
DELETE FROM BARANG
WHERE id_barang = $1;
