# Catatan Perubahan Sesi — Refactor Material List

## Konteks

Sesi pengerjaan branch `feat/work-order`. Branch awal sudah punya WO Shell/Trim + provided_by fields dan migrasi rename `fabric` → `deskripsi`. Sesi ini fokus pada:

1. Memperbaiki error `make migrate-down` / `make migrate-up` (cycle tidak idempotent)
2. Refactor `material_list` menjadi container grouping yang membawahi `material_list_item`
3. Menambah CRUD endpoint untuk Material List + Material List Item
4. Memasang lock semantics di PR Internal terhadap Material List

## Commit Log

| Hash | Pesan |
|------|-------|
| `eec3b6f` | fix(migrations): make rename and constraint operations idempotent |
| `25160a0` | refactor(material-list): turn material_list into grouping container |
| `7c59a4c` | feat(material-list): CRUD endpoints + PR Internal locks linked ML |
| `cdf8af8` | fix(seeder): use new material_list grouping schema |

## 1. Migration Idempotency Fixes

Tiga file migration dimodifikasi supaya `migrate-down` lalu `migrate-up` aman dijalankan berulang tanpa harus `force` manual:

- `db/migrations/000018_rename_qty_plan_to_ratio_plan.down.sql` — `RENAME COLUMN` dibungkus DO block + cek `information_schema.columns`
- `db/migrations/000020_explicit_packing_list_sizes.down.sql` — semua `ALTER TABLE` diberi guard `ALTER TABLE IF EXISTS`
- `db/migrations/000025_unique_po_client_item_wo.up.sql` — `ADD CONSTRAINT` dibungkus DO block + cek `pg_constraint` (Postgres tidak support `ADD CONSTRAINT IF NOT EXISTS`)

Cycle test: `make migrate-up` → `make migrate-down` (dengan `y`) → `make migrate-up` berjalan bersih 2x berturut-turut.

## 2. Refactor Material List → Grouping (Migration 027)

### Sebelum

```
MATERIAL_LIST             MATERIAL_LIST_ITEM
├ id_material_list        ├ id_material_list_item
└ id_material_list_item   ├ description
  (FK 1:1)                ├ id_wo (NOT NULL)
                          ├ id_wo_shell (nullable)
                          └ id_wo_trim (nullable)
```

`material_list` hanyalah wrapper 1:1 dari satu item. Tidak benar-benar grouping apa pun.

### Sesudah

```
MATERIAL_LIST                  MATERIAL_LIST_ITEM
├ id_material_list             ├ id_material_list_item
├ id_wo (FK NOT NULL)          ├ id_material_list (FK NOT NULL)  ← arah dibalik
├ name VARCHAR(150)            ├ item VARCHAR(100)
├ is_locked BOOLEAN            ├ description TEXT
└ created_at                   ├ qty INT
                               ├ unit VARCHAR(20)
                               ├ est_price NUMERIC(15,2)
                               ├ id_wo_shell (nullable)
                               ├ id_wo_trim (nullable)
                               └ created_at
```

Sekarang relasi `MATERIAL_LIST` ↔ `MATERIAL_LIST_ITEM` adalah 1:N. ML jadi container yang dimiliki oleh WO; satu WO bisa memiliki banyak ML; tiap ML berisi banyak item.

### Backfill (migration 027 up)

1. `MATERIAL_LIST.id_wo` di-fill dari `mli.id_wo` lewat link `id_material_list_item` lama
2. `MATERIAL_LIST_ITEM.id_material_list` di-fill dari `ml.id_material_list` lewat link yang sama
3. `MATERIAL_LIST_ITEM.item` dan `MATERIAL_LIST_ITEM.unit` di-best-effort populate dari WO_SHELL/WO_TRIM yang terkait
4. Verifikasi NOT NULL sebelum drop kolom lama

### Down migration

Mengembalikan schema 1:1; warning di file: kalau saat ini sudah ada multi-item per ML, item kedua dan seterusnya akan hilang setelah down (referenced item-id-nya tetap ada di MLI, tapi ML hanya simpan satu link).

## 3. Query Layer

### `db/query/work_order.sql`

