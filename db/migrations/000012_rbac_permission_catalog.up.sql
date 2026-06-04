WITH inserted_permissions (kode_permission, nama_halaman, deskripsi, domain_permission, aksi_permission) AS (
    VALUES
        ('AUTH_CHANGE_PASSWORD', 'Auth Change Password', 'Allows authenticated users to change their own password', 'auth', 'change_password'),
        ('USER_TEMP_PASSWORD_CREATE', 'User Temporary Password Create', 'Allows operators to generate a temporary password while creating or resetting users', 'user_temp_password', 'create'),
        ('PASSWORD_RESET_REQUEST_CREATE', 'Password Reset Request Create', 'Allows users to submit a password reset request', 'password_reset_request', 'create'),
        ('MASTER_WARNA_READ', 'Master Warna Read', 'Allows reading warna master data', 'master_warna', 'read'),
        ('MASTER_WARNA_CREATE', 'Master Warna Create', 'Allows creating warna master data', 'master_warna', 'create'),
        ('MASTER_WARNA_UPDATE', 'Master Warna Update', 'Allows updating warna master data', 'master_warna', 'update'),
        ('MASTER_WARNA_DELETE', 'Master Warna Delete', 'Allows deleting warna master data', 'master_warna', 'delete')
)
INSERT INTO HAK_AKSES (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION)
SELECT kode_permission, nama_halaman, deskripsi, domain_permission, aksi_permission
FROM inserted_permissions
ON CONFLICT (KODE_PERMISSION) DO UPDATE
SET NAMA_HALAMAN = EXCLUDED.NAMA_HALAMAN,
    DESKRIPSI = EXCLUDED.DESKRIPSI,
    DOMAIN_PERMISSION = EXCLUDED.DOMAIN_PERMISSION,
    AKSI_PERMISSION = EXCLUDED.AKSI_PERMISSION;

WITH obsolete_permissions (kode_permission) AS (
    VALUES
        ('ITEM_CREATE'),
        ('ITEM_READ'),
        ('ITEM_UPDATE'),
        ('ITEM_DELETE'),
        ('MASTER_CREATE'),
        ('MASTER_READ'),
        ('MASTER_UPDATE'),
        ('MASTER_DELETE'),
        ('PO_CREATE'),
        ('PO_READ'),
        ('PO_UPDATE'),
        ('PR_APPROVE'),
        ('REPORT_CREATE')
)
DELETE FROM ROLE_HAK_AKSES
WHERE ID_HAK_AKSES IN (
    SELECT h.ID_HAK_AKSES
    FROM HAK_AKSES h
    JOIN obsolete_permissions o ON o.kode_permission = h.KODE_PERMISSION
);

WITH obsolete_permissions (kode_permission) AS (
    VALUES
        ('ITEM_CREATE'),
        ('ITEM_READ'),
        ('ITEM_UPDATE'),
        ('ITEM_DELETE'),
        ('MASTER_CREATE'),
        ('MASTER_READ'),
        ('MASTER_UPDATE'),
        ('MASTER_DELETE'),
        ('PO_CREATE'),
        ('PO_READ'),
        ('PO_UPDATE'),
        ('PR_APPROVE'),
        ('REPORT_CREATE')
)
DELETE FROM USER_AKSES
WHERE ID_HAK_AKSES IN (
    SELECT h.ID_HAK_AKSES
    FROM HAK_AKSES h
    JOIN obsolete_permissions o ON o.kode_permission = h.KODE_PERMISSION
);

WITH obsolete_permissions (kode_permission) AS (
    VALUES
        ('ITEM_CREATE'),
        ('ITEM_READ'),
        ('ITEM_UPDATE'),
        ('ITEM_DELETE'),
        ('MASTER_CREATE'),
        ('MASTER_READ'),
        ('MASTER_UPDATE'),
        ('MASTER_DELETE'),
        ('PO_CREATE'),
        ('PO_READ'),
        ('PO_UPDATE'),
        ('PR_APPROVE'),
        ('REPORT_CREATE')
)
DELETE FROM HAK_AKSES
WHERE KODE_PERMISSION IN (
    SELECT kode_permission
    FROM obsolete_permissions
);

INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
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
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;

INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
SELECT r.ID_ROLE, h.ID_HAK_AKSES
FROM ROLES r
JOIN HAK_AKSES h ON h.KODE_PERMISSION = 'USER_TEMP_PASSWORD_CREATE'
WHERE r.NAMA_ROLE = 'OPERATOR'
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;

INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
SELECT r.ID_ROLE, h.ID_HAK_AKSES
FROM ROLES r
JOIN HAK_AKSES h ON h.KODE_PERMISSION = 'MASTER_WARNA_READ'
WHERE r.NAMA_ROLE IN ('ADMIN_PRODUKSI', 'MANAGER')
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;

INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
SELECT r.ID_ROLE, h.ID_HAK_AKSES
FROM ROLES r
JOIN HAK_AKSES h ON h.KODE_PERMISSION = 'USER_APPROVE'
WHERE r.NAMA_ROLE = 'MANAGER'
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;

UPDATE USERS
SET MUST_CHANGE_PASSWORD = FALSE
WHERE USERNAME = 'super-admin';

INSERT INTO USERS (
    USERNAME,
    PASSWORD,
    ID_ROLE,
    ID_DEPARTEMEN,
    STATUS,
    MUST_CHANGE_PASSWORD,
    CREATED_BY,
    UPDATED_BY
)
SELECT
    'operator',
    '$2a$10$Dxq6jWDNwCvzxx1OQnQJjOqn59JaSVxJsPbN8DUmjacJgAg/oWQc2',
    r.ID_ROLE,
    d.ID_DEPARTEMEN,
    'active',
    TRUE,
    su.ID_USER,
    su.ID_USER
FROM ROLES r
LEFT JOIN DEPARTEMEN d ON d.NAMA_DEPARTEMEN = 'IT'
LEFT JOIN USERS su ON su.USERNAME = 'super-admin'
WHERE r.NAMA_ROLE = 'OPERATOR'
ON CONFLICT (USERNAME) DO UPDATE
SET ID_ROLE = EXCLUDED.ID_ROLE,
    ID_DEPARTEMEN = COALESCE(USERS.ID_DEPARTEMEN, EXCLUDED.ID_DEPARTEMEN),
    STATUS = 'active',
    UPDATED_BY = COALESCE(EXCLUDED.UPDATED_BY, USERS.UPDATED_BY);
