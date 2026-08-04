WITH owned_permissions AS (
    SELECT ID_HAK_AKSES
    FROM HAK_AKSES
    WHERE (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION) IN (
        ('SYSTEM_BACKUP_CREATE', 'System Backup Create', 'Allows starting an encrypted system backup', 'system_backup', 'create'),
        ('SYSTEM_BACKUP_READ', 'System Backup Read', 'Allows reading backup status and completed backup history', 'system_backup', 'read'),
        ('SYSTEM_BACKUP_DOWNLOAD', 'System Backup Download', 'Allows downloading a completed encrypted backup', 'system_backup', 'download')
    )
)
DELETE FROM ROLE_HAK_AKSES
WHERE ID_HAK_AKSES IN (SELECT ID_HAK_AKSES FROM owned_permissions);

DELETE FROM HAK_AKSES
WHERE (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION) IN (
    ('SYSTEM_BACKUP_CREATE', 'System Backup Create', 'Allows starting an encrypted system backup', 'system_backup', 'create'),
    ('SYSTEM_BACKUP_READ', 'System Backup Read', 'Allows reading backup status and completed backup history', 'system_backup', 'read'),
    ('SYSTEM_BACKUP_DOWNLOAD', 'System Backup Download', 'Allows downloading a completed encrypted backup', 'system_backup', 'download')
);
