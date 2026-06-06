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

-- name: CreateReportPengiriman :one
INSERT INTO REPORT_PENGIRIMAN (
    report_date,
    qty,
    id_wo_shell_size
) VALUES (
    sqlc.arg(report_date)::date,
    sqlc.arg(qty),
    sqlc.arg(id_wo_shell_size)
)
RETURNING id_report_pengiriman, report_date, qty, id_wo_shell_size, created_at;

-- name: GetDailyReportsByWorkOrder :many
SELECT 'cutting'::varchar(20) AS division, rc.tanggal, rc.qty, rc.id_wo_shell_size
FROM REPORT_CUTTING rc
JOIN WORK_ORDER_SHELL_SIZE woss ON woss.id_wo_shell_size = rc.id_wo_shell_size
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = woss.id_wo_shell
WHERE wos.id_wo = sqlc.arg(id_wo)

UNION ALL

SELECT 'sewing'::varchar(20) AS division, rs.tanggal, rs.qty, rs.id_wo_shell_size
FROM REPORT_SEWING rs
JOIN WORK_ORDER_SHELL_SIZE woss ON woss.id_wo_shell_size = rs.id_wo_shell_size
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = woss.id_wo_shell
WHERE wos.id_wo = sqlc.arg(id_wo)

UNION ALL

SELECT 'qc-finish'::varchar(20) AS division, rq.tanggal, rq.qty, rq.id_wo_shell_size
FROM REPORT_QC_FINISH rq
JOIN WORK_ORDER_SHELL_SIZE woss ON woss.id_wo_shell_size = rq.id_wo_shell_size
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = woss.id_wo_shell
WHERE wos.id_wo = sqlc.arg(id_wo)

UNION ALL

SELECT 'packing'::varchar(20) AS division, rp.tanggal, rp.qty, rp.id_wo_shell_size
FROM REPORT_PACKING rp
JOIN WORK_ORDER_SHELL_SIZE woss ON woss.id_wo_shell_size = rp.id_wo_shell_size
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = woss.id_wo_shell
WHERE wos.id_wo = sqlc.arg(id_wo)

UNION ALL

SELECT 'pengiriman'::varchar(20) AS division, rpe.report_date AS tanggal, rpe.qty, rpe.id_wo_shell_size
FROM REPORT_PENGIRIMAN rpe
JOIN WORK_ORDER_SHELL_SIZE woss ON woss.id_wo_shell_size = rpe.id_wo_shell_size
JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = woss.id_wo_shell
WHERE wos.id_wo = sqlc.arg(id_wo)

ORDER BY tanggal DESC, id_wo_shell_size ASC;
