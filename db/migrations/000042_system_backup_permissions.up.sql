-- KODE_PERMISSION is unique. Any collision must abort this migration so these
-- records are never adopted from or merged with pre-existing permissions.
INSERT INTO HAK_AKSES (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION)
VALUES
    ('SYSTEM_BACKUP_CREATE', 'System Backup Create', 'Allows starting an encrypted system backup', 'system_backup', 'create'),
    ('SYSTEM_BACKUP_READ', 'System Backup Read', 'Allows reading backup status and completed backup history', 'system_backup', 'read'),
    ('SYSTEM_BACKUP_DOWNLOAD', 'System Backup Download', 'Allows downloading a completed encrypted backup', 'system_backup', 'download');

INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
SELECT r.ID_ROLE, h.ID_HAK_AKSES
FROM ROLES r
JOIN HAK_AKSES h ON h.KODE_PERMISSION IN (
    'SYSTEM_BACKUP_CREATE',
    'SYSTEM_BACKUP_READ',
    'SYSTEM_BACKUP_DOWNLOAD'
)
WHERE r.NAMA_ROLE = 'ADMIN_SISTEM'
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;

INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
SELECT r.ID_ROLE, h.ID_HAK_AKSES
FROM ROLES r
JOIN HAK_AKSES h ON h.KODE_PERMISSION = 'SYSTEM_BACKUP_READ'
WHERE r.NAMA_ROLE = 'MANAGER'
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;
