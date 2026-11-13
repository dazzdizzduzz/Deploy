import { useState } from 'react';
import { ActivityIndicator, RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useAuth } from '../auth/AuthProvider';
import { Alert } from '../components/ui/Alert';
import { Avatar } from '../components/ui/Avatar';
import { Button } from '../components/ui/Button';
import { Card } from '../components/ui/Card';
import { Dialog } from '../components/ui/Dialog';
import { employmentLabels, formatDate, initialsOf, roleLabels } from '../profile/format';
import type { NativeStackScreenProps } from '@react-navigation/native-stack';
import type { RootStackParamList } from '../navigation/types';
import { statusLabels, usePermissions } from '../permissions/PermissionsProvider';
import { useProfile } from '../profile/ProfileProvider';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';

function InfoRow({ label, value, first }: { label: string; value: string | null; first?: boolean }) {
  const styles = useStyles(makeStyles);
  return (
    <View accessible style={[styles.infoRow, !first && styles.infoBorder]}>
      <Text style={styles.infoLabel}>{label}</Text>
      <Text style={[styles.infoValue, !value && styles.infoEmpty]}>{value ?? 'Belum diisi'}</Text>
    </View>
  );
}

/** Profil Saya: data diri pengguna yang sedang login (FR-03) dan tombol keluar. */
type Props = NativeStackScreenProps<RootStackParamList, 'Profile'>;

export function ProfileScreen({ navigation }: Props) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const { signOut } = useAuth();
  const { profile, loading, error, reload } = useProfile();
  const permissions = usePermissions();
  const [confirmLogout, setConfirmLogout] = useState(false);

  if (loading && !profile) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color={color.text.primary} />
      </View>
    );
  }

  const role = profile ? (roleLabels[profile.role] ?? profile.role) : null;
  const subtitle = [profile?.position, profile?.department].filter(Boolean).join(' · ');

  return (
    <ScrollView
      style={styles.screen}
      contentContainerStyle={styles.content}
      refreshControl={<RefreshControl refreshing={loading} onRefresh={() => void reload()} />}
    >
      {error && (
        <Alert tone="danger" title="Profil belum bisa dimuat" action={{ label: 'Coba lagi', onPress: () => void reload() }}>
          {error}
        </Alert>
      )}

      {profile && (
        <>
          <View accessible style={styles.identity}>
            <Avatar initials={initialsOf(profile.fullName)} size="L" />
            <Text accessibilityRole="header" style={styles.name}>
              {profile.fullName ?? role}
            </Text>
            {subtitle ? <Text style={styles.subtitle}>{subtitle}</Text> : null}
            {profile.tenantName && <Text style={styles.tenant}>{profile.tenantName}</Text>}
          </View>

          {profile.source === 'token' && (
            <Alert tone="info" title="Profil lengkap belum tersedia">
              Server baru mengirim peran akun. Data lain tampil setelah endpoint profil (GET /auth/me) aktif.
            </Alert>
          )}

          <Card title="Data diri">
            <View>
              <InfoRow first label="Nama lengkap" value={profile.fullName} />
              <InfoRow label="Email" value={profile.email} />
              <InfoRow label="Nomor HP" value={profile.phone} />
            </View>
          </Card>

          <Card title="Pekerjaan">
            <View>
              <InfoRow first label="Perusahaan" value={profile.tenantName} />
              <InfoRow label="Jabatan" value={profile.position} />
              <InfoRow label="Departemen" value={profile.department} />
              <InfoRow label="Peran akun" value={role} />
              <InfoRow
                label="Status kerja"
                value={profile.employmentType ? (employmentLabels[profile.employmentType] ?? profile.employmentType) : null}
              />
              <InfoRow label="Bergabung" value={formatDate(profile.joinedAt)} />
            </View>
          </Card>
        </>
      )}

      <Card title="Izin aplikasi" description="Dipakai saat Clock In/Out.">
        <View>
          <InfoRow first label="Lokasi (GPS)" value={statusLabels[permissions.location]} />
          <InfoRow label="Kamera" value={statusLabels[permissions.camera]} />
        </View>
        <Button
          fullWidth
          variant="secondary"
          label="Kelola izin"
          onPress={() => navigation.navigate('Permissions', { from: 'profile' })}
        />
      </Card>

      <Button fullWidth variant="secondary" label="Keluar" onPress={() => setConfirmLogout(true)} />
      <Text style={styles.note}>Data diri diubah oleh HRD. Hubungi HRD bila ada yang keliru.</Text>

      <Dialog
        visible={confirmLogout}
        title="Keluar dari On-Track?"
        description="Kamu perlu masuk lagi dengan email dan kata sandi."
        onClose={() => setConfirmLogout(false)}
        actions={
          <>
            <Button variant="secondary" label="Batal" onPress={() => setConfirmLogout(false)} />
            <Button
              variant="danger"
              label="Keluar"
              onPress={() => {
                setConfirmLogout(false);
                void signOut();
              }}
            />
          </>
        }
      />
    </ScrollView>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    screen: { flex: 1, backgroundColor: color.bg.canvas },
    center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: color.bg.canvas },
    content: { gap: space[16], padding: space[20], paddingBottom: space[48] },
    identity: { alignItems: 'center', gap: space[4], paddingVertical: space[8] },
    name: { ...text.headingH1, marginTop: space[8], color: color.text.primary, textAlign: 'center' },
    subtitle: { ...text.bodyS, color: color.text.secondary, textAlign: 'center' },
    tenant: { ...text.labelM, color: color.text.accent, textAlign: 'center' },
    infoRow: { gap: space[2], paddingVertical: space[12] },
    infoBorder: { borderTopWidth: 1, borderTopColor: color.border.default },
    infoLabel: { ...text.caption, color: color.text.secondary },
    infoValue: { ...text.bodyM, color: color.text.primary },
    infoEmpty: { color: color.text.secondary },
    note: { ...text.caption, color: color.text.secondary, textAlign: 'center' },
  });
