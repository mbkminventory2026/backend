-- name: CreateMarkerPlan :one
INSERT INTO MARKER_PLAN (
    NO_DOKUMEN, TANGGAL_EFEKTIF, ID_WO_SHELL
) VALUES (
    $1, $2, $3
) RETURNING ID_MARKER_PLAN, NO_DOKUMEN, TANGGAL_EFEKTIF, ID_WO_SHELL, created_at;

-- name: CreateKomponenMarkerPlan :one
INSERT INTO KOMPONEN_MARKER_PLAN (
    ID_MARKER_PLAN, NAMA_KOMPONEN
) VALUES (
    $1, $2
) RETURNING ID_KOMPONEN_MARKER, ID_MARKER_PLAN, NAMA_KOMPONEN, created_at;

-- name: CreateRatioMarker :one
INSERT INTO RATIO_MARKER (
    ID_KOMPONEN_MARKER, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN,
    PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER,
    PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING ID_RATIO_MARKER, ID_KOMPONEN_MARKER, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN, PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER, PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET, created_at;

-- name: CreateRatioSizeMarker :copyfrom
INSERT INTO RATIO_SIZE_MARKER (
    ID_RATIO_MARKER, ID_WO_SHELL_SIZE, RATIO_PLAN
) VALUES (
    $1, $2, $3
);

-- name: GetMarkerPlanByID :one
SELECT 
    mp.id_marker_plan, 
    mp.no_dokumen, 
    mp.tanggal_efektif, 
    mp.id_wo_shell, 
    mp.created_at,
    wos.color,
    wos.deskripsi AS fabric_description,
    pci.style,
    wo.model
FROM MARKER_PLAN mp
JOIN WORK_ORDER_SHELL wos ON mp.id_wo_shell = wos.id_wo_shell
JOIN WORK_ORDER wo ON wos.id_wo = wo.id_wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
WHERE mp.id_marker_plan = $1 LIMIT 1;

-- name: ListKomponenByMarkerPlanID :many
SELECT ID_KOMPONEN_MARKER, ID_MARKER_PLAN, NAMA_KOMPONEN, created_at
FROM KOMPONEN_MARKER_PLAN
WHERE ID_MARKER_PLAN = $1
ORDER BY ID_KOMPONEN_MARKER ASC;

-- name: ListRatioByKomponenID :many
SELECT ID_RATIO_MARKER, ID_KOMPONEN_MARKER, ID_WO_SHELL, CONS, PLAN_SPREADING_GELARAN,
       PANJANG_MARKER, EFFICIENCY_MARKER, ALLOWANCE, CONS_BUYER,
       PLOT, LEBAR_KAIN, PANJANG_MARKER_UNIT, KET, created_at
FROM RATIO_MARKER
WHERE ID_KOMPONEN_MARKER = $1
ORDER BY ID_RATIO_MARKER ASC;

-- name: ListRatioSizeByRatioID :many
SELECT rsm.ID_RATIO_SIZE_MARKER, rsm.ID_RATIO_MARKER, rsm.ID_WO_SHELL_SIZE, rsm.RATIO_PLAN, ms.NAMA_SIZE AS SIZE, wss.QTY AS size_qty
FROM RATIO_SIZE_MARKER rsm
JOIN WORK_ORDER_SHELL_SIZE wss ON rsm.ID_WO_SHELL_SIZE = wss.ID_WO_SHELL_SIZE
JOIN MASTER_SIZE ms ON ms.ID_SIZE = wss.ID_SIZE
WHERE rsm.ID_RATIO_MARKER = $1
ORDER BY rsm.ID_RATIO_SIZE_MARKER ASC;

-- name: ListMarkerPlans :many
SELECT
    mp.id_marker_plan,
    mp.no_dokumen,
    mp.tanggal_efektif,
    mp.id_wo_shell,
    mp.created_at,
    wos.deskripsi,
    wos.color,
    wo.id_wo,
    wo.buyer,
    wo.model,
    COUNT(*) OVER() AS total_count
FROM MARKER_PLAN mp
JOIN WORK_ORDER_SHELL wos ON mp.id_wo_shell = wos.id_wo_shell
JOIN WORK_ORDER wo ON wos.id_wo = wo.id_wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
WHERE (
    sqlc.arg(search_term) = '' OR
    mp.no_dokumen ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wo.buyer ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wo.model ILIKE '%' || sqlc.arg(search_term) || '%' OR
    wos.deskripsi ILIKE '%' || sqlc.arg(search_term) || '%'
) AND (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN mp.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN mp.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'no_dokumen' AND NOT sqlc.arg(sort_desc)::bool THEN mp.no_dokumen END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'no_dokumen' AND sqlc.arg(sort_desc)::bool THEN mp.no_dokumen END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal_efektif' AND NOT sqlc.arg(sort_desc)::bool THEN mp.tanggal_efektif END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'tanggal_efektif' AND sqlc.arg(sort_desc)::bool THEN mp.tanggal_efektif END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model' AND NOT sqlc.arg(sort_desc)::bool THEN wo.model END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'model' AND sqlc.arg(sort_desc)::bool THEN wo.model END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND NOT sqlc.arg(sort_desc)::bool THEN wo.buyer END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'buyer' AND sqlc.arg(sort_desc)::bool THEN wo.buyer END DESC,
    mp.id_marker_plan DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: GetReceivedQtyByWOShellID :one
SELECT COALESCE(SUM(r.QTY), 0)::bigint AS total_qty_received
FROM RECEIVED r
JOIN MATERIAL_LIST_ITEM mli ON r.ID_MATERIAL_LIST_ITEM = mli.ID_MATERIAL_LIST_ITEM
WHERE mli.ID_WO_SHELL = $1;

