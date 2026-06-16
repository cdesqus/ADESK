import { Ticket } from './index';

export interface DashboardSummary {
  stats: {
    total: number;
    open: number;
    resolved: number;
    in_progress: number;
    closed: number;
    waiting_customer: number;
  };
  recent_tickets: Ticket[];
}
