-- name: CreatePRInternal :one
INSERT INTO PR_INTERNAL (
    tanggal,
    nama,
    departemen,
    vendor_name,
    vendor_address,
    vendor_telp,
    projek,
    id_wo,
    id_user
) VALUES (
    sqlc.arg(tanggal)::date,
    sqlc.arg(nama),
    sqlc.arg(departemen),
    sqlc.arg(vendor_name),
    sqlc.arg(vendor_address),
    sqlc.arg(vendor_telp),
    sqlc.arg(projek),
    sqlc.arg(id_wo),
    sqlc.arg(id_user)
)
RETURNING id_pr_internal, tanggal, nama, departemen, vendor_name, vendor_address, vendor_telp, projek, id_wo, id_user, created_at;

-- name: CreatePRInternalItem :one
INSERT INTO PR_INTERNAL_ITEM (
    id_pr_internal,
    item,
    description,
    qty,
    unit,
    est_price
) VALUES (
    sqlc.arg(id_pr_internal),
    sqlc.arg(item),
    sqlc.arg(description),
    sqlc.arg(qty),
    sqlc.arg(unit),
    sqlc.arg(est_price)::numeric
)
RETURNING id_pr_internal_item, id_pr_internal, item, description, qty, unit, est_price, created_at;
