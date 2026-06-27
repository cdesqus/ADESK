export interface EngineerStat {
  engineerId: string;
  name: string;
  ticketsHandled: number;
  avgTime: number;
  resolutionRate: number;
}

export interface ReportMetrics {
  totalTickets: number;
  resolvedTickets: number;
  openTickets: number;
  inProgressTickets: number;
  averageResolutionTime: number; // hours
  slaCompliance: number; // percentage
  byStatus: Record<string, number>;
  byPriority: Record<string, number>;
  bySource: Record<string, number>;
  engineerStats: EngineerStat[];
}

export interface TicketSummary {
  id: string;
  title: string;
  description?: string;
  status: string;
  priority: string;
  createdAt: string;
  resolvedAt?: string;
  ticketNumber?: string;
  timeToResolve?: number;
  engineer?: string;
}

export interface ReportData {
  id: string;
  customerName: string;
  month: string;
  metrics: ReportMetrics;
  ticketsList: TicketSummary[];
  generatedAt: string;
  sentAt?: string;
}

export interface Report {
  id: string;
  customerId: string;
  customerName: string;
  month: number;
  year: number;
  generatedAt: string;
  sentAt?: string;
  sent: boolean;
  csvSize?: number;
  pdfSize?: number;
}

export interface ReportFilter {
  customerId?: string;
  month?: number;
  year?: number;
  startDate?: string;
  endDate?: string;
  page?: number;
  pageSize?: number;
}
