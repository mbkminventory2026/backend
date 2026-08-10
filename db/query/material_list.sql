-- name: ListMaterialListsPaginated :many
SELECT
    ml.id_material_list,
    ml.id_wo,
    ml.name,
    ml.is_locked,
    ml.created_at,
    wo.buyer,
    wo.model,
    wo.qty AS wo_qty,
    (SELECT COUNT(*) FROM MATERIAL_LIST_ITEM WHERE id_material_list = ml.id_material_list)::integer AS item_count,
    (SELECT COALESCE(SUM(sjc.qty), 0)
     FROM SURAT_JALAN_CLIENT sjc
     JOIN MATERIAL_LIST_ITEM mli2 ON mli2.id_material_list_item = sjc.id_material_list_item
     WHERE mli2.id_material_list = ml.id_material_list)::integer AS total_qty_sj,
    (SELECT COALESCE(SUM(r.qty), 0)
     FROM RECEIVED r
     JOIN MATERIAL_LIST_ITEM mli3 ON mli3.id_material_list_item = r.id_material_list_item
     WHERE mli3.id_material_list = ml.id_material_list)::integer AS total_qty_received,
    COUNT(*) OVER () AS total_count
FROM MATERIAL_LIST ml
JOIN WORK_ORDER wo ON wo.id_wo = ml.id_wo
WHERE (NOT sqlc.arg(locked_only)::boolean OR ml.is_locked = TRUE)
  AND (
    sqlc.arg(search)::text = ''
    OR LOWER(wo.buyer) LIKE '%' || LOWER(sqlc.arg(search)::text) || '%'
    OR LOWER(wo.model) LIKE '%' || LOWER(sqlc.arg(search)::text) || '%'
    OR LOWER(ml.name) LIKE '%' || LOWER(sqlc.arg(search)::text) || '%'
  )
ORDER BY ml.created_at DESC
LIMIT sqlc.arg(lim)
OFFSET sqlc.arg(off);

-- name: GetMaterialListItemDetail :one
SELECT
    mli.id_material_list_item,
    mli.id_material_list,
    mli.item,
    mli.description,
    mli.qty,
    mli.unit,
    mli.est_price,
    mli.id_wo_shell,
    mli.id_wo_trim,
    mli.category,
    mli.cons_per_pc,
    mli.qty_wo_scope,
    mli.id_qty_wo_shell,
    mli.id_qty_wo_size,
    mli.created_at,
    COALESCE((SELECT SUM(sjc.qty) FROM SURAT_JALAN_CLIENT sjc WHERE sjc.id_material_list_item = mli.id_material_list_item), 0)::integer AS qty_surat_jalan,
    COALESCE((SELECT SUM(r.qty) FROM RECEIVED r WHERE r.id_material_list_item = mli.id_material_list_item), 0)::integer AS qty_received,
    ml.name AS ml_name,
    ml.is_locked AS ml_is_locked,
    wo.id_wo,
    wo.buyer,
    wo.model
FROM MATERIAL_LIST_ITEM mli
JOIN MATERIAL_LIST ml ON ml.id_material_list = mli.id_material_list
JOIN WORK_ORDER wo ON wo.id_wo = ml.id_wo
WHERE mli.id_material_list_item = sqlc.arg(id_material_list_item);

-- name: GetMaterialList :one
SELECT id_material_list, id_wo, name, is_locked, created_at
FROM MATERIAL_LIST
WHERE id_material_list = sqlc.arg(id_material_list);

-- name: GetMaterialListForUpdate :one
SELECT id_material_list, id_wo, name, is_locked, created_at
FROM MATERIAL_LIST
WHERE id_material_list = sqlc.arg(id_material_list)
FOR UPDATE;

-- name: ListUnlockedMaterialListsByWO :many
SELECT id_material_list, id_wo, name, is_locked, created_at
FROM MATERIAL_LIST
WHERE id_wo = sqlc.arg(id_wo) AND is_locked = FALSE
ORDER BY id_material_list ASC;

