import { useCallback, useEffect, useState } from 'react';
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native';
import { API_BASE_URL } from '../api/config';
import { getHealth } from '../api/health';
import { Button } from '../components/ui/Button';
import { useStyles, useTheme, type Theme } from '../theme/ThemeProvider';

type State = { kind: 'loading' } | { kind: 'ok' } | { kind: 'error'; message: string };

export function ApiStatusScreen() {
  const [state, setState] = useState<State>({ kind: 'loading' });
  const styles = useStyles(makeStyles);
  const { color } = useTheme();

  const check = useCallback(async (signal?: AbortSignal) => {
    setState({ kind: 'loading' });
    try {
      const { status } = await getHealth(signal);
      setState(status === 'ok' ? { kind: 'ok' } : { kind: 'error', message: `Status server: ${status}` });
    } catch (error) {
      if (signal?.aborted) return;
      setState({ kind: 'error', message: error instanceof Error ? error.message : 'Terjadi kesalahan.' });
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void check(controller.signal);
    return () => controller.abort();
  }, [check]);

  return (
    <View style={styles.container}>
      <Text style={styles.url}>{API_BASE_URL}</Text>
      {state.kind === 'loading' && <ActivityIndicator size="large" color={color.text.primary} />}
      {state.kind === 'ok' && <Text style={styles.ok}>Terhubung ke server</Text>}
      {state.kind === 'error' && <Text style={styles.error}>{state.message}</Text>}
      <Button label="Coba lagi" loading={state.kind === 'loading'} onPress={() => void check()} />
    </View>
  );
}

const makeStyles = ({ color, space, text }: Theme) =>
  StyleSheet.create({
    container: {
      flex: 1,
      alignItems: 'center',
      justifyContent: 'center',
      gap: space[16],
      padding: space[24],
      backgroundColor: color.bg.surface,
    },
    url: { ...text.bodyS, color: color.text.secondary },
    ok: { ...text.headingH3, color: color.status.successFg },
    error: { ...text.bodyM, textAlign: 'center', color: color.status.dangerFg },
  });
