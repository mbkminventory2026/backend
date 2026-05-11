-- name: GetAktivitasLogs :many
SELECT 
    la.ID_LOG,
    la.AKSI,
    la.WAKTU,
    lad.NAMA AS detail_nama,
    lad.TABLE_NAME AS detail_table,
    lad.DESKRIPSI AS detail_deskripsi
FROM 
    LOG_AKTIVITAS la
LEFT JOIN 
    LOG_AKTIVITAS_DETAIL lad ON la.ID_LOG = lad.ID_LOG
ORDER BY 
    la.WAKTU DESC
LIMIT $1 OFFSET $2;