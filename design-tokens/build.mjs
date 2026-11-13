#!/usr/bin/env node
/**
 * Membuat token untuk web dan mobile dari design-tokens/tokens.json (ekspor Figma Variables).
 *   node design-tokens/build.mjs          → tulis ulang kedua file
 *   node design-tokens/build.mjs --check  → gagal bila file hasil tidak sama dengan tokens.json
 * Tanpa dependensi; cukup Node.js 18+.
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const tokens = JSON.parse(readFileSync(join(root, 'design-tokens/tokens.json'), 'utf8'));
const MODES = ['light', 'dark'];

const kebab = (name) => name.replace(/\//g, '-');
const camel = (name) => name.replace(/[-/](\w)/g, (_, c) => c.toUpperCase());
const px = (value) => (value === 0 ? '0' : `${value}px`);
const round = (value) => Math.round(value * 100) / 100;

const colorEntries = (mode) => [
  ...Object.entries(tokens.color).map(([name, value]) => [name, value[mode]]),
  ...Object.entries(tokens.overlay).map(([name, value]) => [name, value[mode]]),
];

// ---------- Web: CSS variables ----------
function cssColors(mode, indent) {
  return colorEntries(mode).map(([name, value]) => `${indent}--color-${kebab(name)}: ${value};`).join('\n');
}

function buildCss() {
  const lines = [
    '/* Dihasilkan oleh design-tokens/build.mjs dari design-tokens/tokens.json (Figma Variables). Jangan ubah manual. */',
    ':root {',
    '  color-scheme: light;',
    cssColors('light', '  '),
    ...Object.entries(tokens.space).map(([key, value]) => `  --space-${key}: ${px(value)};`),
    ...Object.entries(tokens.radius).map(([key, value]) => `  --radius-${key}: ${px(value)};`),
    ...Object.entries(tokens.size).map(([key, value]) => `  --size-${key}: ${px(value)};`),
    ...Object.entries(tokens.shadow).map(([key, value]) => `  --shadow-${key}: ${value};`),
    ...Object.entries(tokens.typography).flatMap(([key, t]) => [
      `  --type-${key}-font: ${t.weight} ${t.size}px/${t.lineHeight} var(--font-sans);`,
      `  --type-${key}-tracking: ${t.tracking}em;`,
    ]),
    '}',
    '',
    '/*',
    ' * Mode gelap aktif lewat <html data-theme="dark">. Belum otomatis mengikuti sistem karena',
    ' * sebagian ikon SVG masih memakai warna garis tetap untuk latar terang.',
    ' */',
    ':root[data-theme="dark"] {',
    '  color-scheme: dark;',
    cssColors('dark', '  '),
    '}',
    '',
  ];
  return lines.join('\n');
}

// ---------- Mobile: modul TypeScript ----------
function palette(mode) {
  const groups = {};
  for (const [name, value] of colorEntries(mode)) {
    const [group, ...rest] = name.split('/');
    (groups[group] ??= {})[camel(rest.join('-'))] = value;
  }
  return groups;
}

/** Nama font Inter dari @expo-google-fonts/inter per ketebalan. */
const INTER = { 400: 'Inter_400Regular', 500: 'Inter_500Medium', 600: 'Inter_600SemiBold', 700: 'Inter_700Bold' };

function buildTs() {
  // Di React Native, ketebalan font custom dipilih lewat fontFamily, bukan fontWeight.
  const text = Object.fromEntries(
    Object.entries(tokens.typography).map(([key, t]) => {
      if (!INTER[t.weight]) throw new Error(`Ketebalan ${t.weight} belum dipetakan ke font Inter`);
      return [
        camel(key),
        {
          fontFamily: INTER[t.weight],
          fontSize: t.size,
          lineHeight: round(t.size * t.lineHeight),
          letterSpacing: round(t.size * t.tracking),
        },
      ];
    }),
  );
  const json = (value) => JSON.stringify(value, null, 2);
  return [
    '/* Dihasilkan oleh design-tokens/build.mjs dari design-tokens/tokens.json (Figma Variables). Jangan ubah manual. */',
    '',
    `export const palettes = ${json({ light: palette('light'), dark: palette('dark') })} as const;`,
    '',
    'export type ColorScheme = keyof typeof palettes;',
    'export type Palette = (typeof palettes)[ColorScheme];',
    '',
    `export const space = ${json(Object.fromEntries(Object.entries(tokens.space).map(([k, v]) => [k, v])))} as const;`,
    '',
    `export const radius = ${json(tokens.radius)} as const;`,
    '',
    `export const size = ${json(Object.fromEntries(Object.entries(tokens.size).map(([k, v]) => [camel(k), v])))} as const;`,
    '',
    `/** Font Inter yang dimuat saat aplikasi dibuka (lihat App.tsx). */`,
    `export const fontFamily = ${json({ regular: INTER[400], medium: INTER[500], semiBold: INTER[600], bold: INTER[700] })} as const;`,
    '',
    '/** Tipografi Figma (Inter). */',
    `export const text = ${json(text)} as const;`,
    '',
  ].join('\n');
}

const outputs = {
  'frontend/src/app/tokens.css': buildCss(),
  'mobile/src/theme/tokens.ts': buildTs(),
};

const check = process.argv.includes('--check');
let stale = false;
for (const [path, content] of Object.entries(outputs)) {
  const file = join(root, path);
  if (check) {
    let current = '';
    try {
      current = readFileSync(file, 'utf8');
    } catch {
      // file belum ada
    }
    if (current !== content) {
      stale = true;
      console.error(`Belum sinkron dengan tokens.json: ${path}`);
    }
  } else {
    writeFileSync(file, content);
    console.log(`Ditulis: ${path}`);
  }
}
if (check && stale) {
  console.error('Jalankan: node design-tokens/build.mjs');
  process.exit(1);
}
