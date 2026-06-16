import { useEffect, useState } from 'react';
import { Customer, PaginatedResponse } from '@/types';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { apiService } from '@/services/api';
import { Edit2, Trash2, Plus } from 'lucide-react';

export const CustomersPage: React.FC = () => {
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pagination, setPagination] = useState({ page: 1, pageSize: 20, total: 0, totalPages: 0 });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState({ name: '', domain: '', email_support: '', address: '' });
  const [isAdding, setIsAdding] = useState(false);

  useEffect(() => {
    fetchCustomers();
  }, []);

  const fetchCustomers = async (page: number = 1) => {
    try {
      setIsLoading(true);
      setError(null);
      const response: PaginatedResponse<Customer> = await apiService.getCustomers(page, 20);
      setCustomers(response.data);
      setPagination({
        page: response.page,
        pageSize: response.pageSize,
        total: response.total,
        totalPages: response.totalPages,
      });
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to load customers';
      setError(errorMsg);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAdd = async () => {
    if (!formData.name) return;
    try {
      await apiService.createCustomer(formData);
      setFormData({ name: '', domain: '', email_support: '', address: '' });
      setIsAdding(false);
      fetchCustomers();
    } catch (err) {
      console.error('Failed to create customer:', err);
    }
  };

  const handleUpdate = async (id: string) => {
    try {
      await apiService.updateCustomer(id, formData);
      setEditingId(null);
      setFormData({ name: '', domain: '', email_support: '', address: '' });
      fetchCustomers();
    } catch (err) {
      console.error('Failed to update customer:', err);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this customer?')) return;
    try {
      await apiService.deleteCustomer(id);
      fetchCustomers();
    } catch (err) {
      console.error('Failed to delete customer:', err);
    }
  };

  const handleEdit = (customer: Customer) => {
    setEditingId(customer.id);
    setFormData({ name: customer.name, domain: customer.domain || '', email_support: customer.email_support || '', address: customer.address || '' });
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-semibold text-gray-900">Customers</h1>
          <p className="text-gray-600 mt-1">Manage customer information</p>
        </div>
        <Button onClick={() => setIsAdding(true)} className="flex items-center gap-2">
          <Plus className="w-4 h-4" />
          Add Customer
        </Button>
      </div>

      {error && (
        <div className="rounded-lg bg-red-50 p-4 border border-red-200 mb-6">
          <p className="text-sm text-red-800">Failed to load customers: {error}</p>
        </div>
      )}

      {isAdding && (
        <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6 mb-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Add New Customer</h3>
          <div className="space-y-4">
            <Input
              placeholder="Name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            />
            <Input
              placeholder="Email Domain (e.g. kaumtech.com) - Used for auto-routing tickets"
              value={formData.domain}
              onChange={(e) => setFormData({ ...formData, domain: e.target.value })}
            />
            <Input
              placeholder="Support Email - For monthly reports"
              type="email"
              value={formData.email_support}
              onChange={(e) => setFormData({ ...formData, email_support: e.target.value })}
            />
            <Input
              placeholder="Address"
              value={formData.address || ''}
              onChange={(e) => setFormData({ ...formData, address: e.target.value })}
            />
            <div className="flex gap-3">
              <Button onClick={handleAdd}>Create</Button>
              <Button variant="outline" onClick={() => { setIsAdding(false); setFormData({ name: '', domain: '', email_support: '', address: '' }); }}>
                Cancel
              </Button>
            </div>
          </div>
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="text-center">
            <div className="inline-block w-8 h-8 border-4 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
            <p className="mt-4 text-gray-600">Loading customers...</p>
          </div>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-gray-200 shadow-xs overflow-hidden">
          <div className="hidden md:block overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Name</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Domain</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Support Email</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Address</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {customers.map((customer) => (
                  <tr key={customer.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 text-sm font-medium text-gray-900">{customer.name}</td>
                    <td className="px-6 py-4 text-sm text-gray-600">{customer.domain || '-'}</td>
                    <td className="px-6 py-4 text-sm text-gray-600">{customer.email_support || '-'}</td>
                    <td className="px-6 py-4 text-sm text-gray-600">{customer.address || '-'}</td>
                    <td className="px-6 py-4 text-sm">
                      <div className="flex gap-2">
                        <Button size="sm" variant="ghost" onClick={() => handleEdit(customer)}>
                          <Edit2 className="w-4 h-4" />
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => handleDelete(customer.id)}>
                          <Trash2 className="w-4 h-4 text-red-600" />
                        </Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {editingId && (
            <div className="bg-gray-50 border-t border-gray-200 p-6">
              <h4 className="text-sm font-semibold text-gray-900 mb-4">Edit Customer</h4>
              <div className="space-y-4">
                <Input value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} placeholder="Name" />
                <Input value={formData.domain} onChange={(e) => setFormData({ ...formData, domain: e.target.value })} placeholder="Email Domain (e.g. kaumtech.com)" />
                <Input value={formData.email_support} onChange={(e) => setFormData({ ...formData, email_support: e.target.value })} placeholder="Support Email" />
                <Input value={formData.address || ''} onChange={(e) => setFormData({ ...formData, address: e.target.value })} placeholder="Address" />
                <div className="flex gap-3">
                  <Button onClick={() => handleUpdate(editingId)}>Save</Button>
                  <Button variant="outline" onClick={() => { setEditingId(null); setFormData({ name: '', domain: '', email_support: '', address: '' }); }}>
                    Cancel
                  </Button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
