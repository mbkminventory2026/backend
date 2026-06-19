# Rekonsiliasi Material Design

Tanggal: 2026-06-15
Branch: `feat/rekon`
Status: Approved in chat, pending implementation

## Latar Belakang

Fitur rekonsiliasi material akan dipakai oleh:

- `ADMIN_PRODUKSI`: create, read, update
- `ADMIN_KEUANGAN`: read only
- `ADMIN_GUDANG`: read only

Rekonsiliasi ini bukan laporan tunggal biasa, tetapi dokumen campuran yang menggabungkan:

- snapshot data `WO`
- snapshot data `material list`
- agregasi `report pengiriman`
- total `spreading cutting plan`
- input manual admin produksi

Output detail harus cukup untuk membentuk tiga blok utama:

1. ringkasan warna per WO
2. tabel detail material per baris
3. tabel penerimaan dan kalkulasi konsumsi/balance

## Tujuan

Menyediakan modul backend rekonsiliasi material yang:

- dapat dibuat dari `id_wo`
- menghasilkan snapshot awal otomatis dari data upstream
- tetap mendukung input manual admin produksi
- dapat di-refresh dari data sumber tanpa menghilangkan input manual
- siap dipakai frontend untuk render tabel campuran yang kompleks

## Ruang Lingkup Batch Pertama

Termasuk:

- desain tabel baru rekonsiliasi
- migration database
- query `sqlc`
- usecase backend create/list/detail/update/refresh
- role access backend
- swagger docs

Belum termasuk:

- export excel/pdf
- soft delete/archive
- manager/client access
- realtime update
- historical versioning per save

## Keputusan Desain

### 1. Struktur Lean 3 Tabel

Batch pertama menggunakan tiga tabel baru:

1. `rekonsiliasi`
2. `rekonsiliasi_material_row`
3. `rekonsiliasi_terima_entry`

Ringkasan warna tidak dibuat tabel sendiri. Section warna akan dihitung saat read dari sumber data upstream.

### 2. Backend-First Snapshot Document

Saat dokumen rekonsiliasi dibuat:

- backend mengambil snapshot header dari `WO` dan relasi terkait
- backend membentuk `material rows` awal
- backend menghitung `color summaries` saat detail dibaca

Dokumen hasil rekonsiliasi dianggap sebagai snapshot bisnis yang boleh di-refresh dari data sumber, tetapi input manual user harus dipertahankan.

### 3. Full-Document Update

Update rekonsiliasi dilakukan lewat satu endpoint detail:

- `PUT /api/v1/rekonsiliasi/:id`

Pendekatan ini dipilih agar:

- FE lebih mudah menyimpan form nested
- perhitungan bisa direfresh ulang sekali setelah save
- implementasi batch pertama lebih stabil

## Model Data

### Tabel `rekonsiliasi`

Satu row per dokumen rekonsiliasi untuk satu `WO`.

Kolom inti:

- `id_rekonsiliasi`
- `id_wo`
- `nama_wo`
- `jasa`
- `no_po`
- `delivery_date`
- `buyer`
- `brand`
- `style`
- `qty_po`
- `plan_cut_total`
- `cons_baju_summary`
- `nama_bahan`
- `warna_kain_summary` `jsonb`
- `created_by`
- `updated_by`
- `created_at`
- `updated_at`

Catatan:

- field di atas adalah snapshot bisnis saat create/refresh
- `warna_kain_summary` boleh berupa array string sederhana untuk header kiri

### Tabel `rekonsiliasi_material_row`

Satu row per baris detail material seperti nomor `1..9` pada contoh.

Kolom inti:

- `id_rekonsiliasi_material_row`
- `id_rekonsiliasi`
- `row_no`
- `kategori`
- `description`
- `size_label`
- `ratio_source`
- `ratio_input`
- `qty_per_pcs_input`
- `qty_wo`
- `toleransi`
- `satuan`
- `qty_actual_kirim_manual`
- `reject`
- `retur`
- `keterangan`
- `created_at`
- `updated_at`

Kolom turunan yang disimpan:

- `total_terima`
- `qty_actual_kirim_source`
- `qty_actual_kirim`
- `cons_actual`
- `balance`
- `last_balance`

