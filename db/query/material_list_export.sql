-- Dedicated, batched reads for the Material List export composition.  These
-- queries deliberately expose source and applicability separately.

-- name: GetMaterialListExportHeader :one
SELECT
    ml.id_material_list,
    ml.name AS material_list_name,
    wo.id_wo,
    wo.buyer,
    wo.model,
    pci.style,
    wo.qty AS wo_qty,
    wo.fob_cmt,
    wo.delivery
FROM material_list ml
JOIN work_order wo ON wo.id_wo = ml.id_wo
JOIN po_client_item pci ON pci.id_po_client_item = wo.id_po_client_item
WHERE ml.id_material_list = sqlc.arg(id_material_list);

-- name: ListMaterialListExportItems :many
SELECT
    mli.id_material_list_item,
    mli.item,
    mli.description,
    mli.category,
    mli.unit,
    mli.cons_per_pc,
    mli.qty_wo_scope,
    mli.id_qty_wo_shell,
    mli.id_qty_wo_size,
    source_shell.id_wo_shell AS source_shell_id,
    source_shell.deskripsi AS source_shell_description,
    source_shell.color AS source_shell_color,
    source_shell.allow AS source_shell_allow,
    source_trim.id_wo_trim AS source_trim_id,
    source_trim.item AS source_trim_item,
    source_trim.description AS source_trim_description,
    source_trim.color AS source_trim_color,
    source_trim.allow AS source_trim_allow
FROM material_list_item mli
JOIN material_list ml ON ml.id_material_list = mli.id_material_list
LEFT JOIN work_order_shell source_shell ON source_shell.id_wo_shell = mli.id_wo_shell AND source_shell.id_wo = ml.id_wo
LEFT JOIN work_order_trim source_trim ON source_trim.id_wo_trim = mli.id_wo_trim AND source_trim.id_wo = ml.id_wo
WHERE mli.id_material_list = sqlc.arg(id_material_list)
ORDER BY mli.id_material_list_item ASC;

-- name: ListMaterialListExportShellSizes :many
SELECT
    wos.id_wo_shell,
    wos.color,
    wos.deskripsi AS shell_description,
    woss.id_wo_shell_size,
    woss.id_size,
    ms.nama_size,
    woss.qty AS order_qty
FROM work_order_shell wos
LEFT JOIN work_order_shell_size woss ON woss.id_wo_shell = wos.id_wo_shell
LEFT JOIN master_size ms ON ms.id_size = woss.id_size
WHERE wos.id_wo = sqlc.arg(id_wo)
ORDER BY wos.id_wo_shell ASC, ms.nama_size ASC NULLS LAST, woss.id_wo_shell_size ASC;

-- name: ListMaterialListExportMarkerPlans :many
SELECT id_marker_plan, id_wo_shell
FROM marker_plan
WHERE id_wo_shell IN (
    SELECT id_wo_shell FROM work_order_shell WHERE id_wo = sqlc.arg(id_wo)
)
ORDER BY id_wo_shell ASC, id_marker_plan ASC;

-- name: ListMaterialListExportMarkerRatios :many
SELECT
    mp.id_marker_plan,
    mp.id_wo_shell AS marker_plan_shell_id,
    rm.id_ratio_marker,
    rm.id_wo_shell AS ratio_shell_id,
    rm.plan_spreading_gelaran,
    rsm.id_ratio_size_marker,
    rsm.id_wo_shell_size,
    rsm.ratio_plan,
    woss.id_wo_shell AS ratio_size_shell_id,
    woss.id_size
FROM marker_plan mp
JOIN komponen_marker_plan kmp ON kmp.id_marker_plan = mp.id_marker_plan
JOIN ratio_marker rm ON rm.id_komponen_marker = kmp.id_komponen_marker
JOIN ratio_size_marker rsm ON rsm.id_ratio_marker = rm.id_ratio_marker
JOIN work_order_shell_size woss ON woss.id_wo_shell_size = rsm.id_wo_shell_size
JOIN work_order_shell shell ON shell.id_wo_shell = mp.id_wo_shell
WHERE shell.id_wo = sqlc.arg(id_wo)
ORDER BY mp.id_marker_plan ASC, rm.id_ratio_marker ASC, rsm.id_ratio_size_marker ASC;

-- name: ListMaterialListExportCutting :many
SELECT
    woss.id_wo_shell_size,
    COALESCE(SUM(rc.qty), 0)::bigint AS actual_cut_qty
FROM report_cutting rc
JOIN work_order_shell_size woss ON woss.id_wo_shell_size = rc.id_wo_shell_size
JOIN work_order_shell wos ON wos.id_wo_shell = woss.id_wo_shell
WHERE wos.id_wo = sqlc.arg(id_wo)
GROUP BY woss.id_wo_shell_size
ORDER BY woss.id_wo_shell_size ASC;

-- name: ListMaterialListExportSuratJalanByDate :many
SELECT
    sjc.id_material_list_item,
    sjc.tanggal,
    SUM(sjc.qty)::bigint AS qty
FROM surat_jalan_client sjc
JOIN material_list_item mli ON mli.id_material_list_item = sjc.id_material_list_item
WHERE mli.id_material_list = sqlc.arg(id_material_list)
GROUP BY sjc.id_material_list_item, sjc.tanggal
ORDER BY sjc.tanggal ASC, sjc.id_material_list_item ASC;

-- name: ListMaterialListExportReceivedByDate :many
SELECT
    r.id_material_list_item,
    r.tanggal,
    SUM(r.qty)::bigint AS qty
FROM received r
JOIN material_list_item mli ON mli.id_material_list_item = r.id_material_list_item
WHERE mli.id_material_list = sqlc.arg(id_material_list)
GROUP BY r.id_material_list_item, r.tanggal
ORDER BY r.tanggal ASC, r.id_material_list_item ASC;

-- name: ListMaterialListExportReconciliationRows :many
SELECT
    rmr.id_material_list_item,
    rmr.reject_qty,
    rmr.retur_qty,
    rmr.keterangan
FROM rekonsiliasi_material_row rmr
JOIN rekonsiliasi r ON r.id_rekonsiliasi = rmr.id_rekonsiliasi
JOIN material_list_item mli ON mli.id_material_list_item = rmr.id_material_list_item
WHERE r.id_wo = sqlc.arg(id_wo)
  AND mli.id_material_list = sqlc.arg(id_material_list)
ORDER BY rmr.id_material_list_item ASC, rmr.id_rekonsiliasi_material_row ASC;
