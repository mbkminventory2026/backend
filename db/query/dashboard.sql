-- name: GetWorkOrderForAIEstimation :many
-- Mengambil data WO dan Production Internal untuk diolah model Regresi Linier di Golang
SELECT 
    wo.ID_WO,
    wo.BUYER,
    wo.MODEL,
    wo.QTY,
    wo.DELIVERY AS target_delivery,
    pr.TANGGAL AS tanggal_pr,
    pr.NAMA AS pic_pr
FROM 
    WORK_ORDER wo
JOIN 
    PR_INTERNAL pr ON wo.ID_WO = pr.ID_WO
ORDER BY 
    wo.DELIVERY DESC;

-- name: GetLowStockAlerts :many
-- Mengecek material yang BALANCE-nya di bawah standar untuk trigger WebSocket layar berkedip
SELECT 
    rm.ID_REKONSILIASI_MATERIAL,
    rm.DESCRIPTION,
    rm.SIZE,
    rm.BALANCE,
    rm.LAST_BALANCE,
    rm.SATUAN,
    COALESCE(b.stok_minimum, 50)::int AS min_stock
FROM 
    REKONSILIASI_MATERIAL rm
LEFT JOIN 
    BARANG b ON rm.description = b.nama_barang OR rm.description = b.kode
WHERE 
    rm.BALANCE < COALESCE(b.stok_minimum, 50); -- Asumsi threshold low stock adalah 50, bisa kita ubah nanti lewat argumen sqlc jika dinamis