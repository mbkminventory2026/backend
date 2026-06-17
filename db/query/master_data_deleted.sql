-- name: SoftDeleteMasterData :exec
INSERT INTO MASTER_DATA_DELETED (nama_tabel, id_record, deleted_by)
VALUES ($1, $2, $3)
ON CONFLICT (nama_tabel, id_record) DO NOTHING;

-- name: IsMasterDataDeleted :one
SELECT EXISTS(
    SELECT 1 FROM MASTER_DATA_DELETED
    WHERE nama_tabel = $1 AND id_record = $2
)::boolean;
