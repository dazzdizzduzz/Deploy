import { createNavigationContainerRef } from '@react-navigation/native';
import type { RootStackParamList } from './types';

/** Navigasi dari luar layar, misalnya saat notifikasi push di-tap. */
export const navigationRef = createNavigationContainerRef<RootStackParamList>();
