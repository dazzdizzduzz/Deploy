import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import * as DocumentPicker from 'expo-document-picker';
import { useEffect, useRef, useState } from 'react';
import { KeyboardAvoidingView, Platform, Alert as NativeAlert, ScrollView, StyleSheet, Text, View } from 'react-native';
import { ApiError } from '../api/client';
import {
  createLeaveRequest,
  previewLeaveRequest,
  uploadLeaveAttachment,
  type LeavePreview,
  type LeaveType,
  type PickedFile,
} from '../api/leave';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { DateField } from '../components/ui/DateField';
import { TextField } from '../components/ui/Field';
import { Select } from '../components/ui/Select';
import {
  ATTACHMENT_MAX_BYTES,
  ATTACHMENT_TYPES,
  countWorkingDays,
  formatBytes,
  formatDateKey,
  REASON_MAX,
  leaveTypeOptions,
  toDateKey,
} from '../leave/labels';
import type { RootStackParamList } from '../navigation/types';
import { useStyles, type Theme } from '../theme/ThemeProvider';

type Props = NativeStackScreenProps<RootStackParamList, 'LeaveForm'>;

type FieldErrors = Partial<Record<'type' | 'start' | 'end' | 'reason' | 'attachment', string>>;
type Failure = { title: string; body: string };
/** `server` = hasil endpoint pratinjau; `local` = perkiraan perangkat saat endpoint belum aktif. */
type Preview = { source: 'server'; data: LeavePreview } | { source: 'local'; workingDays: number };

const formatDate = (date: Date) => formatDateKey(toDateKey(date));

