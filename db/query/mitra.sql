-- name: GetMitraByID :one
SELECT * FROM MITRA
WHERE id_mitra = $1 LIMIT 1;

-- name: ListMitra :many
SELECT * FROM MITRA
ORDER BY nama_perusahaan ASC;

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
