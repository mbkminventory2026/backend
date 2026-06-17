CREATE TABLE IF NOT EXISTS MASTER_PLAN (
    id_master_plan     SERIAL PRIMARY KEY,
    id_departemen      INT NOT NULL,
    id_production_line INT NOT NULL,
    nama               VARCHAR(200) NOT NULL DEFAULT '',
    created_by         INT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (id_departemen)      REFERENCES DEPARTEMEN(ID_DEPARTEMEN),
    FOREIGN KEY (id_production_line) REFERENCES PRODUCTION_LINE(id_production_line),
    FOREIGN KEY (created_by)         REFERENCES USERS(ID_USER) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS MASTER_PLAN_ITEM (
    id_master_plan_item SERIAL PRIMARY KEY,
    id_master_plan      INT NOT NULL,
    id_wo_shell         INT NOT NULL,
    no_urut             INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id_master_plan, id_wo_shell),
    FOREIGN KEY (id_master_plan) REFERENCES MASTER_PLAN(id_master_plan) ON DELETE CASCADE,
    FOREIGN KEY (id_wo_shell)    REFERENCES WORK_ORDER_SHELL(ID_WO_SHELL)
);

CREATE TABLE IF NOT EXISTS MASTER_PLAN_TARGET_HARIAN (
    id_target_harian    SERIAL PRIMARY KEY,
    id_master_plan_item INT NOT NULL,
    tanggal             DATE NOT NULL,
    target              INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id_master_plan_item, tanggal),
    FOREIGN KEY (id_master_plan_item) REFERENCES MASTER_PLAN_ITEM(id_master_plan_item) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS MASTER_PLAN_OUTPUT_HARIAN (
    id_output_harian    SERIAL PRIMARY KEY,
    id_master_plan_item INT NOT NULL,
    tanggal             DATE NOT NULL,
    output              INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id_master_plan_item, tanggal),
    FOREIGN KEY (id_master_plan_item) REFERENCES MASTER_PLAN_ITEM(id_master_plan_item) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS MASTER_PLAN_TARGET_PROSES (
    id_target_proses    SERIAL PRIMARY KEY,
    id_master_plan_item INT NOT NULL,
    tanggal             DATE NOT NULL,
    nama_proses         VARCHAR(100) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id_master_plan_item, tanggal),
    FOREIGN KEY (id_master_plan_item) REFERENCES MASTER_PLAN_ITEM(id_master_plan_item) ON DELETE CASCADE
);
