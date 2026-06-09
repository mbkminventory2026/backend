-- name: CreateTimelinePlan :one
INSERT INTO TIMELINE_PLAN_PRODUKSI (
    ID_PO_CLIENT, TANGGAL_DISUSUN, NOTES
) VALUES (
    $1, $2, $3
) RETURNING ID_TIMELINE, ID_PO_CLIENT, TANGGAL_DISUSUN, NOTES, created_at;

-- name: CreateWOShellPlan :copyfrom
INSERT INTO WO_SHELL_PLAN (
    ID_TIMELINE, ID_WO_SHELL, IN_LINE, 
    TGL_GELAR_CUTTING, STATUS_GELAR_CUTTING, 
    TGL_EMBROO, STATUS_EMBROO, 
    TGL_LOADING_SEWING, STATUS_LOADING_SEWING, 
    TGL_FINISHING_PACKING, STATUS_FINISHING_PACKING
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
);

-- name: GetTimelinePlanByID :one
SELECT ID_TIMELINE, ID_PO_CLIENT, TANGGAL_DISUSUN, NOTES, created_at
FROM TIMELINE_PLAN_PRODUKSI
WHERE ID_TIMELINE = $1 LIMIT 1;

-- name: GetWOShellPlansByTimelineID :many
SELECT 
    wsp.ID_WO_SHELL_PLAN, wsp.ID_TIMELINE, wsp.ID_WO_SHELL, wsp.IN_LINE,
    wsp.TGL_GELAR_CUTTING, wsp.STATUS_GELAR_CUTTING,
    wsp.TGL_EMBROO, wsp.STATUS_EMBROO,
    wsp.TGL_LOADING_SEWING, wsp.STATUS_LOADING_SEWING,
    wsp.TGL_FINISHING_PACKING, wsp.STATUS_FINISHING_PACKING,
    wos.DESKRIPSI, wos.COLOR
FROM WO_SHELL_PLAN wsp
JOIN WORK_ORDER_SHELL wos ON wsp.ID_WO_SHELL = wos.ID_WO_SHELL
WHERE wsp.ID_TIMELINE = $1
ORDER BY wsp.ID_WO_SHELL_PLAN ASC;

-- name: UpdateWOShellPlanStatus :exec
UPDATE WO_SHELL_PLAN
SET 
    IN_LINE = COALESCE(NULLIF($2, ''), IN_LINE),
    STATUS_GELAR_CUTTING = COALESCE(NULLIF($3, ''), STATUS_GELAR_CUTTING),
    STATUS_EMBROO = COALESCE(NULLIF($4, ''), STATUS_EMBROO),
    STATUS_LOADING_SEWING = COALESCE(NULLIF($5, ''), STATUS_LOADING_SEWING),
    STATUS_FINISHING_PACKING = COALESCE(NULLIF($6, ''), STATUS_FINISHING_PACKING)
WHERE ID_WO_SHELL_PLAN = $1;

-- name: ListTimelinePlans :many
SELECT 
    tpp.ID_TIMELINE, tpp.ID_PO_CLIENT, tpp.TANGGAL_DISUSUN, tpp.NOTES, tpp.created_at,
    m.NAMA_PERUSAHAAN AS client_name, pc.PO_NUMBER AS po_number
FROM TIMELINE_PLAN_PRODUKSI tpp
JOIN PO_CLIENT pc ON tpp.ID_PO_CLIENT = pc.ID_PO_CLIENT
JOIN MITRA m ON pc.ID_MITRA = m.ID_MITRA
WHERE (
    sqlc.arg(search_term)::text = '' OR
    m.NAMA_PERUSAHAAN ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    pc.PO_NUMBER ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    tpp.NOTES ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND NOT sqlc.arg(sort_desc)::bool THEN tpp.created_at END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'created_at' AND sqlc.arg(sort_desc)::bool THEN tpp.created_at END DESC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_timeline' AND NOT sqlc.arg(sort_desc)::bool THEN tpp.ID_TIMELINE END ASC,
    CASE WHEN sqlc.arg(sort_by)::text = 'id_timeline' AND sqlc.arg(sort_desc)::bool THEN tpp.ID_TIMELINE END DESC,
    tpp.ID_TIMELINE DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountTimelinePlans :one
SELECT COUNT(*)
FROM TIMELINE_PLAN_PRODUKSI tpp
JOIN PO_CLIENT pc ON tpp.ID_PO_CLIENT = pc.ID_PO_CLIENT
JOIN MITRA m ON pc.ID_MITRA = m.ID_MITRA
WHERE (
    sqlc.arg(search_term)::text = '' OR
    m.NAMA_PERUSAHAAN ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    pc.PO_NUMBER ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    tpp.NOTES ILIKE '%' || sqlc.arg(search_term)::text || '%'
);
