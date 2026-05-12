ALTER TABLE WORK_ORDER
DROP CONSTRAINT IF EXISTS work_order_closed_by_user_id_fkey,
DROP COLUMN IF EXISTS closed_at,
DROP COLUMN IF EXISTS closed_by_user_id,
DROP COLUMN IF EXISTS status;

ALTER TABLE PR_INTERNAL
DROP CONSTRAINT IF EXISTS pr_internal_approved_by_user_id_fkey,
DROP COLUMN IF EXISTS approved_at,
DROP COLUMN IF EXISTS approved_by_user_id,
DROP COLUMN IF EXISTS status;