-- name: UpdateMaterialList :one
UPDATE MATERIAL_LIST
SET name = sqlc.arg(name)
WHERE id_material_list = sqlc.arg(id_material_list)
  AND is_locked = FALSE
RETURNING id_material_list, id_wo, name, is_locked, created_at;

-- name: LockMaterialList :one
UPDATE MATERIAL_LIST
SET is_locked = TRUE
WHERE id_material_list = sqlc.arg(id_material_list)
RETURNING id_material_list, id_wo, name, is_locked, created_at;

-- name: DeleteMaterialList :exec
DELETE FROM MATERIAL_LIST
WHERE id_material_list = sqlc.arg(id_material_list)
  AND is_locked = FALSE;

-- name: GetMaterialListItem :one
SELECT 
    mli.id_material_list_item, 
    mli.id_material_list, 
    mli.item, 
    mli.description, 
    mli.qty, 
    mli.unit, 
    mli.est_price, 
    mli.id_wo_shell, 
    mli.id_wo_trim, 
    mli.category,
    mli.cons_per_pc,
    mli.qty_wo_scope,
    mli.id_qty_wo_shell,
    mli.id_qty_wo_size,
    mli.created_at,
    COALESCE((SELECT SUM(sjc.qty) FROM SURAT_JALAN_CLIENT sjc WHERE sjc.id_material_list_item = mli.id_material_list_item), 0)::integer AS qty_surat_jalan,
    COALESCE((SELECT SUM(r.qty) FROM RECEIVED r WHERE r.id_material_list_item = mli.id_material_list_item), 0)::integer AS qty_received
FROM MATERIAL_LIST_ITEM mli
WHERE mli.id_material_list_item = sqlc.arg(id_material_list_item);

-- name: ListMaterialListItemsByML :many
SELECT 
    mli.id_material_list_item, 
    mli.id_material_list, 
    mli.item, 
    mli.description, 
    mli.qty, 
    mli.unit, 
    mli.est_price, 
    mli.id_wo_shell, 
    mli.id_wo_trim, 
    mli.category,
    mli.cons_per_pc,
    mli.qty_wo_scope,
    mli.id_qty_wo_shell,
    mli.id_qty_wo_size,
    mli.created_at,
    COALESCE((SELECT SUM(sjc.qty) FROM SURAT_JALAN_CLIENT sjc WHERE sjc.id_material_list_item = mli.id_material_list_item), 0)::integer AS qty_surat_jalan,
    COALESCE((SELECT SUM(r.qty) FROM RECEIVED r WHERE r.id_material_list_item = mli.id_material_list_item), 0)::integer AS qty_received
FROM MATERIAL_LIST_ITEM mli
WHERE mli.id_material_list = sqlc.arg(id_material_list)
ORDER BY mli.id_material_list_item ASC;

-- name: UpdateMaterialListItem :one
UPDATE MATERIAL_LIST_ITEM
SET
    item = sqlc.arg(item),
    description = sqlc.arg(description),
    qty = sqlc.arg(qty),
    unit = sqlc.arg(unit),
    est_price = sqlc.arg(est_price)::numeric,
    id_wo_shell = sqlc.narg(id_wo_shell),
    id_wo_trim = sqlc.narg(id_wo_trim),
    category = CASE WHEN sqlc.arg(set_category)::boolean THEN sqlc.narg(category)::text ELSE category END,
    cons_per_pc = CASE WHEN sqlc.arg(set_cons_per_pc)::boolean THEN sqlc.narg(cons_per_pc)::numeric ELSE cons_per_pc END,
    qty_wo_scope = CASE WHEN sqlc.arg(set_qty_wo_scope)::boolean THEN sqlc.narg(qty_wo_scope)::text ELSE qty_wo_scope END,
    id_qty_wo_shell = CASE WHEN sqlc.arg(set_id_qty_wo_shell)::boolean THEN sqlc.narg(id_qty_wo_shell)::integer ELSE id_qty_wo_shell END,
    id_qty_wo_size = CASE WHEN sqlc.arg(set_id_qty_wo_size)::boolean THEN sqlc.narg(id_qty_wo_size)::integer ELSE id_qty_wo_size END
