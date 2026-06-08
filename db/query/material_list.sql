-- name: GetMaterialList :one
SELECT id_material_list, id_wo, name, is_locked, created_at
FROM MATERIAL_LIST
WHERE id_material_list = sqlc.arg(id_material_list);

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
SELECT id_material_list_item, id_material_list, item, description, qty, unit, est_price, id_wo_shell, id_wo_trim, created_at
FROM MATERIAL_LIST_ITEM
WHERE id_material_list_item = sqlc.arg(id_material_list_item);

-- name: ListMaterialListItemsByML :many
SELECT id_material_list_item, id_material_list, item, description, qty, unit, est_price, id_wo_shell, id_wo_trim, created_at
FROM MATERIAL_LIST_ITEM
WHERE id_material_list = sqlc.arg(id_material_list)
ORDER BY id_material_list_item ASC;

-- name: UpdateMaterialListItem :one
UPDATE MATERIAL_LIST_ITEM
SET
    item = sqlc.arg(item),
    description = sqlc.arg(description),
    qty = sqlc.arg(qty),
    unit = sqlc.arg(unit),
    est_price = sqlc.arg(est_price)::numeric,
    id_wo_shell = sqlc.narg(id_wo_shell),
    id_wo_trim = sqlc.narg(id_wo_trim)
WHERE id_material_list_item = sqlc.arg(id_material_list_item)
  AND id_material_list IN (
      SELECT id_material_list FROM MATERIAL_LIST WHERE is_locked = FALSE
  )
RETURNING id_material_list_item, id_material_list, item, description, qty, unit, est_price, id_wo_shell, id_wo_trim, created_at;

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
