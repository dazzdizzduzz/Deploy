import { forwardRef, useState, type ComponentRef, type ReactNode } from 'react';
import { Pressable, Text, TextInput, View, type TextInputProps } from 'react-native';
import { useStyles, useTheme, type Theme } from '../../theme/ThemeProvider';

type FieldProps = Omit<TextInputProps, 'style' | 'secureTextEntry'> & {
  label: string;
  /** Teks bantuan. Teks kosong (" ") hanya menjaga ruang baris, sesuai Figma. */
  helper?: string;
  /** Pesan galat; menggantikan helper dan menandai field tidak valid. */
  error?: string;
};

type InputRef = ComponentRef<typeof TextInput>;

type ShellProps = FieldProps & { trailing?: ReactNode; secureTextEntry?: boolean };

/** Bingkai field: label, kontrol, dan pesan bantuan/galat. Dipakai juga oleh Select. */
export function FieldFrame({
  label,
  helper,
  error,
  focused = false,
  children,
}: {
  label: string;
  helper?: string;
  error?: string;
  focused?: boolean;
  children: ReactNode;
}) {
  const styles = useStyles(makeStyles);
  const { color, space } = useTheme();
  const message = error ?? helper;
  const borderWidth = error || focused ? 2 : 1;
  const borderColor = error ? color.status.dangerFg : focused ? color.border.focus : color.border.control;

  return (
    <View style={styles.field}>
      <Text style={styles.label}>{label}</Text>
      <View
        style={[
          styles.control,
          {
            borderWidth,
            borderColor,
            // Tinggi field tetap saat border menebal, supaya layar tidak bergeser.
            paddingHorizontal: space[16] - (borderWidth - 1),
            paddingVertical: space[12] - (borderWidth - 1),
          },
        ]}
      >
        {children}
      </View>
      {message !== undefined && (
        <Text
          style={[styles.message, error ? styles.messageError : null]}
          accessibilityLiveRegion={error ? 'polite' : 'none'}
        >
          {message}
        </Text>
      )}
    </View>
  );
}

const FieldShell = forwardRef<InputRef, ShellProps>(function FieldShell(
  { label, helper, error, trailing, secureTextEntry, ...inputProps },
  ref,
) {
  const styles = useStyles(makeStyles);
  const { color } = useTheme();
  const [focused, setFocused] = useState(false);
  const message = error ?? helper;

  return (
    <FieldFrame label={label} helper={helper} error={error} focused={focused}>
      <TextInput
        ref={ref}
        {...inputProps}
        style={styles.input}
        accessibilityLabel={label}
        accessibilityHint={message?.trim() ? message : undefined}
        secureTextEntry={secureTextEntry}
        selectionColor={color.text.primary}
        placeholderTextColor={color.text.secondary}
        onFocus={(event) => {
          setFocused(true);
          inputProps.onFocus?.(event);
        }}
        onBlur={(event) => {
          setFocused(false);
          inputProps.onBlur?.(event);
        }}
      />
      {trailing}
    </FieldFrame>
  );
});

export const TextField = forwardRef<InputRef, FieldProps>(function TextField(props, ref) {
  return <FieldShell ref={ref} {...props} />;
});

export const PasswordField = forwardRef<InputRef, FieldProps>(function PasswordField(props, ref) {
  const styles = useStyles(makeStyles);
  const [visible, setVisible] = useState(false);

  return (
    <FieldShell
      ref={ref}
      autoCapitalize="none"
      autoCorrect={false}
      {...props}
      secureTextEntry={!visible}
      trailing={
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={visible ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
          onPress={() => setVisible((value) => !value)}
          style={styles.toggle}
        >
          <Text style={styles.toggleLabel}>{visible ? 'Sembunyikan' : 'Tampilkan'}</Text>
        </Pressable>
      }
    />
  );
});

const makeStyles = ({ color, radius, size, space, text }: Theme) => ({
  field: { alignSelf: 'stretch' as const, gap: space[8] },
  label: { ...text.labelM, color: color.text.primary },
  control: {
    flexDirection: 'row' as const,
    alignItems: 'center' as const,
    gap: space[8],
    minHeight: 48,
    borderRadius: radius.md,
    backgroundColor: color.bg.surface,
  },
  input: {
    flex: 1,
    minWidth: 0,
    minHeight: 24,
    padding: 0,
    color: color.text.primary,
    fontFamily: text.bodyM.fontFamily,
    fontSize: text.bodyM.fontSize,
  },
  // Area sentuh minimal 44px tanpa menambah tinggi field.
  toggle: {
    alignItems: 'center' as const,
    justifyContent: 'center' as const,
    minWidth: size.iconTarget,
    height: size.iconTarget,
    marginVertical: -space[12],
  },
  toggleLabel: { ...text.labelM, color: color.text.primary },
  message: { ...text.caption, minHeight: text.caption.lineHeight, color: color.text.secondary },
  messageError: { color: color.status.dangerFg },
});
