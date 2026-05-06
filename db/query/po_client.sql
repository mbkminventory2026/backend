-- name: CreatePOClient :one
INSERT INTO PO_CLIENT (
    po_number,
    tanggal,
    season,
    delivery,
    payment_term,
    file,
    id_mitra
) VALUES (
    sqlc.arg(po_number),
    sqlc.arg(tanggal)::date,
    sqlc.arg(season),
    sqlc.arg(delivery)::date,
    sqlc.arg(payment_term),
    sqlc.arg(file),
    sqlc.arg(id_mitra)
)
RETURNING id_po_client, po_number, tanggal, season, delivery, payment_term, file, id_mitra, created_at;

-- name: CreatePOClientItem :one
INSERT INTO PO_CLIENT_ITEM (
    id_po_client,
    style,
    colour,
    description,
    qty,
    price
) VALUES (
    sqlc.arg(id_po_client),
    sqlc.arg(style),
    sqlc.arg(colour),
    sqlc.arg(description),
    sqlc.arg(qty),
    sqlc.arg(price)::numeric
)
RETURNING id_po_client_item, id_po_client, style, colour, description, qty, price, created_at;

-- name: CreatePenanggungJawab :one
INSERT INTO PENANGGUNG_JAWAB (
    nama,
    no_telp,
    email,
    id_po_client
) VALUES (
    sqlc.arg(nama),
    sqlc.arg(no_telp),
    sqlc.arg(email),
    sqlc.arg(id_po_client)
)
RETURNING id_penanggung_jawab, nama, no_telp, email, id_po_client, created_at;
