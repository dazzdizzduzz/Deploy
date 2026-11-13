import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { useColorScheme } from 'react-native';
import { fontFamily, palettes, radius, size, space, text, type ColorScheme, type Palette } from './tokens';

export type Theme = {
  scheme: ColorScheme;
  color: Palette;
  space: typeof space;
  radius: typeof radius;
  size: typeof size;
  text: typeof text;
  fontFamily: typeof fontFamily;
};

const themeFor = (scheme: ColorScheme): Theme => ({ scheme, color: palettes[scheme], space, radius, size, text, fontFamily });

const ThemeContext = createContext<Theme>(themeFor('light'));

/**
 * Menyediakan token sesuai skema warna. Tanpa `scheme`, mengikuti sistem; app.json saat ini
 * mengunci `userInterfaceStyle: "light"`, jadi aplikasi tetap terang sampai mode gelap diaktifkan.
 */
export function ThemeProvider({ scheme, children }: { scheme?: ColorScheme; children: ReactNode }) {
  const system = useColorScheme();
  const active: ColorScheme = scheme ?? (system === 'dark' ? 'dark' : 'light');
  const theme = useMemo(() => themeFor(active), [active]);
  return <ThemeContext.Provider value={theme}>{children}</ThemeContext.Provider>;
}

export const useTheme = () => useContext(ThemeContext);

/**
 * Membuat style dari token tema aktif. `factory` harus didefinisikan di luar komponen
 * agar style hanya dibuat ulang saat tema berganti.
 */
export function useStyles<T>(factory: (theme: Theme) => T): T {
  const theme = useTheme();
  return useMemo(() => factory(theme), [factory, theme]);
}
