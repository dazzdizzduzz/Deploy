'use client';

import { useState, type FormEvent } from 'react';
import { Alert } from '@/components/ui/Alert';
import { Avatar } from '@/components/ui/Avatar';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Drawer } from '@/components/ui/Drawer';
import { TextareaField } from '@/components/ui/Field';
import { initialsOf } from '@/features/hr/labels';
import { ApiProblem } from '@/lib/api/client';
import { formatDateTime } from '@/lib/format';
import { useAsync } from '@/lib/hooks/useAsync';
import { leaveApprovalApi, type LeaveApprovalItem } from './api';
import { formatRange, leaveStatusBadges, leaveTypeLabels, NOTE_MAX } from './labels';
import styles from './leave.module.css';

type Props = {
  /** Data dari baris tabel; ditampilkan selagi detail lengkap dimuat. */
  item: LeaveApprovalItem;
  onClose: () => void;
  onDecided: (item: LeaveApprovalItem) => void;
};

type Failure = { title: string; detail: string };

function decisionFailure(error: unknown): Failure {
  if (!(error instanceof ApiProblem)) return { title: 'Terjadi kesalahan', detail: 'Coba lagi.' };
  const { status, title, detail } = error.problem;
  if (status === 404 || status === 405) {
    return { title: 'Belum tersedia di server', detail: 'Endpoint persetujuan izin belum aktif di backend.' };
  }
  if (status === 409) return { title: 'Sudah diputuskan', detail: detail ?? 'Pengajuan ini sudah diproses orang lain.' };
  if (status === 0 || status >= 500) return { title, detail: detail ?? 'Keputusan belum tersimpan. Coba lagi.' };
  return { title, detail: detail ?? 'Keputusan belum tersimpan.' };
}

