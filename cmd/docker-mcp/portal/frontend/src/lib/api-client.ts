import axios, {
  type AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
} from 'axios';
import { toast } from 'sonner';
import { apiLogger } from '@/lib/logger';
import { authService } from '@/services/auth.service';
import type {
  ApiError,
  ApiResponse,
  BulkOperationResult,
  BulkServerOperation,
  Catalog,
  CatalogServer,
  GatewayStartRequest,
  GatewayStatus,
  MCPConfig,
  Server,
  ServerDetails,
} from '@/types/api';

// API client configuration
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const REQUEST_TIMEOUT = 30000; // 30 seconds
const MAX_RETRIES = 3;
const RETRY_DELAY = 1000; // 1 second

class ApiClient {
  private client: AxiosInstance;
  private retryCount = new Map<string, number>();

  constructor() {
    this.client = axios.create({
      baseURL: `${API_BASE_URL}/api`,
      timeout: REQUEST_TIMEOUT,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    this.setupInterceptors();
  }

  private setupInterceptors(): void {
    // Request interceptor for auth and logging
    this.client.interceptors.request.use(
      async config => {
        // Add auth token
        const token = await authService.getAccessToken();
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }

        // Add request ID for tracking
        const requestId = this.generateRequestId();
        config.headers['X-Request-ID'] = requestId;

        // Log request in development
        if (process.env.NODE_ENV === 'development') {
          apiLogger.debug('API Request', {
            method: config.method?.toUpperCase(),
            url: config.url,
            requestId,
            data: config.data,
            params: config.params,
          });
        }

        return config;
      },
      error => {
        apiLogger.error('Request interceptor error', error);
        return Promise.reject(error);
      }
    );

    // Response interceptor for error handling and retries
    this.client.interceptors.response.use(
      (response: AxiosResponse<ApiResponse>) => {
        // Log successful response in development
        if (process.env.NODE_ENV === 'development') {
          apiLogger.debug('API Response', {
            status: response.status,
            requestId: response.config.headers['X-Request-ID'],
            data: response.data,
          });
        }

        // Clear retry count on success
        const requestKey = this.getRequestKey(response.config);
        this.retryCount.delete(requestKey);

        return response;
      },
      async (error: AxiosError<ApiError>) => {
        const requestKey = this.getRequestKey(error.config);
        const retryCount = this.retryCount.get(requestKey) || 0;

        // Handle auth errors
        if (error.response?.status === 401) {
          apiLogger.warn('Authentication error, redirecting to login');
          await authService.signOut();
          window.location.href = '/auth/login';
          return Promise.reject(error);
        }

        // Handle retry logic for network errors and 5xx errors
        if (this.shouldRetry(error, retryCount)) {
          this.retryCount.set(requestKey, retryCount + 1);

          apiLogger.info(
            `Retrying request (${retryCount + 1}/${MAX_RETRIES})`,
            {
              url: error.config?.url,
              error: error.message,
            }
          );

          await this.delay(RETRY_DELAY * Math.pow(2, retryCount)); // Exponential backoff
          if (!error.config) {
            throw new Error('Request configuration is missing');
          }
          return this.client.request(error.config);
        }

        // Clear retry count and handle error
        this.retryCount.delete(requestKey);
        this.handleApiError(error);
        return Promise.reject(error);
      }
    );
  }

  private shouldRetry(error: AxiosError, retryCount: number): boolean {
    if (retryCount >= MAX_RETRIES) return false;

    // Retry on network errors
    if (!error.response) return true;

    // Retry on 5xx server errors
    if (error.response.status >= 500) return true;

    // Retry on rate limiting
    if (error.response.status === 429) return true;

    return false;
  }

  private handleApiError(error: AxiosError<ApiError>): void {
    const message =
      error.response?.data?.message ||
      error.message ||
      'An unexpected error occurred';
    const code = error.response?.data?.code || 'UNKNOWN_ERROR';

    console.error('[API] Error:', {
      code,
      message,
      status: error.response?.status,
      url: error.config?.url,
      details: error.response?.data?.details,
    });

    // Show user-friendly error messages
    if (error.response?.status !== 401) {
      // Don't show toast for auth errors
      toast.error(this.getErrorMessage(error.response?.status, message));
    }
  }

  private getErrorMessage(status?: number, message?: string): string {
    switch (status) {
      case 400:
        return `Invalid request: ${message}`;
      case 403:
        return 'You do not have permission to perform this action';
      case 404:
        return 'The requested resource was not found';
      case 409:
        return `Conflict: ${message}`;
      case 429:
        return 'Too many requests. Please try again later';
      case 500:
        return 'Server error. Please try again later';
      case 503:
        return 'Service temporarily unavailable';
      default:
        return message || 'An unexpected error occurred';
    }
  }

  private generateRequestId(): string {
    return Math.random().toString(36).substring(2, 15);
  }

  private getRequestKey(config: AxiosResponse['config'] | undefined): string {
    return `${config?.method || 'GET'}-${config?.url || 'unknown'}`;
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Generic HTTP methods for use by hooks
  async get<T = unknown>(
    url: string,
    config?: AxiosRequestConfig
  ): Promise<AxiosResponse<T>> {
    return this.client.get<T>(url, config);
  }

  async post<T = unknown>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<AxiosResponse<T>> {
    return this.client.post<T>(url, data, config);
  }

  async put<T = unknown>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<AxiosResponse<T>> {
    return this.client.put<T>(url, data, config);
  }

  async patch<T = unknown>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<AxiosResponse<T>> {
    return this.client.patch<T>(url, data, config);
  }

  async delete<T = unknown>(
    url: string,
    config?: AxiosRequestConfig
  ): Promise<AxiosResponse<T>> {
    return this.client.delete<T>(url, config);
  }

  // Server management methods
  async getServers(): Promise<Server[]> {
    const response = await this.client.get<ApiResponse<Server[]>>('/servers');
    return response.data.data || [];
  }

  async getServerDetails(name: string): Promise<ServerDetails> {
    const response = await this.client.get<ApiResponse<ServerDetails>>(
      `/servers/${encodeURIComponent(name)}/inspect`
    );
    if (!response.data.data) {
      throw new Error('Server not found');
    }
    return response.data.data;
  }

  async enableServer(name: string): Promise<void> {
    await this.client.post<ApiResponse>(
      `/servers/${encodeURIComponent(name)}/enable`
    );
  }

  async disableServer(name: string): Promise<void> {
    await this.client.post<ApiResponse>(
      `/servers/${encodeURIComponent(name)}/disable`
    );
  }

  async toggleServer(name: string, enabled: boolean): Promise<void> {
    if (enabled) {
      await this.enableServer(name);
    } else {
      await this.disableServer(name);
    }
  }

  async bulkServerOperation(
    operation: BulkServerOperation
  ): Promise<BulkOperationResult> {
    const response = await this.client.post<ApiResponse<BulkOperationResult>>(
      '/servers/bulk',
      operation
    );
    if (!response.data.data) {
      throw new Error('Bulk operation failed - no data returned');
    }
    return response.data.data;
  }

  // Gateway management methods
  async getGatewayStatus(): Promise<GatewayStatus> {
    const response =
      await this.client.get<ApiResponse<GatewayStatus>>('/gateway/status');
    return response.data.data || { running: false };
  }

  async startGateway(options?: GatewayStartRequest): Promise<void> {
    await this.client.post<ApiResponse>('/gateway/start', options);
  }

  async stopGateway(): Promise<void> {
    await this.client.post<ApiResponse>('/gateway/stop');
  }

  // Configuration management methods
  async getConfig(): Promise<MCPConfig> {
    const response = await this.client.get<ApiResponse<MCPConfig>>('/config');
    return response.data.data || {};
  }

  async updateConfig(config: Partial<MCPConfig>): Promise<void> {
    await this.client.put<ApiResponse>('/config', { config });
  }

  // Catalog methods
  async getCatalogs(): Promise<Catalog[]> {
    const response = await this.client.get<ApiResponse<Catalog[]>>('/catalogs');
    return response.data.data || [];
  }

  async getCatalogServers(catalogName?: string): Promise<CatalogServer[]> {
    const url = catalogName
      ? `/catalogs/${encodeURIComponent(catalogName)}/servers`
      : '/catalog/servers';
    const response = await this.client.get<ApiResponse<CatalogServer[]>>(url);
    return response.data.data || [];
  }

  async installCatalogServer(
    serverName: string,
    catalogName?: string
  ): Promise<void> {
    const data = catalogName ? { catalog: catalogName } : {};
    await this.client.post<ApiResponse>(
      `/catalog/servers/${encodeURIComponent(serverName)}/install`,
      data
    );
  }

  // Health check
  async healthCheck(): Promise<boolean> {
    try {
      const response = await this.client.get<ApiResponse>('/health', {
        timeout: 5000,
      });
      return response.data.success;
    } catch (error) {
      console.warn('[API] Health check failed:', error);
      return false;
    }
  }

  // Utility method to check if API is available
  async isApiAvailable(): Promise<boolean> {
    return this.healthCheck();
  }
}

// Create singleton instance
export const apiClient = new ApiClient();

// Export individual methods for easier usage
export const serverApi = {
  getServers: () => apiClient.getServers(),
  getServerDetails: (name: string) => apiClient.getServerDetails(name),
  enableServer: (name: string) => apiClient.enableServer(name),
  disableServer: (name: string) => apiClient.disableServer(name),
  toggleServer: (name: string, enabled: boolean) =>
    apiClient.toggleServer(name, enabled),
  bulkOperation: (operation: BulkServerOperation) =>
    apiClient.bulkServerOperation(operation),
};

export const gatewayApi = {
  getStatus: () => apiClient.getGatewayStatus(),
  start: (options?: GatewayStartRequest) => apiClient.startGateway(options),
  stop: () => apiClient.stopGateway(),
};

export const configApi = {
  get: () => apiClient.getConfig(),
  update: (config: Partial<MCPConfig>) => apiClient.updateConfig(config),
};

export const catalogApi = {
  getCatalogs: () => apiClient.getCatalogs(),
  getServers: (catalogName?: string) =>
    apiClient.getCatalogServers(catalogName),
  installServer: (serverName: string, catalogName?: string) =>
    apiClient.installCatalogServer(serverName, catalogName),
};

export default apiClient;
