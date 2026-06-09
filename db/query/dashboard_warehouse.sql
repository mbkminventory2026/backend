-- name: GetWarehouseTotalItems :one
SELECT COUNT(*) FROM BARANG;

-- name: GetWarehouseTotalSuratJalanClientThisMonth :one
SELECT COUNT(*) 
FROM SURAT_JALAN_CLIENT 
WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE);

-- name: GetWarehouseTotalSuratJalanInternalThisMonth :one
SELECT COUNT(*) 
FROM SURAT_JALAN_INTERNAL 
WHERE date_trunc('month', created_at) = date_trunc('month', CURRENT_DATE);

-- name: GetWarehouseRecentSuratJalanClient :many
SELECT 
    sjc.id_surat_jalan_client,
    sjc.tanggal,
    sjc.keterangan,
    mli.description AS material_description
FROM SURAT_JALAN_CLIENT sjc
JOIN MATERIAL_LIST_ITEM mli ON sjc.id_material_list_item = mli.id_material_list_item
ORDER BY sjc.created_at DESC 
LIMIT 5;

-- name: GetWarehouseRecentSuratJalanInternal :many
SELECT 
    sji.id_surat_jalan_internal,
    sji.created_at
FROM SURAT_JALAN_INTERNAL sji
ORDER BY sji.created_at DESC 
LIMIT 5;

-- name: GetWarehouseRecentBarang :many
SELECT 
    id_barang,
    nama_barang,
    kode,
    stok_minimum,
    created_at
FROM BARANG
ORDER BY created_at DESC 
LIMIT 5;
