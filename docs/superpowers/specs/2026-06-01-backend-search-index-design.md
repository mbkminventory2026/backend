# Backend Search Index Design

Tanggal: 2026-06-01
Topik: server-side search, sorting, pagination, dan database indexing untuk tabel master dan transaksi utama

## Latar Belakang

Frontend saat ini sudah memakai URL query params table seperti `filter`, `sortBy`, `sortDesc`, `page`, dan `pageSize`. Namun sebagian besar endpoint backend masih:

- hanya mengembalikan seluruh data tanpa server-side filtering,
- atau hanya mendukung pagination dasar tanpa sorting dinamis,
- atau belum memiliki index database yang cocok untuk pola pencarian dan pengurutan dari frontend.

Akibatnya, filtering dan sorting masih terlalu banyak dilakukan di frontend, sementara backend belum membantu mengurangi beban query dan transfer data.

## Tujuan

- Memindahkan search, sorting, dan pagination utama ke backend.
- Menyediakan query yang konsisten untuk halaman table di frontend.
- Menambahkan index database untuk mempercepat query list yang sering dipakai.
- Menjaga kompatibilitas dengan arsitektur proyek: `migration -> sql query -> sqlc -> usecase -> handler -> swagger -> lint`.

## Non-Goals

- Tidak mengubah bentuk response sukses global dari `pkg/response`.
- Tidak memperkenalkan full-text search `tsvector` pada batch ini.
- Tidak membuat generic endpoint lintas tabel.
- Tidak mengubah flow create/update/delete.
- Tidak memasukkan halaman yang di frontend belum punya kontrak list backend yang nyata.

## Scope Endpoint

### Master Data

- `GET /api/v1/master/departemen`
- `GET /api/v1/master/jenis-barang`
- `GET /api/v1/master/mitra`
- `GET /api/v1/master/barang`
- `GET /api/v1/master/permissions`
- `GET /api/v1/users`

### Transaksi dan Operasional

- `GET /api/v1/po-clients`
- `GET /api/v1/pr-internals`
- `GET /api/v1/po-internals`
- `GET /api/v1/work-orders`
- `GET /api/v1/production/summary`
- `GET /api/v1/packing-lists`
- `GET /api/v1/surat-jalan-clients`
- `GET /api/v1/surat-jalan-internals`

### Out of Scope untuk batch ini

- `company`, karena bukan list table.
- `report-penerimaan` dan `report-pengiriman`, karena frontend memiliki page, tetapi backend saat ini belum memiliki list endpoint yang benar-benar tersedia untuk kontrak tersebut. Menambahkan keduanya berarti membuat API baru, bukan sekadar upgrade search/index pada endpoint yang sudah ada.

## Kontrak Query yang Ditargetkan

Semua endpoint list dalam scope akan menerima query params dengan perilaku berikut:

- `page`: nomor halaman, default `1`
- `pageSize`: ukuran halaman untuk pola frontend table
- `limit`: tetap didukung untuk kompatibilitas endpoint transaksi lama
- `q`: kata kunci pencarian untuk pola frontend master-data
- `search`: alias pencarian untuk endpoint transaksi yang sudah ada
- `sortBy`: nama kolom sort yang diizinkan
- `sortDesc`: boolean untuk arah sort descending

### Aturan kompatibilitas

- Jika `pageSize` ada, backend akan memprioritaskan `pageSize` sebagai limit.
- Jika `pageSize` tidak ada, backend memakai `limit`.
- Jika `q` ada dan `search` kosong, backend memakai `q` sebagai search term.
- Jika `search` ada, backend memakai `search`.
- Jika `sortBy` tidak valid, backend fallback ke default sort per endpoint.

## Bentuk Response

### Master data dan users

Tetap mengikuti pola FE saat ini:

- body `data` berisi array item
- total item dikirim melalui header `x-total-count`

Tujuan keputusan ini adalah menghindari refactor besar di halaman master-data yang sekarang membaca total dari header.

