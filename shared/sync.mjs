#!/usr/bin/env node
/**
 * Menyalin shared/format.ts ke web dan mobile.
 *   node shared/sync.mjs          → tulis ulang salinan
 *   node shared/sync.mjs --check  → gagal bila salinan berbeda dari sumber
 * Kedua aplikasi tidak berada dalam satu workspace, jadi helper disalin, bukan diimpor lintas folder.
 */
import { readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const source = readFileSync(join(root, 'shared/format.ts'), 'utf8');
const header = '/* Disalin oleh shared/sync.mjs dari shared/format.ts. Ubah sumbernya, bukan file ini. */\n';
const targets = ['frontend/src/lib/format.ts', 'mobile/src/lib/format.ts'];

const check = process.argv.includes('--check');
let stale = false;
for (const target of targets) {
  const file = join(root, target);
  const content = header + source;
  if (check) {
    let current = '';
    try {
      current = readFileSync(file, 'utf8');
    } catch {
      // belum ada
    }
    if (current !== content) {
      stale = true;
      console.error(`Belum sinkron dengan shared/format.ts: ${target}`);
    }
  } else {
    writeFileSync(file, content);
    console.log(`Ditulis: ${target}`);
  }
}
if (check && stale) {
  console.error('Jalankan: node shared/sync.mjs');
  process.exit(1);
}