Catatan:

- field source berasal dari `WO`, `material list`, atau query agregat lain
- field manual hanya diisi admin produksi
- field turunan dihitung backend setiap create/update/refresh

### Tabel `rekonsiliasi_terima_entry`

Menampung sub-entry penerimaan seperti:

- `I`
- `U/KK BB 005`
- `Ambil ACC DAUD`

Kolom inti:

- `id_rekonsiliasi_terima_entry`
- `id_rekonsiliasi_material_row`
- `entry_type`
- `entry_label`
- `qty`
- `note`
- `created_at`
- `updated_at`

Nilai `entry_type`:

- `awal`
- `untuk`
- `ambil`

Aturan tanda:

- `awal` menambah total terima
- `ambil` menambah total terima
- `untuk` mengurangi total terima

## Sumber Data

### Header Rekonsiliasi

Diambil dari:

- `WO`
- relasi PO/client jika tersedia

Field yang dibentuk:

- `nama_wo`
- `no_po`
- `buyer`
- `brand`
- `style`
- `qty_po`
- `delivery`
- `nama_bahan`
- `cons_baju_summary`

### Ringkasan Warna

Tidak disimpan ke tabel.

Dihitung dari:

- warna pada `WO`
- qty per warna dari `WO`
- total qty `report_pengiriman` per warna dan size

Output:

- `color`
- `qty_order`
- `qty_kirim`
- `balance`
- `size_breakdown[]`

### Plan Cut Total

Diambil dari agregasi `spreading_cutting_plan`.

### Material Rows

Dibentuk dari kombinasi:

- `WO shell`
- `WO trim`
- `material list`

## Aturan Hitung

### Ringkasan Warna

- `qty_order = qty warna pada WO`
- `qty_kirim = total qty report_pengiriman`
- `balance = qty_order - qty_kirim`

### Qty Actual Kirim

`qty_actual_kirim` dihitung sebagai:

- `qty_actual_kirim_source + qty_actual_kirim_manual`

Keterangan:

- `qty_actual_kirim_source` berasal dari agregasi `report_pengiriman`
- `qty_actual_kirim_manual` berasal dari input user di row

### Total Terima

`total_terima` dihitung dari seluruh `rekonsiliasi_terima_entry` pada row:

- `awal` ditambah
- `ambil` ditambah
- `untuk` dikurang

### Formula Material Row

Formula batch pertama mengikuti definisi user secara eksplisit:

- `cons_actual = total_terima - (qty_actual_kirim * qty_per_pcs_input)`
- `balance = total_terima - cons_actual`
- `last_balance = balance - reject - retur`

Catatan:

- formula ini dipertahankan apa adanya agar sesuai cara hitung bisnis yang sedang dipakai
- validasi domain lanjutan dapat dibahas di batch berikutnya bila diperlukan

## Role Access

### Admin Produksi

- `GET /api/v1/rekonsiliasi`
- `POST /api/v1/rekonsiliasi`
- `GET /api/v1/rekonsiliasi/:id`
- `PUT /api/v1/rekonsiliasi/:id`
- `POST /api/v1/rekonsiliasi/:id/refresh`

### Admin Keuangan

- `GET /api/v1/rekonsiliasi`
- `GET /api/v1/rekonsiliasi/:id`

### Admin Gudang

- `GET /api/v1/rekonsiliasi`
- `GET /api/v1/rekonsiliasi/:id`

## API Design

### 1. List Rekonsiliasi

`GET /api/v1/rekonsiliasi`

Query:

- `page`
- `pageSize`
- `q`
- `sortBy`
- `sortDesc`
- `idWo`

Response item:

- `id_rekonsiliasi`
- `id_wo`
- `nama_wo`
- `buyer`
- `brand`
- `style`
- `qty_po`
- `plan_cut_total`
- `created_at`
- `updated_at`
- `created_by_username`
- `updated_by_username`

### 2. Create Rekonsiliasi

`POST /api/v1/rekonsiliasi`

Request:

```json
{
  "id_wo": 76
}
```

Perilaku:

