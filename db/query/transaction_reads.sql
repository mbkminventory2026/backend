-- name: ListWorkOrders :many
SELECT
    wo.id_wo,
    wo.buyer,
    wo.model,
    wo.qty,
    wo.fob_cmt,
    wo.delivery,
    wo.id_po_client_item,
    wo.status,
    wo.closed_by_user_id,
    wo.closed_at,
    wo.created_at,
    pc.po_number,
    pci.style AS po_client_item_style,
    COUNT(*) OVER() AS total_count
FROM WORK_ORDER wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE (
    sqlc.arg(search_term) = '' OR
    wo.buyer ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wo.model ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pc.po_number ILIKE '%' || sqlc.arg(search_term) || '%'
)
ORDER BY wo.id_wo DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetWorkOrderDetail :one
SELECT
    wo.id_wo,
    wo.buyer,
    wo.model,
    wo.qty,
    wo.fob_cmt,
    wo.delivery,
    wo.id_po_client_item,
    wo.status,
    wo.closed_by_user_id,
    wo.closed_at,
    wo.created_at,
    pc.po_number,
    pci.style AS po_client_item_style
FROM WORK_ORDER wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE wo.id_wo = sqlc.arg(id_wo)
LIMIT 1;

-- name: ListWorkOrderShellsByWorkOrderID :many
SELECT
    id_wo_shell,
    fabric,
    cons,
    color,
    allow,
    berat_1_yd,
    id_wo,
    created_at
FROM WORK_ORDER_SHELL
WHERE id_wo = sqlc.arg(id_wo)
ORDER BY id_wo_shell ASC;

-- name: ListWorkOrderShellSizesByWorkOrderID :many
SELECT
    woss.id_wo_shell_size,
    woss.size,
    woss.qty,
    woss.ratio,
    woss.id_wo_shell,
    woss.created_at
FROM WORK_ORDER_SHELL_SIZE woss
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = woss.id_wo_shell
WHERE wos.id_wo = sqlc.arg(id_wo)
ORDER BY woss.id_wo_shell_size ASC;

-- name: ListWorkOrderTrimsByWorkOrderID :many
SELECT
    id_wo_trim,
    item,
    description,
    color,
    code,
    cons,
    qty,
    uom,
    position,
    created_by,
    allow,
    id_wo,
    created_at
FROM WORK_ORDER_TRIM
WHERE id_wo = sqlc.arg(id_wo)
ORDER BY id_wo_trim ASC;

-- name: ListMaterialListsByWorkOrderID :many
SELECT
    id_material_list,
    description,
    size,
    color,
    uom,
    id_wo,
    created_at
FROM MATERIAL_LIST
WHERE id_wo = sqlc.arg(id_wo)
ORDER BY id_material_list ASC;

-- name: ListPOClients :many
SELECT
    pc.id_po_client,
    pc.po_number,
    pc.tanggal,
    pc.season,
    pc.delivery,
    pc.payment_term,
    pc.file,
    pc.id_mitra,
    pc.created_at,
    m.nama_perusahaan AS mitra_name,
    COUNT(*) OVER() AS total_count
FROM PO_CLIENT pc
JOIN MITRA m ON m.id_mitra = pc.id_mitra
WHERE (
    sqlc.arg(search_term) = '' OR
    pc.po_number ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pc.season ILIKE '%' || sqlc.arg(search_term) || '%' OR
    m.nama_perusahaan ILIKE '%' || sqlc.arg(search_term) || '%'
)
ORDER BY pc.id_po_client DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetPOClientDetail :one
SELECT
    pc.id_po_client,
    pc.po_number,
    pc.tanggal,
    pc.season,
    pc.delivery,
    pc.payment_term,
    pc.file,
    pc.id_mitra,
    pc.created_at,
    m.nama_perusahaan AS mitra_name
FROM PO_CLIENT pc
JOIN MITRA m ON m.id_mitra = pc.id_mitra
WHERE pc.id_po_client = sqlc.arg(id_po_client)
LIMIT 1;

-- name: ListPOClientItemsByPOClientID :many
SELECT
    id_po_client_item,
    id_po_client,
    style,
    colour,
    description,
    qty,
    price,
    created_at
FROM PO_CLIENT_ITEM
WHERE id_po_client = sqlc.arg(id_po_client)
ORDER BY id_po_client_item ASC;

-- name: ListPenanggungJawabByPOClientID :many
SELECT
    id_penanggung_jawab,
    nama,
    no_telp,
    email,
    id_po_client,
    created_at
FROM PENANGGUNG_JAWAB
WHERE id_po_client = sqlc.arg(id_po_client)
ORDER BY id_penanggung_jawab ASC;

-- name: ListPRInternals :many
SELECT
    pr.id_pr_internal,
    pr.tanggal,
    pr.nama,
    pr.departemen,
    pr.vendor_name,
    pr.vendor_address,
    pr.vendor_telp,
    pr.projek,
    pr.id_wo,
    pr.id_user,
    pr.status,
    pr.approved_by_user_id,
    pr.approved_at,
    pr.created_at,
    COUNT(*) OVER() AS total_count
FROM PR_INTERNAL pr
WHERE (
    sqlc.arg(search_term) = '' OR
    pr.nama ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pr.vendor_name ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pr.projek ILIKE '%' || sqlc.arg(search_term) || '%'
)
ORDER BY pr.id_pr_internal DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetPRInternalDetail :one
SELECT
    id_pr_internal,
    tanggal,
    nama,
    departemen,
    vendor_name,
    vendor_address,
    vendor_telp,
    projek,
    id_wo,
    id_user,
    status,
    approved_by_user_id,
    approved_at,
    created_at
FROM PR_INTERNAL
WHERE id_pr_internal = sqlc.arg(id_pr_internal)
LIMIT 1;

