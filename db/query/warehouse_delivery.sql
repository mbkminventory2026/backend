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
    id_packing_list_item
) VALUES (
    sqlc.arg(qty),
    sqlc.arg(id_packing_list_item)
)
RETURNING id_packing_list_item_size, qty, id_packing_list_item, created_at;

-- name: CreatePackingListRejectSize :one
INSERT INTO PACKING_LIST_REJECT_SIZE (
    qty,
    id_packing_list
) VALUES (
    sqlc.arg(qty),
    sqlc.arg(id_packing_list)
)
RETURNING id_packing_list_reject_size, qty, id_packing_list, created_at;

-- name: CreateSuratJalanInternal :one
INSERT INTO SURAT_JALAN_INTERNAL DEFAULT VALUES
RETURNING id_surat_jalan_internal, created_at;

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
