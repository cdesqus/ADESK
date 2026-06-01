import React, { useState, useEffect } from 'react';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { QRCodeDisplay } from '@/components/QRCodeDisplay';
import { WhatsAppSession } from '@/types/whatsapp';

interface WhatsAppSessionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSessionCreated?: (session: WhatsAppSession) => void;
  isLoading?: boolean;
  error?: string;
  onCreate?: (name: string) => Promise<WhatsAppSession>;
}

type ModalStep = 'form' | 'qr' | 'waiting';

export const WhatsAppSessionModal: React.FC<WhatsAppSessionModalProps> = ({
  isOpen,
  onClose,
  onSessionCreated,
  isLoading = false,
  error,
  onCreate,
}) => {
  const [step, setStep] = useState<ModalStep>('form');
  const [sessionName, setSessionName] = useState('');
  const [session, setSession] = useState<WhatsAppSession | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [formError, setFormError] = useState('');

  useEffect(() => {
    if (!isOpen) {
      setStep('form');
      setSessionName('');
      setSession(null);
      setFormError('');
    }
  }, [isOpen]);

  const handleCreate = async () => {
    if (!sessionName.trim()) {
      setFormError('Session name is required');
      return;
    }

    if (!/^[a-zA-Z0-9_-]+$/.test(sessionName)) {
      setFormError('Session name must contain only alphanumeric characters, underscores, and hyphens');
      return;
    }

    setFormError('');
    setIsCreating(true);

    try {
      if (onCreate) {
        const newSession = await onCreate(sessionName);
        setSession(newSession);
        setStep('qr');
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create session');
    } finally {
      setIsCreating(false);
    }
  };

  const handleRefreshQR = async () => {
    if (!session || !onCreate) return;
    setStep('qr');
  };

  const handleDismiss = () => {
    if (session && onSessionCreated) {
      onSessionCreated(session);
    }
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-lg w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">
            {step === 'form' ? 'Create New WhatsApp Session' : 'Session QR Code'}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            aria-label="Close modal"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {step === 'form' && (
            <div className="space-y-4">
              <div>
                <label htmlFor="sessionName" className="block text-sm font-medium text-gray-700 mb-1">
                  Session Name
                </label>
                <Input
                  id="sessionName"
                  type="text"
                  placeholder="e.g., support, business"
                  value={sessionName}
                  onChange={(e) => {
                    setSessionName(e.target.value);
                    setFormError('');
                  }}
                  disabled={isCreating || isLoading}
                />
                <p className="text-xs text-gray-500 mt-1">
                  Alphanumeric characters, underscores, and hyphens only
                </p>
              </div>

              {formError && (
                <div className="p-3 bg-red-50 border border-red-200 rounded-md">
                  <p className="text-sm text-red-700">{formError}</p>
                </div>
              )}

              {error && (
                <div className="p-3 bg-red-50 border border-red-200 rounded-md">
                  <p className="text-sm text-red-700">{error}</p>
                </div>
              )}
            </div>
          )}

          {step === 'qr' && session && (
            <div className="space-y-4">
              <div className="text-center">
                <p className="text-sm text-gray-600 mb-4">
                  Session: <span className="font-semibold">{session.name}</span>
                </p>
              </div>
              <QRCodeDisplay
                qrCode={session.qrCode || null}
                isLoading={isLoading}
                error={error}
                onRefresh={handleRefreshQR}
              />
              <div className="p-3 bg-blue-50 border border-blue-200 rounded-md">
                <p className="text-xs text-blue-700">
                  Use your WhatsApp app to scan this QR code. The session will be marked as connected once scanned.
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex gap-3 p-6 border-t border-gray-200 bg-gray-50">
          <Button
            onClick={onClose}
            variant="outline"
            className="flex-1"
            disabled={isCreating || isLoading}
          >
            Cancel
          </Button>
          {step === 'form' && (
            <Button
              onClick={handleCreate}
              className="flex-1 bg-green-600 hover:bg-green-700 text-white"
              disabled={isCreating || isLoading}
            >
              {isCreating ? 'Creating...' : 'Create Session'}
            </Button>
          )}
          {step === 'qr' && (
            <Button
              onClick={handleDismiss}
              className="flex-1 bg-green-600 hover:bg-green-700 text-white"
              disabled={isLoading}
            >
              {isLoading ? 'Checking...' : 'Done'}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
};
