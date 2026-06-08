-- name: GetStockReportPerKategori :many
-- Mengambil laporan stok material yang teragregasi per kategori barang garmen
SELECT 
    COALESCE(jb.nama_jenis_barang, 'Lain-lain')::text AS kategori,
    rm.description::text AS nama_barang,
    rm.size::text AS size,
    SUM(rm.balance)::bigint AS total_stok,
    rm.satuan::text AS satuan
FROM 
    REKONSILIASI_MATERIAL rm
LEFT JOIN 
    BARANG b ON rm.description = b.nama_barang OR rm.description = b.kode
LEFT JOIN 
    JENIS_BARANG jb ON b.id_jenis_barang = jb.id_jenis_barang
GROUP BY 
    jb.nama_jenis_barang, rm.description, rm.size, rm.satuan
ORDER BY 
    kategori ASC, nama_barang ASC;

-- name: GetStockReportPerLokasi :many
-- Mengambil laporan stok material yang teragregasi per lokasi penyimpanan rak
SELECT 
    COALESCE(b.lokasi_rak, 'Belum Diatur')::text AS lokasi_rak,
    rm.description::text AS nama_barang,
    rm.size::text AS size,
    SUM(rm.balance)::bigint AS total_stok,
    rm.satuan::text AS satuan
FROM 
    REKONSILIASI_MATERIAL rm
LEFT JOIN 
    BARANG b ON rm.description = b.nama_barang OR rm.description = b.kode
GROUP BY 
    b.lokasi_rak, rm.description, rm.size, rm.satuan
ORDER BY 
    lokasi_rak ASC, nama_barang ASC;

-- name: GetMovementReport :many
-- Mengambil laporan riwayat pergerakan stok masuk (received) dan keluar (surat jalan) secara kronologis
SELECT
    'IN'::text AS tipe,
    r.tanggal,
    r.qty::int AS qty,
    r.keterangan,
    mli.description::text AS nama_material,
    mli.unit::text AS uom,
    wo.model::text AS work_order_model
FROM
    RECEIVED r
JOIN
    MATERIAL_LIST_ITEM mli ON r.id_material_list_item = mli.id_material_list_item
JOIN
    MATERIAL_LIST ml ON ml.id_material_list = mli.id_material_list
JOIN
    WORK_ORDER wo ON ml.id_wo = wo.id_wo

UNION ALL

SELECT
    'OUT'::text AS tipe,
    sjc.tanggal,
    sjc.qty::int AS qty,
    sjc.keterangan,
    mli.description::text AS nama_material,
    mli.unit::text AS uom,
    wo.model::text AS work_order_model
FROM
    SURAT_JALAN_CLIENT sjc
JOIN
    MATERIAL_LIST_ITEM mli ON sjc.id_material_list_item = mli.id_material_list_item
JOIN
    MATERIAL_LIST ml ON ml.id_material_list = mli.id_material_list
JOIN
    WORK_ORDER wo ON ml.id_wo = wo.id_wo

ORDER BY
    tanggal DESC, nama_material ASC;