`CreateMaterialList` lama (yang inline INSERT ke 2 tabel) dipecah jadi:

- `CreateMaterialList :one` — INSERT ke `MATERIAL_LIST` saja (id_wo + name)
- `CreateMaterialListItem :one` — INSERT ke `MATERIAL_LIST_ITEM` (id_material_list + item + qty + unit + est_price + shell/trim)

`CountConfiguredWorkOrdersByPOClientID` direvisi: join `MATERIAL_LIST` langsung via `ml.id_wo` (tidak lagi via `mli.id_material_list_item`).

### `db/query/transaction_reads.sql`

`ListMaterialListsByWorkOrderID` direvisi: sekarang return baris ML dengan `id_wo, name, is_locked, created_at, item_count`. Items dijemput terpisah (lihat usecase).

`ListSuratJalanClients` dan `GetSuratJalanClientDetail`: join ke `ml.id_wo` ditambahkan; field lama `mli.id_wo` (sudah tidak ada) digantikan.

### `db/query/report.sql`

`GetMovementReport`: join `RECEIVED`/`SURAT_JALAN_CLIENT` → `MATERIAL_LIST_ITEM` → `MATERIAL_LIST` → `WORK_ORDER`. Field `ml.description/size/uom` (sudah lama tidak ada) digantikan dengan `mli.description/unit`.

### `db/query/material_list.sql` (file baru)

CRUD lengkap untuk ML dan MLI:

- `GetMaterialList`, `ListUnlockedMaterialListsByWO`, `UpdateMaterialList`, `LockMaterialList`, `DeleteMaterialList`
- `GetMaterialListItem`, `ListMaterialListItemsByML`, `UpdateMaterialListItem`, `DeleteMaterialListItem`
- `CheckMaterialListBelongsToWO` — validasi untuk PR Internal flow

Lock semantics ada di level query: `UpdateMaterialList`, `DeleteMaterialList`, `UpdateMaterialListItem`, `DeleteMaterialListItem` semuanya filter `WHERE is_locked = FALSE`.

## 4. Entity / Model / Usecase

### Generated entities (sqlc)

`make db-gen` me-regenerate `internal/entity/material_list.sql.go` (baru) dan update `internal/entity/work_order.sql.go`, `report.sql.go`, `transaction_reads.sql.go`, `models.go`, `querier.go`.

### Model (request/response) di `internal/model/`

- `internal/model/work_order_production.go`
  - `CreateMaterialListRequest` (lama, description/size/color/uom) **dihapus**
  - Pengganti: `CreateMaterialListItemRequest` dengan field `item, description, qty, unit, est_price, color` (color sebagai matching hint, tidak disimpan)
  - `CreateWorkOrderRequest.MaterialLists` → `MaterialListItems`
  - `MaterialListResponse` direstrukturisasi jadi grouping shape: `{id, id_wo, name, is_locked, created_at, items: []MaterialListItemResponse}`
  - `MaterialListItemResponse` baru: `{id, item, description, qty, unit, est_price, id_wo_shell?, id_wo_trim?, created_at}`

- `internal/model/material_list.go` (file baru) — body request untuk endpoint ML CRUD:
  - `CreateMaterialListRequest { name }`
  - `UpdateMaterialListRequest { name }`
  - `CreateMaterialListItemBody`, `UpdateMaterialListItemBody`
  - `MaterialListListResponse { items: []MaterialListResponse }`

- `internal/model/transaction_document.go`
  - `CreatePRInternalRequest` menerima field opsional `id_material_list` (pointer ke int32)

- `internal/model/report.go`
  - `MovementReportResponse.Size` dihapus (tidak ada lagi di skema)

### Usecase

- `internal/usecase/work_order_production_usecase.go` — `CreateWorkOrder`:
  - Setelah membuat WO + shell + trim, auto-create satu `MATERIAL_LIST` dengan `name="Material List Utama"`
  - Loop `req.MaterialListItems`: matching ke shell/trim (logika lama dipertahankan), lalu `CreateMaterialListItem`
  - `GetWorkOrderDetail` me-fetch list ML lalu untuk tiap ML me-fetch items (N+1 sederhana; bisa dioptimasi nanti)

