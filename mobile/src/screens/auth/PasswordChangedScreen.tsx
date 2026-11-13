import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import { Button } from '../../components/ui/Button';
import type { RootStackParamList } from '../../navigation/types';
import { AuthHeading, AuthLayout } from './AuthLayout';

type Props = NativeStackScreenProps<RootStackParamList, 'PasswordChanged'>;

/** M25 · Kata Sandi Berhasil Diubah */
export function PasswordChangedScreen({ navigation }: Props) {
  return (
    <AuthLayout footer={<Button fullWidth label="Masuk sekarang" onPress={() => navigation.popTo('Login')} />}>
      <AuthHeading title="Kata sandi berhasil diubah">
        Silakan masuk dengan email kantor dan kata sandi baru kamu.
      </AuthHeading>
    </AuthLayout>
  );
}
