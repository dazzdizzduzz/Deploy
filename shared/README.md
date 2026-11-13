# Helper bersama web & mobile

`format.ts` adalah sumber tunggal helper format lokal (NFR-12). `frontend/` dan `mobile/` bukan satu
workspace, jadi file ini **disalin** ke `frontend/src/lib/format.ts` dan `mobile/src/lib/format.ts`.

```bash
node shared/sync.mjs              # salin ulang setelah mengubah shared/format.ts
node shared/sync.mjs --check      # gagal bila salinan belum sinkron
node --test shared/format.test.ts # uji (Node 22.6+)
```

## Isi

| Fungsi | Contoh |
|---|---|
| `formatRupiah(value, { cents? })` | `"7250000.00"` → `Rp 7.250.000` · `"1500.5"` + `cents` → `Rp 1.500,50` |
| `formatRupiahCompact(value)` | `"118000000.00"` → `Rp 118 jt` |
| `toDecimalString(input)` | `"Rp 7.250.000"` → `"7250000.00"` (untuk dikirim ke API) |
| `formatDate(value, timeZone?)` | `"2026-11-10T16:30:00Z"` → `11/11/2026` (WITA) |
| `formatTime(value, timeZone?)` | `"2026-11-10T00:05:00Z"` → `08:05` (24 jam) |
| `formatDateTime(value, timeZone?)` | `11/11/2026 00:30` |
| `formatLongDate(value, timeZone?)` | `Rabu, 11 November 2026` |
| `toLocalDateKey(value, timeZone?)` | `2027-01-01` untuk mengelompokkan data per hari lokal |
| `timeZoneLabel(timeZone?)` | `WITA` |

## Aturan

- Uang dari API berupa string desimal dan **tidak pernah** diubah ke `number`; perhitungan memakai `BigInt`.
- Waktu dari API (ISO 8601 UTC) ditampilkan di zona waktu tenant. Bawaan `Asia/Makassar` sampai backend
  mengirim zona waktu tenant (lihat draf `tenant.timezone` di `api/drafts/auth-me.openapi.yaml`).
- WIB, WITA, dan WIT memakai selisih jam tetap (Indonesia tanpa DST), zona lain memakai `Intl`.
- Tanggal tanpa jam (`"2025-01-02"`) adalah tanggal kalender dan tidak digeser zona waktu.
- Nilai kosong atau tidak valid ditampilkan sebagai `—` (bisa diganti lewat parameter `fallback`).
