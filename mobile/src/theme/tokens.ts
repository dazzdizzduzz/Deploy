/* Dihasilkan oleh design-tokens/build.mjs dari design-tokens/tokens.json (Figma Variables). Jangan ubah manual. */

export const palettes = {
  "light": {
    "bg": {
      "canvas": "#F5F5F4",
      "surface": "#FFFFFF",
      "subtle": "#FAFAF9",
      "hover": "#F5F5F4",
      "disabled": "#E7E7E5",
      "inverse": "#0A0A0A",
      "inverseHover": "#1F1F1F",
      "inverseOverlay": "rgba(255, 255, 255, 0.08)",
      "scrim": "rgba(10, 10, 10, 0.45)"
    },
    "text": {
      "primary": "#0A0A0A",
      "secondary": "#666666",
      "accent": "#3F5F00",
      "disabled": "#858583",
      "inverse": "#FFFFFF",
      "onAccent": "#0A0A0A"
    },
    "accent": {
      "default": "#C6F135",
      "hover": "#B2DC22",
      "subtle": "#F1FBD6"
    },
    "border": {
      "default": "#E7E7E5",
      "strong": "#D4D4D2",
      "control": "#858583",
      "focus": "#0A0A0A"
    },
    "status": {
      "successFg": "#3F5F00",
      "successBg": "#F1FBD6",
      "warningFg": "#8A5A00",
      "warningBg": "#FFF4DB",
      "dangerFg": "#B42318",
      "dangerBg": "#FEECEB",
      "dangerHover": "#912018"
    },
    "badge": {
      "successBorder": "#2F4700",
      "warningBorder": "#6B4500",
      "dangerBorder": "#7A1A13",
      "neutralBorder": "#1F1F1F",
      "liveBorder": "#0A0A0A"
    }
  },
  "dark": {
    "bg": {
      "canvas": "#0A0A0A",
      "surface": "#141414",
      "subtle": "#1F1F1F",
      "hover": "#1F1F1F",
      "disabled": "#3A3A3A",
      "inverse": "#FFFFFF",
      "inverseHover": "#F5F5F4",
      "inverseOverlay": "rgba(10, 10, 10, 0.08)",
      "scrim": "rgba(0, 0, 0, 0.6)"
    },
    "text": {
      "primary": "#FFFFFF",
      "secondary": "#A3A3A1",
      "accent": "#C6F135",
      "disabled": "#666666",
      "inverse": "#0A0A0A",
      "onAccent": "#0A0A0A"
    },
    "accent": {
      "default": "#C6F135",
      "hover": "#B2DC22",
      "subtle": "#1E2A00"
    },
    "border": {
      "default": "#3A3A3A",
      "strong": "#7A7A78",
      "control": "#7A7A78",
      "focus": "#FFFFFF"
    },
    "status": {
      "successFg": "#C6F135",
      "successBg": "#1E2A00",
      "warningFg": "#F5C46B",
      "warningBg": "#2E2000",
      "dangerFg": "#FF8A80",
      "dangerBg": "#3A0D0A",
      "dangerHover": "#FFA59D"
    },
    "badge": {
      "successBorder": "#C6F135",
      "warningBorder": "#F5C46B",
      "dangerBorder": "#FF8A80",
      "neutralBorder": "#A3A3A1",
      "liveBorder": "#FFFFFF"
    }
  }
} as const;

export type ColorScheme = keyof typeof palettes;
export type Palette = (typeof palettes)[ColorScheme];

export const space = {
  "2": 2,
  "4": 4,
  "8": 8,
  "12": 12,
  "16": 16,
  "20": 20,
  "24": 24,
  "32": 32,
  "40": 40,
  "48": 48,
  "64": 64
} as const;

export const radius = {
  "sm": 8,
  "md": 12,
  "lg": 20,
  "full": 999
} as const;

export const size = {
  "buttonL": 52,
  "iconTarget": 44
} as const;

/** Font Inter yang dimuat saat aplikasi dibuka (lihat App.tsx). */
export const fontFamily = {
  "regular": "Inter_400Regular",
  "medium": "Inter_500Medium",
  "semiBold": "Inter_600SemiBold",
  "bold": "Inter_700Bold"
} as const;

/** Tipografi Figma (Inter). */
export const text = {
  "numericClock": {
    "fontFamily": "Inter_600SemiBold",
    "fontSize": 48,
    "lineHeight": 52.8,
    "letterSpacing": -1.44
  },
  "displayL": {
    "fontFamily": "Inter_700Bold",
    "fontSize": 40,
    "lineHeight": 44,
    "letterSpacing": -0.8
  },
  "numericKpi": {
    "fontFamily": "Inter_600SemiBold",
    "fontSize": 32,
    "lineHeight": 35.2,
    "letterSpacing": -0.64
  },
  "headingH1": {
    "fontFamily": "Inter_700Bold",
    "fontSize": 28,
    "lineHeight": 33.6,
    "letterSpacing": -0.42
  },
  "headingH2": {
    "fontFamily": "Inter_600SemiBold",
    "fontSize": 20,
    "lineHeight": 26,
    "letterSpacing": -0.2
  },
  "headingH3": {
    "fontFamily": "Inter_600SemiBold",
    "fontSize": 16,
    "lineHeight": 22.4,
    "letterSpacing": -0.08
  },
  "bodyM": {
    "fontFamily": "Inter_400Regular",
    "fontSize": 16,
    "lineHeight": 24,
    "letterSpacing": 0
  },
  "bodyS": {
    "fontFamily": "Inter_400Regular",
    "fontSize": 14,
    "lineHeight": 21,
    "letterSpacing": 0
  },
  "labelM": {
    "fontFamily": "Inter_500Medium",
    "fontSize": 14,
    "lineHeight": 19.6,
    "letterSpacing": 0
  },
  "labelS": {
    "fontFamily": "Inter_500Medium",
    "fontSize": 12,
    "lineHeight": 16.8,
    "letterSpacing": 0.24
  },
  "caption": {
    "fontFamily": "Inter_400Regular",
    "fontSize": 12,
    "lineHeight": 16.8,
    "letterSpacing": 0
  }
} as const;
