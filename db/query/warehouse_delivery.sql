-- name: ReceiveInventory :one
WITH inserted_received AS (
    INSERT INTO RECEIVED (
        tanggal,
        qty,
        keterangan,
        id_material_list_item
    ) VALUES (
        sqlc.arg(tanggal)::date,
        sqlc.arg(qty),
        sqlc.arg(keterangan),
        sqlc.arg(id_material_list_item)
    )
    RETURNING id_received, tanggal, qty, keterangan, id_material_list_item AS id_material_list, created_at
),
inserted_rekonsiliasi_terima AS (
    INSERT INTO REKONSILIASI_MATERIAL_TERIMA (
        keterangan,
        qty,
        id_rekonsiliasi_material
    ) VALUES (
        sqlc.arg(keterangan),
        sqlc.arg(qty),
        sqlc.arg(id_rekonsiliasi_material)
    )
    RETURNING id_rekonsiliasi_material_terima
),
updated_rekonsiliasi AS (
    UPDATE REKONSILIASI_MATERIAL
    SET
        actual_kirim = actual_kirim + sqlc.arg(qty),
        balance = balance + sqlc.arg(qty)
    WHERE id_rekonsiliasi_material = sqlc.arg(id_rekonsiliasi_material)
    RETURNING id_rekonsiliasi_material, actual_kirim, balance
)
SELECT
    r.id_received,
    r.tanggal,
    r.qty,
    r.keterangan,
    r.id_material_list,
    r.created_at,
    t.id_rekonsiliasi_material_terima,
    u.id_rekonsiliasi_material,
    u.actual_kirim,
    u.balance
FROM inserted_received r
CROSS JOIN inserted_rekonsiliasi_terima t
CROSS JOIN updated_rekonsiliasi u;

-- name: CreatePackingList :one
INSERT INTO PACKING_LIST (
    total_garment_per_box,
    total_reject,
    id_wo,
    id_surat_jalan_internal
) VALUES (
    sqlc.arg(total_garment_per_box),
    sqlc.arg(total_reject),
    sqlc.arg(id_wo),
    sqlc.narg(id_surat_jalan_internal)
)
RETURNING id_packing_list, total_garment_per_box, total_reject, id_wo, id_surat_jalan_internal, created_at;

-- name: CreatePackingListItem :one
INSERT INTO PACKING_LIST_ITEM (
    id_packing_list,
    color,
    qty_box,
    qty_per_box,
    box_no_start,
    box_no_end,
    note
) VALUES (
    sqlc.arg(id_packing_list),
    sqlc.arg(color),
    sqlc.arg(qty_box),
    sqlc.arg(qty_per_box),
    sqlc.arg(box_no_start),
    sqlc.arg(box_no_end),
    sqlc.arg(note)
)
RETURNING id_packing_list_item, id_packing_list, color, qty_box, qty_per_box, box_no_start, box_no_end, note, created_at;

-- name: CreatePackingListItemSize :one
INSERT INTO PACKING_LIST_ITEM_SIZE (
    qty,
    id_packing_list_item,
    id_wo_shell_size
) VALUES (
    sqlc.arg(qty),
    sqlc.arg(id_packing_list_item),
    sqlc.arg(id_wo_shell_size)
)
RETURNING id_packing_list_item_size, qty, id_packing_list_item, id_wo_shell_size, created_at;

-- name: CreatePackingListRejectSize :one
INSERT INTO PACKING_LIST_REJECT_SIZE (
    qty,
    id_packing_list,
    id_wo_shell_size
) VALUES (
    sqlc.arg(qty),
    sqlc.arg(id_packing_list),
    sqlc.arg(id_wo_shell_size)
)
RETURNING id_packing_list_reject_size, qty, id_packing_list, id_wo_shell_size, created_at;

-- name: CreateSuratJalanInternal :one
INSERT INTO SURAT_JALAN_INTERNAL (
    id_wo,
    no_dokumen,
    deskripsi
) VALUES (
    sqlc.arg(id_wo),
    sqlc.arg(no_dokumen),
    sqlc.arg(deskripsi)
)
RETURNING id_surat_jalan_internal, id_wo, no_dokumen, deskripsi, created_at;

-- name: CreateSuratJalanInternalItem :one
INSERT INTO SURAT_JALAN_INTERNAL_ITEM (
    id_surat_jalan_internal,
    no_urut,
    deskripsi,
    qty,
    note
) VALUES (
    sqlc.arg(id_surat_jalan_internal),
    sqlc.arg(no_urut),
    sqlc.arg(deskripsi),
    sqlc.arg(qty),
    sqlc.arg(note)
)
RETURNING id_surat_jalan_internal_item, id_surat_jalan_internal, no_urut, deskripsi, qty, note, created_at;

-- name: ListSuratJalanInternalItemsBySJID :many
SELECT
    id_surat_jalan_internal_item,
    id_surat_jalan_internal,
    no_urut,
    deskripsi,
    qty,
    note,
    created_at
FROM SURAT_JALAN_INTERNAL_ITEM
WHERE id_surat_jalan_internal = sqlc.arg(id_surat_jalan_internal)
ORDER BY no_urut ASC, id_surat_jalan_internal_item ASC;

