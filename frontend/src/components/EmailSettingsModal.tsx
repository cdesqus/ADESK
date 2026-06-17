import React, { useState, useEffect } from 'react';
import { EmailSettings } from '@/types/email';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { X, Mail } from 'lucide-react';

interface EmailSettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (settings: Partial<EmailSettings>) => Promise<void>;
  currentSettings: EmailSettings | null;
}

export const EmailSettingsModal: React.FC<EmailSettingsModalProps> = ({
  isOpen,
  onClose,
  onSave,
  currentSettings,
}) => {
  const [formData, setFormData] = useState({
    host: '',
    port: '',
    username: '',
    password: '',
    pollingInterval: '5',
  });
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen && currentSettings) {
      setFormData({
        host: currentSettings.host || '',
        port: currentSettings.port?.toString() || '',
        username: currentSettings.username || '',
        password: '', // Don't populate password for security
        pollingInterval: currentSettings.pollingInterval?.toString() || '5',
      });
      setError('');
    }
  }, [isOpen, currentSettings]);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.host || !formData.port || !formData.username) {
      setError('Host, Port, and Username are required');
      return;
    }

    try {
      setIsSaving(true);
      setError('');
      
      const payload: Partial<EmailSettings> = {
        host: formData.host,
        port: formData.port.toString(),
        username: formData.username,
        pollingInterval: parseInt(formData.pollingInterval, 10),
      };
      
      if (formData.password) {
        payload.password = formData.password;
      }
      
      await onSave(payload);
      onClose();
    } catch (err) {
      setError('Failed to save email settings');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md overflow-hidden">
        <div className="flex items-center justify-between p-6 border-b border-gray-100">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-primary-50 text-primary-600 rounded-lg">
              <Mail className="w-5 h-5" />
            </div>
            <h2 className="text-xl font-semibold text-gray-900">Email Configuration</h2>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-500 transition-colors"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="p-3 text-sm text-red-600 bg-red-50 border border-red-100 rounded-lg">
              {error}
            </div>
          )}

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                IMAP Host
              </label>
              <Input
                value={formData.host}
                onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                placeholder="imap.example.com"
                required
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  IMAP Port
                </label>
                <Input
                  value={formData.port}
                  onChange={(e) => setFormData({ ...formData, port: e.target.value })}
                  placeholder="993"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Polling Interval (min)
                </label>
                <Input
                  type="number"
                  min="1"
                  max="60"
                  value={formData.pollingInterval}
                  onChange={(e) => setFormData({ ...formData, pollingInterval: e.target.value })}
                  required
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Email / Username
              </label>
              <Input
                type="email"
                value={formData.username}
                onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                placeholder="support@example.com"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Password / App Password
              </label>
              <Input
                type="password"
                value={formData.password}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                placeholder={currentSettings?.isConfigured ? 'Leave blank to keep unchanged' : 'Enter password'}
                required={!currentSettings?.isConfigured}
              />
              <p className="mt-1 text-xs text-gray-500">
                For Gmail or Microsoft 365, use an App Password.
              </p>
            </div>
          </div>

          <div className="flex justify-end gap-3 mt-6 pt-4 border-t border-gray-100">
            <Button
              type="button"
              variant="outline"
              onClick={onClose}
              disabled={isSaving}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={isSaving}
              className="min-w-[100px]"
            >
              {isSaving ? 'Saving...' : 'Save Settings'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};