WHERE id_material_list_item = sqlc.arg(id_material_list_item)
  AND id_material_list IN (
      SELECT id_material_list FROM MATERIAL_LIST WHERE is_locked = FALSE
  )
RETURNING id_material_list_item, id_material_list, item, description, qty, unit, est_price, id_wo_shell, id_wo_trim, category, cons_per_pc, qty_wo_scope, id_qty_wo_shell, id_qty_wo_size, created_at;

-- name: GetMaterialListItemConsumptionPrefill :one
SELECT COALESCE(
    (SELECT cons FROM WORK_ORDER_TRIM WHERE id_wo_trim = sqlc.narg(id_wo_trim)),
    (SELECT cons FROM WORK_ORDER_SHELL WHERE id_wo_shell = sqlc.narg(id_wo_shell))
)::numeric AS cons_per_pc;

-- name: MaterialListHasApplicabilityShell :one
SELECT EXISTS (
    SELECT 1
    FROM MATERIAL_LIST ml
    JOIN WORK_ORDER_SHELL wos ON wos.id_wo = ml.id_wo
    WHERE ml.id_material_list = sqlc.arg(id_material_list)
      AND wos.id_wo_shell = sqlc.arg(id_qty_wo_shell)
) AS exists;

-- name: MaterialListHasApplicabilitySize :one
SELECT EXISTS (
    SELECT 1
    FROM MATERIAL_LIST ml
    JOIN WORK_ORDER_SHELL wos ON wos.id_wo = ml.id_wo
    JOIN WORK_ORDER_SHELL_SIZE woss ON woss.id_wo_shell = wos.id_wo_shell
    WHERE ml.id_material_list = sqlc.arg(id_material_list)
      AND woss.id_size = sqlc.arg(id_qty_wo_size)
) AS exists;

-- name: MaterialListHasApplicabilityShellSize :one
SELECT EXISTS (
    SELECT 1
    FROM MATERIAL_LIST ml
    JOIN WORK_ORDER_SHELL wos ON wos.id_wo = ml.id_wo
    JOIN WORK_ORDER_SHELL_SIZE woss ON woss.id_wo_shell = wos.id_wo_shell
    WHERE ml.id_material_list = sqlc.arg(id_material_list)
      AND wos.id_wo_shell = sqlc.arg(id_qty_wo_shell)
      AND woss.id_size = sqlc.arg(id_qty_wo_size)
) AS exists;

-- name: MaterialListHasSourceShell :one
SELECT EXISTS (
    SELECT 1
    FROM MATERIAL_LIST ml
    JOIN WORK_ORDER_SHELL wos ON wos.id_wo = ml.id_wo
    WHERE ml.id_material_list = sqlc.arg(id_material_list)
      AND wos.id_wo_shell = sqlc.arg(id_wo_shell)
) AS exists;

-- name: MaterialListHasSourceTrim :one
SELECT EXISTS (
    SELECT 1
    FROM MATERIAL_LIST ml
    JOIN WORK_ORDER_TRIM wot ON wot.id_wo = ml.id_wo
    WHERE ml.id_material_list = sqlc.arg(id_material_list)
      AND wot.id_wo_trim = sqlc.arg(id_wo_trim)
) AS exists;

-- name: DeleteMaterialListItem :exec
DELETE FROM MATERIAL_LIST_ITEM
WHERE id_material_list_item = sqlc.arg(id_material_list_item)
  AND id_material_list IN (
      SELECT id_material_list FROM MATERIAL_LIST WHERE is_locked = FALSE
  );

-- name: CheckMaterialListBelongsToWO :one
SELECT EXISTS (
    SELECT 1 FROM MATERIAL_LIST
    WHERE id_material_list = sqlc.arg(id_material_list)
      AND id_wo = sqlc.arg(id_wo)
) AS exists;
