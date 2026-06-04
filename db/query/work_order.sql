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
INSERT INTO MATERIAL_LIST (
    description,
    size,
    color,
    uom,
    id_wo
) VALUES (
    sqlc.arg(description),
    sqlc.arg(size),
    sqlc.arg(color),
    sqlc.arg(uom),
    sqlc.arg(id_wo)
)
RETURNING id_material_list, description, size, color, uom, id_wo, created_at;

-- name: WorkOrderShellTotalQty :one
SELECT COALESCE(SUM(qty), 0)::bigint AS total_qty
FROM WORK_ORDER_SHELL_SIZE
WHERE id_wo_shell = sqlc.arg(id_wo_shell);