### Transaksi

Tetap memakai bentuk response yang sudah ada:

- `data.items`
- `data.pagination`

Perubahan hanya pada kemampuan search dan sorting, bukan pada struktur payload.

## Desain Backend

## 1. Model Filter Baru

Akan ditambahkan model filter list generik untuk endpoint table, minimal memuat:

- `Page`
- `Limit`
- `Search`
- `SortBy`
- `SortDesc`

Untuk endpoint yang sudah punya filter khusus seperti `ProductionSummaryFilter`, field sort akan ditambahkan tanpa menghapus filter existing.

## 2. Handler Parsing

Handler list akan diubah agar:

- membaca `page`, `pageSize`, `limit`
- membaca `q` dan `search`
- membaca `sortBy`, `sortDesc`
- melakukan validasi angka dan fallback default
- mengisi header `x-total-count` untuk master data dan users

Validasi kolom sort tidak dilakukan langsung dari raw query. Handler atau usecase akan meneruskan value ke whitelist sorter per endpoint.

## 3. Usecase Normalization

Usecase akan memiliki helper untuk:

- normalisasi pagination dari `page/pageSize/limit`
- normalisasi search dari `q/search`
- normalisasi `sortDesc`
- whitelist `sortBy` per endpoint

Setiap endpoint akan punya daftar kolom sort yang eksplisit, supaya query tetap aman dan perilaku FE stabil.

## 4. Query sqlc

Pendekatan yang digunakan:

- query list khusus per endpoint
- query count khusus per endpoint untuk master data dan users
- sorting dinamis aman dengan `CASE WHEN` di `ORDER BY`
- pencarian menggunakan `ILIKE '%term%'`

Contoh pola:

- master data: query `ListBarangIndexed`, `CountBarangIndexed`
- transaksi: upgrade query existing `ListPOClients`, `ListWorkOrders`, dan lain-lain agar menerima `sort_by` dan `sort_desc`

Alasan memakai query spesifik:

- natural untuk `sqlc`
- lebih mudah dijaga dibanding endpoint generik lintas tabel
- tiap endpoint punya join dan kolom pencarian berbeda

## 5. Database Indexing

Migration baru akan menambah index untuk pola query aktual dari frontend.

Prinsip index:

- btree index untuk kolom default sort seperti `created_at` dan primary relation
- functional index `lower(column)` untuk kolom teks yang sering dicari
- index foreign key dan kolom join yang sering dipakai pada list transaksi

Contoh target index:

### Master data

- `DEPARTEMEN(lower(nama_departemen))`
- `JENIS_BARANG(lower(nama_jenis_barang))`
- `JENIS_BARANG(lower(kode))`
- `MITRA(lower(nama_perusahaan))`
- `MITRA(lower(email))`
- `HAK_AKSES(lower(nama_halaman))`
- `BARANG(lower(nama_barang))`
- `BARANG(lower(kode))`
- `BARANG(created_at)`
- `USERS(lower(username))`
- `USERS(status)`

### Transaksi

- `PO_CLIENT(lower(po_number))`
- `PO_CLIENT(lower(season))`
- `PO_CLIENT(created_at)`
- `PR_INTERNAL(lower(nama))`
- `PR_INTERNAL(lower(vendor_name))`
- `PR_INTERNAL(lower(projek))`
- `PO_INTERNAL(lower(nama_po))`
- `PO_INTERNAL(lower(supplier_name))`
- `PO_INTERNAL(lower(cpo))`
- `WORK_ORDER(lower(buyer))`
- `WORK_ORDER(lower(model))`
- `WORK_ORDER(status)`
- `PACKING_LIST(created_at)`
- `SURAT_JALAN_CLIENT(lower(keterangan))`
- join-support index seperti foreign key pada `id_mitra`, `id_po_client`, `id_po_client_item`, `id_wo`, `id_material_list`, `id_wo_shell_size` bila belum optimal

Catatan:

