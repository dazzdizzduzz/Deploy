import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useState } from 'react';
import { ApiError } from '../../api/client';
import { requestPasswordReset } from '../../api/password';
import { isValidEmail, RESET_TOKEN_MINUTES } from '../../auth/password';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import { TextField } from '../../components/ui/Field';
import type { RootStackParamList } from '../../navigation/types';
import { AuthHeading, AuthLayout } from './AuthLayout';

type Props = NativeStackScreenProps<RootStackParamList, 'ForgotPassword'>;

/** M10 · Lupa Kata Sandi: minta tautan reset ke email kantor. */
export function ForgotPasswordScreen({ navigation, route }: Props) {
  const [email, setEmail] = useState(route.params?.email ?? '');
  const [error, setError] = useState<string>();
  const [failure, setFailure] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit() {
    if (submitting) return;
    if (!isValidEmail(email)) {
      setError('Masukkan email yang valid');
      return;
    }
    setError(undefined);
    setFailure(null);
    setSubmitting(true);
    try {
      await requestPasswordReset(email.trim());
      navigation.navigate('CheckEmail', { email: email.trim() });
    } catch (cause) {
      if (cause instanceof ApiError && cause.status === 400) setError('Format email tidak valid');
      else setFailure(cause instanceof ApiError && cause.status === 0 ? cause.message : 'Server sedang bermasalah. Coba lagi.');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <AuthLayout
      footer={
        <>
          <Button fullWidth label={submitting ? 'Mengirim…' : 'Kirim tautan reset'} loading={submitting} onPress={() => void handleSubmit()} />
          <Button fullWidth variant="ghost" label="Kembali ke login" onPress={() => navigation.popTo('Login')} />
        </>
      }
    >
      <AuthHeading title="Atur ulang kata sandi">
        Masukkan email kantor yang terdaftar. Kami akan mengirim tautan untuk membuat kata sandi baru.
      </AuthHeading>
      {failure && (
        <Alert tone="danger" title="Tautan belum terkirim" action={{ label: 'Coba lagi', onPress: () => void handleSubmit() }}>
          {failure}
        </Alert>
      )}
      <TextField
        label="Email kantor"
        helper={`Tautan berlaku ${RESET_TOKEN_MINUTES} menit`}
        error={error}
        value={email}
        onChangeText={(value) => {
          setEmail(value);
          if (error) setError(undefined);
        }}
        placeholder="nama@perusahaan.com"
        keyboardType="email-address"
        autoCapitalize="none"
        autoCorrect={false}
        autoComplete="email"
        textContentType="username"
        returnKeyType="send"
        onSubmitEditing={() => void handleSubmit()}
        editable={!submitting}
      />
    </AuthLayout>
  );
}
