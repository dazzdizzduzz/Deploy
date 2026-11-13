# Deployment Frontend Web (Next.js)

Frontend dikemas sebagai image Docker mandiri (`output: "standalone"`), berjalan sebagai user non-root di port 3000, dan punya healthcheck ke `/login`. Package manager: **npm** (`package-lock.json`).

## Environment

Semua variabel dibaca saat runtime oleh server Next.js, jadi image yang sama bisa dipakai untuk staging dan production.

| Variabel | Wajib | Contoh | Keterangan |
|---|---|---|---|
| `API_URL` | Ya | `http://backend:8080` | Alamat backend Go dari sisi server Next.js. Tidak pernah dikirim ke browser. |
| `APP_ORIGIN` | Tidak | `https://ontrack.example.com` | Alamat publik web. Diperlukan hanya jika reverse proxy tidak meneruskan header `Host` atau `X-Forwarded-Host`. |

## Build dan jalankan image

```bash
docker build -t ontrack-frontend ./frontend
docker run -d --name ontrack_frontend -p 3000:3000 \
  -e API_URL=https://api.example.com \
  ontrack-frontend
```

Build membutuhkan akses internet karena `next build` mengunduh font Inter dari Google Fonts.

Dengan Docker Compose dari root repo, cukup jalankan `docker compose up -d --build frontend`. `API_URL` sudah diatur ke `http://backend:8080`.

## Reverse proxy HTTPS

Jalankan aplikasi di belakang reverse proxy yang menangani HTTPS dan meneruskan header `Host` dan `X-Forwarded-Proto`. Cookie sesi otomatis bertanda `Secure` jika permintaan datang lewat HTTPS.

Contoh Caddy (sertifikat HTTPS otomatis):

```
ontrack.example.com {
  reverse_proxy localhost:3000
}
```

Contoh Nginx:

```nginx
location / {
  proxy_pass http://127.0.0.1:3000;
  proxy_set_header Host $host;
  proxy_set_header X-Forwarded-Host $host;
  proxy_set_header X-Forwarded-Proto $scheme;
  proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

## CI/CD

- Setiap PR dan push ke `staging` atau `main` menjalankan lint, `next build`, lalu build image Docker frontend tanpa push.
- Push tag `v*` (misalnya `v1.0.0`, buat dari commit di `main`) mem-build lalu push image `<DOCKER_USERNAME>/ontrack-frontend:<versi>` dan `:latest`. Secret `DOCKER_USERNAME` dan `DOCKER_PASSWORD` harus diisi di Settings → Secrets.

## Cek setelah deploy

1. `curl -I https://ontrack.example.com/login` mengembalikan `200`.
2. Login dengan akun HRD, lalu pastikan diarahkan ke `/dashboard`.
3. Simpan satu perubahan, misalnya mengubah data karyawan. Jika gagal dengan "Asal permintaan tidak dikenal", atur `APP_ORIGIN` atau perbaiki header proxy.
4. Login dengan akun `cashier` atau `super_admin` harus menampilkan pesan "Akun tidak memakai portal web".

## Menu yang disembunyikan

Rekap Kehadiran, Persetujuan Izin, dan Struktur Organisasi disembunyikan dari sidebar karena endpoint backend-nya belum ada. Halamannya tetap bisa dibuka lewat URL. Untuk menampilkannya lagi, hapus path dari `PENDING_BACKEND` di `src/app/dashboard/layout.tsx`.
