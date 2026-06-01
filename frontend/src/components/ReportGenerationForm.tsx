import React, { useState, useEffect } from 'react';
import { Button } from '@/components/ui/Button';
import { Select } from '@/components/ui/Select';
import { Customer } from '@/types';
import { AlertCircle, CheckCircle } from 'lucide-react';

interface ReportGenerationFormProps {
  customers: Customer[];
  onGenerate: (customerId: string, month: number, year: number) => Promise<void>;
  isLoading?: boolean;
  error?: string;
}

const months = [
  { value: 1, label: 'January' },
  { value: 2, label: 'February' },
  { value: 3, label: 'March' },
  { value: 4, label: 'April' },
  { value: 5, label: 'May' },
  { value: 6, label: 'June' },
  { value: 7, label: 'July' },
  { value: 8, label: 'August' },
  { value: 9, label: 'September' },
  { value: 10, label: 'October' },
  { value: 11, label: 'November' },
  { value: 12, label: 'December' },
];

export const ReportGenerationForm: React.FC<ReportGenerationFormProps> = ({
  customers,
  onGenerate,
  isLoading = false,
  error,
}) => {
  const [customerId, setCustomerId] = useState('');
  const [month, setMonth] = useState(new Date().getMonth() + 1);
  const [year, setYear] = useState(new Date().getFullYear());
  const [success, setSuccess] = useState(false);
  const [successTime, setSuccessTime] = useState('');
  const [localError, setLocalError] = useState('');

  const currentYear = new Date().getFullYear();
  const years = Array.from({ length: 4 }, (_, i) => currentYear - 2 + i);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLocalError('');
    setSuccess(false);

    if (!customerId) {
      setLocalError('Please select a customer');
      return;
    }

    try {
      await onGenerate(customerId, month, year);
      setSuccess(true);
      setSuccessTime(new Date().toLocaleTimeString());
      setCustomerId('');
      setMonth(new Date().getMonth() + 1);
      setYear(new Date().getFullYear());
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : 'Failed to generate report');
    }
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Generate Report</h3>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Customer
          </label>
          <Select
            value={customerId}
            onChange={(e) => setCustomerId(e.target.value)}
            disabled={isLoading}
            className="w-full"
          >
            <option value="">Select a customer...</option>
            {customers.map((customer) => (
              <option key={customer.id} value={customer.id}>
                {customer.name}
              </option>
            ))}
          </Select>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Month
            </label>
            <Select
              value={month.toString()}
              onChange={(e) => setMonth(parseInt(e.target.value))}
              disabled={isLoading}
              className="w-full"
            >
              {months.map((m) => (
                <option key={m.value} value={m.value.toString()}>
                  {m.label}
                </option>
              ))}
            </Select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Year
            </label>
            <Select
              value={year.toString()}
              onChange={(e) => setYear(parseInt(e.target.value))}
              disabled={isLoading}
              className="w-full"
            >
              {years.map((y) => (
                <option key={y} value={y.toString()}>
                  {y}
                </option>
              ))}
            </Select>
          </div>
        </div>

        {(error || localError) && (
          <div className="flex items-center gap-2 p-4 bg-red-50 rounded-lg border border-red-200">
            <AlertCircle className="w-4 h-4 text-red-600 flex-shrink-0" />
            <p className="text-sm text-red-700">{error || localError}</p>
          </div>
        )}

        {success && (
          <div className="flex items-center gap-2 p-4 bg-green-50 rounded-lg border border-green-200">
            <CheckCircle className="w-4 h-4 text-green-600 flex-shrink-0" />
            <p className="text-sm text-green-700">
              Report generated successfully at {successTime}
            </p>
          </div>
        )}

        <Button
          type="submit"
          disabled={isLoading || !customerId}
          className="w-full"
        >
          {isLoading ? (
            <span className="flex items-center gap-2">
              <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
              Generating...
            </span>
          ) : (
            'Generate Report'
          )}
        </Button>
      </form>
    </div>
  );
};
