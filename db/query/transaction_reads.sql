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
    pc.id_po_client,
    pci.style AS po_client_item_style,
    EXISTS (
        SELECT 1 FROM RETUR_CLIENT rc WHERE rc.id_wo = wo.id_wo
    )::boolean AS has_retur,
    COALESCE(
        (SELECT rc.file FROM RETUR_CLIENT rc WHERE rc.id_wo = wo.id_wo LIMIT 1),
        ''
    )::text AS retur_file,
    COUNT(*) OVER() AS total_count
FROM v_work_order wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE (
    sqlc.arg(search_term) = '' OR
    wo.buyer ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wo.model ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pc.po_number ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pci.style ILIKE '%' || sqlc.arg(search_term) || '%'
 ) AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN wo.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN wo.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_wo' AND NOT sqlc.arg(sort_desc)::bool THEN wo.id_wo END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_wo' AND sqlc.arg(sort_desc)::bool THEN wo.id_wo END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND NOT sqlc.arg(sort_desc)::bool THEN wo.buyer END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND sqlc.arg(sort_desc)::bool THEN wo.buyer END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model' AND NOT sqlc.arg(sort_desc)::bool THEN wo.model END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model' AND sqlc.arg(sort_desc)::bool THEN wo.model END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'qty' AND NOT sqlc.arg(sort_desc)::bool THEN wo.qty END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'qty' AND sqlc.arg(sort_desc)::bool THEN wo.qty END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'status' AND NOT sqlc.arg(sort_desc)::bool THEN wo.status END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'status' AND sqlc.arg(sort_desc)::bool THEN wo.status END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'po_number' AND NOT sqlc.arg(sort_desc)::bool THEN pc.po_number END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'po_number' AND sqlc.arg(sort_desc)::bool THEN pc.po_number END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'po_client_item_style' AND NOT sqlc.arg(sort_desc)::bool THEN pci.style END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'po_client_item_style' AND sqlc.arg(sort_desc)::bool THEN pci.style END DESC,
    wo.id_wo DESC
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
FROM v_work_order wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE wo.id_wo = sqlc.arg(id_wo)
AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
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
    ml.id_material_list,
    mli.description,
    ''::text AS size,
    COALESCE(wos.color, wot.color)::text AS color,
    COALESCE(wot.uom, 'yds')::text AS uom,
    COALESCE(wos.id_wo, wot.id_wo, mli.id_wo)::integer AS id_wo,
    ml.created_at
FROM MATERIAL_LIST ml
JOIN MATERIAL_LIST_ITEM mli ON mli.id_material_list_item = ml.id_material_list_item
LEFT JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = mli.id_wo_shell
LEFT JOIN WORK_ORDER_TRIM wot ON wot.id_wo_trim = mli.id_wo_trim
WHERE COALESCE(wos.id_wo, wot.id_wo, mli.id_wo) = sqlc.arg(id_wo)
ORDER BY ml.id_material_list ASC;

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
    EXISTS (
        SELECT 1
        FROM PO_CLIENT_ITEM pci
        JOIN WORK_ORDER wo ON wo.id_po_client_item = pci.id_po_client_item
        JOIN RETUR_CLIENT rc ON rc.id_wo = wo.id_wo
        WHERE pci.id_po_client = pc.id_po_client
    )::boolean AS has_retur,
    COUNT(*) OVER() AS total_count
FROM PO_CLIENT pc
JOIN MITRA m ON m.id_mitra = pc.id_mitra
WHERE (
    sqlc.arg(search_term) = '' OR
    pc.po_number ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pc.season ILIKE '%' || sqlc.arg(search_term) || '%' OR
    m.nama_perusahaan ILIKE '%' || sqlc.arg(search_term) || '%'
) AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN pc.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN pc.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_po_client' AND NOT sqlc.arg(sort_desc)::bool THEN pc.id_po_client END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_po_client' AND sqlc.arg(sort_desc)::bool THEN pc.id_po_client END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'po_number' AND NOT sqlc.arg(sort_desc)::bool THEN pc.po_number END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'po_number' AND sqlc.arg(sort_desc)::bool THEN pc.po_number END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal' AND NOT sqlc.arg(sort_desc)::bool THEN pc.tanggal END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal' AND sqlc.arg(sort_desc)::bool THEN pc.tanggal END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'season' AND NOT sqlc.arg(sort_desc)::bool THEN pc.season END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'season' AND sqlc.arg(sort_desc)::bool THEN pc.season END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'delivery' AND NOT sqlc.arg(sort_desc)::bool THEN pc.delivery END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'delivery' AND sqlc.arg(sort_desc)::bool THEN pc.delivery END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'mitra_name' AND NOT sqlc.arg(sort_desc)::bool THEN m.nama_perusahaan END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'mitra_name' AND sqlc.arg(sort_desc)::bool THEN m.nama_perusahaan END DESC,
    pc.id_po_client DESC
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
AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
LIMIT 1;

