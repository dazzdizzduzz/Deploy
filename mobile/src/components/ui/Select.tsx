import { useState } from 'react';
import { FlatList, Modal, Pressable, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useStyles, type Theme } from '../../theme/ThemeProvider';
import { Button } from './Button';
import { FieldFrame } from './Field';

export type SelectOption = { value: string; label: string };

type SelectProps = {
  label: string;
  options: SelectOption[];
  value: string | null;
  onChange: (value: string) => void;
  placeholder?: string;
  helper?: string;
  error?: string;
  disabled?: boolean;
};

/** Pilihan tunggal: field yang membuka daftar opsi dari bawah layar. */
export function Select({ label, options, value, onChange, placeholder = 'Pilih', helper, error, disabled }: SelectProps) {
  const styles = useStyles(makeStyles);
  const { bottom } = useSafeAreaInsets();
  const [open, setOpen] = useState(false);
  const selected = options.find((option) => option.value === value);

  return (
    <>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel={`${label}: ${selected?.label ?? 'belum dipilih'}`}
        accessibilityHint="Buka daftar pilihan"
        accessibilityState={{ disabled, expanded: open }}
        disabled={disabled}
        onPress={() => setOpen(true)}
      >
        <FieldFrame label={label} helper={helper} error={error} focused={open}>
          <Text style={[styles.value, !selected && styles.placeholder, disabled && styles.disabled]} numberOfLines={1}>
            {selected?.label ?? placeholder}
          </Text>
          <Text style={styles.chevron} importantForAccessibility="no">
            ▾
          </Text>
        </FieldFrame>
      </Pressable>

      <Modal visible={open} transparent animationType="slide" onRequestClose={() => setOpen(false)}>
        <Pressable style={styles.scrim} onPress={() => setOpen(false)} accessibilityLabel="Tutup daftar pilihan" />
        <View style={[styles.sheet, { paddingBottom: bottom + 16 }]} accessibilityViewIsModal>
          <Text accessibilityRole="header" style={styles.sheetTitle}>
            {label}
          </Text>
          <FlatList
            data={options}
            keyExtractor={(option) => option.value}
            renderItem={({ item }) => {
              const active = item.value === value;
              return (
                <Pressable
                  accessibilityRole="radio"
                  accessibilityState={{ checked: active }}
                  onPress={() => {
                    onChange(item.value);
                    setOpen(false);
                  }}
                  style={({ pressed }) => [styles.option, pressed && styles.optionPressed]}
                >
                  <Text style={[styles.optionLabel, active && styles.optionActive]}>{item.label}</Text>
                  {active && <Text style={styles.check}>✓</Text>}
                </Pressable>
              );
            }}
          />
          <Button variant="secondary" fullWidth label="Batal" onPress={() => setOpen(false)} />
        </View>
      </Modal>
    </>
  );
}

const makeStyles = ({ color, radius, size, space, text }: Theme) => ({
  value: { ...text.bodyM, flex: 1, color: color.text.primary },
  placeholder: { color: color.text.secondary },
  disabled: { color: color.text.disabled },
  chevron: { ...text.bodyM, color: color.text.secondary },
  scrim: { flex: 1, backgroundColor: color.bg.scrim },
  sheet: {
    maxHeight: '70%' as const,
    gap: space[12],
    paddingTop: space[20],
    paddingHorizontal: space[20],
    borderTopLeftRadius: radius.lg,
    borderTopRightRadius: radius.lg,
    backgroundColor: color.bg.surface,
  },
  sheetTitle: { ...text.headingH3, color: color.text.primary },
  option: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    justifyContent: 'space-between' as const,
    minHeight: size.iconTarget + 4,
    paddingHorizontal: space[12],
    borderRadius: radius.md,
  },
  optionPressed: { backgroundColor: color.bg.hover },
  optionLabel: { ...text.bodyM, color: color.text.primary },
  optionActive: { ...text.labelM, fontSize: text.bodyM.fontSize },
  check: { ...text.labelM, color: color.text.accent },
});
