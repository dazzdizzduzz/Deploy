#!/usr/bin/env bash
# scripts/backup-db.sh
# Skrip otomatisasi backup PostgreSQL Aiven menggunakan format custom pg_dump (-Fc)
#
# Spesifikasi:
# - Format dump: pg_dump -Fc
# - Sumber koneksi: Dibaca dari environment variable (AIVEN_HOST, AIVEN_PORT, PGPASSWORD)
# - Verifikasi: Memeriksa integritas berkas dump dengan pg_restore --list
# - Retensi: Menghapus berkas dump yang berusia lebih dari 14 hari
# - Keamanan: Password dibangun dalam connection URL di memori, tidak pernah dicetak atau disimpan ke file

set -euo pipefail

# Tambahkan path PostgreSQL di Windows jika belum ada di PATH
if ! command -v pg_dump &>/dev/null; then
    if [[ -d "/c/Program Files/PostgreSQL/18/bin" ]]; then
        export PATH="/c/Program Files/PostgreSQL/18/bin:$PATH"
    elif [[ -d "C:/Program Files/PostgreSQL/18/bin" ]]; then
        export PATH="C:/Program Files/PostgreSQL/18/bin:$PATH"
    fi
fi

# Validasi ketersediaan utilitas pg_dump dan pg_restore
if ! command -v pg_dump &>/dev/null; then
    echo "ERROR: 'pg_dump' tidak ditemukan di PATH sistem." >&2
    exit 1
fi

if ! command -v pg_restore &>/dev/null; then
    echo "ERROR: 'pg_restore' tidak ditemukan di PATH sistem." >&2
    exit 1
fi

# Konfigurasi koneksi dari environment variables
DB_HOST="${AIVEN_HOST:-${PGHOST:-}}"
DB_PORT="${AIVEN_PORT:-${PGPORT:-}}"
DB_USER="${AIVEN_USER:-${PGUSER:-avnadmin}}"
DB_NAME="${AIVEN_DB:-${PGDATABASE:-defaultdb}}"
DB_PASS="${PGPASSWORD:-}"

if [[ -z "$DB_HOST" || -z "$DB_PORT" || -z "$DB_PASS" ]]; then
    echo "ERROR: Environment variable AIVEN_HOST, AIVEN_PORT, dan PGPASSWORD wajib diisi." >&2
    exit 1
fi

# Direktori dan nama berkas backup
BACKUP_DIR="${BACKUP_DIR:-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
TIMESTAMP="$(date +'%Y%m%d_%H%M%S')"
BACKUP_FILE="${BACKUP_DIR}/${DB_NAME}_${TIMESTAMP}.dump"

mkdir -p "$BACKUP_DIR"

echo "=== [1/4] Memulai Proses Backup Database ==="
echo "Host     : ${DB_HOST}:${DB_PORT}"
echo "User     : ${DB_USER}"
echo "Database : ${DB_NAME}"
echo "Target   : ${BACKUP_FILE}"

# Bangun connection URL hanya di memori
DB_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=require"

echo "=== [2/4] Menjalankan pg_dump (-Fc) ==="
pg_dump -Fc -d "$DB_URL" -f "$BACKUP_FILE"

echo "=== [3/4] Memverifikasi Integritas Dump dengan pg_restore --list ==="
if pg_restore --list "$BACKUP_FILE" > /dev/null 2>&1; then
    FILE_SIZE="$(du -h "$BACKUP_FILE" | cut -f1)"
    echo "Verifikasi BERHASIL: Arsip dump valid (Ukuran berkas: ${FILE_SIZE})."
else
    echo "ERROR: Verifikasi GAGAL! Berkas dump rusak atau tidak valid." >&2
    rm -f "$BACKUP_FILE"
    exit 1
fi

echo "=== [4/4] Membersihkan Berkas Dump Berusia > ${RETENTION_DAYS} Hari ==="
DELETED_COUNT=0
while IFS= read -r -d '' file; do
    echo "Menghapus dump lama: $file"
    rm -f "$file"
    DELETED_COUNT=$((DELETED_COUNT + 1))
done < <(find "$BACKUP_DIR" -type f -name "*.dump" -mtime +"$RETENTION_DAYS" -print0)

echo "Pembersihan selesai (${DELETED_COUNT} berkas lama dihapus)."
echo "=== Backup Database Selesai dengan Sukses ==="