- index `lower(...)` tidak mengubah `ILIKE` menjadi sempurna untuk semua kondisi wildcard, tetapi tetap membantu untuk banyak workload pencarian terarah.
- batch ini mengutamakan perubahan aman tanpa masuk ke `pg_trgm` atau `tsvector`.

## Sort Whitelist Awal

Setiap endpoint akan dibatasi ke kolom FE yang memang dipakai route schema.

### Master data

- `departemen`: `created_at`, `id_departemen`, `nama_departemen`
- `jenis-barang`: `created_at`, `id_jenis_barang`, `kode`, `nama_jenis_barang`
- `mitra`: `created_at`, `id_mitra`, `nama_perusahaan`, `email`, `no_telp`, `tipe_perusahaan`
- `barang`: `created_at`, `id_barang`, `kode`, `nama_barang`, `nama_jenis_barang`, `nama_perusahaan`
- `permissions`: `created_at`, `id_hak_akses`, `nama_halaman`
- `users`: `created_at`, `id_user`, `username`, `status`, `is_manager`

### Transaksi

- `po-clients`: `created_at`, `id_po_client`, `po_number`, `tanggal`, `season`, `delivery`, `mitra_name`
- `work-orders`: `created_at`, `id_wo`, `buyer`, `model`, `qty`, `status`, `po_number`, `po_client_item_style`
- endpoint transaksi lain mengikuti kolom list response yang sudah terekspos saat ini, dengan default aman bila frontend belum mengirim sort tertentu

## Perubahan Frontend

Module API frontend akan disesuaikan agar:

- mengirim `page` dan `pageSize` secara eksplisit
- mengirim `q` untuk endpoint master-data dan users yang mengikuti pola table FE sekarang
- mengirim `sortBy` dan `sortDesc`
- tetap kompatibel dengan endpoint transaksi yang sudah memakai `page`

Tujuannya agar frontend tidak lagi mengandalkan sort/filter lokal selain state URL table.

## Urutan Implementasi

1. Tambah migration index
2. Ubah query SQL per endpoint
3. Jalankan `make db-gen`
4. Ubah model filter, usecase, dan handler
5. Rapikan module API frontend yang relevan
6. Jalankan `make swag`
7. Jalankan `make lint`
8. Verifikasi endpoint utama

## Verifikasi Minimum

Minimal akan diverifikasi pada:

- `barang`
- `departemen`
- `users`
- `po-clients`
- `work-orders`

Aspek yang dicek:

- filter keyword bekerja dari query URL
- sort valid bekerja
- fallback sort default aman
- total count tetap benar
- pagination tidak regresi

## Risiko dan Mitigasi

### Risiko 1: query menjadi panjang karena dynamic sort

Mitigasi:

- batasi `sortBy` dengan whitelist
- gunakan helper normalization di usecase
- pertahankan default sort sederhana

### Risiko 2: FE dan BE punya nama param berbeda

Mitigasi:

- backend menerima dua alias `q/search` dan `pageSize/limit`
- frontend tetap disesuaikan agar kontrak akhirnya konsisten

### Risiko 3: list transaksi dan master data punya bentuk response berbeda

Mitigasi:

- jangan dipaksa diseragamkan pada batch ini
- fokus pada kemampuan query, bukan perubahan payload besar

### Risiko 4: endpoint FE yang belum punya backend list nyata

Mitigasi:

- keluarkan dari batch ini
- buat task terpisah bila `report-penerimaan` dan `report-pengiriman` memang harus dibangun penuh di backend

## Keputusan Akhir

Pendekatan yang dipilih adalah:

- query spesifik per endpoint,
- search berbasis `ILIKE`,
- sort dinamis via whitelist,
- pagination dinormalisasi di backend,
- index database ditambah sesuai pola list aktif FE.

Pendekatan ini paling cocok dengan arsitektur proyek saat ini, aman untuk `sqlc`, dan memberikan peningkatan performa nyata tanpa refactor besar pada payload response.
