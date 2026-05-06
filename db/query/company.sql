-- name: GetCompanyByID :one
SELECT * FROM COMPANY
WHERE id_company = $1 LIMIT 1;

-- name: GetCompany :one
SELECT * FROM COMPANY
ORDER BY created_at DESC LIMIT 1;

-- name: CreateCompany :one
INSERT INTO COMPANY (
    nama, alamat, email, no_telp, about, logo
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateCompany :one
UPDATE COMPANY
SET nama = $2, alamat = $3, email = $4, no_telp = $5, about = $6, logo = $7
WHERE id_company = $1
RETURNING *;

-- name: DeleteCompany :execrows
DELETE FROM COMPANY
WHERE id_company = $1;