-- name: ListPOClientItemsByPOClientID :many
SELECT
    pci.id_po_client_item,
    pci.id_po_client,
    pci.style,
    pci.colour,
    pci.description,
    pci.qty,
    pci.price,
    pci.created_at,
    wo.id_wo,
    wo.status AS wo_status,
    EXISTS (
        SELECT 1 FROM RETUR_CLIENT rc WHERE rc.id_wo = wo.id_wo
    )::boolean AS has_retur
FROM PO_CLIENT_ITEM pci
LEFT JOIN v_work_order wo ON wo.id_po_client_item = pci.id_po_client_item
WHERE pci.id_po_client = sqlc.arg(id_po_client)
ORDER BY pci.id_po_client_item ASC;



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
FROM v_pr_internal pr
WHERE (
    sqlc.arg(search_term) = '' OR
    pr.nama ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pr.departemen ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pr.vendor_name ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pr.projek ILIKE '%' || sqlc.arg(search_term) || '%' OR
    pr.status ILIKE '%' || sqlc.arg(search_term) || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN pr.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN pr.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_pr_internal' AND NOT sqlc.arg(sort_desc)::bool THEN pr.id_pr_internal END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_pr_internal' AND sqlc.arg(sort_desc)::bool THEN pr.id_pr_internal END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal' AND NOT sqlc.arg(sort_desc)::bool THEN pr.tanggal END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal' AND sqlc.arg(sort_desc)::bool THEN pr.tanggal END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama' AND NOT sqlc.arg(sort_desc)::bool THEN pr.nama END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama' AND sqlc.arg(sort_desc)::bool THEN pr.nama END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'departemen' AND NOT sqlc.arg(sort_desc)::bool THEN pr.departemen END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'departemen' AND sqlc.arg(sort_desc)::bool THEN pr.departemen END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'vendor_name' AND NOT sqlc.arg(sort_desc)::bool THEN pr.vendor_name END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'vendor_name' AND sqlc.arg(sort_desc)::bool THEN pr.vendor_name END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'projek' AND NOT sqlc.arg(sort_desc)::bool THEN pr.projek END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'projek' AND sqlc.arg(sort_desc)::bool THEN pr.projek END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'status' AND NOT sqlc.arg(sort_desc)::bool THEN pr.status END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'status' AND sqlc.arg(sort_desc)::bool THEN pr.status END DESC,
    pr.id_pr_internal DESC
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
FROM v_pr_internal
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
    poi.cpo ILIKE '%' || sqlc.arg(search_term) || '%' OR
    poi.currency ILIKE '%' || sqlc.arg(search_term) || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN poi.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN poi.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_po_internal' AND NOT sqlc.arg(sort_desc)::bool THEN poi.id_po_internal END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_po_internal' AND sqlc.arg(sort_desc)::bool THEN poi.id_po_internal END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal' AND NOT sqlc.arg(sort_desc)::bool THEN poi.tanggal END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal' AND sqlc.arg(sort_desc)::bool THEN poi.tanggal END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_po' AND NOT sqlc.arg(sort_desc)::bool THEN poi.nama_po END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'nama_po' AND sqlc.arg(sort_desc)::bool THEN poi.nama_po END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'supplier_name' AND NOT sqlc.arg(sort_desc)::bool THEN poi.supplier_name END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'supplier_name' AND sqlc.arg(sort_desc)::bool THEN poi.supplier_name END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'currency' AND NOT sqlc.arg(sort_desc)::bool THEN poi.currency END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'currency' AND sqlc.arg(sort_desc)::bool THEN poi.currency END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'cpo' AND NOT sqlc.arg(sort_desc)::bool THEN poi.cpo END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'cpo' AND sqlc.arg(sort_desc)::bool THEN poi.cpo END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'ship_date' AND NOT sqlc.arg(sort_desc)::bool THEN poi.ship_date END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'ship_date' AND sqlc.arg(sort_desc)::bool THEN poi.ship_date END DESC,
    poi.id_po_internal DESC
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
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE (
    sqlc.arg(search_term) = '' OR
    wo.buyer ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wo.model ILIKE '%' || sqlc.arg(search_term) || '%'
 ) AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN pl.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN pl.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_packing_list' AND NOT sqlc.arg(sort_desc)::bool THEN pl.id_packing_list END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_packing_list' AND sqlc.arg(sort_desc)::bool THEN pl.id_packing_list END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'total_garment_per_box' AND NOT sqlc.arg(sort_desc)::bool THEN pl.total_garment_per_box END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'total_garment_per_box' AND sqlc.arg(sort_desc)::bool THEN pl.total_garment_per_box END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'total_reject' AND NOT sqlc.arg(sort_desc)::bool THEN pl.total_reject END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'total_reject' AND sqlc.arg(sort_desc)::bool THEN pl.total_reject END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND NOT sqlc.arg(sort_desc)::bool THEN wo.buyer END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND sqlc.arg(sort_desc)::bool THEN wo.buyer END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model' AND NOT sqlc.arg(sort_desc)::bool THEN wo.model END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model' AND sqlc.arg(sort_desc)::bool THEN wo.model END DESC,
    pl.id_packing_list DESC
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
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE pl.id_packing_list = sqlc.arg(id_packing_list)
AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
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
    plis.id_wo_shell_size,
    plis.created_at
FROM PACKING_LIST_ITEM_SIZE plis
JOIN PACKING_LIST_ITEM pli ON pli.id_packing_list_item = plis.id_packing_list_item
WHERE pli.id_packing_list = sqlc.arg(id_packing_list)
ORDER BY plis.id_packing_list_item_size ASC;

-- name: ListPackingListRejectSizesByPackingListID :many
SELECT
    plrs.id_packing_list_reject_size,
    plrs.qty,
    plrs.id_packing_list,
    plrs.id_wo_shell_size,
    plrs.created_at
FROM PACKING_LIST_REJECT_SIZE plrs
WHERE plrs.id_packing_list = sqlc.arg(id_packing_list)
ORDER BY plrs.id_packing_list_reject_size ASC;

-- name: ListSuratJalanClients :many
SELECT
    sjc.id_surat_jalan_client,
    sjc.tanggal,
    sjc.qty,
    sjc.keterangan,
    sjc.id_material_list_item AS id_material_list,
    sjc.created_at,
    mli.description AS material_description,
    COALESCE(wos.id_wo, wot.id_wo, mli.id_wo)::integer AS id_wo,
    COUNT(*) OVER() AS total_count
FROM SURAT_JALAN_CLIENT sjc
JOIN MATERIAL_LIST_ITEM mli ON mli.id_material_list_item = sjc.id_material_list_item
LEFT JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = mli.id_wo_shell
LEFT JOIN WORK_ORDER_TRIM wot ON wot.id_wo_trim = mli.id_wo_trim
JOIN WORK_ORDER wo ON wo.id_wo = COALESCE(wos.id_wo, wot.id_wo, mli.id_wo)
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE (
    sqlc.arg(search_term) = '' OR
    sjc.keterangan ILIKE '%' || sqlc.arg(search_term) || '%' OR
    mli.description ILIKE '%' || sqlc.arg(search_term) || '%'
 ) AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN sjc.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN sjc.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_surat_jalan_client' AND NOT sqlc.arg(sort_desc)::bool THEN sjc.id_surat_jalan_client END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_surat_jalan_client' AND sqlc.arg(sort_desc)::bool THEN sjc.id_surat_jalan_client END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal' AND NOT sqlc.arg(sort_desc)::bool THEN sjc.tanggal END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal' AND sqlc.arg(sort_desc)::bool THEN sjc.tanggal END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'qty' AND NOT sqlc.arg(sort_desc)::bool THEN sjc.qty END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'qty' AND sqlc.arg(sort_desc)::bool THEN sjc.qty END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'keterangan' AND NOT sqlc.arg(sort_desc)::bool THEN sjc.keterangan END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'keterangan' AND sqlc.arg(sort_desc)::bool THEN sjc.keterangan END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'material_description' AND NOT sqlc.arg(sort_desc)::bool THEN mli.description END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'material_description' AND sqlc.arg(sort_desc)::bool THEN mli.description END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_wo' AND NOT sqlc.arg(sort_desc)::bool THEN COALESCE(wos.id_wo, wot.id_wo, mli.id_wo) END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_wo' AND sqlc.arg(sort_desc)::bool THEN COALESCE(wos.id_wo, wot.id_wo, mli.id_wo) END DESC,
    sjc.id_surat_jalan_client DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetSuratJalanClientDetail :one
SELECT
    sjc.id_surat_jalan_client,
    sjc.tanggal,
    sjc.qty,
    sjc.keterangan,
    sjc.id_material_list_item AS id_material_list,
    sjc.created_at,
    mli.description AS material_description,
    COALESCE(wos.id_wo, wot.id_wo, mli.id_wo)::integer AS id_wo
FROM SURAT_JALAN_CLIENT sjc
JOIN MATERIAL_LIST_ITEM mli ON mli.id_material_list_item = sjc.id_material_list_item
LEFT JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = mli.id_wo_shell
LEFT JOIN WORK_ORDER_TRIM wot ON wot.id_wo_trim = mli.id_wo_trim
JOIN WORK_ORDER wo ON wo.id_wo = COALESCE(wos.id_wo, wot.id_wo, mli.id_wo)
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE sjc.id_surat_jalan_client = sqlc.arg(id_surat_jalan_client)
AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
LIMIT 1;

-- name: ListSuratJalanInternals :many
SELECT
    sji.id_surat_jalan_internal,
    sji.created_at,
    COUNT(*) OVER() AS total_count
FROM SURAT_JALAN_INTERNAL sji
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN sji.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN sji.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_surat_jalan_internal' AND NOT sqlc.arg(sort_desc)::bool THEN sji.id_surat_jalan_internal END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_surat_jalan_internal' AND sqlc.arg(sort_desc)::bool THEN sji.id_surat_jalan_internal END DESC,
    sji.id_surat_jalan_internal DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetSuratJalanInternalDetail :one
SELECT
    id_surat_jalan_internal,
    created_at
FROM SURAT_JALAN_INTERNAL
WHERE id_surat_jalan_internal = sqlc.arg(id_surat_jalan_internal)
LIMIT 1;

