# Migrasi Upload Production ke Storage Persisten

Runbook ini digunakan satu kali untuk memindahkan file upload dari filesystem container backend lama ke bind mount host. Target production yang direkomendasikan adalah `/srv/permatatex/uploads`, dipasang ke `/app/uploads`.

Jangan menjalankan prosedur ini tanpa maintenance window, persetujuan penanggung jawab teknis, dan backup production terverifikasi. Jangan menggunakan folder repository backend sebagai storage production.

## Prasyarat Backup Wajib

Sebelum maintenance dimulai, penanggung jawab teknis harus memastikan:

- [ ] Backup mencakup database PostgreSQL format custom, globals/roles, dan seluruh upload dari container lama.
- [ ] Dump dapat dibaca dengan `pg_restore --list`, archive upload dapat dibuka, serta jumlah file, ukuran, dan seluruh checksum sudah diverifikasi.
- [ ] Paket backup dienkripsi sebelum keluar dari VPS; jangan membuat fallback plaintext.
- [ ] Archive terenkripsi dan checksum sudah disalin ke media offline terenkripsi, dibaca kembali, dan checksum salinannya cocok.
- [ ] Backup ID, waktu UTC, commit aplikasi, versi migration, operator, dan media penyimpanan sudah dicatat.
- [ ] Backup lama dan container lama tidak dihapus sebelum backup baru serta migrasi upload selesai diverifikasi.
- [ ] Restore percobaan tidak dilakukan langsung ke production.

## Checklist Admin Rutin

Admin nonteknis tidak menjalankan command Docker atau mengubah konfigurasi server.

- [ ] Jadwal maintenance dan perkiraan downtime sudah diumumkan.
- [ ] Penanggung jawab teknis yang menjalankan migrasi sudah ditentukan.
- [ ] Backup database, globals, dan uploads sudah terenkripsi dan checksum salinan flashdisk sudah cocok.
- [ ] Daftar beberapa URL/file upload lama untuk smoke test sudah disiapkan.
- [ ] Selama maintenance, admin tidak menambah atau mengubah data/upload.
- [ ] Setelah teknisi menyatakan aplikasi siap diuji, admin membuka file lama yang sudah dipilih dan mencatat hasilnya.
- [ ] Maintenance hanya ditutup setelah penanggung jawab teknis menyetujui hasil verifikasi.

## Checklist Teknisi

Semua command berikut dijalankan di VPS oleh teknisi. Sesuaikan nama direktori repository bila instalasi VPS berbeda. Jangan mencetak credential atau isi `.env`.

### 1. Pre-check tanpa mengubah container

- [ ] Pastikan container lama masih ada dan jangan recreate, remove, atau menjalankan `docker compose down`.
- [ ] Catat container aplikasi lama dan image yang sedang berjalan.
- [ ] Pastikan `/app/uploads` tersedia di container lama.
- [ ] Pastikan ruang disk host cukup untuk target, salinan rollback, dan backup terenkripsi.

```bash
cd backend
git status
OLD_APP=permatatex-backend
docker inspect "$OLD_APP" --format '{{.Id}} {{.Image}} {{.State.Status}}'
docker exec "$OLD_APP" test -d /app/uploads
```

### 2. Mulai maintenance window

- [ ] Aktifkan maintenance pada reverse proxy atau mekanisme operasional yang menghentikan semua penulisan baru.
- [ ] Konfirmasi tidak ada request upload/transaksi yang masih berjalan.
- [ ] Biarkan container lama tetap tersedia sampai copy dan verifikasi selesai.

### 3. Buat manifest sumber dan salin file

Gunakan ID waktu UTC agar bukti migrasi tidak tertukar. Direktori bukti dan rollback berada di luar repository.

```bash
MIGRATION_ID="$(date -u +%Y%m%dT%H%M%SZ)"
MIGRATION_DIR="/srv/permatatex/upload-migration-$MIGRATION_ID"
TARGET_DIR=/srv/permatatex/uploads

sudo install -d -m 0750 "$MIGRATION_DIR/source-copy" "$TARGET_DIR"

if sudo find "$TARGET_DIR" -mindepth 1 -print -quit | grep -q .; then
  echo "ERROR: target uploads tidak kosong; hentikan dan review isinya"
  exit 1
fi

docker exec "$OLD_APP" sh -c \
  'cd /app/uploads && find . -type f -exec sha256sum {} \; | sort' \
  | sudo tee "$MIGRATION_DIR/source.sha256" >/dev/null
docker exec "$OLD_APP" sh -c \
  'cd /app/uploads && find . -type f | wc -l' \
  | sudo tee "$MIGRATION_DIR/source.count" >/dev/null
docker exec "$OLD_APP" sh -c \
  'find /app/uploads -type f -exec stat -c %s {} \; | awk "{total += \$1} END {print total + 0}"' \
  | sudo tee "$MIGRATION_DIR/source.bytes" >/dev/null

docker cp "$OLD_APP:/app/uploads/." "$MIGRATION_DIR/source-copy/"
sudo cp -a "$MIGRATION_DIR/source-copy/." "$TARGET_DIR/"
```

