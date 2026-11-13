/** Syarat kata sandi baru (Figma W21). Backend belum memvalidasi; ini hanya panduan di form. */
export function checkPassword(password: string) {
  return {
    minLength: password.length >= 8,
    lettersAndDigits: /\p{L}/u.test(password) && /\d/.test(password),
    uppercase: /\p{Lu}/u.test(password),
  };
}

/** Waktu tunggu sebelum tautan reset boleh diminta ulang. */
export const RESEND_RESET_SECONDS = 45;

/** Masa berlaku token reset di backend (menit). */
export const RESET_TOKEN_MINUTES = 15;

export function formatCountdown(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

/** "daze@studiotigapagi.com" -> "d***@studiotigapagi.com" */
export function maskEmail(email: string): string {
  const at = email.indexOf('@');
  if (at < 1) return email;
  return `${email.charAt(0)}***${email.slice(at)}`;
}
