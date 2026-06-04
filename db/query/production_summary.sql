-- name: ListProductionSummary :many
WITH cutting_agg AS (
    SELECT
        id_wo_shell_size,
        SUM(qty)::bigint AS cutting_qty,
        MAX(created_at) AS last_created_at
    FROM REPORT_CUTTING
    GROUP BY id_wo_shell_size
),
sewing_agg AS (
    SELECT
        id_wo_shell_size,
        SUM(qty)::bigint AS sewing_qty,
        MAX(created_at) AS last_created_at
    FROM REPORT_SEWING
    GROUP BY id_wo_shell_size
),
qc_agg AS (
    SELECT
        id_wo_shell_size,
        SUM(qty)::bigint AS qc_pass_qty,
        MAX(created_at) AS last_created_at
    FROM REPORT_QC_FINISH
    GROUP BY id_wo_shell_size
),
packing_agg AS (
    SELECT
        id_wo_shell_size,
        SUM(qty)::bigint AS packing_qty,
        MAX(created_at) AS last_created_at
    FROM REPORT_PACKING
    GROUP BY id_wo_shell_size
),
shipping_agg AS (
    SELECT
        id_wo_shell_size,
        SUM(qty)::bigint AS shipped_qty,
        MAX(created_at) AS last_created_at
    FROM REPORT_PENGIRIMAN
    GROUP BY id_wo_shell_size
)
SELECT
    woss.id_wo_shell_size,
    wo.id_wo,
    wo.model AS model_name,
    woss.size,
    woss.qty AS target_qty,
    COALESCE(ca.cutting_qty, 0)::int AS cutting_qty,
    COALESCE(sa.sewing_qty, 0)::int AS sewing_qty,
    COALESCE(qa.qc_pass_qty, 0)::int AS qc_pass_qty,
    COALESCE(pa.packing_qty, 0)::int AS packing_qty,
    COALESCE(sha.shipped_qty, 0)::int AS shipped_qty,
    GREATEST(
        woss.created_at,
        COALESCE(ca.last_created_at, woss.created_at),
        COALESCE(sa.last_created_at, woss.created_at),
        COALESCE(qa.last_created_at, woss.created_at),
        COALESCE(pa.last_created_at, woss.created_at),
        COALESCE(sha.last_created_at, woss.created_at)
    )::timestamptz AS last_updated,
    COUNT(*) OVER() AS total_count
FROM WORK_ORDER_SHELL_SIZE woss
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = woss.id_wo_shell
JOIN WORK_ORDER wo ON wo.id_wo = wos.id_wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
LEFT JOIN cutting_agg ca ON ca.id_wo_shell_size = woss.id_wo_shell_size
LEFT JOIN sewing_agg sa ON sa.id_wo_shell_size = woss.id_wo_shell_size
LEFT JOIN qc_agg qa ON qa.id_wo_shell_size = woss.id_wo_shell_size
LEFT JOIN packing_agg pa ON pa.id_wo_shell_size = woss.id_wo_shell_size
LEFT JOIN shipping_agg sha ON sha.id_wo_shell_size = woss.id_wo_shell_size
WHERE (
    sqlc.arg(id_wo)::int = 0 OR
    wo.id_wo = sqlc.arg(id_wo)::int
) AND (
    sqlc.arg(id_wo_shell_size)::int = 0 OR
    woss.id_wo_shell_size = sqlc.arg(id_wo_shell_size)::int
) AND (
    sqlc.arg(search_term)::text = '' OR
    wo.model ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    woss.size ILIKE '%' || sqlc.arg(search_term)::text || '%'
 ) AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'last_updated' AND NOT sqlc.arg(sort_desc)::bool THEN GREATEST(
        woss.created_at,
        COALESCE(ca.last_created_at, woss.created_at),
        COALESCE(sa.last_created_at, woss.created_at),
        COALESCE(qa.last_created_at, woss.created_at),
        COALESCE(pa.last_created_at, woss.created_at),
        COALESCE(sha.last_created_at, woss.created_at)
    ) END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'last_updated' AND sqlc.arg(sort_desc)::bool THEN GREATEST(
        woss.created_at,
        COALESCE(ca.last_created_at, woss.created_at),
        COALESCE(sa.last_created_at, woss.created_at),
        COALESCE(qa.last_created_at, woss.created_at),
        COALESCE(pa.last_created_at, woss.created_at),
        COALESCE(sha.last_created_at, woss.created_at)
    ) END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_wo_shell_size' AND NOT sqlc.arg(sort_desc)::bool THEN woss.id_wo_shell_size END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_wo_shell_size' AND sqlc.arg(sort_desc)::bool THEN woss.id_wo_shell_size END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model_name' AND NOT sqlc.arg(sort_desc)::bool THEN wo.model END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model_name' AND sqlc.arg(sort_desc)::bool THEN wo.model END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'size' AND NOT sqlc.arg(sort_desc)::bool THEN woss.size END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'size' AND sqlc.arg(sort_desc)::bool THEN woss.size END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'target_qty' AND NOT sqlc.arg(sort_desc)::bool THEN woss.qty END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'target_qty' AND sqlc.arg(sort_desc)::bool THEN woss.qty END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'cutting_qty' AND NOT sqlc.arg(sort_desc)::bool THEN COALESCE(ca.cutting_qty, 0)::int END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'cutting_qty' AND sqlc.arg(sort_desc)::bool THEN COALESCE(ca.cutting_qty, 0)::int END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'sewing_qty' AND NOT sqlc.arg(sort_desc)::bool THEN COALESCE(sa.sewing_qty, 0)::int END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'sewing_qty' AND sqlc.arg(sort_desc)::bool THEN COALESCE(sa.sewing_qty, 0)::int END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'qc_pass_qty' AND NOT sqlc.arg(sort_desc)::bool THEN COALESCE(qa.qc_pass_qty, 0)::int END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'qc_pass_qty' AND sqlc.arg(sort_desc)::bool THEN COALESCE(qa.qc_pass_qty, 0)::int END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'packing_qty' AND NOT sqlc.arg(sort_desc)::bool THEN COALESCE(pa.packing_qty, 0)::int END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'packing_qty' AND sqlc.arg(sort_desc)::bool THEN COALESCE(pa.packing_qty, 0)::int END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'shipped_qty' AND NOT sqlc.arg(sort_desc)::bool THEN COALESCE(sha.shipped_qty, 0)::int END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'shipped_qty' AND sqlc.arg(sort_desc)::bool THEN COALESCE(sha.shipped_qty, 0)::int END DESC,
    last_updated DESC,
    woss.id_wo_shell_size DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);
