import React, { useEffect, useState } from 'react';
import { apiService } from '@/services/api';
import { DashboardSummary } from '@/types/dashboard';
import { TicketStatus, TicketPriority } from '@/types';
import { Link } from 'react-router-dom';
import { 
  Ticket as TicketIcon, 
  CheckCircle2, 
  Clock, 
  AlertCircle, 
  MoreHorizontal,
  ArrowRight
} from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

const statusColors: Record<TicketStatus, { bg: string; text: string }> = {
  open: { bg: 'bg-blue-50', text: 'text-blue-800' },
  in_progress: { bg: 'bg-amber-50', text: 'text-amber-800' },
  waiting_customer: { bg: 'bg-purple-50', text: 'text-purple-800' },
  resolved: { bg: 'bg-green-50', text: 'text-green-800' },
  closed: { bg: 'bg-gray-100', text: 'text-gray-800' },
};

const priorityColors: Record<TicketPriority, { bg: string; text: string }> = {
  low: { bg: 'bg-green-50', text: 'text-green-800' },
  medium: { bg: 'bg-blue-50', text: 'text-blue-800' },
  high: { bg: 'bg-amber-50', text: 'text-amber-800' },
  urgent: { bg: 'bg-red-50', text: 'text-red-800' },
};

export const DashboardPage: React.FC = () => {
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadSummary();
  }, []);

  const loadSummary = async () => {
    try {
      setIsLoading(true);
      const data = await apiService.getDashboardSummary();
      setSummary(data);
      setError(null);
    } catch (err) {
      setError('Failed to load dashboard summary');
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="text-center">
          <div className="inline-block w-8 h-8 border-4 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
          <p className="mt-4 text-gray-600">Loading dashboard...</p>
        </div>
      </div>
    );
  }

  if (error || !summary) {
    return (
      <div className="rounded-lg bg-red-50 p-4 border border-red-200">
        <p className="text-sm text-red-800">{error || 'Something went wrong'}</p>
        <button 
          onClick={loadSummary}
          className="mt-2 text-sm font-medium text-red-800 hover:text-red-900 underline"
        >
          Try again
        </button>
      </div>
    );
  }

  const statCards = [
    {
      title: 'Total Tickets',
      value: summary.stats.total,
      icon: TicketIcon,
      color: 'text-gray-700',
      bgColor: 'bg-gray-100',
    },
    {
      title: 'Open',
      value: summary.stats.open,
      icon: AlertCircle,
      color: 'text-blue-700',
      bgColor: 'bg-blue-100',
    },
    {
      title: 'In Progress',
      value: summary.stats.in_progress,
      icon: Clock,
      color: 'text-amber-700',
      bgColor: 'bg-amber-100',
    },
    {
      title: 'Resolved & Closed',
      value: summary.stats.resolved + summary.stats.closed,
      icon: CheckCircle2,
      color: 'text-green-700',
      bgColor: 'bg-green-100',
    },
  ];

  return (
    <div className="space-y-6">
      <div className="mb-8 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-semibold text-gray-900 tracking-tight">Dashboard Overview</h1>
          <p className="text-gray-500 mt-1">Here's what's happening with your support tickets today.</p>
        </div>
        <Link 
          to="/tickets" 
          className="inline-flex items-center px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-md transition-colors shadow-sm"
        >
          View All Tickets
        </Link>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {statCards.map((stat, index) => (
          <div key={index} className="bg-white rounded-xl border border-gray-200 p-6 shadow-sm hover:shadow-md transition-shadow">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-medium text-gray-500">{stat.title}</h3>
              <div className={`p-2 rounded-lg ${stat.bgColor}`}>
                <stat.icon className={`w-5 h-5 ${stat.color}`} />
              </div>
            </div>
            <div className="flex items-baseline gap-2">
              <p className="text-3xl font-bold text-gray-900">{stat.value}</p>
            </div>
          </div>
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mt-8">
        {/* Recent Tickets Table */}
        <div className="lg:col-span-2 bg-white rounded-xl border border-gray-200 shadow-sm overflow-hidden">
          <div className="px-6 py-5 border-b border-gray-200 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Recent Tickets</h2>
            <Link to="/tickets" className="text-sm font-medium text-primary-600 hover:text-primary-700 flex items-center gap-1">
              See all <ArrowRight className="w-4 h-4" />
            </Link>
          </div>
          
          {summary.recent_tickets.length === 0 ? (
            <div className="p-6 text-center text-gray-500">
              No recent tickets found.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead className="bg-gray-50 border-b border-gray-200">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Ticket</th>
                    <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                    <th className="px-6 py-3 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Created</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {summary.recent_tickets.map((ticket) => (
                    <tr key={ticket.id} className="hover:bg-gray-50 transition-colors group">
                      <td className="px-6 py-4">
                        <Link to={`/tickets/${ticket.id}`} className="block">
                          <div className="flex items-center gap-3">
                            <div className="w-8 h-8 rounded-full bg-primary-100 text-primary-700 flex items-center justify-center font-bold text-xs shrink-0">
                              {ticket.customer?.name?.[0]?.toUpperCase() || '?'}
                            </div>
                            <div>
                              <p className="text-sm font-medium text-gray-900 group-hover:text-primary-600 transition-colors line-clamp-1">
                                {ticket.title}
                              </p>
                              <p className="text-xs text-gray-500 mt-0.5">
                                #{ticket.ticket_number || ticket.id} • {ticket.customer?.name || 'Unknown Customer'}
                              </p>
                            </div>
                          </div>
                        </Link>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className={`inline-flex px-2.5 py-1 rounded-full text-xs font-medium ${
                          (statusColors[ticket.status.toLowerCase() as TicketStatus] || statusColors.open).bg
                        } ${(statusColors[ticket.status.toLowerCase() as TicketStatus] || statusColors.open).text}`}>
                          {ticket.status.replace('_', ' ')}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {formatDistanceToNow(new Date(ticket.createdAt || (ticket as any).created_at), { addSuffix: true })}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Action Panel */}
        <div className="bg-white rounded-xl border border-gray-200 shadow-sm p-6 flex flex-col">
          <h2 className="text-lg font-semibold text-gray-900 mb-6">Quick Actions</h2>
          
          <div className="space-y-3 flex-1">
            <Link 
              to="/tickets?status=open" 
              className="flex items-center justify-between p-4 rounded-lg border border-gray-200 hover:border-primary-300 hover:bg-primary-50 transition-all group"
            >
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-100 text-blue-700 rounded-md group-hover:bg-blue-200 transition-colors">
                  <AlertCircle className="w-5 h-5" />
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">View Open Tickets</p>
                  <p className="text-xs text-gray-500">Tickets needing attention</p>
                </div>
              </div>
              <ArrowRight className="w-4 h-4 text-gray-400 group-hover:text-primary-600" />
            </Link>

            <Link 
              to="/tickets?status=waiting_customer" 
              className="flex items-center justify-between p-4 rounded-lg border border-gray-200 hover:border-primary-300 hover:bg-primary-50 transition-all group"
            >
              <div className="flex items-center gap-3">
                <div className="p-2 bg-purple-100 text-purple-700 rounded-md group-hover:bg-purple-200 transition-colors">
                  <MoreHorizontal className="w-5 h-5" />
                </div>
                <div>
                  <p className="text-sm font-medium text-gray-900">Waiting on Customer</p>
                  <p className="text-xs text-gray-500">Follow up required</p>
                </div>
              </div>
              <ArrowRight className="w-4 h-4 text-gray-400 group-hover:text-primary-600" />
            </Link>
          </div>
          
          <div className="mt-6 pt-6 border-t border-gray-100">
            <div className="bg-gray-50 rounded-lg p-4 text-center border border-gray-200">
              <p className="text-sm font-medium text-gray-900">AI-DESK Support System</p>
              <p className="text-xs text-gray-500 mt-1">Version 1.0.0</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
