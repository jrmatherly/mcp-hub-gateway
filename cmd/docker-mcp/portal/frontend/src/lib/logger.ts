/**
 * Frontend Logger Utility
 *
 * A centralized logging system for the MCP Portal frontend that:
 * - Respects environment settings (dev vs production)
 * - Provides structured logging with levels
 * - Supports context and metadata
 * - Can be extended for remote logging services
 */

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
  NONE = 4,
}

interface LogContext {
  module?: string;
  userId?: string;
  sessionId?: string;
  [key: string]: unknown;
}

interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  context?: LogContext;
  data?: unknown;
  error?: Error;
}

class Logger {
  private level: LogLevel;
  private context: LogContext;
  private isDevelopment: boolean;
  private logBuffer: LogEntry[] = [];
  private maxBufferSize = 100;

  constructor(context: LogContext = {}) {
    this.context = context;
    this.isDevelopment = process.env.NODE_ENV === 'development';

    // Set log level based on environment
    if (this.isDevelopment) {
      this.level = LogLevel.DEBUG;
    } else {
      // In production, only log warnings and errors
      this.level = LogLevel.WARN;
    }

    // Override with environment variable if set
    const envLogLevel = process.env.NEXT_PUBLIC_LOG_LEVEL?.toUpperCase();
    if (
      envLogLevel &&
      LogLevel[envLogLevel as keyof typeof LogLevel] !== undefined
    ) {
      this.level = LogLevel[envLogLevel as keyof typeof LogLevel];
    }
  }

  /**
   * Create a child logger with additional context
   */
  child(additionalContext: LogContext): Logger {
    const childLogger = new Logger({
      ...this.context,
      ...additionalContext,
    });
    childLogger.level = this.level;
    return childLogger;
  }

  /**
   * Set the minimum log level
   */
  setLevel(level: LogLevel): void {
    this.level = level;
  }

  /**
   * Add context that will be included with all logs
   */
  setContext(context: LogContext): void {
    this.context = { ...this.context, ...context };
  }

  /**
   * Debug level logging - detailed information for debugging
   */
  debug(message: string, data?: unknown): void {
    this.log(LogLevel.DEBUG, message, data);
  }

  /**
   * Info level logging - general informational messages
   */
  info(message: string, data?: unknown): void {
    this.log(LogLevel.INFO, message, data);
  }

  /**
   * Warning level logging - potentially harmful situations
   */
  warn(message: string, data?: unknown): void {
    this.log(LogLevel.WARN, message, data);
  }

  /**
   * Error level logging - error events
   */
  error(message: string, error?: Error | unknown, data?: unknown): void {
    if (error instanceof Error) {
      this.log(LogLevel.ERROR, message, data, error);
    } else if (error && typeof error === 'object') {
      // Safely spread object data
      const spreadData =
        data && typeof data === 'object' && data !== null ? { ...data } : {};
      this.log(LogLevel.ERROR, message, { error, ...spreadData });
    } else {
      // Handle primitive error types
      this.log(LogLevel.ERROR, message, { error, data });
    }
  }

  /**
   * Log with performance timing
   */
  time(label: string): () => void {
    const start = performance.now();
    return () => {
      const duration = performance.now() - start;
      this.debug(`${label} took ${duration.toFixed(2)}ms`, { duration });
    };
  }

  /**
   * Log grouped messages (development only)
   */
  group(label: string, fn: () => void): void {
    if (this.isDevelopment && this.level <= LogLevel.DEBUG) {
      // eslint-disable-next-line no-console
      console.group(label);
      fn();
      // eslint-disable-next-line no-console
      console.groupEnd();
    } else {
      fn();
    }
  }

  /**
   * Core logging method
   */
  private log(
    level: LogLevel,
    message: string,
    data?: unknown,
    error?: Error
  ): void {
    // Skip if below minimum level
    if (level < this.level) return;

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      message,
      context: this.context,
      data,
      error,
    };

    // Buffer the log entry
    this.bufferLog(entry);

    // Output to console based on level
    const levelName = LogLevel[level];
    const prefix = `[${levelName}]`;
    const formattedMessage = this.formatMessage(prefix, message, this.context);

    switch (level) {
      case LogLevel.DEBUG:
        if (this.isDevelopment) {
          // Debug statements allowed in development environment
          // eslint-disable-next-line no-console
          console.debug(formattedMessage, data || '');
        }
        break;
      case LogLevel.INFO:
        if (this.isDevelopment) {
          // Info statements allowed in development environment
          // eslint-disable-next-line no-console
          console.info(formattedMessage, data || '');
        }
        break;
      case LogLevel.WARN:
        console.warn(formattedMessage, data || '');
        break;
      case LogLevel.ERROR:
        console.error(formattedMessage, error || data || '');
        if (error?.stack && this.isDevelopment) {
          console.error(error.stack);
        }
        break;
    }

