-- name: CreateMasterPlan :one
INSERT INTO MASTER_PLAN (id_departemen, id_production_line, nama, created_by)
VALUES (
    sqlc.arg(id_departemen),
    sqlc.arg(id_production_line),
    sqlc.arg(nama),
    sqlc.narg(created_by)
)
RETURNING *;

-- name: GetMasterPlanByID :one
SELECT
    mp.id_master_plan,
    mp.id_departemen,
    mp.id_production_line,
    mp.nama,
    mp.created_by,
    mp.created_at,
    mp.updated_at,
    d.nama_departemen,
    pl.name AS nama_line
FROM MASTER_PLAN mp
JOIN DEPARTEMEN d ON d.id_departemen = mp.id_departemen
JOIN PRODUCTION_LINE pl ON pl.id_production_line = mp.id_production_line
WHERE mp.id_master_plan = $1;

-- name: ListMasterPlans :many
SELECT
    mp.id_master_plan,
    mp.id_departemen,
    mp.id_production_line,
    mp.nama,
    mp.created_at,
    d.nama_departemen,
    pl.name AS nama_line,
    COUNT(*) OVER() AS total_count
FROM MASTER_PLAN mp
JOIN DEPARTEMEN d ON d.id_departemen = mp.id_departemen
JOIN PRODUCTION_LINE pl ON pl.id_production_line = mp.id_production_line
WHERE (
    sqlc.arg(search_term)::text = '' OR
    mp.nama ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    d.nama_departemen ILIKE '%' || sqlc.arg(search_term)::text || '%' OR
    pl.name ILIKE '%' || sqlc.arg(search_term)::text || '%'
)
ORDER BY mp.created_at DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: UpdateMasterPlan :one
UPDATE MASTER_PLAN
SET nama = $2, updated_at = NOW()
WHERE id_master_plan = $1
RETURNING *;

-- name: DeleteMasterPlan :exec
DELETE FROM MASTER_PLAN WHERE id_master_plan = $1;

-- name: AddMasterPlanItem :one
INSERT INTO MASTER_PLAN_ITEM (id_master_plan, id_wo_shell, no_urut)
VALUES ($1, $2, $3)
RETURNING *;

-- name: RemoveMasterPlanItem :execrows
DELETE FROM MASTER_PLAN_ITEM
WHERE id_master_plan_item = $1 AND id_master_plan = $2;

-- name: GetMasterPlanItemByID :one
SELECT
    mpi.id_master_plan_item,
    mpi.id_master_plan,
    mpi.id_wo_shell,
    mpi.no_urut,
    mpi.created_at,
    wos.id_wo,
    wos.color,
    wos.deskripsi,
    wo.buyer,
    wo.model AS style,
    wo.qty
FROM MASTER_PLAN_ITEM mpi
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = mpi.id_wo_shell
JOIN WORK_ORDER wo ON wo.id_wo = wos.id_wo
WHERE mpi.id_master_plan_item = $1;

-- name: ListMasterPlanItems :many
SELECT
    mpi.id_master_plan_item,
    mpi.id_master_plan,
    mpi.id_wo_shell,
    mpi.no_urut,
    mpi.created_at,
    wos.id_wo,
    wos.color,
    wos.deskripsi,
    wo.buyer,
    wo.model AS style,
    wo.qty
FROM MASTER_PLAN_ITEM mpi
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = mpi.id_wo_shell
JOIN WORK_ORDER wo ON wo.id_wo = wos.id_wo
WHERE mpi.id_master_plan = $1
ORDER BY mpi.no_urut ASC, mpi.created_at ASC;

-- name: UpsertTargetHarian :one
INSERT INTO MASTER_PLAN_TARGET_HARIAN (id_master_plan_item, tanggal, target)
VALUES ($1, $2::date, $3)
ON CONFLICT (id_master_plan_item, tanggal)
DO UPDATE SET target = EXCLUDED.target, updated_at = NOW()
RETURNING *;

-- name: UpsertOutputHarian :one
INSERT INTO MASTER_PLAN_OUTPUT_HARIAN (id_master_plan_item, tanggal, output)
VALUES ($1, $2::date, $3)
ON CONFLICT (id_master_plan_item, tanggal)
DO UPDATE SET output = EXCLUDED.output, updated_at = NOW()
RETURNING *;

-- name: UpsertTargetProses :one
INSERT INTO MASTER_PLAN_TARGET_PROSES (id_master_plan_item, tanggal, nama_proses)
VALUES ($1, $2::date, $3)
ON CONFLICT (id_master_plan_item, tanggal)
DO UPDATE SET nama_proses = EXCLUDED.nama_proses
RETURNING *;

-- name: DeleteTargetProses :exec
DELETE FROM MASTER_PLAN_TARGET_PROSES
WHERE id_master_plan_item = $1 AND tanggal = $2::date;

-- name: ListTargetHarianByItem :many
SELECT * FROM MASTER_PLAN_TARGET_HARIAN
WHERE id_master_plan_item = $1
ORDER BY tanggal ASC;

-- name: ListOutputHarianByItem :many
SELECT * FROM MASTER_PLAN_OUTPUT_HARIAN
WHERE id_master_plan_item = $1
ORDER BY tanggal ASC;

-- name: ListTargetProsesByItem :many
SELECT * FROM MASTER_PLAN_TARGET_PROSES
WHERE id_master_plan_item = $1
ORDER BY tanggal ASC;
