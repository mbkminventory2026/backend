-- name: GetRekonsiliasiByWOID :one
SELECT
    r.id_rekonsiliasi,
    r.id_wo,
    r.jasa,
    r.no_po,
    r.delivery,
    r.buyer,
    r.brand,
    r.style,
    r.qty_po,
    r.plan_cut_total,
    r.cons_baju_summary,
    r.nama_bahan,
    r.warna_kain_summary,
    r.created_by,
    r.updated_by,
    r.created_at,
    r.updated_at
FROM REKONSILIASI r
WHERE r.id_wo = sqlc.arg(id_wo)
LIMIT 1;

-- name: CreateRekonsiliasi :one
INSERT INTO REKONSILIASI (
    id_wo,
    jasa,
    no_po,
    delivery,
    buyer,
    brand,
    style,
    qty_po,
    plan_cut_total,
    cons_baju_summary,
    nama_bahan,
    warna_kain_summary,
    created_by,
    updated_by
) VALUES (
    sqlc.arg(id_wo),
    sqlc.arg(jasa),
    sqlc.arg(no_po),
    sqlc.arg(delivery)::date,
    sqlc.arg(buyer),
    sqlc.arg(brand),
    sqlc.arg(style),
    sqlc.arg(qty_po),
    sqlc.arg(plan_cut_total),
    sqlc.arg(cons_baju_summary),
    sqlc.arg(nama_bahan),
    to_jsonb(sqlc.arg(warna_kain_summary)::text[]),
    sqlc.narg(created_by),
    sqlc.narg(updated_by)
)
RETURNING
    id_rekonsiliasi,
    id_wo,
    jasa,
    no_po,
    delivery,
    buyer,
    brand,
    style,
    qty_po,
    plan_cut_total,
    cons_baju_summary,
    nama_bahan,
    warna_kain_summary,
    created_by,
    updated_by,
    created_at,
    updated_at;

-- name: UpdateRekonsiliasiSnapshot :one
UPDATE REKONSILIASI
SET
    jasa = sqlc.arg(jasa),
    no_po = sqlc.arg(no_po),
    delivery = sqlc.arg(delivery)::date,
    buyer = sqlc.arg(buyer),
    brand = sqlc.arg(brand),
    style = sqlc.arg(style),
    qty_po = sqlc.arg(qty_po),
    plan_cut_total = sqlc.arg(plan_cut_total),
    cons_baju_summary = sqlc.arg(cons_baju_summary),
    nama_bahan = sqlc.arg(nama_bahan),
    warna_kain_summary = to_jsonb(sqlc.arg(warna_kain_summary)::text[]),
    updated_by = sqlc.narg(updated_by),
    updated_at = NOW()
WHERE id_rekonsiliasi = sqlc.arg(id_rekonsiliasi)
RETURNING
    id_rekonsiliasi,
    id_wo,
    jasa,
    no_po,
    delivery,
    buyer,
    brand,
    style,
    qty_po,
    plan_cut_total,
    cons_baju_summary,
    nama_bahan,
    warna_kain_summary,
    created_by,
    updated_by,
    created_at,
    updated_at;

-- name: TouchRekonsiliasi :one
UPDATE REKONSILIASI
SET
    updated_by = sqlc.narg(updated_by),
    updated_at = NOW()
WHERE id_rekonsiliasi = sqlc.arg(id_rekonsiliasi)
RETURNING
    id_rekonsiliasi,
    id_wo,
    jasa,
    no_po,
    delivery,
    buyer,
    brand,
    style,
    qty_po,
    plan_cut_total,
    cons_baju_summary,
    nama_bahan,
    warna_kain_summary,
    created_by,
    updated_by,
    created_at,
    updated_at;

-- name: ListRekonsiliasis :many
SELECT
    r.id_rekonsiliasi,
    r.id_wo,
    wo.model AS nama_wo,
    r.buyer,
    r.brand,
    r.style,
    r.qty_po,
    r.plan_cut_total,
    r.created_at,
    r.updated_at,
    COALESCE(cu.username, '') AS created_by_username,
    COALESCE(uu.username, '') AS updated_by_username,
    COUNT(*) OVER() AS total_count
