#!/usr/bin/env node
/**
 * Memeriksa kontras pasangan warna token terhadap WCAG 2.1 AA di mode terang dan gelap.
 *   Teks biasa: minimal 4.5:1 · Komponen UI dan garis fokus: minimal 3:1.
 * Teks berstatus nonaktif (text/disabled) dikecualikan oleh WCAG 1.4.3.
 *   node design-tokens/contrast.mjs   → keluar dengan kode 1 bila ada pasangan yang gagal
 */
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const { color } = JSON.parse(readFileSync(join(dirname(fileURLToPath(import.meta.url)), 'tokens.json'), 'utf8'));

/** [foreground, background, minimum, keterangan] */
const PAIRS = [
  ['text/primary', 'bg/surface', 4.5, 'Teks utama di kartu'],
  ['text/primary', 'bg/canvas', 4.5, 'Teks utama di latar halaman'],
  ['text/primary', 'bg/subtle', 4.5, 'Teks di header tabel'],
  ['text/secondary', 'bg/surface', 4.5, 'Teks sekunder di kartu'],
  ['text/secondary', 'bg/canvas', 4.5, 'Teks sekunder di latar halaman'],
  ['text/secondary', 'bg/subtle', 4.5, 'Label header tabel'],
  ['text/accent', 'bg/surface', 4.5, 'Teks aksen'],
  ['text/accent', 'accent/subtle', 4.5, 'Teks aksen di latar aksen lembut'],
  ['text/inverse', 'bg/inverse', 4.5, 'Tombol primary'],
  ['text/inverse', 'bg/inverse-hover', 4.5, 'Tombol primary saat hover'],
  ['text/on-accent', 'accent/default', 4.5, 'Tombol accent'],
  ['text/on-accent', 'accent/hover', 4.5, 'Tombol accent saat hover'],
  ['text/inverse', 'status/danger-fg', 4.5, 'Tombol danger'],
  ['text/inverse', 'status/danger-hover', 4.5, 'Tombol danger saat hover'],
  ['status/success-fg', 'status/success-bg', 4.5, 'Badge/alert sukses'],
  ['status/warning-fg', 'status/warning-bg', 4.5, 'Badge/alert peringatan'],
  ['status/danger-fg', 'status/danger-bg', 4.5, 'Badge/alert bahaya'],
  ['status/danger-fg', 'bg/surface', 4.5, 'Pesan galat di form'],
  ['text/primary', 'status/warning-bg', 4.5, 'Isi alert peringatan'],
  ['text/primary', 'status/danger-bg', 4.5, 'Isi alert bahaya'],
  ['border/control', 'bg/surface', 3, 'Garis input/select'],
  ['border/focus', 'bg/surface', 3, 'Garis fokus'],
  ['status/danger-fg', 'bg/surface', 3, 'Garis input galat'],
];

function luminance(hex) {
  const [r, g, b] = [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255);
  const linear = (c) => (c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4);
  return 0.2126 * linear(r) + 0.7152 * linear(g) + 0.0722 * linear(b);
}

function ratio(a, b) {
  const [light, dark] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (light + 0.05) / (dark + 0.05);
}

let failed = 0;
for (const mode of ['light', 'dark']) {
  console.log(`\n${mode === 'light' ? 'Mode terang' : 'Mode gelap'}`);
  for (const [fg, bg, min, label] of PAIRS) {
    const value = ratio(color[fg][mode], color[bg][mode]);
    const ok = value >= min;
    if (!ok) failed += 1;
    console.log(`${ok ? 'OK  ' : 'GAGAL'} ${value.toFixed(2).padStart(5)}:1 (min ${min}) ${fg} di ${bg} · ${label}`);
  }
}
console.log(failed ? `\n${failed} pasangan belum memenuhi WCAG AA.` : '\nSemua pasangan memenuhi WCAG AA.');
process.exit(failed ? 1 : 0);
