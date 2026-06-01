import { useEffect, useState } from 'react';
import { Engineer, PaginatedResponse } from '@/types';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { apiService } from '@/services/api';
import { Edit2, Trash2, Plus } from 'lucide-react';

export const EngineersPage: React.FC = () => {
  const [engineers, setEngineers] = useState<Engineer[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pagination, setPagination] = useState({ page: 1, pageSize: 20, total: 0, totalPages: 0 });
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState({ name: '', email: '', specialty: '', status: 'active' as 'active' | 'inactive' });
  const [isAdding, setIsAdding] = useState(false);

  useEffect(() => {
    fetchEngineers();
  }, []);

  const fetchEngineers = async (page: number = 1) => {
    try {
      setIsLoading(true);
      setError(null);
      const response: PaginatedResponse<Engineer> = await apiService.getEngineers(page, 20);
      setEngineers(response.data);
      setPagination({
        page: response.page,
        pageSize: response.pageSize,
        total: response.total,
        totalPages: response.totalPages,
      });
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to load engineers';
      setError(errorMsg);
    } finally {
      setIsLoading(false);
    }
  };

  const handleAdd = async () => {
    if (!formData.name || !formData.email) return;
    try {
      await apiService.createEngineer(formData);
      setFormData({ name: '', email: '', specialty: '', status: 'active' as 'active' | 'inactive' });
      setIsAdding(false);
      fetchEngineers();
    } catch (err) {
      console.error('Failed to create engineer:', err);
    }
  };

  const handleUpdate = async (id: string) => {
    try {
      await apiService.updateEngineer(id, formData);
      setEditingId(null);
      setFormData({ name: '', email: '', specialty: '', status: 'active' as 'active' | 'inactive' });
      fetchEngineers();
    } catch (err) {
      console.error('Failed to update engineer:', err);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this engineer?')) return;
    try {
      await apiService.deleteEngineer(id);
      fetchEngineers();
    } catch (err) {
      console.error('Failed to delete engineer:', err);
    }
  };

  const handleEdit = (engineer: Engineer) => {
    setEditingId(engineer.id);
    setFormData({ name: engineer.name, email: engineer.email, specialty: engineer.specialization || '', status: engineer.status || 'active' });
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-semibold text-gray-900">Engineers</h1>
          <p className="text-gray-600 mt-1">Manage support team members</p>
        </div>
        <Button onClick={() => setIsAdding(true)} className="flex items-center gap-2">
          <Plus className="w-4 h-4" />
          Add Engineer
        </Button>
      </div>

      {error && (
        <div className="rounded-lg bg-red-50 p-4 border border-red-200 mb-6">
          <p className="text-sm text-red-800">Failed to load engineers: {error}</p>
        </div>
      )}

      {isAdding && (
        <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6 mb-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Add New Engineer</h3>
          <div className="space-y-4">
            <Input
              placeholder="Name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            />
            <Input
              placeholder="Email"
              type="email"
              value={formData.email}
              onChange={(e) => setFormData({ ...formData, email: e.target.value })}
            />
            <Input
              placeholder="Specialty"
              value={formData.specialty}
              onChange={(e) => setFormData({ ...formData, specialty: e.target.value })}
            />
            <Select
              value={formData.status}
              onChange={(e) => setFormData({ ...formData, status: e.target.value as 'active' | 'inactive' })}
            >
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </Select>
            <div className="flex gap-3">
              <Button onClick={handleAdd}>Create</Button>
              <Button variant="outline" onClick={() => { setIsAdding(false); setFormData({ name: '', email: '', specialty: '', status: 'active' as 'active' | 'inactive' }); }}>
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
            <p className="mt-4 text-gray-600">Loading engineers...</p>
          </div>
        </div>
      ) : (
        <div className="bg-white rounded-lg border border-gray-200 shadow-xs overflow-hidden">
          <div className="hidden md:block overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Name</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Email</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Specialty</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Status</th>
                  <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {engineers.map((engineer) => (
                  <tr key={engineer.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 text-sm font-medium text-gray-900">{engineer.name}</td>
                    <td className="px-6 py-4 text-sm text-gray-600">{engineer.email}</td>
                    <td className="px-6 py-4 text-sm text-gray-600">{engineer.specialization || '-'}</td>
                    <td className="px-6 py-4 text-sm">
                      <span className={`inline-flex px-2.5 py-0.5 rounded-full text-xs font-medium ${(engineer.status || 'active') === 'active' ? 'bg-green-50 text-green-800' : 'bg-gray-100 text-gray-800'}`}>
                        {engineer.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm">
                      <div className="flex gap-2">
                        <Button size="sm" variant="ghost" onClick={() => handleEdit(engineer)}>
                          <Edit2 className="w-4 h-4" />
                        </Button>
                        <Button size="sm" variant="ghost" onClick={() => handleDelete(engineer.id)}>
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
              <h4 className="text-sm font-semibold text-gray-900 mb-4">Edit Engineer</h4>
              <div className="space-y-4">
                <Input value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} placeholder="Name" />
                <Input value={formData.email} onChange={(e) => setFormData({ ...formData, email: e.target.value })} placeholder="Email" />
                <Input value={formData.specialty} onChange={(e) => setFormData({ ...formData, specialty: e.target.value })} placeholder="Specialty" />
                <Select value={formData.status} onChange={(e) => setFormData({ ...formData, status: e.target.value as 'active' | 'inactive' })}>
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                </Select>
                <div className="flex gap-3">
                  <Button onClick={() => handleUpdate(editingId)}>Save</Button>
                  <Button variant="outline" onClick={() => { setEditingId(null); setFormData({ name: '', email: '', specialty: '', status: 'active' as 'active' | 'inactive' }); }}>
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
