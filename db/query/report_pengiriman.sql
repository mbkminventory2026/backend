-- name: GetNextReportPengirimanID :one
SELECT COALESCE(MAX(ID_REPORT_PENGIRIMAN), 0) + 1 AS next_id
FROM REPORT_PENGIRIMAN;

-- name: CreateReportPengiriman :one
INSERT INTO REPORT_PENGIRIMAN (
    ID_REPORT_PENGIRIMAN,
    "DATE",
    Quantity,
    ID_WO_SHELL_SIZE
) VALUES ($1, $2, $3, $4)
RETURNING ID_REPORT_PENGIRIMAN, "DATE", Quantity, ID_WO_SHELL_SIZE, created_at;

-- name: GetReportPengirimanByID :one
SELECT ID_REPORT_PENGIRIMAN, "DATE", Quantity, ID_WO_SHELL_SIZE, created_at
FROM REPORT_PENGIRIMAN
WHERE ID_REPORT_PENGIRIMAN = $1
LIMIT 1;

-- name: ListReportPengiriman :many
SELECT ID_REPORT_PENGIRIMAN, "DATE", Quantity, ID_WO_SHELL_SIZE, created_at
FROM REPORT_PENGIRIMAN
WHERE
    ($1::date IS NULL OR "DATE" >= $1::date)
    AND ($2::date IS NULL OR "DATE" <= $2::date)
    AND ($3::int IS NULL OR ID_WO_SHELL_SIZE = $3::int)
ORDER BY ID_REPORT_PENGIRIMAN DESC
LIMIT $4 OFFSET $5;

-- name: DeleteReportPengirimanByID :execrows
DELETE FROM REPORT_PENGIRIMAN
WHERE ID_REPORT_PENGIRIMAN = $1;

-- name: WorkOrderShellSizeExists :one
SELECT EXISTS(
    SELECT 1
    FROM WORK_ORDER_SHELL_SIZE
    WHERE ID_WO_SHELL_SIZE = $1
) AS exists;
