import { useState, useEffect } from 'react';
import { Customer, Engineer, TicketPriority, TicketStatus } from '@/types';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { apiService } from '@/services/api';

interface CreateTicketModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: any) => Promise<void>;
}

export const CreateTicketModal: React.FC<CreateTicketModalProps> = ({
  isOpen,
  onClose,
  onSubmit,
}) => {
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [engineers, setEngineers] = useState<Engineer[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [formData, setFormData] = useState({
    title: '',
    description: '',
    customer_id: '',
    engineer_id: '',
    priority: 'medium' as TicketPriority,
    status: 'open' as TicketStatus,
    source: 'MANUAL',
  });

  useEffect(() => {
    if (isOpen) {
      fetchCustomers();
      fetchEngineers();
      // Reset form
      setFormData({
        title: '',
        description: '',
        customer_id: '',
        engineer_id: '',
        priority: 'medium',
        status: 'open',
        source: 'MANUAL',
      });
      setError(null);
    }
  }, [isOpen]);

  const fetchCustomers = async () => {
    try {
      const response = await apiService.getCustomers(1, 100);
      setCustomers(response.data);
    } catch (err) {
      console.error('Failed to fetch customers:', err);
    }
  };

  const fetchEngineers = async () => {
    try {
      const response = await apiService.getEngineers(1, 100);
      setEngineers(response.data);
    } catch (err) {
      console.error('Failed to fetch engineers:', err);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.title || !formData.customer_id) {
      setError('Title and Customer are required');
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      
      const payload = {
        title: formData.title,
        description: formData.description,
        customer_id: parseInt(formData.customer_id),
        engineer_id: formData.engineer_id ? parseInt(formData.engineer_id) : undefined,
        priority: formData.priority,
        status: formData.status,
        source: formData.source,
      };

      await onSubmit(payload);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create ticket');
    } finally {
      setIsLoading(false);
    }
  };

  // Filter engineers based on selected customer
  const filteredEngineers = engineers.filter(
    (eng) => !eng.customer_id || eng.customer_id === parseInt(formData.customer_id)
  );

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex items-center justify-center min-h-screen px-4 pt-4 pb-20 text-center sm:block sm:p-0">
        <div className="fixed inset-0 transition-opacity bg-gray-500 bg-opacity-75" onClick={onClose} />
        <span className="hidden sm:inline-block sm:align-middle sm:h-screen">&#8203;</span>
        <div className="inline-block w-full max-w-2xl px-4 pt-5 pb-4 overflow-hidden text-left align-bottom transition-all transform bg-white rounded-lg shadow-xl sm:my-8 sm:align-middle sm:p-6">
          <div className="flex justify-between items-center mb-5">
            <h3 className="text-lg font-medium leading-6 text-gray-900">Create Manual Ticket</h3>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-500">
              <span className="sr-only">Close</span>
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="p-3 text-sm text-red-600 bg-red-50 border border-red-200 rounded-md">
                {error}
              </div>
            )}

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Title *</label>
              <Input
                required
                value={formData.title}
                onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                placeholder="Brief summary of the issue"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
              <textarea
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-primary-500 focus:border-primary-500 sm:text-sm"
                rows={4}
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                placeholder="Detailed explanation of the request or problem..."
              />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Customer *</label>
                <Select
                  required
                  value={formData.customer_id}
                  onChange={(e) => {
                    setFormData({ ...formData, customer_id: e.target.value, engineer_id: '' });
                  }}
                >
                  <option value="">Select a customer</option>
                  {customers.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name}
                    </option>
                  ))}
                </Select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Assign Engineer</label>
                <Select
                  value={formData.engineer_id}
                  onChange={(e) => setFormData({ ...formData, engineer_id: e.target.value })}
                  disabled={!formData.customer_id}
                >
                  <option value="">Unassigned</option>
                  {filteredEngineers.map((e) => (
                    <option key={e.id} value={e.id}>
                      {e.name} {e.customer_id ? '' : '(All Customers)'}
                    </option>
                  ))}
                </Select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Priority</label>
                <Select
                  value={formData.priority}
                  onChange={(e) => setFormData({ ...formData, priority: e.target.value as TicketPriority })}
                >
                  <option value="low">Low</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                  <option value="urgent">Urgent</option>
                </Select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                <Select
                  value={formData.status}
                  onChange={(e) => setFormData({ ...formData, status: e.target.value as TicketStatus })}
                >
                  <option value="open">Open</option>
                  <option value="in_progress">In Progress</option>
                  <option value="waiting_customer">Waiting Customer</option>
                  <option value="resolved">Resolved</option>
                  <option value="closed">Closed</option>
                </Select>
              </div>
            </div>

            <div className="mt-5 sm:mt-6 sm:flex sm:flex-row-reverse">
              <Button
                type="submit"
                className="w-full sm:ml-3 sm:w-auto"
                disabled={isLoading || !formData.title || !formData.customer_id}
              >
                {isLoading ? 'Creating...' : 'Create Ticket'}
              </Button>
              <Button
                type="button"
                variant="outline"
                className="mt-3 w-full sm:mt-0 sm:w-auto"
                onClick={onClose}
              >
                Cancel
              </Button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
};