- menolak jika `WO` tidak ditemukan
- menolak jika rekonsiliasi untuk `WO` tersebut sudah ada, bila aturan bisnis memang satu WO satu rekonsiliasi
- auto-generate header snapshot
- auto-generate material rows

### 3. Get Detail Rekonsiliasi

`GET /api/v1/rekonsiliasi/:id`

Response:

- `header`
- `color_summaries[]`
- `material_rows[]`
  - nested `terima_entries[]`

### 4. Update Rekonsiliasi

`PUT /api/v1/rekonsiliasi/:id`

Body hanya berisi field editable:

- `material_rows[].ratio_input`
- `material_rows[].qty_per_pcs_input`
- `material_rows[].qty_actual_kirim_manual`
- `material_rows[].reject`
- `material_rows[].retur`
- `material_rows[].keterangan`
- `material_rows[].terima_entries[]`

Field source tidak boleh diubah manual.

### 5. Refresh Rekonsiliasi

`POST /api/v1/rekonsiliasi/:id/refresh`

Perilaku:

- rebuild field source dari data upstream terbaru
- pertahankan field manual user
- hitung ulang seluruh turunan

## Aturan Update dan Refresh

### Update

Saat `PUT`:

- backend menyimpan field manual
- backend replace `terima_entries` per row dari payload terbaru
- backend menghitung ulang semua kolom turunan

### Refresh

Saat `POST refresh`:

- backend baca ulang `WO`
- backend baca ulang `material list`
- backend baca ulang `report_pengiriman`
- backend baca ulang `spreading_cutting_plan`
- backend update snapshot source
- backend tidak menimpa field manual:
  - `ratio_input`
  - `qty_per_pcs_input`
  - `qty_actual_kirim_manual`
  - `reject`
  - `retur`
  - `keterangan`
  - `terima_entries`

## Validasi

- `id_wo` wajib valid saat create
- `ratio_input >= 0`
- `qty_per_pcs_input >= 0`
- `qty_actual_kirim_manual >= 0`
- `reject >= 0`
- `retur >= 0`
- `entry_type` harus salah satu dari:
  - `awal`
  - `untuk`
  - `ambil`
- `qty` pada `terima_entries` harus `>= 0`

## Audit Log

Setelah modul ini aktif, audit log batch berikutnya harus mencatat:

- create rekonsiliasi
- update rekonsiliasi
- refresh rekonsiliasi

Entity type yang direkomendasikan:

- `rekonsiliasi`
- `rekonsiliasi_material_row`

## Integrasi Frontend yang Disiapkan

Backend detail response harus siap untuk page FE yang memuat:

1. section ringkasan warna
2. section detail buyer/header
3. section tabel material rows
4. section nested input `terima`

FE dapat memakai satu endpoint detail untuk merender semua blok.

## Acceptance Criteria

Batch backend pertama dianggap selesai jika:

- migration tabel baru berhasil
- `make db-gen` berhasil
- `POST /api/v1/rekonsiliasi` membuat dokumen dari `WO`
- `GET /api/v1/rekonsiliasi` menampilkan list
- `GET /api/v1/rekonsiliasi/:id` mengembalikan data lengkap
- `PUT /api/v1/rekonsiliasi/:id` menyimpan field manual dan menghitung ulang turunan
- `POST /api/v1/rekonsiliasi/:id/refresh` mengupdate field source tanpa menghapus input manual
- role access sesuai:
  - produksi `CRU`
  - keuangan/gudang `R`
- `go build ./...`, `make swag`, dan `make lint` lolos

## Rencana Implementasi Bertahap

Commit granular yang ditargetkan:

1. `docs: add material reconciliation design spec`
2. `feat(db): add reconciliation tables`
3. `feat(db): add reconciliation sqlc queries`
4. `feat(backend): add reconciliation models and usecase foundation`
5. `feat(backend): add reconciliation create and list endpoints`
6. `feat(backend): add reconciliation detail and update endpoints`
7. `feat(backend): add reconciliation refresh endpoint`
8. `feat(backend): enforce reconciliation role access`
9. `chore: regenerate docs and run backend quality checks`

Frontend dan smoke testing fullstack dibahas setelah backend batch pertama stabil.
