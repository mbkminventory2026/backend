-- name: GetCompany :one
SELECT * FROM COMPANY
ORDER BY created_at DESC LIMIT 1;

-- name: UpdateCompany :one
UPDATE COMPANY
SET nama = $2, alamat = $3, email = $4, no_telp = $5, about = $6, logo = $7
WHERE id_company = $1
RETURNING *;
