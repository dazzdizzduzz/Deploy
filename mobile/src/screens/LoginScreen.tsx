import { useRef, useState, type ComponentRef } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';
import { useNavigation } from '@react-navigation/native';
import type { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { ApiError } from '../api/client';
import { useAuth } from '../auth/AuthProvider';
import { Alert } from '../components/ui/Alert';
import { Button } from '../components/ui/Button';
import { PasswordField, TextField } from '../components/ui/Field';
import type { RootStackParamList } from '../navigation/types';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';

type Failure = {
  tone: 'warning' | 'danger';
  title: string;
  body: string;
  /** Galat jaringan/server bisa dicoba ulang dengan data yang sama. */
  retry?: boolean;
  /** Akun dikunci: arahkan ke atur ulang kata sandi. */
  reset?: boolean;
};

const isValidEmail = (value: string) => /^\S+@\S+\.\S+$/.test(value.trim());

/** Pesan galat per status dari POST /auth/login. Pesan backend dipakai sebagai isi bila ada. */
function describeFailure(error: unknown): Failure {
  if (!(error instanceof ApiError)) {
    return { tone: 'danger', title: 'Gagal masuk', body: 'Terjadi kesalahan. Coba lagi.', retry: true };
  }
  switch (true) {
    case error.status === 401:
      return {
        tone: 'warning',
        title: 'Email atau kata sandi salah',
        body: error.message || 'Periksa kembali email dan kata sandi kamu.',
      };
    case error.status === 423:
      return {
        tone: 'danger',
        title: 'Akun dikunci sementara',
        body: error.message || 'Terlalu banyak percobaan gagal. Coba lagi dalam 15 menit.',
        reset: true,
      };
    case error.status === 400:
      return { tone: 'warning', title: 'Data belum lengkap', body: error.message };
    case error.status === 0:
      return { tone: 'danger', title: 'Tidak dapat terhubung', body: error.message, retry: true };
    default:
      return {
        tone: 'danger',
        title: 'Server bermasalah',
        body: 'Terjadi kesalahan pada server. Coba lagi beberapa saat lagi.',
        retry: true,
      };
  }
}

export function LoginScreen() {
  const { signIn } = useAuth();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList, 'Login'>>();
  const styles = useStyles(makeStyles);
  const { space } = useTheme();
  const { top, bottom } = useSafeAreaInsets();
  const passwordRef = useRef<ComponentRef<typeof TextInput>>(null);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [emailError, setEmailError] = useState<string>();
  const [passwordError, setPasswordError] = useState<string>();
  const [failure, setFailure] = useState<Failure | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const goToReset = () => navigation.navigate('ForgotPassword', isValidEmail(email) ? { email: email.trim() } : undefined);

  async function handleSubmit() {
    if (submitting) return;
    const nextEmailError = isValidEmail(email) ? undefined : 'Masukkan email yang valid';
    const nextPasswordError = password ? undefined : 'Kata sandi wajib diisi';
    setEmailError(nextEmailError);
    setPasswordError(nextPasswordError);
    if (nextEmailError || nextPasswordError) return;

    setSubmitting(true);
    setFailure(null);
    try {
      // Berhasil: AuthProvider menyimpan token di SecureStore dan navigator pindah ke beranda.
      await signIn({ email: email.trim(), password });
    } catch (error) {
      const next = describeFailure(error);
      setFailure(next);
      if (error instanceof ApiError && error.status === 401) {
        setPassword('');
        passwordRef.current?.focus();
      }
      setSubmitting(false);
    }
  }

  return (
    <KeyboardAvoidingView style={styles.screen} behavior={Platform.OS === 'ios' ? 'padding' : undefined}>
      <ScrollView
        contentContainerStyle={[
          styles.content,
          { paddingTop: Math.max(space[48], top + space[16]), paddingBottom: Math.max(space[32], bottom) },
        ]}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.logo}>
          <View style={styles.logoMark} />
          <Text style={styles.logoText}>On-Track</Text>
        </View>

        <View style={styles.heading}>
          <Text accessibilityRole="header" style={styles.title}>
            Masuk ke akun
          </Text>
          <Text style={styles.subtitle}>Absen, izin, dan pantau task dalam satu aplikasi.</Text>
        </View>

        {failure && (
          <Alert
            tone={failure.tone}
            title={failure.title}
            onDismiss={() => setFailure(null)}
            action={
              failure.retry
                ? { label: 'Coba lagi', onPress: () => void handleSubmit() }
                : failure.reset
                  ? { label: 'Atur ulang kata sandi', onPress: goToReset }
                  : undefined
            }
          >
            {failure.body}
          </Alert>
        )}

        <View style={styles.form}>
          <TextField
            label="Email"
            helper="Gunakan email kantor"
            error={emailError}
            value={email}
            onChangeText={(value) => {
              setEmail(value);
              if (emailError) setEmailError(undefined);
            }}
            placeholder="nama@studiotigapagi.com"
            keyboardType="email-address"
            autoCapitalize="none"
            autoCorrect={false}
            autoComplete="email"
            textContentType="username"
            returnKeyType="next"
            onSubmitEditing={() => passwordRef.current?.focus()}
            editable={!submitting}
          />
          <PasswordField
            ref={passwordRef}
            label="Kata sandi"
            helper=" "
            error={passwordError}
            value={password}
            onChangeText={(value) => {
              setPassword(value);
              if (passwordError) setPasswordError(undefined);
            }}
            placeholder="Masukkan kata sandi"
            autoComplete="current-password"
            textContentType="password"
            returnKeyType="go"
            onSubmitEditing={() => void handleSubmit()}
            editable={!submitting}
          />
          <View style={styles.options}>
            <Button variant="ghost" label="Lupa kata sandi?" onPress={goToReset} disabled={submitting} />
          </View>
        </View>

        <Button
          fullWidth
          label={submitting ? 'Memproses…' : 'Masuk'}
          loading={submitting}
          onPress={() => void handleSubmit()}
        />

        <Text style={styles.note}>
          Dengan masuk, kamu menyetujui penggunaan lokasi & kamera untuk absensi sesuai UU PDP No.
          27/2022.
        </Text>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
  screen: {
    flex: 1,
    backgroundColor: color.bg.surface,
  },
  content: {
    flexGrow: 1,
    justifyContent: 'flex-end',
    gap: space[24],
    paddingHorizontal: space[24],
  },
  logo: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: space[8],
    marginBottom: 'auto',
  },
  logoMark: {
    width: 14,
    height: 14,
    borderRadius: 7,
    backgroundColor: color.accent.default,
  },
  logoText: {
    ...text.headingH2,
    color: color.text.primary,
  },
  heading: {
    gap: space[8],
  },
  title: {
    ...text.displayL,
    color: color.text.primary,
  },
  subtitle: {
    ...text.bodyM,
    color: color.text.secondary,
  },
  options: { flexDirection: 'row', justifyContent: 'flex-end', marginTop: -space[12] },
  form: {
    gap: space[20],
  },
  note: {
    ...text.caption,
    color: color.text.secondary,
    textAlign: 'center',
  },
  });
