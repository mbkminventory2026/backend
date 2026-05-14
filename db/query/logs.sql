-- name: CreateAktivitasLog :one
INSERT INTO LOG_AKTIVITAS (
    aksi
) VALUES (
    sqlc.arg(aksi)
)
RETURNING id_log, aksi, waktu, created_at;

-- name: CreateAktivitasLogDetail :one
INSERT INTO LOG_AKTIVITAS_DETAIL (
    nama,
    table_name,
    deskripsi,
    id_log
) VALUES (
    sqlc.arg(nama),
    sqlc.arg(table_name),
    sqlc.arg(deskripsi),
    sqlc.arg(id_log)
)
RETURNING id_log_detail, nama, table_name, deskripsi, id_log, created_at;

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
