import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { login, type LoginRequest } from '../api/auth';
import { ApiError, setAccessToken, setUnauthorizedHandler } from '../api/client';
import { runSignOutTasks } from './signOutTasks';
import { clearSession, loadSession, saveSession, type StoredSession } from './tokenStorage';

type Status = 'loading' | 'signedOut' | 'signedIn';

type AuthContextValue = {
  status: Status;
  signIn: (credentials: LoginRequest) => Promise<void>;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<Status>('loading');
  const expiryTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const signingOut = useRef(false);

  const signOut = useCallback(async () => {
    // Tugas logout bisa kena 401 yang memanggil signOut lagi; cukup jalan sekali.
    if (signingOut.current) return;
    signingOut.current = true;
    if (expiryTimer.current) clearTimeout(expiryTimer.current);
    try {
      // Selagi token masih ada, misalnya untuk menghapus token push perangkat di server.
      await runSignOutTasks();
    } finally {
      setAccessToken(null);
      await clearSession().catch(() => undefined);
      setStatus('signedOut');
      signingOut.current = false;
    }
  }, []);

  const activate = useCallback(
    (session: StoredSession) => {
      setAccessToken(session.accessToken);
      if (expiryTimer.current) clearTimeout(expiryTimer.current);
      // Belum ada refresh token: saat token habis, kembali ke layar login.
      expiryTimer.current = setTimeout(() => void signOut(), Math.max(0, session.expiresAt - Date.now()));
      setStatus('signedIn');
    },
    [signOut],
  );

  useEffect(() => {
    setUnauthorizedHandler(() => void signOut());
    return () => setUnauthorizedHandler(null);
  }, [signOut]);

  useEffect(() => {
    loadSession()
      .then((session) => (session ? activate(session) : setStatus('signedOut')))
      .catch(() => setStatus('signedOut'));
    return () => {
      if (expiryTimer.current) clearTimeout(expiryTimer.current);
    };
  }, [activate]);

  const signIn = useCallback(
    async (credentials: LoginRequest) => {
      const response = await login(credentials);
      if (!response?.access_token || typeof response.expires_in !== 'number') {
        throw new ApiError('Respons login dari server tidak lengkap.', 502);
      }
      const session = { accessToken: response.access_token, expiresAt: Date.now() + response.expires_in * 1000 };
      await saveSession(session);
      activate(session);
    },
    [activate],
  );

  const value = useMemo(() => ({ status, signIn, signOut }), [status, signIn, signOut]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth harus dipakai di dalam <AuthProvider>.');
  return context;
}
