import { useState, useCallback } from 'react';
import { Ticket, PaginatedResponse } from '@/types';
import { apiService } from '@/services/api';

interface UseTicketsOptions {
  page?: number;
  pageSize?: number;
  status?: string;
  priority?: string;
  customerId?: string;
  search?: string;
}

export const useTickets = () => {
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pagination, setPagination] = useState({
    page: 1,
    pageSize: 20,
    total: 0,
    totalPages: 0,
  });

  const fetchTickets = useCallback(async (options: UseTicketsOptions = {}) => {
    try {
      setIsLoading(true);
      setError(null);
      const response: PaginatedResponse<Ticket> = await apiService.getTickets(
        options.page || 1,
        options.pageSize || 20,
        options.status,
        options.priority,
        options.customerId,
        options.search
      );
      setTickets(response.data);
      setPagination({
        page: response.page,
        pageSize: response.pageSize,
        total: response.total,
        totalPages: response.totalPages,
      });
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to fetch tickets';
      setError(errorMsg);
    } finally {
      setIsLoading(false);
    }
  }, []);

  const createTicket = useCallback(async (data: Partial<Ticket>) => {
    try {
      setError(null);
      const newTicket = await apiService.createTicket(data);
      setTickets((prev) => [newTicket, ...prev]);
      return newTicket;
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to create ticket';
      setError(errorMsg);
      throw err;
    }
  }, []);

  const updateTicket = useCallback(async (id: string, data: Partial<Ticket>) => {
    try {
      setError(null);
      const updated = await apiService.updateTicket(id, data);
      setTickets((prev) =>
        prev.map((ticket) => (ticket.id === id ? updated : ticket))
      );
      return updated;
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to update ticket';
      setError(errorMsg);
      throw err;
    }
  }, []);

  const deleteTicket = useCallback(async (id: string) => {
    try {
      setError(null);
      await apiService.deleteTicket(id);
      setTickets((prev) => prev.filter((ticket) => ticket.id.toString() !== id.toString()));
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to delete ticket';
      setError(errorMsg);
      throw err;
    }
  }, []);

  return {
    tickets,
    isLoading,
    error,
    pagination,
    fetchTickets,
    createTicket,
    updateTicket,
    deleteTicket,
  };
};