- `internal/usecase/material_list_usecase.go` (file baru):
  - `CreateMaterialList(idWo, req)` — buat ML tambahan untuk WO
  - `ListByWO(idWo, unlockedOnly)` — list ML dengan items, opsi filter unlocked saja (untuk picker PR Internal)
  - `Get(id)` — ML detail + items
  - `Update(id, req)` — rename, ditolak kalau locked
  - `Delete(id)` — hapus, ditolak kalau locked
  - `CreateItem`, `UpdateItem`, `DeleteItem` — CRUD MLI, semua respek lock

- `internal/usecase/transaction_document_usecase.go` — `CreatePRInternal`:
  - Validasi `req.IDMaterialList` (opsional): cek ML belong ke WO yang sama (`CheckMaterialListBelongsToWO`), cek tidak locked
  - Setelah PR row insert, panggil `LockMaterialList(*req.IDMaterialList)` di tx yang sama
  - Error baru: `ErrMaterialListAlreadyLocked`

- `internal/usecase/report_usecase.go` — drop assignment `row.Size`

### Handler

- `internal/delivery/http/material_list_handler.go` (file baru) — registrasi 8 route ML/MLI
- `internal/delivery/http/transaction_document_handler.go` — `handleError` mapping `ErrMaterialListAlreadyLocked` → 409 `material_list_already_locked`
- `cmd/web/main.go` — wiring `materialListUseCase` dan `materialListHandler`

### Endpoint baru

| Method | Path | Permission |
|--------|------|------------|
| GET | `/api/v1/work-orders/:id/material-lists` (query `unlocked=true`) | `WO_READ` |
| POST | `/api/v1/work-orders/:id/material-lists` | `WO_UPDATE` + internal |
| GET | `/api/v1/material-lists/:id` | `WO_READ` |
| PATCH | `/api/v1/material-lists/:id` | `WO_UPDATE` + internal |
| DELETE | `/api/v1/material-lists/:id` | `WO_UPDATE` + internal |
| POST | `/api/v1/material-lists/:id/items` | `WO_UPDATE` + internal |
| PATCH | `/api/v1/material-list-items/:id` | `WO_UPDATE` + internal |
| DELETE | `/api/v1/material-list-items/:id` | `WO_UPDATE` + internal |

## 5. Seeder

`cmd/seeder/main.go` di-update agar mengisi tabel baru sesuai schema baru:

- Tiap WO sekarang membuat 1 `MATERIAL_LIST` (`name="Material List Utama"`) lalu insert items dengan `id_material_list` dan field item/qty/unit/est_price/shell/trim
- INSERT lama yang nulis `MATERIAL_LIST_ITEM.id_wo` dan `MATERIAL_LIST.id_material_list_item` dihapus

---

# Flow Lengkap: PO Client Item → Work Order → Report → PR/PO Internal

## Aktor

- **Admin Internal** (role `SUPER_ADMIN`, `MANAGER`, `ADMIN_PRODUKSI`, `ADMIN_GUDANG`, `ADMIN_KEUANGAN`) — yang membuat dokumen
- **Mitra Client** (role `CLIENT`) — buyer; melihat WO miliknya, melakukan retur, menandai selesai
- **Approval chain** — RBAC v2 (lihat migration 005 dan `internal/usecase/approval_*`)

## Step 1 — Mitra Register

`POST /api/v1/register-mitra`

Mitra register sebagai user dengan role `CLIENT`. Sistem membuat baris di tabel `MITRA` yang ter-link ke `USERS`.

## Step 2 — Admin Buat PO Client

`POST /api/v1/po-clients`

Body sebagian: `{ po_number, tanggal, season, delivery, id_mitra, penanggung_jawab: [...], items: [...] }`

Hasil:

- 1 baris `PO_CLIENT` (master)
- N baris `PO_CLIENT_ITEM` (style, qty, harga, dll)
- N baris `PENANGGUNG_JAWAB` (siapa PIC untuk PO ini)
- Workflow approval di-inisialisasi (table `OTORITAS_DOKUMEN`)

## Step 3 — Admin Buat Work Order

`POST /api/v1/work-orders`

