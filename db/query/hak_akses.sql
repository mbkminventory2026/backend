-- name: GetHakAksesByID :one
SELECT * FROM HAK_AKSES
WHERE id_hak_akses = $1 LIMIT 1;

-- name: ListHakAkses :many
SELECT * FROM HAK_AKSES
ORDER BY nama_halaman ASC;

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
