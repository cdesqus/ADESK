import React, { useState, useEffect } from 'react';
import { X } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { WhatsAppSession } from '@/types/whatsapp';
import { apiService } from '@/services/api';

interface WhatsAppSessionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSessionCreated?: (session: WhatsAppSession) => void;
  isLoading?: boolean;
  error?: string;
  onCreate?: (name: string) => Promise<WhatsAppSession>;
  existingSession?: WhatsAppSession;
}

type ModalStep = 'form' | 'qr' | 'waiting';

export const WhatsAppSessionModal: React.FC<WhatsAppSessionModalProps> = ({
  isOpen,
  onClose,
  onSessionCreated,
  isLoading = false,
  error,
  onCreate,
  existingSession,
}) => {
  const [step, setStep] = useState<ModalStep>('form');
  const [sessionName, setSessionName] = useState('');
  const [session, setSession] = useState<WhatsAppSession | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [formError, setFormError] = useState('');
  
  const [phoneNumber, setPhoneNumber] = useState('');
  const [pairingCode, setPairingCode] = useState('');
  const [isRequestingPairingCode, setIsRequestingPairingCode] = useState(false);

  const [qrCode, setQrCode] = useState('');
  const [isLoadingQR, setIsLoadingQR] = useState(false);

  useEffect(() => {
    if (!isOpen) {
      setStep('form');
      setSessionName('');
      setSession(null);
      setFormError('');
      setPhoneNumber('');
      setPairingCode('');
      setQrCode('');
    } else if (existingSession) {
      setSession(existingSession);
      setStep('qr');
      fetchQRCode(existingSession.id);
    }
  }, [isOpen, existingSession]);

  const fetchQRCode = async (sessionId: string) => {
    setIsLoadingQR(true);
    setFormError('');
    try {
      const res = await apiService.getSessionQR(sessionId);
      if (res.qrCode) {
        setQrCode(res.qrCode);
      } else {
        setFormError('Failed to load QR code. Session might be starting.');
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to fetch QR code');
    } finally {
      setIsLoadingQR(false);
    }
  };

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
        fetchQRCode(newSession.id);
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create session');
    } finally {
      setIsCreating(false);
    }
  };

  const handleRefreshQR = async () => {
    if (!session) return;
    setPairingCode('');
    fetchQRCode(session.id);
  };

  const handleRequestPairingCode = async () => {
    if (!phoneNumber || !session) return;
    setIsRequestingPairingCode(true);
    setFormError('');
    try {
      const res = await apiService.requestPairingCode(session.id, phoneNumber);
      setPairingCode(res.pairingCode);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to request pairing code');
    } finally {
      setIsRequestingPairingCode(false);
    }
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
            <div className="space-y-6">
              <div className="text-center">
                <p className="text-sm text-gray-600 mb-2">
                  Session: <span className="font-semibold">{session.session_name}</span>
                </p>
                <div className="mb-6 flex flex-col items-center justify-center">
                  {isLoadingQR ? (
                    <div className="w-64 h-64 bg-gray-100 rounded-lg flex items-center justify-center animate-pulse">
                      <div className="w-8 h-8 border-3 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
                    </div>
                  ) : qrCode ? (
                    <div className="border-4 border-white shadow-lg rounded-lg p-2 bg-white inline-block">
                      {qrCode.startsWith('data:image') ? (
                        <img src={qrCode} alt="WhatsApp QR Code" className="w-64 h-64 object-contain" />
                      ) : (
                        <div dangerouslySetInnerHTML={{ __html: qrCode }} className="w-64 h-64" />
                      )}
                    </div>
                  ) : (
                    <div className="w-64 h-64 bg-gray-50 rounded-lg border-2 border-dashed border-gray-300 flex flex-col items-center justify-center p-4">
                      <p className="text-sm text-gray-500 text-center mb-4">
                        QR Code not available yet
                      </p>
                    </div>
                  )}
                  <Button
                    onClick={handleRefreshQR}
                    variant="outline"
                    size="sm"
                    className="mt-4"
                    disabled={isLoadingQR}
                  >
                    Refresh QR Code
                  </Button>
                </div>
              </div>

              {/* Pairing Code Section */}
              <div className="border border-gray-200 rounded-lg p-4">
                <h4 className="text-sm font-medium text-gray-900 mb-3 text-center">Use Pairing Code</h4>
                <div className="space-y-3">
                  {pairingCode ? (
                    <div className="text-center">
                      <p className="text-xs text-gray-600 mb-2">Your pairing code is:</p>
                      <div className="text-2xl font-bold tracking-widest text-primary-600 bg-primary-50 py-3 rounded-lg border border-primary-100">
                        {pairingCode}
                      </div>
                      <p className="text-xs text-gray-500 mt-2">Enter this code in your WhatsApp linked devices screen.</p>
                    </div>
                  ) : (
                    <>
                      <Input
                        type="tel"
                        placeholder="Phone Number (e.g., 628123456789)"
                        value={phoneNumber}
                        onChange={(e) => setPhoneNumber(e.target.value)}
                        disabled={isRequestingPairingCode}
                      />
                      <Button
                        onClick={handleRequestPairingCode}
                        disabled={!phoneNumber || isRequestingPairingCode}
                        className="w-full"
                      >
                        {isRequestingPairingCode ? 'Requesting...' : 'Get Pairing Code'}
                      </Button>
                      <p className="text-xs text-gray-500 text-center">
                        Include country code without '+' sign.
                      </p>
                    </>
                  )}
                  {formError && (
                    <p className="text-sm text-red-600 text-center">{formError}</p>
                  )}
                </div>
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
