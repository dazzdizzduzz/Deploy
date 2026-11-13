import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useEffect, useState } from 'react';
import { requestPasswordReset } from '../../api/password';
import { formatCountdown, maskEmail, RESEND_RESET_SECONDS, RESET_TOKEN_MINUTES } from '../../auth/password';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import type { RootStackParamList } from '../../navigation/types';
import { AuthHeading, AuthLayout } from './AuthLayout';

type Props = NativeStackScreenProps<RootStackParamList, 'CheckEmail'>;

/** M24 · Cek Email: konfirmasi tautan terkirim, dengan kirim ulang setelah jeda. */
export function CheckEmailScreen({ navigation, route }: Props) {
  const { email } = route.params;
  const [seconds, setSeconds] = useState(RESEND_RESET_SECONDS);
  const [resending, setResending] = useState(false);
  const [notice, setNotice] = useState<{ tone: 'success' | 'danger'; text: string } | null>(null);
  const counting = seconds > 0;

  useEffect(() => {
    if (!counting) return;
    const timer = setInterval(() => setSeconds((value) => Math.max(0, value - 1)), 1000);
    return () => clearInterval(timer);
  }, [counting]);

  async function resend() {
    setResending(true);
    setNotice(null);
    try {
      await requestPasswordReset(email);
      setSeconds(RESEND_RESET_SECONDS);
      setNotice({ tone: 'success', text: 'Tautan baru sudah dikirim. Tautan sebelumnya tidak berlaku lagi.' });
    } catch {
      setNotice({ tone: 'danger', text: 'Tautan belum terkirim. Periksa koneksi lalu coba lagi.' });
    } finally {
      setResending(false);
    }
  }

  return (
    <AuthLayout
      footer={
        <>
          <Button fullWidth label="Masukkan kode reset" onPress={() => navigation.navigate('NewPassword')} />
          {counting ? (
            <Button fullWidth disabled label={`Kirim ulang dalam ${formatCountdown(seconds)}`} />
          ) : (
            <Button
              fullWidth
              variant="secondary"
              label={resending ? 'Mengirim…' : 'Kirim ulang tautan'}
              loading={resending}
              onPress={() => void resend()}
            />
          )}
        </>
      }
    >
      <AuthHeading title="Cek email kamu">
        Jika {maskEmail(email)} terdaftar, tautan untuk membuat kata sandi baru sudah dikirim. Buka tautan itu
        di ponsel ini.
      </AuthHeading>
      {notice && (
        <Alert tone={notice.tone} title={notice.tone === 'success' ? 'Terkirim' : 'Gagal mengirim'}>
          {notice.text}
        </Alert>
      )}
      <Alert tone="info" title="Belum menerima email?">
        Cek folder Spam/Promosi. Tautan kedaluwarsa dalam {RESET_TOKEN_MINUTES} menit. Bila email berisi kode,
        ketuk “Masukkan kode reset”.
      </Alert>
    </AuthLayout>
  );
}