Body utama:

```json
{
  "buyer": "...",
  "model": "...",
  "qty": 1000,
  "fob_cmt": true,
  "delivery": "2026-09-30",
  "id_po_client_item": 12,
  "shells": [
    {
      "deskripsi": "Cotton Combed 30s",
      "cons": 1.2, "color": "Navy", "allow": 3,
      "berat_1_yd": 0.22, "provided_by": "permata", "material_type": "fabric",
      "sizes": [{"size":"M","qty":300,"ratio":3}, ...]
    }
  ],
  "trims": [
    { "item": "Kancing", "description": "...", "color": "Navy", "code": "BTN-01",
      "cons": 0.1, "qty": 6, "uom": "pcs", "position": "Front", "created_by": "admin",
      "allow": 0, "provided_by": "permata" }
  ],
  "material_list_items": [
    { "item": "Kain Navy", "description": "Cotton 30s",
      "qty": 1200, "unit": "yds", "est_price": 18000, "color": "Navy" }
  ]
}
```

Di backend (`work_order_production_usecase.CreateWorkOrder`) dalam satu transaksi:

1. Insert `WORK_ORDER` (header)
2. Loop `shells`: insert `WORK_ORDER_SHELL` + nested `WORK_ORDER_SHELL_SIZE` per ukuran
3. Loop `trims`: insert `WORK_ORDER_TRIM`
4. **Auto-create `MATERIAL_LIST` utama** (`name="Material List Utama"`, `is_locked=false`)
5. Loop `material_list_items`:
   - Auto-match ke shell/trim via `color` + kemiripan deskripsi (logika string contains)
   - Fallback: kalau hanya 1 shell di WO, paksa link by color saja
   - Insert `MATERIAL_LIST_ITEM` dengan `id_material_list` ke ML utama, plus `id_wo_shell` atau `id_wo_trim` hasil match
6. Initialize approval workflow untuk WORK_ORDER
7. Commit

Hasil response: WO header + shells (dengan sizes) + trims + 1 material list (dengan items).

## Step 4 — Tambah Material List Tambahan (opsional, di tengah produksi)

Kalau di tengah produksi butuh material tambahan:

```
POST /api/v1/work-orders/:id/material-lists
{ "name": "Tambahan Buttons Bulan 7" }
```

Lalu tambah item:

```
POST /api/v1/material-lists/:id/items
{ "item": "Kancing emergency", "qty": 50, "unit": "pcs", "est_price": 500, "id_wo_trim": 12 }
```

ML tambahan ini independent dari ML utama; bisa di-edit, di-delete, atau di-lock terpisah.

## Step 5 — Report per Tahap

WO punya 5 divisi produksi: `cutting`, `sewing`, `qc_finish`, `packing`, `pengiriman`.

`POST /api/v1/reports/:divisi`

Body: `{ id_wo_shell_size, tanggal, qty }`

Backend insert ke tabel report yang sesuai (`REPORT_CUTTING`, `REPORT_SEWING`, `REPORT_QC_FINISH`, `REPORT_PACKING`, `REPORT_PENGIRIMAN`). Status produksi WO di-hitung dari progress 5 divisi (lihat `deriveProductionStatus` di usecase).

## Step 6 — Mitra Tandai Selesai / Ajukan Retur

Setelah WO mencapai 100% (semua size shell selesai pengiriman):

- Mitra POV: `PATCH /api/v1/work-orders/:id/client-close` → set flag client-closed
- Atau: `POST /api/v1/work-orders/:id/retur` (multipart upload file retur)

## Step 7 — Admin Close WO

Setelah mitra menandai selesai:

`PATCH /api/v1/work-orders/:id/close` — admin menutup WO secara final.

## Step 8 — PR Internal (Procurement Request)

Untuk material yang perlu dibeli (cek kolom `provided_by` di shell/trim — kalau `client` berarti disuplai mitra, tidak perlu PR):

`POST /api/v1/pr-internals`

Flow yang direkomendasikan untuk frontend:

