export interface EmailSettings {
  emailAddress: string;
  status: 'connected' | 'disconnected';
  lastSync: string; // ISO timestamp
  pollingInterval: string;
}

export interface EmailLog {
  id: string;
  messageId: string;
  from: string;
  subject: string;
  body?: string;
  domainMatched: string;
  customerId: string;
  ticketId?: string;
  status: 'success' | 'failed' | 'unknown_domain';
  error?: string;
  createdAt: string; // ISO timestamp
}

export interface DomainMapping {
  customerId: string;
  customerName: string;
  domain: string;
}

export interface AutoReplyTemplate {
  template: string;
  variables: string[];
}

export interface EmailHistoryFilter {
  status?: 'success' | 'failed' | 'unknown_domain' | 'all';
  customerId?: string;
  startDate?: string;
  endDate?: string;
  page?: number;
  pageSize?: number;
}

export interface EmailHistoryResponse {
  data: EmailLog[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}
