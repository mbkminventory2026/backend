-- Dedicated fixed reads for the renderer-neutral Spreading & Cutting Plan
-- export. Marker Plan, Data Approve Cutting Plan, and REPORT_CUTTING are
-- intentionally outside this document's source graph.

-- name: GetSpreadingCuttingPlanExportHeader :one
SELECT
    scp.id_spreading_cutting_plan,
    scp.no_dokumen,
    scp.tanggal_efektif,
    wo.id_wo,
    wo.buyer,
    wo.model,
    pci.style
FROM spreading_cutting_plan scp
JOIN work_order wo ON wo.id_wo = scp.id_wo
JOIN po_client_item pci ON pci.id_po_client_item = wo.id_po_client_item
WHERE scp.id_spreading_cutting_plan = sqlc.arg(id_spreading_cutting_plan);

-- name: ListSpreadingCuttingPlanExportShellSizes :many
SELECT
    wos.id_wo_shell,
    wos.color,
    woss.id_wo_shell_size,
    woss.id_size,
    ms.nama_size,
    woss.qty AS order_qty
FROM work_order_shell wos
LEFT JOIN work_order_shell_size woss ON woss.id_wo_shell = wos.id_wo_shell
LEFT JOIN master_size ms ON ms.id_size = woss.id_size
WHERE wos.id_wo = sqlc.arg(id_wo)
ORDER BY wos.id_wo_shell ASC, woss.id_wo_shell_size ASC NULLS LAST;

-- name: ListSpreadingCuttingPlanExportRatios :many
SELECT
    kscp.id_komponen_spreading,
    kscp.nama_komponen,
    rs.id_ratio_spreading,
    rs.id_wo_shell AS ratio_shell_id,
    ratio_shell.id_wo AS ratio_shell_wo_id,
    rs.cons,
    rs.plan_spreading_gelaran,
    rs.allowance,
    rs.roll_qty,
    rs.sambungan_roll,
    rs.reject,
    rs.lebar_kain,
    rs.ket,
    rss.id_ratio_size_spreading,
    rss.id_wo_shell_size,
    rss.ratio_plan,
    ratio_size.id_wo_shell AS ratio_size_shell_id,
    ratio_size_shell.id_wo AS ratio_size_shell_wo_id,
    ratio_size.id_size
FROM komponen_spreading_cutting_plan kscp
JOIN ratio_spreading rs ON rs.id_komponen_spreading = kscp.id_komponen_spreading
JOIN work_order_shell ratio_shell ON ratio_shell.id_wo_shell = rs.id_wo_shell
LEFT JOIN ratio_size_spreading rss ON rss.id_ratio_spreading = rs.id_ratio_spreading
LEFT JOIN work_order_shell_size ratio_size ON ratio_size.id_wo_shell_size = rss.id_wo_shell_size
LEFT JOIN work_order_shell ratio_size_shell ON ratio_size_shell.id_wo_shell = ratio_size.id_wo_shell
WHERE kscp.id_spreading_cutting_plan = sqlc.arg(id_spreading_cutting_plan)
ORDER BY kscp.id_komponen_spreading ASC, rs.id_ratio_spreading ASC,
         rss.id_ratio_size_spreading ASC NULLS LAST;

-- name: ListSpreadingCuttingPlanExportReceivedByShell :many
SELECT
    mli.id_wo_shell,
    source_shell.id_wo AS source_shell_wo_id,
    SUM(received.qty)::bigint AS received_actual
FROM material_list ml
JOIN material_list_item mli ON mli.id_material_list = ml.id_material_list
JOIN received ON received.id_material_list_item = mli.id_material_list_item
JOIN work_order_shell source_shell ON source_shell.id_wo_shell = mli.id_wo_shell
WHERE ml.id_wo = sqlc.arg(id_wo)
GROUP BY mli.id_wo_shell, source_shell.id_wo
ORDER BY mli.id_wo_shell ASC;
