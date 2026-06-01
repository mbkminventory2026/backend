CREATE INDEX IF NOT EXISTS idx_departemen_nama_lower ON DEPARTEMEN (LOWER(NAMA_DEPARTEMEN));
CREATE INDEX IF NOT EXISTS idx_departemen_created_at ON DEPARTEMEN (created_at);

CREATE INDEX IF NOT EXISTS idx_jenis_barang_nama_lower ON JENIS_BARANG (LOWER(NAMA_JENIS_BARANG));
CREATE INDEX IF NOT EXISTS idx_jenis_barang_kode_lower ON JENIS_BARANG (LOWER(KODE));
CREATE INDEX IF NOT EXISTS idx_jenis_barang_created_at ON JENIS_BARANG (created_at);

CREATE INDEX IF NOT EXISTS idx_mitra_nama_lower ON MITRA (LOWER(NAMA_PERUSAHAAN));
CREATE INDEX IF NOT EXISTS idx_mitra_tipe_lower ON MITRA (LOWER(TIPE_PERUSAHAAN));
CREATE INDEX IF NOT EXISTS idx_mitra_email_lower ON MITRA (LOWER(EMAIL));
CREATE INDEX IF NOT EXISTS idx_mitra_no_telp_lower ON MITRA (LOWER(NO_TELP));
CREATE INDEX IF NOT EXISTS idx_mitra_created_at ON MITRA (created_at);

CREATE INDEX IF NOT EXISTS idx_hak_akses_nama_lower ON HAK_AKSES (LOWER(NAMA_HALAMAN));
CREATE INDEX IF NOT EXISTS idx_hak_akses_created_at ON HAK_AKSES (created_at);

CREATE INDEX IF NOT EXISTS idx_barang_nama_lower ON BARANG (LOWER(NAMA_BARANG));
CREATE INDEX IF NOT EXISTS idx_barang_kode_lower ON BARANG (LOWER(KODE));
CREATE INDEX IF NOT EXISTS idx_barang_created_at ON BARANG (created_at);
CREATE INDEX IF NOT EXISTS idx_barang_id_jenis_barang ON BARANG (ID_JENIS_BARANG);
CREATE INDEX IF NOT EXISTS idx_barang_id_mitra ON BARANG (ID_MITRA);

CREATE INDEX IF NOT EXISTS idx_users_username_lower ON USERS (LOWER(USERNAME));
CREATE INDEX IF NOT EXISTS idx_users_status_lower ON USERS (LOWER(STATUS));
CREATE INDEX IF NOT EXISTS idx_users_created_at ON USERS (created_at);
CREATE INDEX IF NOT EXISTS idx_users_id_departemen ON USERS (ID_DEPARTEMEN);
CREATE INDEX IF NOT EXISTS idx_users_id_mitra ON USERS (ID_MITRA);

CREATE INDEX IF NOT EXISTS idx_po_client_po_number_lower ON PO_CLIENT (LOWER(PO_NUMBER));
CREATE INDEX IF NOT EXISTS idx_po_client_season_lower ON PO_CLIENT (LOWER(SEASON));
CREATE INDEX IF NOT EXISTS idx_po_client_created_at ON PO_CLIENT (created_at);
CREATE INDEX IF NOT EXISTS idx_po_client_tanggal ON PO_CLIENT (TANGGAL);
CREATE INDEX IF NOT EXISTS idx_po_client_delivery ON PO_CLIENT (DELIVERY);
CREATE INDEX IF NOT EXISTS idx_po_client_id_mitra ON PO_CLIENT (ID_MITRA);

CREATE INDEX IF NOT EXISTS idx_po_client_item_id_po_client ON PO_CLIENT_ITEM (ID_PO_CLIENT);
CREATE INDEX IF NOT EXISTS idx_po_client_item_style_lower ON PO_CLIENT_ITEM (LOWER(STYLE));

CREATE INDEX IF NOT EXISTS idx_penanggung_jawab_id_po_client ON PENANGGUNG_JAWAB (ID_PO_CLIENT);

CREATE INDEX IF NOT EXISTS idx_work_order_buyer_lower ON WORK_ORDER (LOWER(BUYER));
CREATE INDEX IF NOT EXISTS idx_work_order_model_lower ON WORK_ORDER (LOWER(MODEL));
CREATE INDEX IF NOT EXISTS idx_work_order_status_lower ON WORK_ORDER (LOWER(STATUS));
CREATE INDEX IF NOT EXISTS idx_work_order_created_at ON WORK_ORDER (created_at);
CREATE INDEX IF NOT EXISTS idx_work_order_id_po_client_item ON WORK_ORDER (ID_PO_CLIENT_ITEM);
CREATE INDEX IF NOT EXISTS idx_work_order_delivery ON WORK_ORDER (DELIVERY);

