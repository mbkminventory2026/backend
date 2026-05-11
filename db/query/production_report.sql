-- name: CreateReportCutting :one
INSERT INTO REPORT_CUTTING (
    tanggal,
    qty,
    id_wo_shell_size
) VALUES (
    sqlc.arg(tanggal)::date,
    sqlc.arg(qty),
    sqlc.arg(id_wo_shell_size)
)
RETURNING id_report_cutting, tanggal, qty, id_wo_shell_size, created_at;

-- name: CreateReportSewing :one
INSERT INTO REPORT_SEWING (
    tanggal,
    qty,
    id_wo_shell_size
) VALUES (
    sqlc.arg(tanggal)::date,
    sqlc.arg(qty),
    sqlc.arg(id_wo_shell_size)
)
RETURNING id_report_sewing, tanggal, qty, id_wo_shell_size, created_at;

-- name: CreateReportQCFinish :one
INSERT INTO REPORT_QC_FINISH (
    tanggal,
    qty,
    id_wo_shell_size
) VALUES (
    sqlc.arg(tanggal)::date,
    sqlc.arg(qty),
    sqlc.arg(id_wo_shell_size)
)
RETURNING id_report_qc_finishing, tanggal, qty, id_wo_shell_size, created_at;

-- name: CreateReportPacking :one
INSERT INTO REPORT_PACKING (
    tanggal,
    qty,
    id_wo_shell_size
) VALUES (
    sqlc.arg(tanggal)::date,
    sqlc.arg(qty),
    sqlc.arg(id_wo_shell_size)
)
RETURNING id_report_packing, tanggal, qty, id_wo_shell_size, created_at;
