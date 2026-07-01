-- Seed WO permissions yang belum pernah di-INSERT ke HAK_AKSES via migration
INSERT INTO HAK_AKSES (KODE_PERMISSION, NAMA_HALAMAN, DESKRIPSI, DOMAIN_PERMISSION, AKSI_PERMISSION)
VALUES
    ('WO_READ',   'Work Order Read',   'Allows reading work order list and detail', 'work_order', 'read'),
    ('WO_CREATE', 'Work Order Create', 'Allows creating new work orders',            'work_order', 'create'),
    ('WO_UPDATE', 'Work Order Update', 'Allows updating work orders',                'work_order', 'update'),
    ('WO_CLOSE',  'Work Order Close',  'Allows closing / finalizing work orders',    'work_order', 'close')
ON CONFLICT (KODE_PERMISSION) DO NOTHING;

-- ADMIN_PRODUKSI: baca, buat, dan ubah Work Order
INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
SELECT r.ID_ROLE, h.ID_HAK_AKSES
FROM ROLES r
JOIN HAK_AKSES h ON h.KODE_PERMISSION IN ('WO_READ', 'WO_CREATE', 'WO_UPDATE')
WHERE r.NAMA_ROLE = 'ADMIN_PRODUKSI'
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;

-- ADMIN_KEUANGAN: baca dan tutup Work Order
-- (assignment WO_CLOSE sempat dicoba di migration 000034 namun WO_CLOSE belum ada saat itu)
INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
SELECT r.ID_ROLE, h.ID_HAK_AKSES
FROM ROLES r
JOIN HAK_AKSES h ON h.KODE_PERMISSION IN ('WO_READ', 'WO_CLOSE')
WHERE r.NAMA_ROLE = 'ADMIN_KEUANGAN'
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;

-- MANAGER: hanya baca Work Order
INSERT INTO ROLE_HAK_AKSES (ID_ROLE, ID_HAK_AKSES)
SELECT r.ID_ROLE, h.ID_HAK_AKSES
FROM ROLES r
JOIN HAK_AKSES h ON h.KODE_PERMISSION = 'WO_READ'
WHERE r.NAMA_ROLE = 'MANAGER'
ON CONFLICT (ID_ROLE, ID_HAK_AKSES) DO NOTHING;
