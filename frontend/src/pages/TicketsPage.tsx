import { useEffect, useState } from 'react';
import { useTickets } from '@/hooks/useTickets';
import { Ticket as TicketType, TicketStatus, TicketPriority } from '@/types';
import { Button } from '@/components/ui/Button';
import { Select } from '@/components/ui/Select';
import { Input } from '@/components/ui/Input';
import { Link } from 'react-router-dom';
import { Eye, ChevronLeft, ChevronRight, Plus } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';
import { CreateTicketModal } from '@/components/CreateTicketModal';

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

export const TicketsPage: React.FC = () => {
  const { tickets, isLoading, error, pagination, fetchTickets, createTicket } = useTickets();
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [filters, setFilters] = useState({
    status: '',
    priority: '',
    search: '',
  });

  useEffect(() => {
    fetchTickets({
      page: 1,
      pageSize: 20,
      status: filters.status || undefined,
      priority: filters.priority || undefined,
      search: filters.search || undefined,
    });
  }, [filters]);

  const handleFilterChange = (key: keyof typeof filters, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const handlePageChange = (newPage: number) => {
    fetchTickets({
      page: newPage,
      pageSize: 20,
      status: filters.status || undefined,
      priority: filters.priority || undefined,
      search: filters.search || undefined,
    });
  };

  return (
    <div>
      {/* Header */}
      <div className="mb-8 flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-semibold text-gray-900">Tickets</h1>
          <p className="text-gray-600 mt-1">Manage and track all support tickets</p>
        </div>
        <Button onClick={() => setIsCreateModalOpen(true)} className="flex items-center gap-2">
          <Plus className="w-4 h-4" />
          Add Ticket
        </Button>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-4 mb-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-900 mb-1.5">Search</label>
            <Input
              placeholder="Search tickets..."
              value={filters.search}
              onChange={(e) => handleFilterChange('search', e.target.value)}
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-900 mb-1.5">Status</label>
            <Select
              value={filters.status}
              onChange={(e) => handleFilterChange('status', e.target.value)}
            >
              <option value="">All Status</option>
              <option value="open">Open</option>
              <option value="in_progress">In Progress</option>
              <option value="waiting_customer">Waiting Customer</option>
              <option value="resolved">Resolved</option>
              <option value="closed">Closed</option>
            </Select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-900 mb-1.5">Priority</label>
            <Select
              value={filters.priority}
              onChange={(e) => handleFilterChange('priority', e.target.value)}
            >
              <option value="">All Priority</option>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
              <option value="urgent">Urgent</option>
            </Select>
          </div>

          <div className="flex items-end">
            <Button
              variant="outline"
              className="w-full"
              onClick={() => setFilters({ status: '', priority: '', search: '' })}
            >
              Clear Filters
            </Button>
          </div>
        </div>
      </div>

      {/* Loading State */}
      {isLoading && (
        <div className="flex items-center justify-center py-12">
          <div className="text-center">
            <div className="inline-block w-8 h-8 border-4 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
            <p className="mt-4 text-gray-600">Loading tickets...</p>
          </div>
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className="rounded-lg bg-red-50 p-4 border border-red-200 mb-6">
          <p className="text-sm text-red-800">Failed to load tickets: {error}</p>
        </div>
      )}

      {/* Tickets Table */}
      {!isLoading && tickets.length > 0 && (
        <div className="bg-white rounded-lg border border-gray-200 shadow-xs overflow-hidden">
          <div className="hidden md:block overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">ID</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Title</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Customer</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Priority</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Created</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {tickets.map((ticket) => (
                  <tr key={ticket.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4 text-sm font-medium text-primary-700">
                      #{typeof ticket.id === 'string' ? ticket.id.slice(0, 8) : ticket.id}
                    </td>
                    <td className="px-6 py-4">
                      <p className="text-sm font-medium text-gray-900 truncate max-w-xs">
                        {ticket.title}
                      </p>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600">
                      {ticket.customer?.name || 'N/A'}
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          (statusColors[ticket.status] || statusColors.open).bg
                        } ${(statusColors[ticket.status] || statusColors.open).text}`}
                      >
                        {ticket.status.replace('_', ' ')}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex px-2.5 py-0.5 rounded-full text-xs font-medium ${
                          (priorityColors[ticket.priority] || priorityColors.medium).bg
                        } ${(priorityColors[ticket.priority] || priorityColors.medium).text}`}
                      >
                        {ticket.priority}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-600">
                      {formatDistanceToNow(new Date(ticket.createdAt || (ticket as any).created_at), { addSuffix: true })}
                    </td>
                    <td className="px-6 py-4">
                      <Link to={`/tickets/${ticket.id}`}>
                        <Button size="sm" variant="ghost">
                          <Eye className="w-4 h-4" />
                        </Button>
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Mobile Cards */}
          <div className="md:hidden divide-y divide-gray-200">
            {tickets.map((ticket) => (
              <div key={ticket.id} className="p-4">
                <div className="flex justify-between items-start mb-3">
                  <div className="flex-1">
                    <p className="text-xs font-medium text-primary-700 mb-1">
                      #{typeof ticket.id === 'string' ? ticket.id.slice(0, 8) : ticket.id}
                    </p>
                    <h3 className="text-sm font-medium text-gray-900">{ticket.title}</h3>
                  </div>
                  <Link to={`/tickets/${ticket.id}`}>
                    <Button size="sm" variant="ghost">
                      <Eye className="w-4 h-4" />
                    </Button>
                  </Link>
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-gray-600">Status:</span>
                    <span
                      className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${
                        (statusColors[ticket.status] || statusColors.open).bg
                      } ${(statusColors[ticket.status] || statusColors.open).text}`}
                    >
                      {ticket.status.replace('_', ' ')}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-gray-600">Priority:</span>
                    <span
                      className={`inline-flex px-2 py-0.5 rounded text-xs font-medium ${
                        (priorityColors[ticket.priority] || priorityColors.medium).bg
                      } ${(priorityColors[ticket.priority] || priorityColors.medium).text}`}
                    >
                      {ticket.priority}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Empty State */}
      {!isLoading && tickets.length === 0 && !error && (
        <div className="text-center py-12 bg-white rounded-lg border border-gray-200">
          <p className="text-gray-600 mb-4">No tickets found</p>
        </div>
      )}

      {/* Pagination */}
      {pagination.totalPages > 1 && (
        <div className="flex items-center justify-between mt-6">
          <p className="text-sm text-gray-600">
            Page {pagination.page} of {pagination.totalPages}
          </p>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => handlePageChange(pagination.page - 1)}
              disabled={pagination.page === 1}
            >
              <ChevronLeft className="w-4 h-4" />
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => handlePageChange(pagination.page + 1)}
              disabled={pagination.page === pagination.totalPages}
            >
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}

      {/* Create Ticket Modal */}
      <CreateTicketModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSubmit={async (data) => {
          await createTicket(data);
        }}
      />
    </div>
  );
};
