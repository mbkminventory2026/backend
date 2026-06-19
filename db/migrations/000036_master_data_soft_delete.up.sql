CREATE TABLE IF NOT EXISTS MASTER_DATA_DELETED (
    id_deleted     SERIAL PRIMARY KEY,
    nama_tabel     VARCHAR(100)  NOT NULL,
    id_record      INT           NOT NULL,
    deleted_by     INT,
    deleted_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (nama_tabel, id_record),
    FOREIGN KEY (deleted_by) REFERENCES USERS(ID_USER) ON DELETE SET NULL
);
