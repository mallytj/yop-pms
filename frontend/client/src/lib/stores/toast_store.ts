import { writable } from 'svelte/store';

export type ToastType = 'success' | 'error' | 'info';

export interface Toast {
  id: number;
  message: string;
  type: ToastType;
}

export const toasts = writable<Toast[]>([]);

export const addToast = (message: string, type: ToastType = 'info', duration = 3000) => {
  const id = Date.now();

  // Add the toast
  toasts.update((all) => [...all, { id, message, type }]);

  // Automatically remove it after duration
  if (duration) {
    setTimeout(() => {
      dismissToast(id);
    }, duration);
  }
};

export const dismissToast = (id: number) => {
  toasts.update((all) => all.filter((t) => t.id !== id));
};