-- name: CreatePOInternal :one
INSERT INTO PO_INTERNAL (
    tanggal,
    nama_po,
    supplier_name,
    supplier_addr,
    supplier_contact,
    supplier_email,
    supplier_telp,
    supplier_fax,
    currency,
    cpo,
    term,
    ship_date,
    id_pr_internal
) VALUES (
    sqlc.arg(tanggal)::date,
    sqlc.arg(nama_po),
    sqlc.arg(supplier_name),
    sqlc.arg(supplier_addr),
    sqlc.arg(supplier_contact),
    sqlc.arg(supplier_email),
    sqlc.arg(supplier_telp),
    sqlc.arg(supplier_fax),
    sqlc.arg(currency),
    sqlc.arg(cpo),
    sqlc.arg(term),
    sqlc.arg(ship_date)::date,
    sqlc.arg(id_pr_internal)
)
RETURNING id_po_internal, tanggal, nama_po, supplier_name, supplier_addr, supplier_contact, supplier_email, supplier_telp, supplier_fax, currency, cpo, term, ship_date, id_pr_internal, created_at;

-- name: CreatePOInternalItem :one
INSERT INTO PO_INTERNAL_ITEM (
    id_po_internal,
    item,
    description,
    qty,
    unit,
    unit_price
) VALUES (
    sqlc.arg(id_po_internal),
    sqlc.arg(item),
    sqlc.arg(description),
    sqlc.arg(qty),
    sqlc.arg(unit),
    sqlc.arg(unit_price)::numeric
)
RETURNING id_po_internal_item, id_po_internal, item, description, qty, unit, unit_price, created_at;
