-- name: GetJenisBarangByID :one
SELECT * FROM JENIS_BARANG
WHERE id_jenis_barang = $1 LIMIT 1;

-- name: ListJenisBarang :many
SELECT * FROM JENIS_BARANG
ORDER BY nama_jenis_barang ASC;

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
