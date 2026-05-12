-- name: ApprovePRInternal :one
UPDATE PR_INTERNAL
SET
    status = 'approved',
    approved_by_user_id = sqlc.arg(approved_by_user_id),
    approved_at = NOW()
WHERE id_pr_internal = sqlc.arg(id_pr_internal)
RETURNING id_pr_internal, status, approved_by_user_id, approved_at;

-- name: CloseWorkOrder :one
UPDATE WORK_ORDER
SET
    status = 'closed',
    closed_by_user_id = sqlc.arg(closed_by_user_id),
    closed_at = NOW()
WHERE id_wo = sqlc.arg(id_wo)
RETURNING id_wo, status, closed_by_user_id, closed_at;
