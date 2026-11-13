import { Platform } from 'react-native';

/** Di emulator Android, "localhost" adalah emulator itu sendiri; komputer host ada di 10.0.2.2. */
const defaultHost = Platform.OS === 'android' ? '10.0.2.2' : 'localhost';

export const API_BASE_URL = (
  process.env.EXPO_PUBLIC_API_URL || `http://${defaultHost}:8080`
).replace(/\/$/, '');

export const API_TIMEOUT_MS = 10_000;
