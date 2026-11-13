import DateTimePicker, { DateTimePickerAndroid } from '@react-native-community/datetimepicker';
import { useState } from 'react';
import { Modal, Platform, Pressable, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useStyles, useTheme, type Theme } from '../../theme/ThemeProvider';
import { Button } from './Button';
import { FieldFrame } from './Field';

type DateFieldProps = {
  label: string;
  value: Date | null;
  onChange: (date: Date) => void;
  /** Teks tampilan untuk nilai terpilih, misalnya "Senin, 16 Nov 2026". */
  format: (date: Date) => string;
  placeholder?: string;
  minimumDate?: Date;
  maximumDate?: Date;
  helper?: string;
  error?: string;
};

/**
 * Field tanggal. Android memakai dialog kalender bawaan; iOS memakai kalender inline di lembar bawah
 * dengan tombol Pilih, supaya tanggal baru tersimpan saat dikonfirmasi.
 */
export function DateField({ label, value, onChange, format, placeholder = 'Pilih tanggal', minimumDate, maximumDate, helper, error }: DateFieldProps) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const { bottom } = useSafeAreaInsets();
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState<Date>(value ?? minimumDate ?? new Date());

  function openPicker() {
    const initial = value ?? minimumDate ?? new Date();
    if (Platform.OS === 'android') {
      DateTimePickerAndroid.open({
        value: initial,
        mode: 'date',
        minimumDate,
        maximumDate,
        onValueChange: (_event, date) => onChange(date),
      });
      return;
    }
    setDraft(initial);
    setOpen(true);
  }

  return (
    <>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${label}: ${value ? format(value) : 'belum dipilih'}`}
        accessibilityHint={error ?? 'Buka kalender'}
        onPress={openPicker}
      >
        <FieldFrame label={label} helper={helper} error={error} focused={open}>
          <Text style={[styles.value, !value && styles.placeholder]} numberOfLines={1}>
            {value ? format(value) : placeholder}
          </Text>
        </FieldFrame>
      </Pressable>

      {Platform.OS !== 'android' && (
        <Modal visible={open} transparent animationType="slide" onRequestClose={() => setOpen(false)}>
          <Pressable style={styles.scrim} onPress={() => setOpen(false)} accessibilityLabel="Tutup kalender" />
          <View style={[styles.sheet, { paddingBottom: bottom + 16 }]} accessibilityViewIsModal>
            <Text accessibilityRole="header" style={styles.sheetTitle}>
              {label}
            </Text>
            <DateTimePicker
              value={draft}
              mode="date"
              display="inline"
              locale="id-ID"
              accentColor={color.text.primary}
              minimumDate={minimumDate}
              maximumDate={maximumDate}
              onValueChange={(_event, date) => setDraft(date)}
            />
            <View style={styles.actions}>
              <View style={styles.action}>
                <Button variant="secondary" fullWidth label="Batal" onPress={() => setOpen(false)} />
              </View>
              <View style={styles.action}>
                <Button
                  fullWidth
                  label="Pilih"
                  onPress={() => {
                    onChange(draft);
                    setOpen(false);
                  }}
                />
              </View>
            </View>
          </View>
        </Modal>
      )}
    </>
  );
}

const makeStyles = ({ color, radius, space, text }: Theme) => ({
  value: { ...text.bodyM, flex: 1, color: color.text.primary },
  placeholder: { color: color.text.secondary },
  scrim: { flex: 1, backgroundColor: color.bg.scrim },
  sheet: {
    gap: space[12],
    paddingTop: space[20],
    paddingHorizontal: space[20],
    borderTopLeftRadius: radius.lg,
    borderTopRightRadius: radius.lg,
    backgroundColor: color.bg.surface,
  },
  sheetTitle: { ...text.headingH3, color: color.text.primary },
  actions: { flexDirection: 'row' as const, gap: space[12] },
  action: { flex: 1 },
});
