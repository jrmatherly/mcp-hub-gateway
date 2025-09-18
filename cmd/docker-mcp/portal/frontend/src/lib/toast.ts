// Simple toast utility to match the usage in our hooks
// This provides a compatible interface with the expected toast API

interface ToastOptions {
  description?: string;
  duration?: number;
}

class ToastManager {
  private toasts: Array<{
    id: string;
    type: 'success' | 'error' | 'info' | 'warning';
    title: string;
    description?: string;
  }> = [];

  private createToast(
    type: 'success' | 'error' | 'info' | 'warning',
    title: string,
    options?: ToastOptions
  ) {
    const id = Math.random().toString(36).substring(2, 9);
    const toast = {
      id,
      type,
      title,
      description: options?.description,
    };

    this.toasts.push(toast);

    // In a real implementation, this would show a toast notification
    // For now, we'll just log to console
    console.warn(
      `[TOAST ${type.toUpperCase()}] ${title}`,
      options?.description || ''
    );

    // Auto-remove after duration
    setTimeout(() => {
      this.toasts = this.toasts.filter(t => t.id !== id);
    }, options?.duration || 5000);

    return toast;
  }

  success(title: string, options?: ToastOptions) {
    return this.createToast('success', title, options);
  }

  error(title: string, options?: ToastOptions) {
    return this.createToast('error', title, options);
  }

  info(title: string, options?: ToastOptions) {
    return this.createToast('info', title, options);
  }

  warning(title: string, options?: ToastOptions) {
    return this.createToast('warning', title, options);
  }
}

export const toast = new ToastManager();
