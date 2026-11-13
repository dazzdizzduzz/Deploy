import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import { Badge } from '../components/ui/Badge';
import { Button } from '../components/ui/Button';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';
import type { GeofenceCheck } from './geo';
import type { Position } from './useCurrentPosition';

type LocationStatusProps = {
  granted: boolean;
  loading: boolean;
  error: string | null;
  position: Position | null;
  check: GeofenceCheck | null;
  /** Teks bila tidak ada kantor untuk dibandingkan. */
  emptyText: string;
  onRefresh: () => void;
};

/** Status di dalam/di luar radius kantor untuk langkah lokasi clock-in dan clock-out. */
export function LocationStatus({ granted, loading, error, position, check, emptyText, onRefresh }: LocationStatusProps) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();

  return (
    <>
      {loading || (granted && !position && !error) ? (
        <View style={styles.row}>
          <ActivityIndicator color={color.text.primary} />
          <Text style={styles.value}>Membaca lokasi…</Text>
        </View>
      ) : error ? (
        <Text style={styles.danger}>{error}</Text>
      ) : !check ? (
        <Text style={styles.value}>{emptyText}</Text>
      ) : (
        <View style={styles.geo}>
          <Badge tone={check.inside ? 'success' : 'danger'}>{check.inside ? 'Di dalam radius' : 'Di luar radius'}</Badge>
          <Text style={styles.value}>
            {check.distance} m dari {check.location.name} · batas {check.location.radius_meters} m
          </Text>
          {position?.accuracy != null && <Text style={styles.hint}>Akurasi GPS ± {Math.round(position.accuracy)} m</Text>}
        </View>
      )}
      {granted && <Button variant="secondary" label="Perbarui lokasi" loading={loading} onPress={onRefresh} />}
    </>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    row: { flexDirection: 'row', alignItems: 'center', gap: space[12] },
    geo: { gap: space[8] },
    value: { ...text.bodyM, color: color.text.primary, flexShrink: 1 },
    hint: { ...text.bodyS, color: color.text.secondary },
    danger: { ...text.bodyM, color: color.status.dangerFg },
  });
