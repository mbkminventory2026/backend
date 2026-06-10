-- name: CreateWorkOrder :one
INSERT INTO WORK_ORDER (
    buyer,
    model,
    qty,
    fob_cmt,
    delivery,
    id_po_client_item
) VALUES (
    sqlc.arg(buyer),
    sqlc.arg(model),
    sqlc.arg(qty),
    sqlc.arg(fob_cmt),
    sqlc.arg(delivery)::date,
    sqlc.arg(id_po_client_item)
)
RETURNING id_wo, buyer, model, qty, fob_cmt, delivery, id_po_client_item, created_at;

-- name: CreateWorkOrderShell :one
INSERT INTO WORK_ORDER_SHELL (
    deskripsi,
    cons,
    color,
    allow,
    berat_1_yd,
    id_wo,
    provided_by,
    material_type
) VALUES (
    sqlc.arg(deskripsi),
    sqlc.arg(cons)::numeric,
    sqlc.arg(color),
    sqlc.arg(allow),
    sqlc.arg(berat_1_yd)::numeric,
    sqlc.arg(id_wo),
    sqlc.arg(provided_by),
    sqlc.arg(material_type)
)
RETURNING id_wo_shell, deskripsi, cons, color, allow, berat_1_yd, id_wo, created_at, provided_by, material_type;

-- name: CreateWorkOrderShellSize :one
INSERT INTO WORK_ORDER_SHELL_SIZE (
    id_size,
    qty,
    ratio,
    id_wo_shell
) VALUES (
    sqlc.arg(id_size),
    sqlc.arg(qty),
    sqlc.arg(ratio),
    sqlc.arg(id_wo_shell)
)
RETURNING id_wo_shell_size, id_size, qty, ratio, id_wo_shell, created_at;

-- name: CreateWorkOrderTrim :one
INSERT INTO WORK_ORDER_TRIM (
    item,
    description,
    color,
    code,
    cons,
    qty,
    uom,
    position,
    created_by,
    allow,
    id_wo,
    provided_by
) VALUES (
    sqlc.arg(item),
    sqlc.arg(description),
    sqlc.arg(color),
    sqlc.arg(code),
    sqlc.arg(cons)::numeric,
    sqlc.arg(qty),
    sqlc.arg(uom),
    sqlc.arg(position),
    sqlc.arg(created_by),
    sqlc.arg(allow),
    sqlc.arg(id_wo),
    sqlc.arg(provided_by)
)
RETURNING id_wo_trim, item, description, color, code, cons, qty, uom, position, created_by, allow, id_wo, created_at, provided_by;

-- name: CreateMaterialList :one
INSERT INTO MATERIAL_LIST (id_wo, name)
VALUES (
    sqlc.arg(id_wo),
    sqlc.arg(name)
)
RETURNING id_material_list, id_wo, name, is_locked, created_at;

-- name: CreateMaterialListItem :one
INSERT INTO MATERIAL_LIST_ITEM (
    id_material_list,
    item,
    description,
    qty,
    unit,
    est_price,
    id_wo_shell,
    id_wo_trim
) VALUES (
    sqlc.arg(id_material_list),
    sqlc.arg(item),
    sqlc.arg(description),
    sqlc.arg(qty),
    sqlc.arg(unit),
    sqlc.arg(est_price)::numeric,
    sqlc.narg(id_wo_shell),
    sqlc.narg(id_wo_trim)
)
RETURNING id_material_list_item, id_material_list, item, description, qty, unit, est_price, id_wo_shell, id_wo_trim, created_at;

-- name: WorkOrderShellTotalQty :one
SELECT COALESCE(SUM(qty), 0)::bigint AS total_qty
FROM WORK_ORDER_SHELL_SIZE
WHERE id_wo_shell = sqlc.arg(id_wo_shell);

-- name: DeleteWorkOrdersByPOClientID :exec
DELETE FROM WORK_ORDER
WHERE id_po_client_item IN (
    SELECT id_po_client_item
    FROM PO_CLIENT_ITEM
    WHERE id_po_client = sqlc.arg(id_po_client)
);

-- name: CountConfiguredWorkOrdersByPOClientID :one
SELECT COUNT(*)
FROM WORK_ORDER wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
LEFT JOIN WORK_ORDER_SHELL wos ON wos.id_wo = wo.id_wo
LEFT JOIN WORK_ORDER_TRIM wot ON wot.id_wo = wo.id_wo
LEFT JOIN MATERIAL_LIST ml ON ml.id_wo = wo.id_wo
WHERE pci.id_po_client = sqlc.arg(id_po_client)
  AND (wos.id_wo_shell IS NOT NULL OR wot.id_wo_trim IS NOT NULL OR ml.id_material_list IS NOT NULL);