/** FR-12 · Form pengajuan izin, sakit, dan cuti tahunan dengan lampiran surat dokter. */
export function LeaveFormScreen({ navigation }: Props) {
  const styles = useStyles(makeStyles);
  const [type, setType] = useState<LeaveType | null>(null);
  const [start, setStart] = useState<Date | null>(null);
  const [end, setEnd] = useState<Date | null>(null);
  const [reason, setReason] = useState('');
  const [file, setFile] = useState<PickedFile | null>(null);
  const [errors, setErrors] = useState<FieldErrors>({});
  const [failure, setFailure] = useState<Failure | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const submitted = useRef(false);

  const startKey = start ? toDateKey(start) : null;
  const endKey = end ? toDateKey(end) : null;
  const dirty = Boolean(type || start || end || reason.trim() || file);

  // Pratinjau hari kerja dan kuota setiap kali jenis atau tanggal berubah.
  useEffect(() => {
    if (!type || !startKey || !endKey || endKey < startKey) {
      setPreview(null);
      return;
    }
    let active = true;
    const timer = setTimeout(() => {
      previewLeaveRequest({ type, start_date: startKey, end_date: endKey })
        .then((data) => active && setPreview({ source: 'server', data }))
        .catch(() => active && setPreview({ source: 'local', workingDays: countWorkingDays(startKey, endKey) }));
    }, 300);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [type, startKey, endKey]);

  // Cegah isian hilang saat menekan kembali sebelum mengirim.
  useEffect(
    () =>
      navigation.addListener('beforeRemove', (event) => {
        if (!dirty || submitted.current) return;
        event.preventDefault();
        NativeAlert.alert('Buang pengajuan?', 'Isian yang belum dikirim akan hilang.', [
          { text: 'Lanjut mengisi', style: 'cancel' },
          { text: 'Buang', style: 'destructive', onPress: () => navigation.dispatch(event.data.action) },
        ]);
      }),
    [navigation, dirty],
  );

  const workingDays = preview ? (preview.source === 'server' ? preview.data.working_days : preview.workingDays) : null;
  const needsDoctorNote =
    type === 'sakit' &&
    (preview?.source === 'server' ? preview.data.doctor_note_required : workingDays !== null && workingDays > 1);

  function clearError(field: keyof FieldErrors) {
    setErrors((current) => (current[field] ? { ...current, [field]: undefined } : current));
  }

  function changeStart(date: Date) {
    setStart(date);
    clearError('start');
    // Tanggal selesai mengikuti tanggal mulai bila belum diisi atau jadi lebih awal.
    if (!end || toDateKey(end) < toDateKey(date)) setEnd(date);
    clearError('end');
  }

  async function pickFile() {
    const result = await DocumentPicker.getDocumentAsync({ type: ATTACHMENT_TYPES, copyToCacheDirectory: true });
    if (result.canceled) return;
    const asset = result.assets[0];
    const mimeType = asset.mimeType ?? '';
    if (!ATTACHMENT_TYPES.includes(mimeType)) {
      setErrors((current) => ({ ...current, attachment: 'Gunakan berkas JPG, PNG, atau PDF.' }));
      return;
    }
    if (asset.size !== undefined && asset.size > ATTACHMENT_MAX_BYTES) {
      setErrors((current) => ({ ...current, attachment: `Ukuran berkas ${formatBytes(asset.size ?? 0)}. Maksimal 5 MB.` }));
      return;
    }
    setFile({ uri: asset.uri, name: asset.name, mimeType, size: asset.size });
    clearError('attachment');
  }

  function validate(): FieldErrors {
    const next: FieldErrors = {};
    if (!type) next.type = 'Pilih jenis pengajuan.';
    if (!start) next.start = 'Pilih tanggal mulai.';
    if (!end) next.end = 'Pilih tanggal selesai.';
    else if (startKey && endKey && endKey < startKey) next.end = 'Tanggal selesai tidak boleh sebelum tanggal mulai.';
    else if (workingDays === 0) next.end = 'Rentang ini tidak memuat hari kerja (Senin–Jumat).';
    if (!reason.trim()) next.reason = 'Tulis alasan pengajuan.';
    if (needsDoctorNote && !file) next.attachment = 'Sakit lebih dari 1 hari kerja wajib melampirkan surat dokter.';
    return next;
  }

  async function submit() {
    setFailure(null);
    const found = validate();
    setErrors(found);
    if (Object.values(found).some(Boolean) || !type || !startKey || !endKey) return;

    setSubmitting(true);
    try {
      const attachment = file && type === 'sakit' ? await uploadLeaveAttachment(file) : null;
      const request = await createLeaveRequest({
        type,
        start_date: startKey,
        end_date: endKey,
        reason: reason.trim(),
        ...(attachment && { attachment_url: attachment.attachment_url }),
      });
      submitted.current = true;
      navigation.replace('LeaveDetail', { id: request.id, request, created: true });
    } catch (error) {
      handleError(error);
    } finally {
      setSubmitting(false);
    }
  }

  function handleError(error: unknown) {
    if (!(error instanceof ApiError) || error.status === 0) {
      setFailure({ title: 'Tidak dapat terhubung', body: error instanceof ApiError ? error.message : 'Terjadi kesalahan. Coba lagi.' });
      return;
    }
    if (error.status === 404 || error.status === 405) {
      setFailure({
        title: 'Pengajuan belum bisa dikirim',
        body: 'Endpoint izin dan cuti belum aktif di server. Isianmu masih tersimpan di layar ini.',
      });
      return;
    }
    if (error.status >= 500) {
      setFailure({ title: 'Server sedang bermasalah', body: 'Pengajuan belum terkirim. Coba lagi beberapa saat lagi.' });
      return;
    }
    switch (error.code) {
      case 'DOCTOR_NOTE_REQUIRED':
      case 'ATTACHMENT_TOO_LARGE':
      case 'ATTACHMENT_UNSUPPORTED':
        setErrors((current) => ({ ...current, attachment: error.message }));
        return;
      case 'OVERLAPPING_LEAVE':
      case 'INVALID_DATE_RANGE':
        setErrors((current) => ({ ...current, end: error.message }));
        return;
      default:
        setFailure({ title: 'Pengajuan ditolak', body: error.message });
    }
  }

  return (
    <KeyboardAvoidingView style={styles.screen} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
        <Select
          label="Jenis pengajuan"
          options={leaveTypeOptions}
          value={type}
          onChange={(value) => {
            setType(value as LeaveType);
            clearError('type');
            if (value !== 'sakit') clearError('attachment');
          }}
          placeholder="Pilih jenis"
          helper={type === 'cuti' ? 'Mengurangi kuota cuti tahunan.' : ' '}
          error={errors.type}
        />

        <View style={styles.dates}>
          <View style={styles.date}>
            <DateField label="Tanggal mulai" value={start} onChange={changeStart} format={formatDate} error={errors.start} />
          </View>
          <View style={styles.date}>
            <DateField
              label="Tanggal selesai"
              value={end}
              onChange={(date) => {
                setEnd(date);
                clearError('end');
              }}
              minimumDate={start ?? undefined}
              format={formatDate}
              error={errors.end}
            />
          </View>
        </View>

        {workingDays !== null && (
          <Card>
            <Text style={styles.summary} accessibilityLiveRegion="polite">
              {workingDays} hari kerja
              {preview?.source === 'local' ? ' (perkiraan, belum termasuk libur nasional)' : ''}
            </Text>
            {preview?.source === 'server' && preview.data.deducts_quota && preview.data.remaining_days_after != null && (
              <Text style={styles.meta}>Sisa kuota cuti setelah pengajuan ini: {preview.data.remaining_days_after} hari</Text>
            )}
            {preview?.source === 'server' && !preview.data.quota_sufficient && (
              <Text style={styles.warning}>Kuota cuti tidak cukup untuk rentang ini.</Text>
            )}
            {preview?.source === 'server' && preview.data.overlapping && (
              <Text style={styles.warning}>Tanggal ini bertabrakan dengan pengajuan lain yang menunggu atau disetujui.</Text>
            )}
          </Card>
        )}

        <TextField
          label="Alasan"
          value={reason}
          onChangeText={(value) => {
            setReason(value);
            if (value.trim()) clearError('reason');
          }}
          multiline
          maxLength={REASON_MAX}
          placeholder="Contoh: Acara keluarga di luar kota"
          helper={`${reason.length}/${REASON_MAX} karakter`}
          error={errors.reason}
        />

        {type === 'sakit' && (
          <View style={styles.attachment}>
            <Text style={styles.label}>Surat dokter{needsDoctorNote ? '' : ' (opsional)'}</Text>
            {file ? (
              <View style={styles.file}>
                <View style={styles.fileText}>
                  <Text style={styles.fileName} numberOfLines={1}>
                    {file.name}
                  </Text>
                  {file.size !== undefined && <Text style={styles.meta}>{formatBytes(file.size)}</Text>}
                </View>
                <Button variant="ghost" label="Hapus" onPress={() => setFile(null)} />
              </View>
            ) : (
              <Button variant="secondary" fullWidth label="Pilih berkas" onPress={() => void pickFile()} />
            )}
            <Text style={[styles.meta, errors.attachment ? styles.error : null]} accessibilityLiveRegion="polite">
              {errors.attachment ?? 'JPG, PNG, atau PDF, maksimal 5 MB. Wajib untuk sakit lebih dari 1 hari kerja.'}
            </Text>
          </View>
        )}

        {failure && (
          <Alert tone="danger" title={failure.title} onDismiss={() => setFailure(null)}>
            {failure.body}
          </Alert>
        )}

        <Button fullWidth label="Kirim Pengajuan" loading={submitting} onPress={() => void submit()} />
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const makeStyles = ({ color, radius, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[48] },
    dates: { gap: space[16] },
    date: { alignSelf: 'stretch' },
    summary: { ...text.labelM, color: color.text.primary },
    meta: { ...text.caption, color: color.text.secondary },
    warning: { ...text.caption, color: color.status.warningFg },
    error: { color: color.status.dangerFg },
    attachment: { gap: space[8] },
    label: { ...text.labelM, color: color.text.primary },
    file: {
      flexDirection: 'row',
      alignItems: 'center',
      gap: space[12],
      paddingLeft: space[16],
      paddingRight: space[4],
      minHeight: 52,
      borderWidth: 1,
      borderColor: color.border.default,
      borderRadius: radius.md,
      backgroundColor: color.bg.surface,
    },
    fileText: { flex: 1, gap: space[2] },
    fileName: { ...text.bodyM, color: color.text.primary },
  });
