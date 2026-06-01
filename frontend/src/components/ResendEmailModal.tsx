import React, { useState, useEffect } from 'react';
import { X, CheckCircle, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';

interface ResendEmailModalProps {
  isOpen: boolean;
  onClose: () => void;
  onResend: (additionalEmails?: string[]) => Promise<void>;
  customerEmail?: string;
  isLoading?: boolean;
}

export const ResendEmailModal: React.FC<ResendEmailModalProps> = ({
  isOpen,
  onClose,
  onResend,
  customerEmail,
  isLoading = false,
}) => {
  const [additionalEmails, setAdditionalEmails] = useState('');
  const [sendToTeam, setSendToTeam] = useState(true);
  const [sending, setSending] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isOpen) {
      setAdditionalEmails('');
      setSendToTeam(true);
      setSuccess(false);
      setError('');
    }
  }, [isOpen]);

  const validateEmails = (emails: string): boolean => {
    if (!emails.trim()) return true;
    const emailList = emails.split(',').map((e) => e.trim());
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailList.every((email) => emailRegex.test(email));
  };

  const handleResend = async () => {
    setError('');

    if (!validateEmails(additionalEmails)) {
      setError('Please enter valid email addresses (comma-separated)');
      return;
    }

    setSending(true);

    try {
      const emails = additionalEmails
        .split(',')
        .map((e) => e.trim())
        .filter((e) => e);

      await onResend(emails.length > 0 ? emails : undefined);

      setSuccess(true);
      setTimeout(() => {
        onClose();
      }, 2000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resend email');
    } finally {
      setSending(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-lg max-w-md w-full">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Resend Report Email</h2>
          <button
            onClick={onClose}
            disabled={sending || isLoading}
            className="text-gray-400 hover:text-gray-600 transition-colors disabled:opacity-50"
            aria-label="Close"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4">
          {/* Primary Recipient */}
          {customerEmail && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Primary Recipient (Read-only)
              </label>
              <div className="p-3 bg-gray-50 rounded-lg border border-gray-200">
                <p className="text-sm text-gray-700">{customerEmail}</p>
              </div>
            </div>
          )}

          {/* Send to Team Checkbox */}
          <label className="flex items-center gap-3 cursor-pointer">
            <input
              type="checkbox"
              checked={sendToTeam}
              onChange={(e) => setSendToTeam(e.target.checked)}
              disabled={sending || isLoading}
              className="rounded border-gray-300"
            />
            <span className="text-sm text-gray-700">Send to Support Team</span>
          </label>

          {/* Additional Emails */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Additional Emails (Optional)
            </label>
            <Input
              type="text"
              value={additionalEmails}
              onChange={(e) => setAdditionalEmails(e.target.value)}
              placeholder="email1@example.com, email2@example.com"
              disabled={sending || isLoading}
              className="w-full"
            />
            <p className="text-xs text-gray-500 mt-1">Separate multiple emails with commas</p>
          </div>

          {/* Success Message */}
          {success && (
            <div className="flex items-center gap-2 p-4 bg-green-50 rounded-lg border border-green-200">
              <CheckCircle className="w-4 h-4 text-green-600 flex-shrink-0" />
              <p className="text-sm text-green-700">Email sent successfully!</p>
            </div>
          )}

          {/* Error Message */}
          {error && (
            <div className="flex items-center gap-2 p-4 bg-red-50 rounded-lg border border-red-200">
              <AlertCircle className="w-4 h-4 text-red-600 flex-shrink-0" />
              <p className="text-sm text-red-700">{error}</p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex gap-3 p-6 border-t border-gray-200">
          <Button
            variant="outline"
            onClick={onClose}
            className="flex-1"
            disabled={sending || isLoading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleResend}
            className="flex-1"
            disabled={sending || isLoading}
          >
            {sending ? (
              <span className="flex items-center gap-2">
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                Sending...
              </span>
            ) : (
              'Send Email'
            )}
          </Button>
        </div>
      </div>
    </div>
  );
};
