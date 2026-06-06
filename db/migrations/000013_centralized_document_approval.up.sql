-- 1. Drop legacy indexes
DROP INDEX IF EXISTS idx_pr_internal_status_lower;
DROP INDEX IF EXISTS idx_work_order_status_lower;

-- 2. Drop columns and constraints from PR_INTERNAL
ALTER TABLE PR_INTERNAL DROP CONSTRAINT IF EXISTS pr_internal_approved_by_user_id_fkey;
ALTER TABLE PR_INTERNAL DROP COLUMN IF EXISTS status;
ALTER TABLE PR_INTERNAL DROP COLUMN IF EXISTS approved_by_user_id;
ALTER TABLE PR_INTERNAL DROP COLUMN IF EXISTS approved_at;

-- 3. Drop columns and constraints from WORK_ORDER
ALTER TABLE WORK_ORDER DROP CONSTRAINT IF EXISTS work_order_closed_by_user_id_fkey;
ALTER TABLE WORK_ORDER DROP COLUMN IF EXISTS status;
ALTER TABLE WORK_ORDER DROP COLUMN IF EXISTS closed_by_user_id;
ALTER TABLE WORK_ORDER DROP COLUMN IF EXISTS closed_at;

-- 4. Clean duplicates and add UNIQUE constraint to OTORITAS_DOKUMEN
DELETE FROM OTORITAS_DOKUMEN a
USING OTORITAS_DOKUMEN b
WHERE a.ID_OTORITAS < b.ID_OTORITAS
  AND a.NAMA_TABEL_DOKUMEN = b.NAMA_TABEL_DOKUMEN
  AND a.ID_DOKUMEN = b.ID_DOKUMEN;

ALTER TABLE OTORITAS_DOKUMEN ADD CONSTRAINT uq_otoritas_dokumen UNIQUE (NAMA_TABEL_DOKUMEN, ID_DOKUMEN);

-- 5. Create Database Views for centralized status compatibility with sqlc
CREATE OR REPLACE VIEW v_pr_internal AS
SELECT 
    pr.id_pr_internal,
    pr.tanggal,
    pr.nama,
    pr.departemen,
    pr.vendor_name,
    pr.vendor_address,
    pr.vendor_telp,
    pr.projek,
    pr.id_wo,
    pr.id_user,
    pr.created_at,
    COALESCE(LOWER(od.STATUS_GLOBAL), 'draft')::varchar AS status,
    NULL::integer AS approved_by_user_id,
    NULL::timestamptz AS approved_at
FROM PR_INTERNAL pr
LEFT JOIN OTORITAS_DOKUMEN od 
  ON od.NAMA_TABEL_DOKUMEN = 'PR_INTERNAL' AND od.ID_DOKUMEN = pr.id_pr_internal;

CREATE OR REPLACE VIEW v_work_order AS
SELECT 
    wo.id_wo,
    wo.buyer,
    wo.model,
    wo.qty,
    wo.fob_cmt,
    wo.delivery,
    wo.id_po_client_item,
    wo.created_at,
    COALESCE(LOWER(od.STATUS_GLOBAL), 'open')::varchar AS status,
    NULL::integer AS closed_by_user_id,
    NULL::timestamptz AS closed_at
FROM WORK_ORDER wo
LEFT JOIN OTORITAS_DOKUMEN od 
  ON od.NAMA_TABEL_DOKUMEN = 'WORK_ORDER' AND od.ID_DOKUMEN = wo.id_wo;
