import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { ApiError } from '../api/client';
import { getProfile, type Profile } from '../api/me';

type ProfileState = {
  profile: Profile | null;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
};

const ProfileContext = createContext<ProfileState | null>(null);

/** Memuat profil pengguna sekali setelah login dan membagikannya ke Beranda dan Profil Saya. */
export function ProfileProvider({ children }: { children: ReactNode }) {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      setProfile(await getProfile());
    } catch (cause) {
      // Jangan tampilkan pesan internal server (5xx) ke pengguna.
      const serverError = cause instanceof ApiError && cause.status >= 500;
      setError(
        cause instanceof ApiError && !serverError
          ? cause.message
          : 'Server sedang bermasalah. Tarik layar ke bawah untuk mencoba lagi.',
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const value = useMemo(() => ({ profile, loading, error, reload }), [profile, loading, error, reload]);
  return <ProfileContext.Provider value={value}>{children}</ProfileContext.Provider>;
}

export function useProfile() {
  const context = useContext(ProfileContext);
  if (!context) throw new Error('useProfile harus dipakai di dalam <ProfileProvider>.');
  return context;
}
