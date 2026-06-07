-- name: CreateReturClient :one
INSERT INTO RETUR_CLIENT (
    id_wo,
    file,
    deskripsi
) VALUES (
    sqlc.arg(id_wo),
    sqlc.arg(file),
    sqlc.arg(deskripsi)
)
RETURNING id_retur_client, id_wo, file, deskripsi, created_at;

-- name: GetReturClientByWorkOrderID :one
SELECT id_retur_client, id_wo, file, deskripsi, created_at
FROM RETUR_CLIENT
WHERE id_wo = $1 LIMIT 1;

-- name: ClientCloseWorkOrder :one
INSERT INTO OTORITAS_DOKUMEN (NAMA_TABEL_DOKUMEN, ID_DOKUMEN, STATUS_GLOBAL)
VALUES ('WORK_ORDER', sqlc.arg(id_wo), 'client_closed')
ON CONFLICT (NAMA_TABEL_DOKUMEN, ID_DOKUMEN)
DO UPDATE SET STATUS_GLOBAL = 'client_closed'
RETURNING ID_DOKUMEN AS id_wo, STATUS_GLOBAL AS status;

-- name: AutoCloseWorkOrders :exec
INSERT INTO OTORITAS_DOKUMEN (NAMA_TABEL_DOKUMEN, ID_DOKUMEN, STATUS_GLOBAL)
SELECT 'WORK_ORDER', wo.id_wo, 'client_closed'
FROM WORK_ORDER wo
JOIN (
    SELECT wos.id_wo, MAX(rp.REPORT_DATE) AS last_delivery
    FROM REPORT_PENGIRIMAN rp
    JOIN WORK_ORDER_SHELL_SIZE woss ON woss.id_wo_shell_size = rp.id_wo_shell_size
    JOIN WORK_ORDER_SHELL wos ON wos.id_wo_shell = woss.id_wo_shell
    GROUP BY wos.id_wo
) delivery ON delivery.id_wo = wo.id_wo
LEFT JOIN RETUR_CLIENT rc ON rc.id_wo = wo.id_wo
LEFT JOIN OTORITAS_DOKUMEN od ON od.NAMA_TABEL_DOKUMEN = 'WORK_ORDER' AND od.ID_DOKUMEN = wo.id_wo
WHERE COALESCE(od.STATUS_GLOBAL, 'open') = 'open'
  AND rc.id_retur_client IS NULL
  AND delivery.last_delivery < NOW() - INTERVAL '2 months'
ON CONFLICT (NAMA_TABEL_DOKUMEN, ID_DOKUMEN)
DO UPDATE SET STATUS_GLOBAL = 'client_closed';

-- name: ListReturClients :many
SELECT rc.id_retur_client, rc.id_wo, rc.file, rc.deskripsi, rc.created_at,
       wo.buyer, wo.model, wo.qty AS wo_qty,
       pc.po_number, pc.id_mitra, m.nama_perusahaan as mitra_name,
       pc.id_po_client,
       COUNT(*) OVER() as total_count
FROM RETUR_CLIENT rc
JOIN WORK_ORDER wo ON wo.id_wo = rc.id_wo
JOIN PO_CLIENT_ITEM pci ON pci.id_po_client_item = wo.id_po_client_item
JOIN PO_CLIENT pc ON pc.id_po_client = pci.id_po_client
JOIN MITRA m ON m.id_mitra = pc.id_mitra
WHERE (
    sqlc.narg(id_mitra)::integer IS NULL OR
    pc.id_mitra = sqlc.narg(id_mitra)::integer
) AND (
    sqlc.arg(search)::text = '' OR
    pc.po_number ILIKE '%' || sqlc.arg(search)::text || '%' OR
    wo.model ILIKE '%' || sqlc.arg(search)::text || '%'
)
ORDER BY rc.created_at DESC
LIMIT sqlc.arg(page_limit)::integer
OFFSET sqlc.arg(page_offset)::integer;