    // Send to remote logging service in production
    if (!this.isDevelopment && level >= LogLevel.WARN) {
      this.sendToRemote(entry);
    }
  }

  /**
   * Format log message with context
   */
  private formatMessage(
    prefix: string,
    message: string,
    context?: LogContext
  ): string {
    const contextStr = context?.module ? `[${context.module}]` : '';
    return `${prefix}${contextStr} ${message}`;
  }

  /**
   * Buffer logs for potential batch sending
   */
  private bufferLog(entry: LogEntry): void {
    this.logBuffer.push(entry);
    if (this.logBuffer.length > this.maxBufferSize) {
      this.logBuffer.shift();
    }
  }

  /**
   * Send logs to remote service (placeholder for future implementation)
   */
  private sendToRemote(entry: LogEntry): void {
    // TODO: Implement remote logging service integration
    // This could be Sentry, LogRocket, DataDog, etc.

    // For now, we'll just track critical errors in localStorage for debugging
    if (entry.level === LogLevel.ERROR) {
      try {
        const errors = JSON.parse(
          localStorage.getItem('mcp_portal_errors') || '[]'
        ) as LogEntry[];

        // Keep only last 10 errors
        if (errors.length >= 10) {
          errors.shift();
        }

        errors.push({
          ...entry,
          // Don't store the full error object to avoid circular references
          error: entry.error
            ? ({
                message: entry.error.message || '',
                stack: entry.error.stack,
                name: entry.error.name || 'Error',
              } as Error)
            : undefined,
        });

        localStorage.setItem('mcp_portal_errors', JSON.stringify(errors));
      } catch {
        // Silently fail if localStorage is not available
      }
    }
  }

  /**
   * Get buffered logs (useful for debugging)
   */
  getBuffer(): LogEntry[] {
    return [...this.logBuffer];
  }

  /**
   * Clear log buffer
   */
  clearBuffer(): void {
    this.logBuffer = [];
  }

  /**
   * Export logs (for debugging or support)
   */
  exportLogs(): string {
    return JSON.stringify(this.logBuffer, null, 2);
  }
}

// Create default logger instance
const defaultLogger = new Logger({ module: 'app' });

// Export singleton instance and Logger class
export default defaultLogger;
export { Logger };

// Convenience functions for quick logging
export const logger = {
  debug: (message: string, data?: unknown) =>
    defaultLogger.debug(message, data),
  info: (message: string, data?: unknown) => defaultLogger.info(message, data),
  warn: (message: string, data?: unknown) => defaultLogger.warn(message, data),
  error: (message: string, error?: Error | unknown, data?: unknown) =>
    defaultLogger.error(message, error, data),
  time: (label: string) => defaultLogger.time(label),
  group: (label: string, fn: () => void) => defaultLogger.group(label, fn),
  child: (context: LogContext) => defaultLogger.child(context),
};

// Specialized loggers for different modules
export const authLogger = defaultLogger.child({ module: 'auth' });
export const apiLogger = defaultLogger.child({ module: 'api' });
export const uiLogger = defaultLogger.child({ module: 'ui' });
export const wsLogger = defaultLogger.child({ module: 'websocket' });

// Performance logger
export const performanceLogger = defaultLogger.child({ module: 'performance' });

// Development-only logger (completely disabled in production)
export const devLogger = {
  debug: (message: string, data?: unknown) => {
    if (process.env.NODE_ENV === 'development') {
      defaultLogger.debug(message, data);
    }
  },
  info: (message: string, data?: unknown) => {
    if (process.env.NODE_ENV === 'development') {
      defaultLogger.info(message, data);
    }
  },
  warn: (message: string, data?: unknown) => {
    if (process.env.NODE_ENV === 'development') {
      defaultLogger.warn(message, data);
    }
  },
  error: (message: string, error?: Error | unknown, data?: unknown) => {
    if (process.env.NODE_ENV === 'development') {
      defaultLogger.error(message, error, data);
    }
  },
};

// Browser console wrapper for migration
export const browserConsole = {
  log: (...args: unknown[]) => {
    if (process.env.NODE_ENV === 'development') {
      // Debug logging allowed in development
      // eslint-disable-next-line no-console
      console.debug(...args);
    }
  },
  warn: (...args: unknown[]) => console.warn(...args),
  error: (...args: unknown[]) => console.error(...args),
  debug: (...args: unknown[]) => {
    if (process.env.NODE_ENV === 'development') {
      // Debug logging allowed in development
      // eslint-disable-next-line no-console
      console.debug(...args);
    }
  },
};

// Type guard for errors
export function isError(error: unknown): error is Error {
  return error instanceof Error;
}

// Format error for logging
export function formatError(error: unknown): {
  message: string;
  stack?: string;
} {
  if (isError(error)) {
    return {
      message: error.message,
      stack: error.stack,
    };
  }

  if (typeof error === 'string') {
    return { message: error };
  }

  if (typeof error === 'object' && error !== null) {
    return {
      message: JSON.stringify(error),
    };
  }

  return { message: String(error) };
}
