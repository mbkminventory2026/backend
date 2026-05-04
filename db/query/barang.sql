-- name: GetBarangByID :one
SELECT b.*, m.nama_perusahaan, j.nama_jenis_barang
FROM BARANG b
JOIN MITRA m ON b.id_mitra = m.id_mitra
JOIN JENIS_BARANG j ON b.id_jenis_barang = j.id_jenis_barang
WHERE b.id_barang = $1 LIMIT 1;

-- name: ListBarang :many
SELECT b.*, m.nama_perusahaan, j.nama_jenis_barang
FROM BARANG b
JOIN MITRA m ON b.id_mitra = m.id_mitra
JOIN JENIS_BARANG j ON b.id_jenis_barang = j.id_jenis_barang
ORDER BY b.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateBarang :one
INSERT INTO BARANG (
    nama_barang, kode, id_jenis_barang, id_mitra
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: UpdateBarang :one
UPDATE BARANG
SET nama_barang = $2, kode = $3, id_jenis_barang = $4, id_mitra = $5
WHERE id_barang = $1
RETURNING *;

-- name: DeleteBarang :execrows
DELETE FROM BARANG
WHERE id_barang = $1;