/** Detail pengajuan dengan lampiran, serta aksi setujui atau tolak dengan catatan (FR-14). */
export function LeaveDetailDrawer({ item: initial, onClose, onDecided }: Props) {
  const detail = useAsync(() => leaveApprovalApi.get(initial.id), [initial.id]);
  const item = detail.data ?? initial;
  const [mode, setMode] = useState<'view' | 'reject'>('view');
  const [note, setNote] = useState('');
  const [reason, setReason] = useState('');
  const [reasonError, setReasonError] = useState<string | undefined>();
  const [failure, setFailure] = useState<Failure | null>(null);
  const [saving, setSaving] = useState(false);

  const pending = item.status === 'menunggu';
  const badge = leaveStatusBadges[item.status];
  const detailUnavailable = detail.error && (detail.error.status === 404 || detail.error.status === 405);
  const downloadUrl = item.attachment?.download_url;
  const quotaShort = item.type === 'cuti' && item.leave_quota_remaining != null && item.leave_quota_remaining < item.working_days;

  async function decide(action: () => Promise<LeaveApprovalItem>) {
    setSaving(true);
    setFailure(null);
    try {
      onDecided(await action());
    } catch (error) {
      setFailure(decisionFailure(error));
      // Status bisa sudah berubah di server (409): muat ulang detail.
      if (error instanceof ApiProblem && error.status === 409) void detail.reload();
    } finally {
      setSaving(false);
    }
  }

  function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (!pending || saving) return;
    if (mode === 'reject') {
      if (!reason.trim()) {
        setReasonError('Tulis alasan penolakan agar karyawan tahu apa yang perlu diperbaiki.');
        return;
      }
      void decide(() => leaveApprovalApi.reject(item.id, reason.trim()));
      return;
    }
    void decide(() => leaveApprovalApi.approve(item.id, note.trim() || undefined));
  }

  const actions = !pending ? undefined : mode === 'reject' ? (
    <>
      <Button variant="secondary" compact onClick={() => setMode('view')} disabled={saving}>
        Batal
      </Button>
      <Button variant="danger" compact type="submit" disabled={saving}>
        {saving ? 'Menolak…' : 'Tolak pengajuan'}
      </Button>
    </>
  ) : (
    <>
      <Button
        variant="secondary"
        compact
        disabled={saving}
        onClick={() => {
          setFailure(null);
          setMode('reject');
        }}
      >
        Tolak
      </Button>
      <Button compact type="submit" disabled={saving}>
        {saving ? 'Menyetujui…' : 'Setujui'}
      </Button>
    </>
  );

  return (
    <Drawer
      title={item.employee.name}
      subtitle={[item.employee.position_name, item.employee.department_name].filter(Boolean).join(' · ') || undefined}
      leading={<Avatar initials={initialsOf(item.employee.name)} size="M" />}
      onClose={onClose}
      onSubmit={handleSubmit}
      actions={actions}
    >
      <div className={styles.drawerBody}>
        <div className={styles.statusRow}>
          <span className={styles.typeLabel}>{leaveTypeLabels[item.type]}</span>
          <Badge tone={badge.tone}>{badge.label}</Badge>
        </div>

        <dl className={styles.details}>
          <div>
            <dt>Tanggal</dt>
            <dd>{formatRange(item.start_date, item.end_date)}</dd>
          </div>
          <div>
            <dt>Lama</dt>
            <dd>{item.working_days} hari kerja</dd>
          </div>
          {item.type === 'cuti' && item.leave_quota_remaining != null && (
            <div>
              <dt>Sisa kuota cuti</dt>
              <dd className={quotaShort ? styles.danger : undefined}>
                {item.leave_quota_remaining} hari{quotaShort ? ' (tidak cukup)' : ''}
              </dd>
            </div>
          )}
          <div>
            <dt>Diajukan</dt>
            <dd>{formatDateTime(item.created_at)}</dd>
          </div>
          <div className={styles.full}>
            <dt>Alasan</dt>
            <dd className={styles.reason}>{item.reason}</dd>
          </div>
        </dl>

        <section className={styles.attachment} aria-labelledby="lampiran">
          <h3 id="lampiran" className={styles.sectionTitle}>
            Lampiran
          </h3>
          {item.attachment ? (
            <div className={styles.file}>
              <span className={styles.fileName}>{item.attachment.file_name}</span>
              {downloadUrl ? (
                <a href={downloadUrl} target="_blank" rel="noopener noreferrer" className={styles.fileLink}>
                  Buka lampiran
                  <span className={styles.srOnly}> (tab baru)</span>
                </a>
              ) : (
                <span className={styles.muted}>Tautan belum tersedia</span>
              )}
            </div>
          ) : (
            <p className={styles.muted}>
              {item.type === 'sakit' && item.working_days > 1
                ? 'Tidak ada surat dokter, padahal wajib untuk sakit lebih dari 1 hari kerja.'
                : 'Tidak ada lampiran.'}
            </p>
          )}
        </section>

        {!pending && (
          <section className={styles.decision} aria-labelledby="keputusan">
            <h3 id="keputusan" className={styles.sectionTitle}>
              {item.status === 'ditolak' ? 'Alasan penolakan' : 'Catatan persetujuan'}
            </h3>
            <p className={styles.reason}>{item.decision_note || 'Tanpa catatan.'}</p>
            {(item.decided_by_name || item.decided_at) && (
              <p className={styles.muted}>
                {[item.decided_by_name && `Oleh ${item.decided_by_name}`, item.decided_at && formatDateTime(item.decided_at)]
                  .filter(Boolean)
                  .join(' · ')}
              </p>
            )}
          </section>
        )}

        {pending && mode === 'view' && (
          <TextareaField
            label="Catatan untuk karyawan (opsional)"
            value={note}
            maxLength={NOTE_MAX}
            rows={3}
            helper={`${note.length}/${NOTE_MAX} karakter · dikirim bersama persetujuan`}
            onChange={(event) => setNote(event.target.value)}
          />
        )}

        {pending && mode === 'reject' && (
          <TextareaField
            label="Alasan penolakan"
            value={reason}
            maxLength={NOTE_MAX}
            rows={4}
            autoFocus
            required
            helper={`${reason.length}/${NOTE_MAX} karakter · wajib diisi`}
            error={reasonError}
            onChange={(event) => {
              setReason(event.target.value);
              if (event.target.value.trim()) setReasonError(undefined);
            }}
          />
        )}

        {quotaShort && pending && (
          <Alert tone="warning" title="Kuota cuti tidak cukup">
            Sisa kuota {item.leave_quota_remaining} hari, sedangkan pengajuan ini {item.working_days} hari kerja.
          </Alert>
        )}

        {detail.error && !detailUnavailable && (
          <Alert tone="warning" title="Detail belum lengkap">
            {detail.error.problem.detail ?? 'Detail terbaru belum bisa dimuat.'}{' '}
            <button type="button" className={styles.inlineLink} onClick={() => void detail.reload()}>
              Coba lagi
            </button>
          </Alert>
        )}

        <div aria-live="polite">
          {failure && (
            <Alert tone="danger" title={failure.title}>
              {failure.detail}
            </Alert>
          )}
        </div>
      </div>
    </Drawer>
  );
}
