DROP INDEX IF EXISTS idx_report_pengiriman_id_wo_shell_size_created_at;
DROP INDEX IF EXISTS idx_report_packing_id_wo_shell_size_created_at;
DROP INDEX IF EXISTS idx_report_qc_finish_id_wo_shell_size_created_at;
DROP INDEX IF EXISTS idx_report_sewing_id_wo_shell_size_created_at;
DROP INDEX IF EXISTS idx_report_cutting_id_wo_shell_size_created_at;

DROP INDEX IF EXISTS idx_work_order_shell_size_created_at;
DROP INDEX IF EXISTS idx_work_order_shell_size_size_lower;
DROP INDEX IF EXISTS idx_work_order_shell_size_id_wo_shell;
DROP INDEX IF EXISTS idx_work_order_shell_id_wo;

DROP INDEX IF EXISTS idx_material_list_id_wo;
DROP INDEX IF EXISTS idx_material_list_description_lower;

DROP INDEX IF EXISTS idx_surat_jalan_internal_created_at;

DROP INDEX IF EXISTS idx_surat_jalan_client_id_material_list;
DROP INDEX IF EXISTS idx_surat_jalan_client_tanggal;
DROP INDEX IF EXISTS idx_surat_jalan_client_created_at;
DROP INDEX IF EXISTS idx_surat_jalan_client_keterangan_lower;

DROP INDEX IF EXISTS idx_packing_list_id_surat_jalan_internal;
DROP INDEX IF EXISTS idx_packing_list_id_wo;
DROP INDEX IF EXISTS idx_packing_list_created_at;

DROP INDEX IF EXISTS idx_po_internal_id_pr_internal;
DROP INDEX IF EXISTS idx_po_internal_ship_date;
DROP INDEX IF EXISTS idx_po_internal_tanggal;
DROP INDEX IF EXISTS idx_po_internal_created_at;
DROP INDEX IF EXISTS idx_po_internal_cpo_lower;
DROP INDEX IF EXISTS idx_po_internal_currency_lower;
DROP INDEX IF EXISTS idx_po_internal_supplier_name_lower;
DROP INDEX IF EXISTS idx_po_internal_nama_po_lower;

DROP INDEX IF EXISTS idx_pr_internal_id_user;
DROP INDEX IF EXISTS idx_pr_internal_id_wo;
DROP INDEX IF EXISTS idx_pr_internal_tanggal;
DROP INDEX IF EXISTS idx_pr_internal_created_at;
DROP INDEX IF EXISTS idx_pr_internal_status_lower;
DROP INDEX IF EXISTS idx_pr_internal_projek_lower;
DROP INDEX IF EXISTS idx_pr_internal_vendor_name_lower;
DROP INDEX IF EXISTS idx_pr_internal_departemen_lower;
DROP INDEX IF EXISTS idx_pr_internal_nama_lower;

DROP INDEX IF EXISTS idx_work_order_delivery;
DROP INDEX IF EXISTS idx_work_order_id_po_client_item;
DROP INDEX IF EXISTS idx_work_order_created_at;
DROP INDEX IF EXISTS idx_work_order_status_lower;
DROP INDEX IF EXISTS idx_work_order_model_lower;
DROP INDEX IF EXISTS idx_work_order_buyer_lower;

DROP INDEX IF EXISTS idx_penanggung_jawab_id_po_client;

DROP INDEX IF EXISTS idx_po_client_item_style_lower;
DROP INDEX IF EXISTS idx_po_client_item_id_po_client;

DROP INDEX IF EXISTS idx_po_client_id_mitra;
DROP INDEX IF EXISTS idx_po_client_delivery;
DROP INDEX IF EXISTS idx_po_client_tanggal;
DROP INDEX IF EXISTS idx_po_client_created_at;
DROP INDEX IF EXISTS idx_po_client_season_lower;
DROP INDEX IF EXISTS idx_po_client_po_number_lower;

DROP INDEX IF EXISTS idx_users_id_mitra;
DROP INDEX IF EXISTS idx_users_id_departemen;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_status_lower;
DROP INDEX IF EXISTS idx_users_username_lower;

DROP INDEX IF EXISTS idx_barang_id_mitra;
DROP INDEX IF EXISTS idx_barang_id_jenis_barang;
DROP INDEX IF EXISTS idx_barang_created_at;
DROP INDEX IF EXISTS idx_barang_kode_lower;
DROP INDEX IF EXISTS idx_barang_nama_lower;

DROP INDEX IF EXISTS idx_hak_akses_created_at;
DROP INDEX IF EXISTS idx_hak_akses_nama_lower;

DROP INDEX IF EXISTS idx_mitra_created_at;
DROP INDEX IF EXISTS idx_mitra_no_telp_lower;
DROP INDEX IF EXISTS idx_mitra_email_lower;
DROP INDEX IF EXISTS idx_mitra_tipe_lower;
DROP INDEX IF EXISTS idx_mitra_nama_lower;

DROP INDEX IF EXISTS idx_jenis_barang_created_at;
DROP INDEX IF EXISTS idx_jenis_barang_kode_lower;
DROP INDEX IF EXISTS idx_jenis_barang_nama_lower;

DROP INDEX IF EXISTS idx_departemen_created_at;
DROP INDEX IF EXISTS idx_departemen_nama_lower;
