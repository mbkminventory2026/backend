-- name: GetProfilePerusahaanByID :one
SELECT * FROM PROFILE_PERUSAHAAN
WHERE id_profile_perusahaan = $1 LIMIT 1;

-- name: GetProfilePerusahaan :one
SELECT * FROM PROFILE_PERUSAHAAN
ORDER BY created_at DESC LIMIT 1;

-- name: CreateProfilePerusahaan :one
INSERT INTO PROFILE_PERUSAHAAN (
    nama, alamat, email, no_telp, about, logo
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateProfilePerusahaan :one
UPDATE PROFILE_PERUSAHAAN
SET nama = $2, alamat = $3, email = $4, no_telp = $5, about = $6, logo = $7
WHERE id_profile_perusahaan = $1
RETURNING *;

-- name: DeleteProfilePerusahaan :execrows
DELETE FROM PROFILE_PERUSAHAAN
WHERE id_profile_perusahaan = $1;
