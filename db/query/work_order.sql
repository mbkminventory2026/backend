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
    fabric,
    cons,
    color,
    allow,
    berat_1_yd,
    id_wo
) VALUES (
    sqlc.arg(fabric),
    sqlc.arg(cons)::numeric,
    sqlc.arg(color),
    sqlc.arg(allow),
    sqlc.arg(berat_1_yd)::numeric,
    sqlc.arg(id_wo)
)
RETURNING id_wo_shell, fabric, cons, color, allow, berat_1_yd, id_wo, created_at;

-- name: CreateWorkOrderShellSize :one
INSERT INTO WORK_ORDER_SHELL_SIZE (
    size,
    qty,
    ratio,
    id_wo_shell
) VALUES (
    sqlc.arg(size),
    sqlc.arg(qty),
    sqlc.arg(ratio),
    sqlc.arg(id_wo_shell)
)
RETURNING id_wo_shell_size, size, qty, ratio, id_wo_shell, created_at;

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
    id_wo
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
    sqlc.arg(id_wo)
)
RETURNING id_wo_trim, item, description, color, code, cons, qty, uom, position, created_by, allow, id_wo, created_at;

-- name: CreateMaterialList :one
WITH inserted_item AS (
    INSERT INTO MATERIAL_LIST_ITEM (description)
    VALUES (sqlc.arg(description))
    RETURNING id_material_list_item, description
)
INSERT INTO MATERIAL_LIST (id_material_list_item)
SELECT id_material_list_item FROM inserted_item
RETURNING id_material_list,
          (SELECT description FROM inserted_item) AS description,
          sqlc.arg(size)::text AS size,
          sqlc.arg(color)::text AS color,
          sqlc.arg(uom)::text AS uom,
          sqlc.arg(id_wo)::integer AS id_wo,
          created_at;

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
