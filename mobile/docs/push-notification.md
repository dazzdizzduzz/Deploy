# Push notification (FR-15)

Notifikasi status pengajuan izin dikirim backend lewat FCM (Android) dan APNs (iOS). Aplikasi
mendaftarkan token perangkat ke `POST /me/devices` setelah login dan menghapusnya lewat
`POST /me/devices/unregister` saat logout. Kontrak: `api/drafts/notifications.openapi.yaml`.

## Expo Go

Sejak Expo SDK 53, Expo Go di Android tidak menerima push dari server. Di Expo Go aplikasi melewati
pendaftaran token tanpa error, dan daftar notifikasi in-app (menu Notifikasi) tetap berjalan.

## Development build (untuk mencoba push sungguhan)

1. Minta file `google-services.json` dari pemegang project Firebase tim. Jangan di-commit
   (sudah masuk `.gitignore`).
2. Taruh di `mobile/google-services.json`, lalu tambahkan ke `app.json`:
   ```json
   "android": { "googleServicesFile": "./google-services.json", "package": "<package id tim>" }
   ```
3. Buat build: `npx eas build --profile development --platform android`, pasang APK-nya di HP,
   lalu jalankan `npx expo start --dev-client`.
4. Backend mengirim push dengan Firebase Admin SDK memakai token dari `/me/devices`. Isi `data`
   wajib `notification_id`, `type`, dan `leave_request_id` agar tap membuka Detail Pengajuan.

iOS butuh akun Apple Developer dan kunci APNs yang diunggah ke EAS (`eas credentials`).