CREATE INDEX IF NOT EXISTS idx_pr_internal_nama_lower ON PR_INTERNAL (LOWER(NAMA));
CREATE INDEX IF NOT EXISTS idx_pr_internal_departemen_lower ON PR_INTERNAL (LOWER(DEPARTEMEN));
CREATE INDEX IF NOT EXISTS idx_pr_internal_vendor_name_lower ON PR_INTERNAL (LOWER(VENDOR_NAME));
CREATE INDEX IF NOT EXISTS idx_pr_internal_projek_lower ON PR_INTERNAL (LOWER(PROJEK));
CREATE INDEX IF NOT EXISTS idx_pr_internal_status_lower ON PR_INTERNAL (LOWER(STATUS));
CREATE INDEX IF NOT EXISTS idx_pr_internal_created_at ON PR_INTERNAL (created_at);
CREATE INDEX IF NOT EXISTS idx_pr_internal_tanggal ON PR_INTERNAL (TANGGAL);
CREATE INDEX IF NOT EXISTS idx_pr_internal_id_wo ON PR_INTERNAL (ID_WO);
CREATE INDEX IF NOT EXISTS idx_pr_internal_id_user ON PR_INTERNAL (ID_USER);

CREATE INDEX IF NOT EXISTS idx_po_internal_nama_po_lower ON PO_INTERNAL (LOWER(NAMA_PO));
CREATE INDEX IF NOT EXISTS idx_po_internal_supplier_name_lower ON PO_INTERNAL (LOWER(SUPPLIER_NAME));
CREATE INDEX IF NOT EXISTS idx_po_internal_currency_lower ON PO_INTERNAL (LOWER(CURRENCY));
CREATE INDEX IF NOT EXISTS idx_po_internal_cpo_lower ON PO_INTERNAL (LOWER(CPO));
CREATE INDEX IF NOT EXISTS idx_po_internal_created_at ON PO_INTERNAL (created_at);
CREATE INDEX IF NOT EXISTS idx_po_internal_tanggal ON PO_INTERNAL (TANGGAL);
CREATE INDEX IF NOT EXISTS idx_po_internal_ship_date ON PO_INTERNAL (SHIP_DATE);
CREATE INDEX IF NOT EXISTS idx_po_internal_id_pr_internal ON PO_INTERNAL (ID_PR_INTERNAL);

CREATE INDEX IF NOT EXISTS idx_packing_list_created_at ON PACKING_LIST (created_at);
CREATE INDEX IF NOT EXISTS idx_packing_list_id_wo ON PACKING_LIST (ID_WO);
CREATE INDEX IF NOT EXISTS idx_packing_list_id_surat_jalan_internal ON PACKING_LIST (ID_SURAT_JALAN_INTERNAL);

CREATE INDEX IF NOT EXISTS idx_surat_jalan_client_keterangan_lower ON SURAT_JALAN_CLIENT (LOWER(KETERANGAN));
CREATE INDEX IF NOT EXISTS idx_surat_jalan_client_created_at ON SURAT_JALAN_CLIENT (created_at);
CREATE INDEX IF NOT EXISTS idx_surat_jalan_client_tanggal ON SURAT_JALAN_CLIENT (TANGGAL);
CREATE INDEX IF NOT EXISTS idx_surat_jalan_client_id_material_list ON SURAT_JALAN_CLIENT (ID_MATERIAL_LIST);

CREATE INDEX IF NOT EXISTS idx_surat_jalan_internal_created_at ON SURAT_JALAN_INTERNAL (created_at);

CREATE INDEX IF NOT EXISTS idx_material_list_description_lower ON MATERIAL_LIST (LOWER(DESCRIPTION));
CREATE INDEX IF NOT EXISTS idx_material_list_id_wo ON MATERIAL_LIST (ID_WO);

CREATE INDEX IF NOT EXISTS idx_work_order_shell_id_wo ON WORK_ORDER_SHELL (ID_WO);
CREATE INDEX IF NOT EXISTS idx_work_order_shell_size_id_wo_shell ON WORK_ORDER_SHELL_SIZE (ID_WO_SHELL);
CREATE INDEX IF NOT EXISTS idx_work_order_shell_size_size_lower ON WORK_ORDER_SHELL_SIZE (LOWER(SIZE));
CREATE INDEX IF NOT EXISTS idx_work_order_shell_size_created_at ON WORK_ORDER_SHELL_SIZE (created_at);

CREATE INDEX IF NOT EXISTS idx_report_cutting_id_wo_shell_size_created_at ON REPORT_CUTTING (ID_WO_SHELL_SIZE, created_at);
CREATE INDEX IF NOT EXISTS idx_report_sewing_id_wo_shell_size_created_at ON REPORT_SEWING (ID_WO_SHELL_SIZE, created_at);
CREATE INDEX IF NOT EXISTS idx_report_qc_finish_id_wo_shell_size_created_at ON REPORT_QC_FINISH (ID_WO_SHELL_SIZE, created_at);
CREATE INDEX IF NOT EXISTS idx_report_packing_id_wo_shell_size_created_at ON REPORT_PACKING (ID_WO_SHELL_SIZE, created_at);
CREATE INDEX IF NOT EXISTS idx_report_pengiriman_id_wo_shell_size_created_at ON REPORT_PENGIRIMAN (ID_WO_SHELL_SIZE, created_at);
