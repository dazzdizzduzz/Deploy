# Design tokens

Satu sumber token untuk web (`frontend/`) dan mobile (`mobile/`), diekspor dari Figma Variables
(halaman **01 · Foundations**): warna mode Light/Dark, spacing, radius, dan tipografi.

| File | Isi |
|---|---|
| `tokens.json` | Nilai token hasil ekspor Figma. Ubah file ini bila Figma berubah. |
| `build.mjs` | Membuat `frontend/src/app/tokens.css` (CSS variables) dan `mobile/src/theme/tokens.ts`. |
| `contrast.mjs` | Memeriksa kontras pasangan warna terhadap WCAG 2.1 AA di kedua mode. |

```bash
node design-tokens/build.mjs           # tulis ulang file token web dan mobile
node design-tokens/build.mjs --check   # gagal bila file hasil belum sinkron dengan tokens.json
node design-tokens/contrast.mjs        # gagal bila ada pasangan warna di bawah WCAG AA
```

Jangan mengubah `tokens.css` atau `tokens.ts` secara manual; perubahan akan tertimpa.

## Font

Semua tipografi memakai **Inter**. Web memuatnya lewat `next/font` di `frontend/src/app/layout.tsx`.
Mobile memuat 4 ketebalan (400, 500, 600, 700) dari `@expo-google-fonts/inter` di `mobile/App.tsx`;
token `text.*` di mobile sudah berisi `fontFamily` yang sesuai, jadi jangan menambahkan `fontWeight`.

## Mode gelap

- **Web:** aktif dengan `<html data-theme="dark">`. Belum otomatis mengikuti sistem karena sebagian
  ikon SVG di `frontend/public/icons/` masih memakai warna garis tetap untuk latar terang.
- **Mobile:** komponen membaca warna lewat `useTheme()` / `useStyles()` dari `ThemeProvider`.
  `app.json` mengunci `userInterfaceStyle: "light"`; ubah ke `"automatic"` untuk mengikuti sistem.
