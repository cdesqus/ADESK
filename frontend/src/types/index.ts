export type UserRole = 'admin' | 'engineer' | 'customer';

// Re-export email types
export type {
  EmailSettings,
  EmailLog,
  DomainMapping,
  AutoReplyTemplate,
  EmailHistoryFilter,
  EmailHistoryResponse,
} from './email';

// Re-export WhatsApp types
export type {
  WhatsAppSession,
  EngineerWAPhone,
  WAHookStatus,
} from './whatsapp';

// Re-export Report types
export type {
  Report,
  ReportData,
  ReportMetrics,
  EngineerStat,
  TicketSummary,
  ReportFilter,
} from './reports';

export interface User {
  id: string;
  email: string;
  name: string;
  role: UserRole;
  avatar?: string;
  createdAt: string;
}

export interface AuthToken {
  access_token: string;
  token_type: string;
  expires_in: number;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export type TicketStatus = 'open' | 'in_progress' | 'waiting_customer' | 'resolved' | 'closed';
export type TicketPriority = 'low' | 'medium' | 'high' | 'urgent';

export interface Ticket {
  id: string;
  title: string;
  description: string;
  status: TicketStatus;
  priority: TicketPriority;
  customerId: string;
  assignedEngineerId?: string;
  createdAt: string;
  updatedAt: string;
  resolvedAt?: string;
  customer?: Customer;
  assignedEngineer?: Engineer;
}

export interface TicketComment {
  id: string;
  ticketId: string;
  userId: string;
  content: string;
  createdAt: string;
  user?: User;
}

export interface TicketUpdate {
  id: string;
  ticketId: string;
  type: 'status_change' | 'priority_change' | 'assignment' | 'comment';
  oldValue?: string;
  newValue?: string;
  createdAt: string;
  user: User;
}

export interface Customer {
  id: string;
  name: string;
  email: string;
  phone?: string;
  domain?: string;
  company?: string;
  createdAt: string;
}

export interface Engineer {
  id: string;
  name: string;
  email: string;
  phone?: string;
  domain?: string;
  specialization?: string;
  status?: 'active' | 'inactive';
  createdAt: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
}