1. Pilih WO (PR harus ter-link ke WO)
2. `GET /api/v1/work-orders/:id/material-lists?unlocked=true` → tampilkan dropdown ML yang belum di-lock
3. Pilih ML → `GET /api/v1/material-lists/:id` → ambil items
4. Pre-fill PR Items dari MLI: `{item, description, qty, unit, est_price}` (admin masih bisa edit)
5. Submit PR dengan `id_material_list` di body

Body:

```json
{
  "tanggal": "2026-06-10",
  "nama": "PR Kain Navy WO 12",
  "departemen": "produksi",
  "vendor_name": "Supplier Tekstil ABC",
  "vendor_address": "...",
  "vendor_telp": "...",
  "projek": "WO 12 - Kemeja Navy",
  "id_wo": 12,
  "id_material_list": 7,
  "items": [
    {"item": "Kain Navy", "description": "Cotton 30s", "qty": 1200, "unit": "yds", "est_price": 18000}
  ]
}
```

Backend (`transaction_document_usecase.CreatePRInternal`) dalam satu tx:

1. Kalau `id_material_list` ada → cek ML ini milik WO yang sama (`CheckMaterialListBelongsToWO`) + cek belum locked
2. Insert `PR_INTERNAL` + N `PR_INTERNAL_ITEM`
3. Kalau `id_material_list` ada → `LockMaterialList(id)` → flip `is_locked=true`
4. Initialize approval workflow

Setelah lock: ML tersebut tidak akan muncul lagi di `?unlocked=true` query, dan semua CRUD MLI-nya akan return 409 `material_list_already_locked`.

## Step 9 — PR Internal Approval → PO Internal

`PATCH /api/v1/pr-internals/:id/approve` — admin keuangan/manager approve PR.

Setelah PR `status='approved'`:

`POST /api/v1/po-internals`

Body: header PO (supplier info, currency, ship_date, dll) + `id_pr_internal` + `items` (snapshot dari PR, plus `unit_price` final).

Backend validate PR `status='approved'` sebelum buat PO. Hasil:

- 1 baris `PO_INTERNAL` + N `PO_INTERNAL_ITEM`

## Step 10 — Received & Surat Jalan

Setelah barang datang dari supplier:

- `POST /api/v1/received` — track item received (link ke `MATERIAL_LIST_ITEM` via `id_material_list_item`)
- Update `REKONSILIASI_MATERIAL.balance`

Saat material dikirim ke mitra:

- `POST /api/v1/surat-jalan/client` — link ke `MATERIAL_LIST_ITEM`

## Ringkasan Relasi

```
MITRA ←─── PO_CLIENT ←──── PO_CLIENT_ITEM ←──── WORK_ORDER ←──┬─── WO_SHELL ←─── WO_SHELL_SIZE
                                                              │
                                                              ├─── WO_TRIM
                                                              │
                                                              ├─── MATERIAL_LIST (1:N)
                                                              │       ↓
                                                              │    MATERIAL_LIST_ITEM
                                                              │       ↑
                                                              │       └── id_wo_shell / id_wo_trim (FK ke spec)
                                                              │       ↑
                                                              │       └── RECEIVED / SURAT_JALAN_CLIENT
                                                              │
                                                              ├─── PR_INTERNAL ←──── PR_INTERNAL_ITEM
                                                              │       ↓ approved
                                                              │     PO_INTERNAL ←──── PO_INTERNAL_ITEM
                                                              │
                                                              ├─── REPORT_CUTTING / SEWING / QC_FINISH / PACKING / PENGIRIMAN
                                                              │
                                                              └─── RETUR_CLIENT
```

## Frontend TODO

1. WO create payload: kunci JSON `material_lists` → `material_list_items`
2. Field per item baru: `{item, description, qty, unit, est_price, color}` (`size` dihapus)
3. Response ML shape: `{id, id_wo, name, is_locked, created_at, items: [...]}`
4. PR Internal create page: 2-step picker (WO → ML unlocked) → prefill items dari MLI
5. Material List management UI:
   - List ML per WO
   - Tambah ML baru (additional)
   - Edit name ML (hanya kalau belum locked)
   - Tambah/edit/delete MLI (hanya kalau ML belum locked)
   - Indicator "Locked" di ML yang sudah punya PR
