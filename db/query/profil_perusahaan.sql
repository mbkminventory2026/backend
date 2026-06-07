-- name: GetProfilPerusahaanByID :one
SELECT * FROM PROFIL_PERUSAHAAN
WHERE id_profil_perusahaan = $1 LIMIT 1;

-- name: GetProfilPerusahaan :one
SELECT * FROM PROFIL_PERUSAHAAN
ORDER BY created_at DESC LIMIT 1;

-- name: CreateProfilPerusahaan :one
INSERT INTO PROFIL_PERUSAHAAN (
    nama, alamat, email, no_telp, about, logo
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateProfilPerusahaan :one
UPDATE PROFIL_PERUSAHAAN
SET nama = $2, alamat = $3, email = $4, no_telp = $5, about = $6, logo = $7
WHERE id_profil_perusahaan = $1
RETURNING *;

-- name: DeleteProfilPerusahaan :execrows
DELETE FROM PROFIL_PERUSAHAAN
WHERE id_profil_perusahaan = $1;