-- name: ListPRInternalItemsByPRInternalID :many
SELECT
    id_pr_internal_item,
    id_pr_internal,
    item,
    description,
    qty,
    unit,
    est_price,
    created_at
FROM PR_INTERNAL_ITEM
WHERE id_pr_internal = sqlc.arg(id_pr_internal)
ORDER BY id_pr_internal_item ASC;

-- name: ListPOInternals :many
SELECT
    poi.id_po_internal,
    poi.tanggal,
    poi.nama_po,
    poi.supplier_name,
    poi.supplier_addr,
    poi.supplier_contact,
    poi.supplier_email,
    poi.supplier_telp,
    poi.supplier_fax,
    poi.currency,
    poi.cpo,
    poi.term,
    poi.ship_date,
    poi.id_pr_internal,
    poi.created_at,
    COUNT(*) OVER() AS total_count
FROM PO_INTERNAL poi
WHERE (
    sqlc.arg(search_term) = '' OR
    poi.nama_po ILIKE '%' || sqlc.arg(search_term) || '%' OR
    poi.supplier_name ILIKE '%' || sqlc.arg(search_term) || '%' OR
    poi.cpo ILIKE '%' || sqlc.arg(search_term) || '%'
)
ORDER BY poi.id_po_internal DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetPOInternalDetail :one
SELECT
    id_po_internal,
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
    id_pr_internal,
    created_at
FROM PO_INTERNAL
WHERE id_po_internal = sqlc.arg(id_po_internal)
LIMIT 1;

-- name: ListPOInternalItemsByPOInternalID :many
SELECT
    id_po_internal_item,
    id_po_internal,
    item,
    description,
    qty,
    unit,
    unit_price,
    created_at
FROM PO_INTERNAL_ITEM
WHERE id_po_internal = sqlc.arg(id_po_internal)
ORDER BY id_po_internal_item ASC;

-- name: ListPackingLists :many
SELECT
    pl.id_packing_list,
    pl.total_garment_per_box,
    pl.total_reject,
    pl.id_wo,
    pl.id_surat_jalan_internal,
    pl.created_at,
    wo.buyer,
    wo.model,
    COUNT(*) OVER() AS total_count
FROM PACKING_LIST pl
JOIN WORK_ORDER wo ON wo.id_wo = pl.id_wo
WHERE (
    sqlc.arg(search_term) = '' OR
    wo.buyer ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wo.model ILIKE '%' || sqlc.arg(search_term) || '%'
)
ORDER BY pl.id_packing_list DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetPackingListDetail :one
SELECT
    pl.id_packing_list,
    pl.total_garment_per_box,
    pl.total_reject,
    pl.id_wo,
    pl.id_surat_jalan_internal,
    pl.created_at,
    wo.buyer,
    wo.model
FROM PACKING_LIST pl
JOIN WORK_ORDER wo ON wo.id_wo = pl.id_wo
WHERE pl.id_packing_list = sqlc.arg(id_packing_list)
LIMIT 1;

-- name: ListPackingListItemsByPackingListID :many
SELECT
    id_packing_list_item,
    id_packing_list,
    color,
    qty_box,
    qty_per_box,
    box_no_start,
    box_no_end,
    note,
    created_at
FROM PACKING_LIST_ITEM
WHERE id_packing_list = sqlc.arg(id_packing_list)
ORDER BY id_packing_list_item ASC;

-- name: ListPackingListItemSizesByPackingListID :many
SELECT
    plis.id_packing_list_item_size,
    plis.qty,
    plis.id_packing_list_item,
    plis.created_at
FROM PACKING_LIST_ITEM_SIZE plis
JOIN PACKING_LIST_ITEM pli ON pli.id_packing_list_item = plis.id_packing_list_item
WHERE pli.id_packing_list = sqlc.arg(id_packing_list)
ORDER BY plis.id_packing_list_item_size ASC;

-- name: ListSuratJalanClients :many
SELECT
    sjc.id_surat_jalan_client,
    sjc.tanggal,
    sjc.qty,
    sjc.keterangan,
    sjc.id_material_list,
    sjc.created_at,
    ml.description AS material_description,
    ml.id_wo,
    COUNT(*) OVER() AS total_count
FROM SURAT_JALAN_CLIENT sjc
JOIN MATERIAL_LIST ml ON ml.id_material_list = sjc.id_material_list
WHERE (
    sqlc.arg(search_term) = '' OR
    sjc.keterangan ILIKE '%' || sqlc.arg(search_term) || '%' OR
    ml.description ILIKE '%' || sqlc.arg(search_term) || '%'
)
ORDER BY sjc.id_surat_jalan_client DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetSuratJalanClientDetail :one
SELECT
    sjc.id_surat_jalan_client,
    sjc.tanggal,
    sjc.qty,
    sjc.keterangan,
    sjc.id_material_list,
    sjc.created_at,
    ml.description AS material_description,
    ml.id_wo
FROM SURAT_JALAN_CLIENT sjc
JOIN MATERIAL_LIST ml ON ml.id_material_list = sjc.id_material_list
WHERE sjc.id_surat_jalan_client = sqlc.arg(id_surat_jalan_client)
LIMIT 1;

-- name: ListSuratJalanInternals :many
SELECT
    sji.id_surat_jalan_internal,
    sji.created_at,
    COUNT(*) OVER() AS total_count
FROM SURAT_JALAN_INTERNAL sji
ORDER BY sji.id_surat_jalan_internal DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetSuratJalanInternalDetail :one
SELECT
    id_surat_jalan_internal,
    created_at
FROM SURAT_JALAN_INTERNAL
WHERE id_surat_jalan_internal = sqlc.arg(id_surat_jalan_internal)
LIMIT 1;

