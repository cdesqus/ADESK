import React, { useState } from 'react';
import { DomainMapping as DomainMappingType } from '@/types/email';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Check, X, Edit2, RefreshCw } from 'lucide-react';

interface DomainMappingProps {
  domains: DomainMappingType[];
  isLoading?: boolean;
  onUpdate?: (customerId: string, domain: string) => Promise<void>;
  onTest?: (domain: string) => Promise<void>;
}

export const DomainMapping: React.FC<DomainMappingProps> = ({
  domains,
  isLoading,
  onUpdate,
  onTest,
}) => {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editValue, setEditValue] = useState('');
  const [isSaving, setIsSaving] = useState<string | null>(null);
  const [isTesting, setIsTesting] = useState<string | null>(null);
  const [testResults, setTestResults] = useState<Record<string, { success: boolean; message: string }>>({});

  const handleEditStart = (customerId: string, currentDomain: string) => {
    setEditingId(customerId);
    setEditValue(currentDomain);
  };

  const handleSave = async (customerId: string) => {
    if (!editValue.trim() || !onUpdate) return;

    try {
      setIsSaving(customerId);
      await onUpdate(customerId, editValue.trim());
      setEditingId(null);
      setEditValue('');
    } finally {
      setIsSaving(null);
    }
  };

  const handleTest = async (customerId: string, domain: string) => {
    if (!onTest) return;

    try {
      setIsTesting(customerId);
      await onTest(domain);
      setTestResults((prev) => ({
        ...prev,
        [customerId]: { success: true, message: 'Domain test passed' },
      }));
    } catch (error) {
      setTestResults((prev) => ({
        ...prev,
        [customerId]: {
          success: false,
          message: error instanceof Error ? error.message : 'Domain test failed',
        },
      }));
    } finally {
      setIsTesting(null);
    }
  };

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="p-6">
          <div className="space-y-3">
            {[...Array(3)].map((_, i) => (
              <div key={i} className="h-12 bg-gray-100 rounded animate-pulse" />
            ))}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                Customer
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                Domain
              </th>
              <th className="px-6 py-3 text-right text-xs font-semibold text-gray-700 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {domains.length === 0 ? (
              <tr>
                <td colSpan={3} className="px-6 py-8 text-center">
                  <p className="text-gray-500 text-sm">No customers configured</p>
                </td>
              </tr>
            ) : (
              domains.map((mapping) => (
                <tr key={mapping.customerId} className="hover:bg-gray-50">
                  <td className="px-6 py-4">
                    <p className="text-sm font-medium text-gray-900">{mapping.customerName}</p>
                    <p className="text-xs text-gray-500 mt-0.5">{mapping.customerId}</p>
                  </td>
                  <td className="px-6 py-4">
                    {editingId === mapping.customerId ? (
                      <div className="flex items-center gap-2">
                        <Input
                          type="text"
                          value={editValue}
                          onChange={(e) => setEditValue(e.target.value)}
                          placeholder="Enter domain"
                          className="flex-1"
                          autoFocus
                        />
                      </div>
                    ) : (
                      <div>
                        <p className="text-sm font-mono text-gray-900">{mapping.domain}</p>
                        {testResults[mapping.customerId] && (
                          <p
                            className={`text-xs mt-1 ${
                              testResults[mapping.customerId].success
                                ? 'text-green-600'
                                : 'text-red-600'
                            }`}
                          >
                            {testResults[mapping.customerId].message}
                          </p>
                        )}
                      </div>
                    )}
                  </td>
                  <td className="px-6 py-4 text-right">
                    {editingId === mapping.customerId ? (
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => handleSave(mapping.customerId)}
                          disabled={isSaving === mapping.customerId}
                          className="p-2 text-green-600 hover:bg-green-50 rounded transition-colors disabled:opacity-50"
                          aria-label="Save"
                        >
                          <Check className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => setEditingId(null)}
                          disabled={isSaving === mapping.customerId}
                          className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded transition-colors disabled:opacity-50"
                          aria-label="Cancel"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center justify-end gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTest(mapping.customerId, mapping.domain)}
                          disabled={isTesting === mapping.customerId}
                          className="flex items-center gap-1"
                        >
                          {isTesting === mapping.customerId ? (
                            <RefreshCw className="w-3 h-3 animate-spin" />
                          ) : (
                            <RefreshCw className="w-3 h-3" />
                          )}
                          <span className="hidden sm:inline">Test</span>
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleEditStart(mapping.customerId, mapping.domain)}
                          className="flex items-center gap-1"
                        >
                          <Edit2 className="w-3 h-3" />
                          <span className="hidden sm:inline">Edit</span>
                        </Button>
                      </div>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};
