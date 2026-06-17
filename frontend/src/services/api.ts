import axios, { AxiosInstance, AxiosError } from 'axios';
import {
  LoginRequest,
  AuthToken,
  Ticket,
  TicketComment,
  TicketUpdate,
  Customer,
  Engineer,
  PaginatedResponse,
  ApiError,
  User,
} from '@/types';
import {
  EmailSettings,
  EmailLog,
  DomainMapping,
  AutoReplyTemplate,
  EmailHistoryFilter,
  EmailHistoryResponse,
} from '@/types/email';
import {
  WhatsAppSession,
  EngineerWAPhone,
  WAHookStatus,
} from '@/types/whatsapp';
import {
  Report,
  ReportData,
  ReportFilter,
} from '@/types/reports';
import { DashboardSummary } from '@/types/dashboard';

class ApiService {
  private api: AxiosInstance;
  private baseURL: string;

  constructor() {
    this.baseURL = import.meta.env.VITE_API_BASE_URL || '/api';

    this.api = axios.create({
      baseURL: this.baseURL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor to add auth token
    this.api.interceptors.request.use(
      (config) => {
        const token = this.getAuthToken();
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => Promise.reject(error)
    );

    // Response interceptor for error handling
    this.api.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        if (error.response?.status === 401) {
          this.clearAuthToken();
          window.location.href = '/login';
        }
        return Promise.reject(this.handleError(error));
      }
    );
  }

  private handleError(error: AxiosError): ApiError {
    if (error.response?.data) {
      const data = error.response.data as Record<string, unknown>;
      return {
        code: data.code as string || error.code || 'UNKNOWN_ERROR',
        message: data.message as string || error.message || 'An error occurred',
        details: data.details as Record<string, unknown>,
      };
    }
    return {
      code: error.code || 'NETWORK_ERROR',
      message: error.message || 'Network error occurred',
    };
  }

  private setAuthToken(token: string): void {
    localStorage.setItem('auth_token', token);
  }

  private getAuthToken(): string | null {
    return localStorage.getItem('auth_token');
  }

  private clearAuthToken(): void {
    localStorage.removeItem('auth_token');
  }

  // Auth endpoints
  async login(credentials: LoginRequest): Promise<AuthToken> {
    const response = await this.api.post<AuthToken>('/auth/login', credentials);
    this.setAuthToken(response.data.token);
    return response.data;
  }

  async logout(): Promise<void> {
    try {
      await this.api.post('/auth/logout');
    } finally {
      this.clearAuthToken();
    }
  }

  async getCurrentUser(): Promise<User> {
    const response = await this.api.get<User>('/auth/me');
    return response.data;
  }

  // Ticket endpoints
  async getTickets(
    page: number = 1,
    pageSize: number = 20,
    status?: string,
    priority?: string,
    customerId?: string,
    search?: string
  ): Promise<PaginatedResponse<Ticket>> {
    const params = new URLSearchParams();
    params.append('page', page.toString());
    params.append('pageSize', pageSize.toString());
    if (status) params.append('status', status);
    if (priority) params.append('priority', priority);
    if (customerId) params.append('customerId', customerId);
    if (search) params.append('search', search);

    const response = await this.api.get<PaginatedResponse<Ticket>>(
      `/tickets?${params.toString()}`
    );
    return response.data;
  }

  // Dashboard endpoint
  async getDashboardSummary(): Promise<DashboardSummary> {
    const response = await this.api.get<DashboardSummary>('/dashboard/summary');
    return response.data;
  }

  async getTicket(id: string): Promise<Ticket> {
    const response = await this.api.get<Ticket>(`/tickets/${id}`);
    return response.data;
  }

  async createTicket(data: Partial<Ticket>): Promise<Ticket> {
    const response = await this.api.post<Ticket>('/tickets', data);
    return response.data;
  }

  async updateTicket(id: string, data: Partial<Ticket>): Promise<Ticket> {
    const response = await this.api.put<Ticket>(`/tickets/${id}`, data);
    return response.data;
  }

  async deleteTicket(id: string): Promise<void> {
    await this.api.delete(`/tickets/${id}`);
  }

  async getTicketUpdates(ticketId: string): Promise<TicketUpdate[]> {
    const response = await this.api.get<TicketUpdate[]>(`/tickets/${ticketId}/updates`);
    return response.data;
  }

  // Comments endpoints
  async getTicketComments(ticketId: string): Promise<TicketComment[]> {
    const response = await this.api.get<TicketComment[]>(`/tickets/${ticketId}/comments`);
    return response.data;
  }

  async addComment(ticketId: string, content: string): Promise<TicketComment> {
    const response = await this.api.post<TicketComment>(
      `/tickets/${ticketId}/comments`,
      { content }
    );
    return response.data;
  }

  // Customer endpoints
  async getCustomers(page: number = 1, pageSize: number = 20): Promise<PaginatedResponse<Customer>> {
    const params = new URLSearchParams();
    params.append('page', page.toString());
    params.append('pageSize', pageSize.toString());

    const response = await this.api.get<PaginatedResponse<Customer>>(
      `/customers?${params.toString()}`
    );
    return response.data;
  }

  async getCustomer(id: string): Promise<Customer> {
    const response = await this.api.get<Customer>(`/customers/${id}`);
    return response.data;
  }

  async createCustomer(data: Partial<Customer>): Promise<Customer> {
    const response = await this.api.post<Customer>('/customers', data);
    return response.data;
  }

  async updateCustomer(id: string, data: Partial<Customer>): Promise<Customer> {
    const response = await this.api.put<Customer>(`/customers/${id}`, data);
    return response.data;
  }

  async deleteCustomer(id: string): Promise<void> {
    await this.api.delete(`/customers/${id}`);
  }

  // Engineer endpoints
  async getEngineers(page: number = 1, pageSize: number = 20): Promise<PaginatedResponse<Engineer>> {
    const params = new URLSearchParams();
    params.append('page', page.toString());
    params.append('pageSize', pageSize.toString());

    const response = await this.api.get<PaginatedResponse<Engineer>>(
      `/engineers?${params.toString()}`
    );
    return response.data;
  }

  async getEngineer(id: string): Promise<Engineer> {
    const response = await this.api.get<Engineer>(`/engineers/${id}`);
    return response.data;
  }

  async createEngineer(data: Partial<Engineer>): Promise<Engineer> {
    const response = await this.api.post<Engineer>('/engineers', data);
    return response.data;
  }

  async updateEngineer(id: string, data: Partial<Engineer>): Promise<Engineer> {
    const response = await this.api.put<Engineer>(`/engineers/${id}`, data);
    return response.data;
  }

  async deleteEngineer(id: string): Promise<void> {
    await this.api.delete(`/engineers/${id}`);
  }

  // Email endpoints
  async getEmailSettings(): Promise<EmailSettings> {
    const response = await this.api.get<EmailSettings>('/email/settings');
    return response.data;
  }

  async updateEmailSettings(settings: Partial<EmailSettings>): Promise<EmailSettings> {
    const response = await this.api.patch<EmailSettings>('/email/settings', settings);
    return response.data;
  }

  async getDomainMappings(): Promise<DomainMapping[]> {
    const response = await this.api.get<DomainMapping[]>('/email/domain-mappings');
    return response.data;
  }

  async getAutoReplyTemplate(): Promise<AutoReplyTemplate> {
    const response = await this.api.get<AutoReplyTemplate>('/email/auto-reply-template');
    return response.data;
  }

  async updateAutoReplyTemplate(template: string): Promise<AutoReplyTemplate> {
    const response = await this.api.patch<AutoReplyTemplate>('/email/auto-reply-template', { template });
    return response.data;
  }

  async getEmailHistory(filters: EmailHistoryFilter): Promise<EmailHistoryResponse> {
    const params = new URLSearchParams();
    if (filters.status) params.append('status', filters.status);
    if (filters.customerId) params.append('customerId', filters.customerId);
    if (filters.startDate) params.append('startDate', filters.startDate);
    if (filters.endDate) params.append('endDate', filters.endDate);
    if (filters.page) params.append('page', filters.page.toString());
    if (filters.pageSize) params.append('pageSize', filters.pageSize.toString());

    const response = await this.api.get<EmailHistoryResponse>(
      `/email/history?${params.toString()}`
    );
    return response.data;
  }

  async testEmailConnection(): Promise<{ success: boolean; message: string }> {
    const response = await this.api.post<{ success: boolean; message: string }>(
      '/email/test-connection'
    );
    return response.data;
  }

  async syncEmails(): Promise<{ synced: number; message: string }> {
    const response = await this.api.post<{ synced: number; message: string }>('/email/sync');
    return response.data;
  }

  async createWASession(name: string): Promise<WhatsAppSession> {
    const response = await this.api.post<WhatsAppSession>('/whatsapp/sessions', { session_name: name });
    return response.data;
  }

  async getWASessions(): Promise<WhatsAppSession[]> {
    const response = await this.api.get<WhatsAppSession[]>('/whatsapp/sessions');
    return response.data;
  }

  async getSessionQR(sessionId: string): Promise<{ qrCode: string, status?: string }> {
    const response = await this.api.get<{ qr_code: string, status?: string }>(`/whatsapp/sessions/${sessionId}/qr`);
    return { qrCode: response.data.qr_code, status: response.data.status };
  }

  async requestPairingCode(sessionId: string, phoneNumber: string): Promise<{ pairingCode: string }> {
    const response = await this.api.post<{ pairing_code: string }>(`/whatsapp/sessions/${sessionId}/pairing-code`, { phone_number: phoneNumber });
    return { pairingCode: response.data.pairing_code };
  }

  async deleteSession(sessionId: string): Promise<void> {
    await this.api.delete(`/whatsapp/sessions/${sessionId}`);
  }

  async verifySession(sessionId: string): Promise<WhatsAppSession> {
    const response = await this.api.post<WhatsAppSession>(
      `/whatsapp/sessions/${sessionId}/verify`
    );
    return response.data;
  }

  async assignPhoneToEngineer(
    engineerId: string,
    phoneNumber: string,
    groupId?: string
  ): Promise<EngineerWAPhone> {
    const response = await this.api.post<EngineerWAPhone>(
      `/whatsapp/engineers/${engineerId}/phone`,
      { phone_number: phoneNumber, group_id: groupId }
    );
    return response.data;
  }

  async getEngineerPhones(): Promise<EngineerWAPhone[]> {
    const response = await this.api.get<EngineerWAPhone[]>('/whatsapp/engineer-phones');
    return response.data;
  }

  async updateEngineerPhone(
    id: string,
    phoneNumber: string,
    groupId?: string
  ): Promise<EngineerWAPhone> {
    const response = await this.api.patch<EngineerWAPhone>(
      `/whatsapp/engineer-phones/${id}`,
      { phoneNumber, groupId }
    );
    return response.data;
  }

  async deleteEngineerPhone(id: string): Promise<void> {
    await this.api.delete(`/whatsapp/engineer-phones/${id}`);
  }

  async testWAMessage(
    sessionId: string,
    phoneNumber: string,
    message: string
  ): Promise<{ success: boolean; message: string }> {
    const response = await this.api.post<{ success: boolean; message: string }>(
      `/whatsapp/test-message`,
      { sessionId, phoneNumber, message }
    );
    return response.data;
  }

  async getWAHookStatus(): Promise<WAHookStatus> {
    const response = await this.api.get<WAHookStatus>('/whatsapp/webhook/status');
    return response.data;
  }

  // Report endpoints
  async generateReport(customerId: string, month: number, year: number): Promise<ReportData> {
    const response = await this.api.post<ReportData>('/reports/generate', {
      customerId,
      month,
      year,
    });
    return response.data;
  }

  async getReports(filters?: ReportFilter): Promise<PaginatedResponse<Report>> {
    const params = new URLSearchParams();
    if (filters?.customerId) params.append('customerId', filters.customerId);
    if (filters?.month) params.append('month', filters.month.toString());
    if (filters?.year) params.append('year', filters.year.toString());
    if (filters?.startDate) params.append('startDate', filters.startDate);
    if (filters?.endDate) params.append('endDate', filters.endDate);
    if (filters?.page) params.append('page', filters.page.toString());
    if (filters?.pageSize) params.append('pageSize', filters.pageSize.toString());

    const response = await this.api.get<PaginatedResponse<Report>>(
      `/reports?${params.toString()}`
    );
    return response.data;
  }

  async getReport(reportId: string): Promise<ReportData> {
    const response = await this.api.get<ReportData>(`/reports/${reportId}`);
    return response.data;
  }

  async downloadReport(reportId: string, format: 'csv' | 'pdf'): Promise<Blob> {
    const response = await this.api.get<Blob>(`/reports/${reportId}/download/${format}`, {
      responseType: 'blob',
    });
    return response.data;
  }

  async resendReportEmail(reportId: string, additionalEmails?: string[]): Promise<{ success: boolean; message: string }> {
    const response = await this.api.post<{ success: boolean; message: string }>(
      `/reports/${reportId}/resend-email`,
      { additionalEmails: additionalEmails || [] }
    );
    return response.data;
  }

  async deleteReport(reportId: string): Promise<void> {
    await this.api.delete(`/reports/${reportId}`);
  }
}

export const apiService = new ApiService();
