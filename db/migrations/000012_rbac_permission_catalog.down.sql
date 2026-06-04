DELETE FROM ROLE_HAK_AKSES
WHERE (ID_ROLE, ID_HAK_AKSES) IN (
    SELECT r.ID_ROLE, h.ID_HAK_AKSES
    FROM ROLES r
    JOIN HAK_AKSES h ON h.KODE_PERMISSION IN ('AUTH_CHANGE_PASSWORD', 'PASSWORD_RESET_REQUEST_CREATE')
    WHERE r.NAMA_ROLE IN (
        'SUPER_ADMIN',
        'OPERATOR',
        'ADMIN_KEUANGAN',
        'ADMIN_PRODUKSI',
        'ADMIN_GUDANG',
        'MANAGER',
        'CLIENT'
    )
);

DELETE FROM ROLE_HAK_AKSES
WHERE (ID_ROLE, ID_HAK_AKSES) IN (
    SELECT r.ID_ROLE, h.ID_HAK_AKSES
    FROM ROLES r
    JOIN HAK_AKSES h ON h.KODE_PERMISSION = 'USER_TEMP_PASSWORD_CREATE'
    WHERE r.NAMA_ROLE = 'OPERATOR'
);

DELETE FROM ROLE_HAK_AKSES
WHERE (ID_ROLE, ID_HAK_AKSES) IN (
    SELECT r.ID_ROLE, h.ID_HAK_AKSES
    FROM ROLES r
    JOIN HAK_AKSES h ON h.KODE_PERMISSION = 'MASTER_WARNA_READ'
    WHERE r.NAMA_ROLE IN ('ADMIN_PRODUKSI', 'MANAGER')
);

DELETE FROM ROLE_HAK_AKSES
WHERE (ID_ROLE, ID_HAK_AKSES) IN (
    SELECT r.ID_ROLE, h.ID_HAK_AKSES
    FROM ROLES r
    JOIN HAK_AKSES h ON h.KODE_PERMISSION = 'USER_APPROVE'
    WHERE r.NAMA_ROLE = 'MANAGER'
);

DELETE FROM USERS
WHERE USERNAME = 'operator';

DELETE FROM ROLE_HAK_AKSES
WHERE ID_HAK_AKSES IN (
    SELECT ID_HAK_AKSES
    FROM HAK_AKSES
    WHERE KODE_PERMISSION IN (
        'AUTH_CHANGE_PASSWORD',
        'USER_TEMP_PASSWORD_CREATE',
        'PASSWORD_RESET_REQUEST_CREATE',
        'MASTER_WARNA_READ',
        'MASTER_WARNA_CREATE',
        'MASTER_WARNA_UPDATE',
        'MASTER_WARNA_DELETE'
    )
);

DELETE FROM USER_AKSES
WHERE ID_HAK_AKSES IN (
    SELECT ID_HAK_AKSES
    FROM HAK_AKSES
    WHERE KODE_PERMISSION IN (
        'AUTH_CHANGE_PASSWORD',
        'USER_TEMP_PASSWORD_CREATE',
        'PASSWORD_RESET_REQUEST_CREATE',
        'MASTER_WARNA_READ',
        'MASTER_WARNA_CREATE',
        'MASTER_WARNA_UPDATE',
        'MASTER_WARNA_DELETE'
    )
);

DELETE FROM HAK_AKSES
WHERE KODE_PERMISSION IN (
    'AUTH_CHANGE_PASSWORD',
    'USER_TEMP_PASSWORD_CREATE',
    'PASSWORD_RESET_REQUEST_CREATE',
    'MASTER_WARNA_READ',
    'MASTER_WARNA_CREATE',
    'MASTER_WARNA_UPDATE',
    'MASTER_WARNA_DELETE'
);

INSERT INTO HAK_AKSES (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION)
VALUES
    ('ITEM_CREATE', 'Item Create', 'Allows item create', 'item', 'create'),
    ('ITEM_READ', 'Item Read', 'Allows item read', 'item', 'read'),
    ('ITEM_UPDATE', 'Item Update', 'Allows item update', 'item', 'update'),
    ('ITEM_DELETE', 'Item Delete', 'Allows item delete', 'item', 'delete'),
    ('MASTER_CREATE', 'Master Create', 'Allows master create', 'master', 'create'),
    ('MASTER_READ', 'Master Read', 'Allows master read', 'master', 'read'),
    ('MASTER_UPDATE', 'Master Update', 'Allows master update', 'master', 'update'),
    ('MASTER_DELETE', 'Master Delete', 'Allows master delete', 'master', 'delete'),
    ('PO_CREATE', 'Po Create', 'Allows po create', 'po', 'create'),
    ('PO_READ', 'Po Read', 'Allows po read', 'po', 'read'),
    ('PO_UPDATE', 'Po Update', 'Allows po update', 'po', 'update'),
    ('PR_APPROVE', 'Pr Approve', 'Allows pr approve', 'pr', 'approve'),
    ('REPORT_CREATE', 'Report Create', 'Allows report create', 'report', 'create')
ON CONFLICT (KODE_PERMISSION) DO NOTHING;
