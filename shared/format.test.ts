/**
 * Uji helper format. Jalankan: node --test shared/format.test.ts (Node 22.6+ membaca TypeScript langsung).
 */
import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  formatDate,
  formatDateTime,
  formatLongDate,
  formatRupiah,
  formatRupiahCompact,
  formatTime,
  parseDecimal,
  timeZoneLabel,
  toDecimalString,
  toLocalDateKey,
} from './format.ts';

test('formatRupiah: string desimal dari API', () => {
  assert.equal(formatRupiah('7250000.00'), 'Rp 7.250.000');
  assert.equal(formatRupiah('18000.00'), 'Rp 18.000');
  assert.equal(formatRupiah('0.00'), 'Rp 0');
  assert.equal(formatRupiah('950'), 'Rp 950');
  assert.equal(formatRupiah('1000'), 'Rp 1.000');
});

test('formatRupiah: pembulatan ke rupiah terdekat dan opsi sen', () => {
  assert.equal(formatRupiah('1500.49'), 'Rp 1.500');
  assert.equal(formatRupiah('1500.50'), 'Rp 1.501');
  assert.equal(formatRupiah('1500.5', { cents: true }), 'Rp 1.500,50');
  assert.equal(formatRupiah('1500.005', { cents: true }), 'Rp 1.500,01');
  assert.equal(formatRupiah('99.995', { cents: true }), 'Rp 100,00');
});

test('formatRupiah: negatif dan nol negatif', () => {
  assert.equal(formatRupiah('-250000.00'), '-Rp 250.000');
  assert.equal(formatRupiah('-0.00'), 'Rp 0');
  assert.equal(formatRupiah('-0.40'), 'Rp 0');
  assert.equal(formatRupiah('-0.40', { cents: true }), '-Rp 0,40');
});

test('formatRupiah: angka besar tetap tepat (tanpa float)', () => {
  assert.equal(formatRupiah('123456789012345678.99'), 'Rp 123.456.789.012.345.679');
  assert.equal(formatRupiah('9007199254740993.00'), 'Rp 9.007.199.254.740.993');
});

test('formatRupiah: input tidak valid', () => {
  assert.equal(formatRupiah(null), '—');
  assert.equal(formatRupiah(undefined), '—');
  assert.equal(formatRupiah(''), '—');
  assert.equal(formatRupiah('abc'), '—');
  assert.equal(formatRupiah('1.000.000'), '—');
  assert.equal(formatRupiah('12,50'), '—');
  assert.equal(formatRupiah('x', { fallback: 'Belum diatur' }), 'Belum diatur');
  assert.equal(parseDecimal(' 42.10 ')?.whole, BigInt(42));
});

test('formatRupiahCompact', () => {
  assert.equal(formatRupiahCompact('118000000.00'), 'Rp 118 jt');
  assert.equal(formatRupiahCompact('2500000000'), 'Rp 2,5 M');
  assert.equal(formatRupiahCompact('1250000'), 'Rp 1,2 jt');
  assert.equal(formatRupiahCompact('75000'), 'Rp 75 rb');
  assert.equal(formatRupiahCompact('950'), 'Rp 950');
  assert.equal(formatRupiahCompact('-3000000'), '-Rp 3 jt');
});

test('toDecimalString: isian form ke format API', () => {
  assert.equal(toDecimalString('7.250.000'), '7250000.00');
  assert.equal(toDecimalString('Rp 18.000'), '18000.00');
  assert.equal(toDecimalString('007'), '7.00');
  assert.equal(toDecimalString('abc'), null);
});

test('tanggal dan jam: UTC ke WITA (bawaan)', () => {
  assert.equal(formatTime('2026-11-10T00:05:00Z'), '08:05');
  assert.equal(formatDate('2026-11-10T00:05:00Z'), '10/11/2026');
  assert.equal(formatDateTime('2026-11-10T13:45:30Z'), '10/11/2026 21:45');
});

test('tanggal berganti hari karena zona waktu', () => {
  // 16:30 UTC = 00:30 WITA keesokan harinya, masih 23:30 WIB.
  assert.equal(formatDateTime('2026-11-10T16:30:00Z', 'Asia/Makassar'), '11/11/2026 00:30');
  assert.equal(formatDateTime('2026-11-10T16:30:00Z', 'Asia/Jakarta'), '10/11/2026 23:30');
  assert.equal(formatDateTime('2026-11-10T16:30:00Z', 'Asia/Jayapura'), '11/11/2026 01:30');
  assert.equal(toLocalDateKey('2026-12-31T16:00:00Z', 'Asia/Makassar'), '2027-01-01');
});

test('offset eksplisit di string ISO dihormati', () => {
  assert.equal(formatTime('2026-11-10T08:00:00+08:00', 'Asia/Makassar'), '08:00');
  assert.equal(formatTime('2026-11-10T08:00:00+08:00', 'Asia/Jakarta'), '07:00');
});

test('zona waktu di luar Indonesia memakai Intl', () => {
  assert.equal(formatTime('2026-07-01T12:00:00Z', 'Europe/Amsterdam'), '14:00');
  assert.equal(formatTime('2026-01-01T12:00:00Z', 'Europe/Amsterdam'), '13:00');
  assert.equal(formatTime('2026-01-01T12:00:00Z', 'Zona/Ngawur'), '—');
});

test('tanggal tanpa jam tidak digeser zona waktu', () => {
  assert.equal(formatDate('2025-01-02'), '02/01/2025');
  assert.equal(formatDate('2025-01-02', 'Asia/Jayapura'), '02/01/2025');
  assert.equal(formatTime('2025-01-02'), '—');
  assert.equal(formatDate('2025-02-30'), '—');
});

test('tanggal panjang Bahasa Indonesia', () => {
  assert.equal(formatLongDate('2026-11-11T02:00:00Z'), 'Rabu, 11 November 2026');
  assert.equal(formatLongDate('2025-01-02'), 'Kamis, 2 Januari 2025');
});

test('jam 24 jam di tengah malam dan siang', () => {
  assert.equal(formatTime('2026-11-10T16:00:00Z'), '00:00');
  assert.equal(formatTime('2026-11-10T04:00:00Z'), '12:00');
  assert.equal(formatTime('2026-11-10T15:59:00Z'), '23:59');
});

test('nilai tidak valid', () => {
  assert.equal(formatDate(null), '—');
  assert.equal(formatDate(''), '—');
  assert.equal(formatDate('bukan tanggal'), '—');
  assert.equal(formatTime(Number.NaN), '—');
  assert.equal(formatDateTime(undefined, undefined, 'Belum ada'), 'Belum ada');
});

test('label zona waktu', () => {
  assert.equal(timeZoneLabel(), 'WITA');
  assert.equal(timeZoneLabel('Asia/Jakarta'), 'WIB');
  assert.equal(timeZoneLabel('Asia/Jayapura'), 'WIT');
  assert.equal(timeZoneLabel('Europe/Amsterdam'), 'Europe/Amsterdam');
});
