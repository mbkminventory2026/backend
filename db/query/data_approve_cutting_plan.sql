-- name: CreateDataApproveCuttingPlan :one
INSERT INTO DATA_APPROVE_CUTTING_PLAN (
    NO_DOKUMEN, TANGGAL, ID_WO
) VALUES (
    sqlc.arg(no_dokumen), sqlc.arg(tanggal)::date, sqlc.arg(id_wo)
)
RETURNING id_dacp, no_dokumen, tanggal, id_wo, created_at;

-- name: GetDataApproveCuttingPlanByID :one
SELECT
    dacp.id_dacp,
    dacp.no_dokumen,
    dacp.tanggal,
    dacp.id_wo,
    dacp.created_at,
    wo.buyer,
    wo.model,
    pci.style,
    pci.colour
FROM DATA_APPROVE_CUTTING_PLAN dacp
JOIN WORK_ORDER wo ON dacp.id_wo = wo.id_wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
WHERE dacp.id_dacp = sqlc.arg(id_dacp)
LIMIT 1;

-- name: GetDataApproveCuttingPlanRows :many
SELECT
    wss.id_wo_shell_size,
    ms.nama_size AS size,
    wss.qty                                                     AS qty_order,
    COALESCE(sp.qty_cutting_plan, 0)::bigint                    AS qty_cutting_plan,
    COALESCE(mk.qty_cutting_actual, 0)::bigint                  AS qty_cutting_actual,
    COALESCE(rc.cutting_report, 0)::bigint                      AS cutting_report,
    (wss.qty - COALESCE(rc.cutting_report, 0))::bigint          AS balance_allowance
FROM WORK_ORDER_SHELL wos
JOIN WORK_ORDER_SHELL_SIZE wss ON wss.id_wo_shell = wos.id_wo_shell
JOIN MASTER_SIZE ms ON ms.id_size = wss.id_size
LEFT JOIN (
    SELECT
        rss.id_wo_shell_size,
        SUM(rss.ratio_plan) AS qty_cutting_plan
    FROM RATIO_SIZE_SPREADING rss
    JOIN RATIO_SPREADING rs ON rs.id_ratio_spreading = rss.id_ratio_spreading
    JOIN WORK_ORDER_SHELL wos2 ON wos2.id_wo_shell = rs.id_wo_shell
    WHERE wos2.id_wo = sqlc.arg(id_wo)
    GROUP BY rss.id_wo_shell_size
) sp ON sp.id_wo_shell_size = wss.id_wo_shell_size
LEFT JOIN (
    SELECT
        rsm.id_wo_shell_size,
        SUM(rsm.ratio_plan) AS qty_cutting_actual
    FROM RATIO_SIZE_MARKER rsm
    JOIN RATIO_MARKER rm ON rm.id_ratio_marker = rsm.id_ratio_marker
    JOIN WORK_ORDER_SHELL wos3 ON wos3.id_wo_shell = rm.id_wo_shell
    WHERE wos3.id_wo = sqlc.arg(id_wo)
    GROUP BY rsm.id_wo_shell_size
) mk ON mk.id_wo_shell_size = wss.id_wo_shell_size
LEFT JOIN (
    SELECT
        rc2.id_wo_shell_size,
        SUM(rc2.qty) AS cutting_report
    FROM REPORT_CUTTING rc2
    GROUP BY rc2.id_wo_shell_size
) rc ON rc.id_wo_shell_size = wss.id_wo_shell_size
WHERE wos.id_wo = sqlc.arg(id_wo)
ORDER BY wos.id_wo_shell ASC, wss.id_wo_shell_size ASC;

-- name: ListDataApproveCuttingPlans :many
SELECT
    dacp.id_dacp,
    dacp.no_dokumen,
    dacp.tanggal,
    dacp.id_wo,
    dacp.created_at,
    wo.buyer,
    wo.model,
    COUNT(*) OVER() AS total_count
FROM DATA_APPROVE_CUTTING_PLAN dacp
JOIN WORK_ORDER wo ON dacp.id_wo = wo.id_wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE (
    sqlc.arg(search_term)::text = ''
    OR dacp.no_dokumen ILIKE '%' || sqlc.arg(search_term)::text || '%'
    OR wo.buyer ILIKE '%' || sqlc.arg(search_term)::text || '%'
    OR wo.model ILIKE '%' || sqlc.arg(search_term)::text || '%'
) AND (
    sqlc.narg(id_mitra)::integer IS NULL
    OR pc.id_mitra = sqlc.narg(id_mitra)::integer
)
ORDER BY dacp.id_dacp DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);