-- name: AssignPackingListToSuratJalan :exec
UPDATE PACKING_LIST
SET id_surat_jalan_internal = sqlc.arg(id_surat_jalan_internal)
WHERE id_packing_list = sqlc.arg(id_packing_list);

-- name: UnassignPackingListFromSuratJalan :exec
UPDATE PACKING_LIST
SET id_surat_jalan_internal = NULL
WHERE id_packing_list = sqlc.arg(id_packing_list);

-- name: ListPackingListsBySuratJalanID :many
SELECT
    pl.id_packing_list,
    pl.total_garment_per_box,
    pl.total_reject,
    pl.id_wo,
    pl.id_surat_jalan_internal,
    pl.created_at
FROM PACKING_LIST pl
WHERE pl.id_surat_jalan_internal = sqlc.arg(id_surat_jalan_internal)
ORDER BY pl.id_packing_list ASC;

-- name: CreateSuratJalanClient :one
INSERT INTO SURAT_JALAN_CLIENT (
    tanggal,
    qty,
    keterangan,
    id_material_list_item
) VALUES (
    sqlc.arg(tanggal)::date,
    sqlc.arg(qty),
    sqlc.arg(keterangan),
    sqlc.arg(id_material_list_item)
)
RETURNING id_surat_jalan_client, tanggal, qty, keterangan, id_material_list_item AS id_material_list, created_at;

-- name: GetRekonsiliasiMaterialStock :one
SELECT
    id_rekonsiliasi_material,
    balance,
    last_balance
FROM REKONSILIASI_MATERIAL
WHERE id_rekonsiliasi_material = sqlc.arg(id_rekonsiliasi_material)
LIMIT 1;

-- name: IssueInventory :one
UPDATE REKONSILIASI_MATERIAL
SET
    last_balance = balance,
    balance = balance - sqlc.arg(qty)
WHERE id_rekonsiliasi_material = sqlc.arg(id_rekonsiliasi_material)
RETURNING id_rekonsiliasi_material, last_balance, balance;

-- name: CreateReceivedSimple :one
INSERT INTO RECEIVED (tanggal, qty, keterangan, id_material_list_item)
VALUES (sqlc.arg(tanggal)::date, sqlc.arg(qty), sqlc.arg(keterangan), sqlc.arg(id_material_list_item))
RETURNING id_received, tanggal, qty, keterangan, id_material_list_item, created_at;

-- name: ListReceived :many
SELECT
    r.id_received,
    r.tanggal,
    r.qty,
    r.keterangan,
    r.id_material_list_item,
    r.created_at,
    mli.item AS material_item,
    mli.description AS material_description,
    ml.id_wo,
    COUNT(*) OVER () AS total_count
FROM RECEIVED r
JOIN MATERIAL_LIST_ITEM mli ON mli.id_material_list_item = r.id_material_list_item
JOIN MATERIAL_LIST ml ON ml.id_material_list = mli.id_material_list
WHERE (
    sqlc.arg(search)::text = ''
    OR LOWER(mli.item) LIKE '%' || LOWER(sqlc.arg(search)::text) || '%'
    OR LOWER(mli.description) LIKE '%' || LOWER(sqlc.arg(search)::text) || '%'
    OR LOWER(r.keterangan) LIKE '%' || LOWER(sqlc.arg(search)::text) || '%'
)
ORDER BY r.created_at DESC
LIMIT sqlc.arg(lim)
OFFSET sqlc.arg(off);

-- name: GetReceivedByID :one
SELECT
    r.id_received,
    r.tanggal,
    r.qty,
    r.keterangan,
    r.id_material_list_item,
    r.created_at,
    mli.item AS material_item,
    mli.description AS material_description,
    ml.id_wo
FROM RECEIVED r
JOIN MATERIAL_LIST_ITEM mli ON mli.id_material_list_item = r.id_material_list_item
JOIN MATERIAL_LIST ml ON ml.id_material_list = mli.id_material_list
WHERE r.id_received = sqlc.arg(id_received);

-- name: UpdateReceivedSimple :one
UPDATE RECEIVED
SET tanggal = sqlc.arg(tanggal)::date, qty = sqlc.arg(qty), keterangan = sqlc.arg(keterangan)
WHERE id_received = sqlc.arg(id_received)
RETURNING id_received, tanggal, qty, keterangan, id_material_list_item, created_at;

-- name: DeleteReceivedSimple :exec
DELETE FROM RECEIVED WHERE id_received = sqlc.arg(id_received);

-- name: ListReceivedByMLI :many
SELECT id_received, tanggal, qty, keterangan, id_material_list_item, created_at
FROM RECEIVED
WHERE id_material_list_item = sqlc.arg(id_material_list_item)
ORDER BY created_at DESC;

-- name: ListSuratJalanClientByMLI :many
SELECT id_surat_jalan_client, tanggal, qty, keterangan, id_material_list_item, created_at
FROM SURAT_JALAN_CLIENT
WHERE id_material_list_item = sqlc.arg(id_material_list_item)
ORDER BY created_at DESC;

-- name: DeleteSuratJalanClient :exec
DELETE FROM SURAT_JALAN_CLIENT WHERE id_surat_jalan_client = sqlc.arg(id_surat_jalan_client);
