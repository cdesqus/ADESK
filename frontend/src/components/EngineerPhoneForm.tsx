import React, { useState, useEffect } from 'react';
import { AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { Engineer } from '@/types';

interface EngineerPhoneFormProps {
  engineers: Engineer[];
  isLoading?: boolean;
  error?: string;
  onSubmit?: (data: {
    engineerId: string;
    phoneNumber: string;
    groupId?: string;
  }) => Promise<void>;
  onCancel?: () => void;
  initialData?: {
    engineerId: string;
    phoneNumber: string;
    groupId?: string;
  };
}

export const EngineerPhoneForm: React.FC<EngineerPhoneFormProps> = ({
  engineers,
  isLoading = false,
  error,
  onSubmit,
  onCancel,
  initialData,
}) => {
  const [engineerId, setEngineerId] = useState(initialData?.engineerId || '');
  const [phoneNumber, setPhoneNumber] = useState(initialData?.phoneNumber || '');
  const [groupId, setGroupId] = useState(initialData?.groupId || '');
  const [formError, setFormError] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const validatePhoneNumber = (phone: string): boolean => {
    return /^\+?[1-9]\d{1,14}$/.test(phone.replace(/\D/g, ''));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError('');

    if (!engineerId) {
      setFormError('Please select an engineer');
      return;
    }

    if (!phoneNumber.trim()) {
      setFormError('Phone number is required');
      return;
    }

    if (!validatePhoneNumber(phoneNumber)) {
      setFormError('Invalid phone number format. Use format like +62xxx or numbers only');
      return;
    }

    setIsSubmitting(true);
    try {
      if (onSubmit) {
        await onSubmit({
          engineerId,
          phoneNumber,
          groupId: groupId || undefined,
        });
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save phone number');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="engineer" className="block text-sm font-medium text-gray-700 mb-1">
          Engineer
        </label>
        <Select
          id="engineer"
          value={engineerId}
          onChange={(e) => setEngineerId(e.target.value)}
          disabled={isSubmitting || isLoading}
        >
          <option value="">Select an engineer</option>
          {engineers.map((eng) => (
            <option key={eng.id} value={eng.id}>
              {eng.name}
            </option>
          ))}
        </Select>
      </div>

      <div>
        <label htmlFor="phone" className="block text-sm font-medium text-gray-700 mb-1">
          Phone Number
        </label>
        <Input
          id="phone"
          type="tel"
          placeholder="+62xxx or numbers only"
          value={phoneNumber}
          onChange={(e) => setPhoneNumber(e.target.value)}
          disabled={isSubmitting || isLoading}
        />
        <p className="text-xs text-gray-500 mt-1">International format recommended (e.g., +62812345678)</p>
      </div>

      <div>
        <label htmlFor="groupId" className="block text-sm font-medium text-gray-700 mb-1">
          Group ID (Optional)
        </label>
        <Input
          id="groupId"
          type="text"
          placeholder="e.g., group-123"
          value={groupId}
          onChange={(e) => setGroupId(e.target.value)}
          disabled={isSubmitting || isLoading}
        />
      </div>

      {(formError || error) && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-md flex gap-2">
          <AlertCircle className="w-4 h-4 text-red-600 flex-shrink-0 mt-0.5" />
          <p className="text-sm text-red-700">{formError || error}</p>
        </div>
      )}

      <div className="flex gap-3 pt-4">
        <Button
          type="button"
          onClick={onCancel}
          variant="outline"
          className="flex-1"
          disabled={isSubmitting || isLoading}
        >
          Cancel
        </Button>
        <Button
          type="submit"
          className="flex-1 bg-green-600 hover:bg-green-700 text-white"
          disabled={isSubmitting || isLoading}
        >
          {isSubmitting ? 'Saving...' : 'Save Phone'}
        </Button>
      </div>
    </form>
  );
};