FROM REKONSILIASI r
JOIN WORK_ORDER wo ON wo.id_wo = r.id_wo
LEFT JOIN USERS cu ON cu.id_user = r.created_by
LEFT JOIN USERS uu ON uu.id_user = r.updated_by
WHERE (
    sqlc.arg(search_term)::text = ''
    OR r.no_po ILIKE '%' || sqlc.arg(search_term)::text || '%'
    OR r.buyer ILIKE '%' || sqlc.arg(search_term)::text || '%'
    OR r.style ILIKE '%' || sqlc.arg(search_term)::text || '%'
    OR wo.model ILIKE '%' || sqlc.arg(search_term)::text || '%'
) AND (
    sqlc.narg(id_wo)::integer IS NULL
    OR r.id_wo = sqlc.narg(id_wo)::integer
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN r.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN r.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'updated_at' AND NOT sqlc.arg(sort_desc)::bool THEN r.updated_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'updated_at' AND sqlc.arg(sort_desc)::bool THEN r.updated_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_rekonsiliasi' AND NOT sqlc.arg(sort_desc)::bool THEN r.id_rekonsiliasi END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_rekonsiliasi' AND sqlc.arg(sort_desc)::bool THEN r.id_rekonsiliasi END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_wo' AND NOT sqlc.arg(sort_desc)::bool THEN r.id_wo END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_wo' AND sqlc.arg(sort_desc)::bool THEN r.id_wo END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND NOT sqlc.arg(sort_desc)::bool THEN r.buyer END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND sqlc.arg(sort_desc)::bool THEN r.buyer END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'style' AND NOT sqlc.arg(sort_desc)::bool THEN r.style END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'style' AND sqlc.arg(sort_desc)::bool THEN r.style END DESC,
    r.id_rekonsiliasi DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetRekonsiliasiByID :one
SELECT
    r.id_rekonsiliasi,
    r.id_wo,
    r.jasa,
    r.no_po,
    r.delivery,
    r.buyer,
    r.brand,
    r.style,
    r.qty_po,
    r.plan_cut_total,
    r.cons_baju_summary,
    r.nama_bahan,
    r.warna_kain_summary,
    r.created_by,
    r.updated_by,
    r.created_at,
    r.updated_at,
    COALESCE(cu.username, '') AS created_by_username,
    COALESCE(uu.username, '') AS updated_by_username
FROM REKONSILIASI r
LEFT JOIN USERS cu ON cu.id_user = r.created_by
LEFT JOIN USERS uu ON uu.id_user = r.updated_by
WHERE r.id_rekonsiliasi = sqlc.arg(id_rekonsiliasi)
LIMIT 1;

-- name: DeleteRekonsiliasiMaterialRowsByRekonsiliasiID :exec
DELETE FROM REKONSILIASI_MATERIAL_ROW
WHERE id_rekonsiliasi = sqlc.arg(id_rekonsiliasi);

-- name: CreateRekonsiliasiMaterialRow :one
INSERT INTO REKONSILIASI_MATERIAL_ROW (
    id_rekonsiliasi,
    row_no,
    kategori,
    description,
    size_label,
    ratio_source,
    ratio_input,
    qty_per_pcs_input,
    qty_wo,
    toleransi,
    satuan,
    qty_actual_kirim_source,
    qty_actual_kirim_manual,
    reject_qty,
    retur_qty,
    keterangan,
    id_material_list_item,
    id_wo_shell,
    id_wo_trim
) VALUES (
    sqlc.arg(id_rekonsiliasi),
    sqlc.arg(row_no),
    sqlc.arg(kategori),
    sqlc.arg(description),
    sqlc.arg(size_label),
    sqlc.arg(ratio_source)::numeric,
    sqlc.arg(ratio_input)::numeric,
    sqlc.arg(qty_per_pcs_input)::numeric,
    sqlc.arg(qty_wo),
    sqlc.arg(toleransi),
    sqlc.arg(satuan),
    sqlc.arg(qty_actual_kirim_source),
    sqlc.arg(qty_actual_kirim_manual),
    sqlc.arg(reject_qty),
    sqlc.arg(retur_qty),
    sqlc.arg(keterangan),
    sqlc.narg(id_material_list_item),
    sqlc.narg(id_wo_shell),
    sqlc.narg(id_wo_trim)
)
RETURNING
    id_rekonsiliasi_material_row,
    id_rekonsiliasi,
    row_no,
    kategori,
    description,
    size_label,
    ratio_source,
    ratio_input,
    qty_per_pcs_input,
    qty_wo,
    toleransi,
    satuan,
    qty_actual_kirim_source,
    qty_actual_kirim_manual,
    reject_qty,
    retur_qty,
    keterangan,
    id_material_list_item,
    id_wo_shell,
    id_wo_trim,
    created_at,
    updated_at;

-- name: ListRekonsiliasiMaterialRowsByRekonsiliasiID :many
SELECT
    id_rekonsiliasi_material_row,
    id_rekonsiliasi,
    row_no,
    kategori,
    description,
    size_label,
    ratio_source,
    ratio_input,
    qty_per_pcs_input,
    qty_wo,
    toleransi,
    satuan,
    qty_actual_kirim_source,
    qty_actual_kirim_manual,
    reject_qty,
    retur_qty,
    keterangan,
    id_material_list_item,
    id_wo_shell,
    id_wo_trim,
    created_at,
    updated_at
FROM REKONSILIASI_MATERIAL_ROW
WHERE id_rekonsiliasi = sqlc.arg(id_rekonsiliasi)
ORDER BY row_no ASC, id_rekonsiliasi_material_row ASC;

-- name: UpdateRekonsiliasiMaterialRowManualFields :one
UPDATE REKONSILIASI_MATERIAL_ROW
SET
    ratio_input = sqlc.arg(ratio_input)::numeric,
    qty_per_pcs_input = sqlc.arg(qty_per_pcs_input)::numeric,
    qty_actual_kirim_manual = sqlc.arg(qty_actual_kirim_manual),
    reject_qty = sqlc.arg(reject_qty),
    retur_qty = sqlc.arg(retur_qty),
    keterangan = sqlc.arg(keterangan),
    updated_at = NOW()
WHERE id_rekonsiliasi_material_row = sqlc.arg(id_rekonsiliasi_material_row)
RETURNING
    id_rekonsiliasi_material_row,
    id_rekonsiliasi,
    row_no,
    kategori,
    description,
    size_label,
    ratio_source,
    ratio_input,
    qty_per_pcs_input,
    qty_wo,
    toleransi,
    satuan,
    qty_actual_kirim_source,
    qty_actual_kirim_manual,
    reject_qty,
    retur_qty,
    keterangan,
    id_material_list_item,
    id_wo_shell,
    id_wo_trim,
    created_at,
    updated_at;

-- name: DeleteRekonsiliasiTerimaEntriesByRowID :exec
DELETE FROM REKONSILIASI_TERIMA_ENTRY
WHERE id_rekonsiliasi_material_row = sqlc.arg(id_rekonsiliasi_material_row);

-- name: CreateRekonsiliasiTerimaEntry :one
INSERT INTO REKONSILIASI_TERIMA_ENTRY (
    id_rekonsiliasi_material_row,
    entry_type,
    entry_label,
    qty,
    note
) VALUES (
    sqlc.arg(id_rekonsiliasi_material_row),
    sqlc.arg(entry_type),
    sqlc.arg(entry_label),
    sqlc.arg(qty),
    sqlc.arg(note)
)
RETURNING
    id_rekonsiliasi_terima_entry,
    id_rekonsiliasi_material_row,
    entry_type,
    entry_label,
    qty,
    note,
    created_at,
    updated_at;

-- name: ListRekonsiliasiTerimaEntriesByRekonsiliasiID :many
SELECT
    rte.id_rekonsiliasi_terima_entry,
    rte.id_rekonsiliasi_material_row,
    rte.entry_type,
    rte.entry_label,
    rte.qty,
    rte.note,
    rte.created_at,
    rte.updated_at
FROM REKONSILIASI_TERIMA_ENTRY rte
JOIN REKONSILIASI_MATERIAL_ROW rmr ON rmr.id_rekonsiliasi_material_row = rte.id_rekonsiliasi_material_row
WHERE rmr.id_rekonsiliasi = sqlc.arg(id_rekonsiliasi)
ORDER BY rmr.row_no ASC, rte.id_rekonsiliasi_terima_entry ASC;

-- name: GetRekonsiliasiSourceHeader :one
SELECT
    wo.id_wo,
    CASE
        WHEN wo.fob_cmt THEN 'FOB'
        ELSE 'CMT'
    END AS jasa,
    pc.po_number AS no_po,
    pc.delivery,
    wo.buyer,
    ''::text AS brand,
    pci.style,
    pci.qty AS qty_po,
    COALESCE((
        SELECT SUM(rss.ratio_plan)::bigint
        FROM RATIO_SIZE_SPREADING rss
        JOIN RATIO_SPREADING rs ON rs.id_ratio_spreading = rss.id_ratio_spreading
        JOIN WORK_ORDER_SHELL wos_plan ON wos_plan.id_wo_shell = rs.id_wo_shell
        WHERE wos_plan.id_wo = wo.id_wo
    ), 0)::bigint AS plan_cut_total
FROM WORK_ORDER wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE wo.id_wo = sqlc.arg(id_wo)
LIMIT 1;

-- name: ListRekonsiliasiShellSourcesByWO :many
SELECT
    wos.id_wo_shell,
    wos.deskripsi,
    wos.color,
    wos.cons,
    wos.allow,
    wos.material_type,
    COALESCE(SUM(wss.qty), 0)::integer AS qty_order,
    COALESCE(shell_ship.qty_shipped, 0)::integer AS qty_actual_kirim_source
FROM WORK_ORDER_SHELL wos
LEFT JOIN WORK_ORDER_SHELL_SIZE wss ON wss.id_wo_shell = wos.id_wo_shell
LEFT JOIN (
    SELECT
        wss2.id_wo_shell,
        SUM(rp.qty) AS qty_shipped
    FROM REPORT_PENGIRIMAN rp
    JOIN WORK_ORDER_SHELL_SIZE wss2 ON wss2.id_wo_shell_size = rp.id_wo_shell_size
    GROUP BY wss2.id_wo_shell
) shell_ship ON shell_ship.id_wo_shell = wos.id_wo_shell
WHERE wos.id_wo = sqlc.arg(id_wo)
GROUP BY
    wos.id_wo_shell,
    wos.deskripsi,
    wos.color,
    wos.cons,
    wos.allow,
    wos.material_type,
    shell_ship.qty_shipped
ORDER BY wos.id_wo_shell ASC;

-- name: ListRekonsiliasiColorSourcesByWO :many
SELECT
    wos.color,
    ms.nama_size AS size,
    wss.qty AS qty_order,
    COALESCE(ship.qty_kirim, 0)::integer AS qty_kirim
FROM WORK_ORDER_SHELL wos
JOIN WORK_ORDER_SHELL_SIZE wss ON wss.id_wo_shell = wos.id_wo_shell
JOIN MASTER_SIZE ms ON ms.id_size = wss.id_size
LEFT JOIN (
    SELECT
        rp.id_wo_shell_size,
        SUM(rp.qty) AS qty_kirim
    FROM REPORT_PENGIRIMAN rp
    GROUP BY rp.id_wo_shell_size
) ship ON ship.id_wo_shell_size = wss.id_wo_shell_size
WHERE wos.id_wo = sqlc.arg(id_wo)
ORDER BY wos.color ASC, wss.id_wo_shell_size ASC;

-- name: ListRekonsiliasiMaterialSourceRowsByWO :many
SELECT
    mli.id_material_list_item,
    mli.id_material_list,
    CASE
        WHEN mli.id_wo_shell IS NOT NULL THEN 'shell'
        ELSE 'trim'
    END AS kategori,
    COALESCE(
        NULLIF(mli.item, ''),
        NULLIF(mli.description, ''),
        CASE
            WHEN mli.id_wo_shell IS NOT NULL THEN wos.deskripsi
            ELSE wot.item
        END,
        ''
    ) AS description,
    CASE
        WHEN mli.id_wo_shell IS NOT NULL THEN COALESCE(wos.color, 'ALL SIZE')
        WHEN COALESCE(wot.color, '') <> '' THEN wot.color
        ELSE 'ALL SIZE'
    END AS size_label,
    CASE
        WHEN mli.id_wo_shell IS NOT NULL THEN COALESCE(shell_qty.qty_order, 0)::numeric
        WHEN mli.id_wo_trim IS NOT NULL THEN COALESCE(wot.qty, mli.qty)::numeric
        ELSE mli.qty::numeric
    END AS ratio_source,
    CASE
        WHEN mli.id_wo_shell IS NOT NULL THEN COALESCE(ROUND(shell_qty.qty_order::numeric * wos.cons), 0)::integer
        WHEN mli.id_wo_trim IS NOT NULL THEN COALESCE(wot.qty, mli.qty)
        ELSE mli.qty
    END AS qty_wo,
    CASE
        WHEN mli.id_wo_shell IS NOT NULL THEN COALESCE(wos.allow, 0)
        WHEN mli.id_wo_trim IS NOT NULL THEN COALESCE(wot.allow, 0)
        ELSE 0
    END AS toleransi,
    CASE
        WHEN COALESCE(NULLIF(mli.unit, ''), '') <> '' THEN mli.unit
        WHEN mli.id_wo_shell IS NOT NULL THEN 'YDS'
        WHEN mli.id_wo_trim IS NOT NULL THEN COALESCE(wot.uom, '')
        ELSE ''
    END AS satuan,
    CASE
        WHEN mli.id_wo_shell IS NOT NULL THEN COALESCE(shell_ship.qty_shipped, 0)
        ELSE COALESCE(wo_ship.qty_shipped, 0)
    END AS qty_actual_kirim_source,
    mli.id_wo_shell,
    mli.id_wo_trim
FROM MATERIAL_LIST_ITEM mli
JOIN MATERIAL_LIST ml ON ml.id_material_list = mli.id_material_list
LEFT JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = mli.id_wo_shell
LEFT JOIN WORK_ORDER_TRIM wot ON wot.id_wo_trim = mli.id_wo_trim
LEFT JOIN (
    SELECT
        wss.id_wo_shell,
        SUM(wss.qty) AS qty_order
    FROM WORK_ORDER_SHELL_SIZE wss
    GROUP BY wss.id_wo_shell
) shell_qty ON shell_qty.id_wo_shell = mli.id_wo_shell
LEFT JOIN (
    SELECT
        wss.id_wo_shell,
        SUM(rp.qty)::integer AS qty_shipped
    FROM REPORT_PENGIRIMAN rp
    JOIN WORK_ORDER_SHELL_SIZE wss ON wss.id_wo_shell_size = rp.id_wo_shell_size
    GROUP BY wss.id_wo_shell
) shell_ship ON shell_ship.id_wo_shell = mli.id_wo_shell
LEFT JOIN (
    SELECT
        wos2.id_wo,
        SUM(rp.qty)::integer AS qty_shipped
    FROM REPORT_PENGIRIMAN rp
    JOIN WORK_ORDER_SHELL_SIZE wss2 ON wss2.id_wo_shell_size = rp.id_wo_shell_size
    JOIN WORK_ORDER_SHELL wos2 ON wos2.id_wo_shell = wss2.id_wo_shell
    GROUP BY wos2.id_wo
) wo_ship ON wo_ship.id_wo = ml.id_wo
WHERE ml.id_wo = sqlc.arg(id_wo)
ORDER BY mli.id_material_list_item ASC;
