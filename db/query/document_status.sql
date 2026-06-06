-- name: ApprovePRInternal :one
INSERT INTO OTORITAS_DOKUMEN (NAMA_TABEL_DOKUMEN, ID_DOKUMEN, STATUS_GLOBAL)
VALUES ('PR_INTERNAL', sqlc.arg(id_pr_internal), 'approved')
ON CONFLICT (NAMA_TABEL_DOKUMEN, ID_DOKUMEN)
DO UPDATE SET STATUS_GLOBAL = 'approved'
WHERE (sqlc.narg(approved_by_user_id)::integer IS NULL OR TRUE)
RETURNING ID_DOKUMEN AS id_pr_internal, STATUS_GLOBAL AS status, NULL::integer AS approved_by_user_id, NULL::timestamptz AS approved_at;

-- name: CloseWorkOrder :one
INSERT INTO OTORITAS_DOKUMEN (NAMA_TABEL_DOKUMEN, ID_DOKUMEN, STATUS_GLOBAL)
VALUES ('WORK_ORDER', sqlc.arg(id_wo), 'closed')
ON CONFLICT (NAMA_TABEL_DOKUMEN, ID_DOKUMEN)
DO UPDATE SET STATUS_GLOBAL = 'closed'
WHERE (sqlc.narg(closed_by_user_id)::integer IS NULL OR TRUE)
RETURNING ID_DOKUMEN AS id_wo, STATUS_GLOBAL AS status, NULL::integer AS closed_by_user_id, NULL::timestamptz AS closed_at;