Jangan menimpa atau menghapus file target yang belum diketahui asalnya.

### 4. Verifikasi copy sebelum recreate

```bash
sudo sh -c "cd '$TARGET_DIR' && find . -type f -exec sha256sum {} \; | sort > '$MIGRATION_DIR/target.sha256'"
sudo sh -c "cd '$TARGET_DIR' && find . -type f | wc -l > '$MIGRATION_DIR/target.count'"
sudo sh -c "find '$TARGET_DIR' -type f -exec stat -c %s {} \; | awk '{total += \$1} END {print total + 0}' > '$MIGRATION_DIR/target.bytes'"

sudo diff -u "$MIGRATION_DIR/source.sha256" "$MIGRATION_DIR/target.sha256"
sudo diff -u "$MIGRATION_DIR/source.count" "$MIGRATION_DIR/target.count"
sudo diff -u "$MIGRATION_DIR/source.bytes" "$MIGRATION_DIR/target.bytes"
```

- [ ] Ketiga perbandingan selesai dengan exit code `0`.
- [ ] Salinan `source-copy` dipertahankan sebagai rollback dan tidak dipindahkan ke repository.
- [ ] Jika ada perbedaan, jangan aktifkan bind mount dan jangan recreate container.

### 5. Set ownership, permission, dan konfigurasi

Tentukan UID/GID yang dipakai proses aplikasi. Jangan menebak; ambil dari image/container yang akan dijalankan atau kebijakan server.

```bash
APP_UID=<uid-aplikasi>
APP_GID=<gid-aplikasi>
sudo chown -R "$APP_UID:$APP_GID" "$TARGET_DIR"
sudo find "$TARGET_DIR" -type d -exec chmod 0750 {} \;
sudo find "$TARGET_DIR" -type f -exec chmod 0640 {} \;
```

- [ ] Simpan `UPLOADS_HOST_PATH=/srv/permatatex/uploads` pada konfigurasi environment production yang terlindungi.
- [ ] Untuk GitHub Actions, set environment variable `UPLOADS_HOST_PATH` pada environment `production`.
- [ ] Jangan memakai `./uploads`, path relatif, atau direktori di dalam checkout repository untuk production.
- [ ] Verifikasi konfigurasi tanpa menampilkan environment:

```bash
export UPLOADS_HOST_PATH=/srv/permatatex/uploads
test "${UPLOADS_HOST_PATH:-}" = /srv/permatatex/uploads
test -d "$UPLOADS_HOST_PATH"
make prod-validate-config
```

`make prod-deploy` sengaja gagal sebelum `PROD_DB_URL` dan `UPLOADS_HOST_PATH` tersedia, target host ada, dan target berada di luar repository.

### 6. Recreate hanya service app

Langkah ini baru boleh dijalankan setelah seluruh checksum cocok dan backup offline terverifikasi.

```bash
docker compose build app
UPLOADS_HOST_PATH=/srv/permatatex/uploads \
  docker compose up -d --no-deps --wait --wait-timeout 60 app
docker compose ps app
```

- [ ] Jangan menjalankan `docker compose down`, `down -v`, migration down, atau seed.
- [ ] Pastikan mount aktif: `docker inspect permatatex-backend` menunjukkan source `/srv/permatatex/uploads` ke destination `/app/uploads`.
- [ ] Pastikan health check lulus.

### 7. Uji file lama dan tutup maintenance

- [ ] Teknisi memeriksa log backend dan memastikan tidak ada error permission pada `/app/uploads`.
- [ ] Admin menguji beberapa URL/file lama yang sudah dicatat sebelum maintenance.
- [ ] Uji satu upload baru dan pastikan file muncul pada `/srv/permatatex/uploads`.
- [ ] Tutup maintenance hanya setelah seluruh pemeriksaan lulus.

### 8. Rollback dan retensi

Jika container baru gagal, hentikan perubahan lanjutan. Gunakan image/config aplikasi sebelumnya sesuai prosedur rollback, tetap mount data yang sudah diverifikasi, dan jangan menjalankan migration down otomatis.

- [ ] Pertahankan `$MIGRATION_DIR/source-copy`, manifest, image ID lama, dan backup terenkripsi sampai masa observasi selesai dan penghapusan disetujui.
- [ ] Jangan menghapus salinan rollback, manifest, atau image lama sebelum checksum, smoke test, dan persetujuan rollback selesai.
- [ ] Catat waktu, operator, jumlah file, total byte, checksum, image lama/baru, dan hasil smoke test.
