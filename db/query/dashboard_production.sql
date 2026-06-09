-- name: GetProductionTotalTimelineThisMonth :one
SELECT COUNT(*) FROM TIMELINE_PLAN_PRODUKSI 
WHERE EXTRACT(MONTH FROM created_at) = EXTRACT(MONTH FROM CURRENT_DATE) 
  AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM CURRENT_DATE);

-- name: GetProductionTotalMarkerPlanThisMonth :one
SELECT COUNT(*) FROM MARKER_PLAN 
WHERE EXTRACT(MONTH FROM tanggal_efektif) = EXTRACT(MONTH FROM CURRENT_DATE) 
  AND EXTRACT(YEAR FROM tanggal_efektif) = EXTRACT(YEAR FROM CURRENT_DATE);

-- name: GetProductionTotalSpreadingCuttingPlanThisMonth :one
SELECT COUNT(*) FROM SPREADING_CUTTING_PLAN 
WHERE EXTRACT(MONTH FROM tanggal_efektif) = EXTRACT(MONTH FROM CURRENT_DATE) 
  AND EXTRACT(YEAR FROM tanggal_efektif) = EXTRACT(YEAR FROM CURRENT_DATE);

-- name: GetProductionRecentTimelines :many
SELECT 
    tl.id_timeline,
    tl.tanggal_disusun,
    tl.notes,
    pc.po_number
FROM TIMELINE_PLAN_PRODUKSI tl
JOIN PO_CLIENT pc ON tl.id_po_client = pc.id_po_client
ORDER BY tl.created_at DESC
LIMIT 5;

-- name: GetProductionRecentMarkerPlans :many
SELECT 
    mp.id_marker_plan,
    mp.no_dokumen,
    mp.tanggal_efektif,
    wos.color,
    wo.model
FROM MARKER_PLAN mp
JOIN WORK_ORDER_SHELL wos ON mp.id_wo_shell = wos.id_wo_shell
JOIN WORK_ORDER wo ON wos.id_wo = wo.id_wo
ORDER BY mp.created_at DESC
LIMIT 5;

-- name: GetProductionRecentSpreadingCuttingPlans :many
SELECT 
    scp.id_spreading_cutting_plan,
    scp.no_dokumen,
    scp.tanggal_efektif,
    wo.model
FROM SPREADING_CUTTING_PLAN scp
JOIN WORK_ORDER wo ON scp.id_wo = wo.id_wo
ORDER BY scp.created_at DESC
LIMIT 5;
