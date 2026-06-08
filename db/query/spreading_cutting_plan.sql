-- name: CreateSpreadingCuttingPlan :one
INSERT INTO SPREADING_CUTTING_PLAN (
    NO_DOKUMEN, TANGGAL_EFEKTIF, ID_WO
) VALUES (
    $1, $2, $3
) RETURNING ID_SPREADING_CUTTING_PLAN, NO_DOKUMEN, TANGGAL_EFEKTIF, ID_WO, created_at;

-- name: CreateKomponenSpreadingCuttingPlan :one
INSERT INTO KOMPONEN_SPREADING_CUTTING_PLAN (
    ID_SPREADING_CUTTING_PLAN, NAMA_KOMPONEN
) VALUES (
    $1, $2
) RETURNING ID_KOMPONEN_SPREADING, ID_SPREADING_CUTTING_PLAN, NAMA_KOMPONEN, created_at;

-- name: CreateRatioSpreading :one
INSERT INTO RATIO_SPREADING (
    ID_KOMPONEN_SPREADING, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN,
    PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER,
    ROLL_QTY, SAMBUNGAN_ROLL, REJECT, PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
) RETURNING ID_RATIO_SPREADING, ID_KOMPONEN_SPREADING, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN, PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER, ROLL_QTY, SAMBUNGAN_ROLL, REJECT, PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET, created_at;

-- name: CreateRatioSizeSpreading :copyfrom
INSERT INTO RATIO_SIZE_SPREADING (
    ID_RATIO_SPREADING, ID_WO_SHELL_SIZE, RATIO_PLAN
) VALUES (
    $1, $2, $3
);

-- name: GetSpreadingCuttingPlanByID :one
SELECT ID_SPREADING_CUTTING_PLAN, NO_DOKUMEN, TANGGAL_EFEKTIF, ID_WO, created_at
FROM SPREADING_CUTTING_PLAN
WHERE ID_SPREADING_CUTTING_PLAN = $1 LIMIT 1;

-- name: ListKomponenBySpreadingPlanID :many
SELECT ID_KOMPONEN_SPREADING, ID_SPREADING_CUTTING_PLAN, NAMA_KOMPONEN, created_at
FROM KOMPONEN_SPREADING_CUTTING_PLAN
WHERE ID_SPREADING_CUTTING_PLAN = $1
ORDER BY ID_KOMPONEN_SPREADING ASC;

-- name: ListRatioByKomponenSpreadingID :many
SELECT ID_RATIO_SPREADING, ID_KOMPONEN_SPREADING, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN,
       PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER, ROLL_QTY, SAMBUNGAN_ROLL,
       REJECT, PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET, created_at
FROM RATIO_SPREADING
WHERE ID_KOMPONEN_SPREADING = $1
ORDER BY ID_RATIO_SPREADING ASC;

-- name: ListRatioSizeByRatioSpreadingID :many
SELECT rss.ID_RATIO_SIZE_SPREADING, rss.ID_RATIO_SPREADING, rss.ID_WO_SHELL_SIZE, rss.RATIO_PLAN, wss.SIZE, wss.QTY AS size_qty
FROM RATIO_SIZE_SPREADING rss
JOIN WORK_ORDER_SHELL_SIZE wss ON rss.ID_WO_SHELL_SIZE = wss.ID_WO_SHELL_SIZE
WHERE rss.ID_RATIO_SPREADING = $1
ORDER BY rss.ID_RATIO_SIZE_SPREADING ASC;

-- name: ListSpreadingCuttingPlans :many
SELECT
    scp.id_spreading_cutting_plan,
    scp.no_dokumen,
    scp.tanggal_efektif,
    scp.id_wo,
    scp.created_at,
    wo.buyer,
    wo.model,
    COUNT(*) OVER() AS total_count
FROM SPREADING_CUTTING_PLAN scp
JOIN WORK_ORDER wo ON scp.id_wo = wo.id_wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE (
    sqlc.arg(search_term) = '' OR
    scp.no_dokumen ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wo.buyer ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wo.model ILIKE '%' || sqlc.arg(search_term) || '%'
) AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN scp.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN scp.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'no_dokumen' AND NOT sqlc.arg(sort_desc)::bool THEN scp.no_dokumen END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'no_dokumen' AND sqlc.arg(sort_desc)::bool THEN scp.no_dokumen END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal_efektif' AND NOT sqlc.arg(sort_desc)::bool THEN scp.tanggal_efektif END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal_efektif' AND sqlc.arg(sort_desc)::bool THEN scp.tanggal_efektif END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model' AND NOT sqlc.arg(sort_desc)::bool THEN wo.model END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model' AND sqlc.arg(sort_desc)::bool THEN wo.model END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND NOT sqlc.arg(sort_desc)::bool THEN wo.buyer END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND sqlc.arg(sort_desc)::bool THEN wo.buyer END DESC,
    scp.id_spreading_cutting_plan DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);
