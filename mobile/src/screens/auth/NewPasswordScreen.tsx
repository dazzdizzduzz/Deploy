import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { useState } from 'react';
import { StyleSheet, Text, View } from 'react-native';
import { ApiError } from '../../api/client';
import { resetPassword } from '../../api/password';
import { checkPassword } from '../../auth/password';
import { Alert } from '../../components/ui/Alert';
import { Button } from '../../components/ui/Button';
import { PasswordField, TextField } from '../../components/ui/Field';
import type { RootStackParamList } from '../../navigation/types';
import { useStyles, type Theme } from '../../theme/ThemeProvider';
import { AuthHeading, AuthLayout } from './AuthLayout';

type Props = NativeStackScreenProps<RootStackParamList, 'NewPassword'>;

function Rule({ label, met }: { label: string; met: boolean }) {
  const styles = useStyles(makeStyles);
  return (
    <View accessible accessibilityLabel={`${label}, ${met ? 'terpenuhi' : 'belum terpenuhi'}`} style={styles.rule}>
      <Text style={[styles.mark, met ? styles.markMet : styles.markUnmet]}>{met ? '✓' : '•'}</Text>
      <Text style={[styles.ruleText, !met && styles.ruleTextUnmet]}>{label}</Text>
    </View>
  );
}

/**
 * M11 · Kata Sandi Baru. Token datang dari tautan aplikasi (ontrack://kata-sandi-baru?token=...)
 * atau ditempel manual dari email.
 */
export function NewPasswordScreen({ navigation, route }: Props) {
  const styles = useStyles(makeStyles);
  const linkToken = route.params?.token?.trim() ?? '';
  const [token, setToken] = useState(linkToken);
  const [password, setPassword] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [failure, setFailure] = useState<{ title: string; body: string; expired?: boolean } | null>(null);

  const { minLength, lettersAndDigits, uppercase } = checkPassword(password);
  const passwordValid = minLength && lettersAndDigits && uppercase;
  const matches = password === confirmation;
  const tokenMissing = !token.trim();

  async function handleSubmit() {
    if (submitting) return;
    setSubmitted(true);
    if (tokenMissing || !passwordValid || !matches) return;
    setSubmitting(true);
    setFailure(null);
    try {
      await resetPassword(token.trim(), password);
      navigation.reset({ index: 0, routes: [{ name: 'Login' }, { name: 'PasswordChanged' }] });
    } catch (cause) {
      setSubmitting(false);
      if (cause instanceof ApiError && cause.status === 400) {
        setFailure({
          title: 'Tautan tidak dapat dipakai',
          body: 'Kode reset salah, sudah kedaluwarsa, atau sudah pernah dipakai. Minta tautan baru.',
          expired: true,
        });
      } else {
        setFailure({
          title: 'Kata sandi belum tersimpan',
          body: cause instanceof ApiError && cause.status === 0 ? cause.message : 'Server sedang bermasalah. Coba lagi.',
        });
      }
    }
  }

  return (
    <AuthLayout
      footer={
        <Button
          fullWidth
          label={submitting ? 'Menyimpan…' : 'Simpan kata sandi'}
          loading={submitting}
          onPress={() => void handleSubmit()}
        />
      }
    >
      <AuthHeading title="Buat kata sandi baru">Pilih kata sandi yang belum pernah kamu pakai di akun ini.</AuthHeading>

      {failure && (
        <Alert
          tone="danger"
          title={failure.title}
          action={
            failure.expired
              ? { label: 'Minta tautan baru', onPress: () => navigation.navigate('ForgotPassword') }
              : { label: 'Coba lagi', onPress: () => void handleSubmit() }
          }
        >
          {failure.body}
        </Alert>
      )}

      {linkToken ? (
        <Alert tone="success" title="Tautan reset terbaca">
          Silakan buat kata sandi baru.
        </Alert>
      ) : (
        <TextField
          label="Kode reset"
          helper="Salin dari email reset kata sandi"
          error={submitted && tokenMissing ? 'Kode reset wajib diisi' : undefined}
          value={token}
          onChangeText={setToken}
          autoCapitalize="none"
          autoCorrect={false}
          autoComplete="one-time-code"
          textContentType="oneTimeCode"
        />
      )}

      <PasswordField
        label="Kata sandi baru"
        helper="Minimal 8 karakter"
        error={submitted && !passwordValid ? 'Kata sandi belum memenuhi syarat' : undefined}
        value={password}
        onChangeText={setPassword}
        autoComplete="new-password"
        textContentType="newPassword"
      />
      <PasswordField
        label="Ulangi kata sandi"
        helper="Harus sama dengan kata sandi baru"
        error={submitted && !matches ? 'Kata sandi tidak sama' : undefined}
        value={confirmation}
        onChangeText={setConfirmation}
        autoComplete="new-password"
        textContentType="newPassword"
        returnKeyType="done"
        onSubmitEditing={() => void handleSubmit()}
      />
      <View style={styles.rules}>
        <Rule label="Minimal 8 karakter" met={minLength} />
        <Rule label="Mengandung huruf dan angka" met={lettersAndDigits} />
        <Rule label="Mengandung huruf besar" met={uppercase} />
      </View>
    </AuthLayout>
  );
}

const makeStyles = ({ color, radius, space, text }: Theme) =>
  StyleSheet.create({
    rules: { gap: space[8], padding: space[16], borderRadius: radius.md, backgroundColor: color.bg.subtle },
    rule: { flexDirection: 'row', alignItems: 'center', gap: space[8] },
    mark: { ...text.labelM, width: 16, textAlign: 'center' },
    markMet: { color: color.status.successFg },
    markUnmet: { color: color.text.secondary },
    ruleText: { ...text.bodyS, color: color.text.primary },
    ruleTextUnmet: { color: color.text.secondary },
  });
