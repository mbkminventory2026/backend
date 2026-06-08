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

-- name: UpdatePOClient :one
UPDATE PO_CLIENT
SET
    po_number = sqlc.arg(po_number),
    tanggal = sqlc.arg(tanggal)::date,
    season = sqlc.arg(season),
    delivery = sqlc.arg(delivery)::date,
    payment_term = sqlc.arg(payment_term),
    file = sqlc.arg(file),
    id_mitra = sqlc.arg(id_mitra)
WHERE id_po_client = sqlc.arg(id_po_client)
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

-- name: DeletePOClientItemsByPOClientID :exec
DELETE FROM PO_CLIENT_ITEM
WHERE id_po_client = sqlc.arg(id_po_client);

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

-- name: DeletePenanggungJawabByPOClientID :exec
DELETE FROM PENANGGUNG_JAWAB
WHERE id_po_client = sqlc.arg(id_po_client);

-- name: CountWorkOrdersByPOClientID :one
SELECT COUNT(*)
FROM WORK_ORDER wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
WHERE pci.id_po_client = sqlc.arg(id_po_client);

-- name: ListAvailablePOClientItems :many
SELECT 
    pci.id_po_client_item,
    pci.style,
    pci.colour,
    pci.qty,
    pci.id_po_client
FROM PO_CLIENT_ITEM pci
WHERE NOT EXISTS (
    SELECT 1 
    FROM WORK_ORDER wo 
    WHERE wo.id_po_client_item = pci.id_po_client_item
)
ORDER BY pci.created_at DESC;