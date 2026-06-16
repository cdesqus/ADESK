import React, { useEffect, useState } from 'react';
import { apiService } from '@/services/api';
import { WhatsAppSession, EngineerWAPhone, WAHookStatus } from '@/types/whatsapp';
import { Engineer } from '@/types';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { WhatsAppSessionModal } from '@/components/WhatsAppSessionModal';
import { SessionStatus } from '@/components/SessionStatus';
import { EngineerPhoneForm } from '@/components/EngineerPhoneForm';
import { AlertCircle, RefreshCw, Trash2, Phone, Check, X, Smartphone } from 'lucide-react';

export const WhatsAppSettingsPage: React.FC = () => {
  const [sessions, setSessions] = useState<WhatsAppSession[]>([]);
  const [engineers, setEngineers] = useState<Engineer[]>([]);
  const [engineerPhones, setEngineerPhones] = useState<EngineerWAPhone[]>([]);
  const [hookStatus, setHookStatus] = useState<WAHookStatus | null>(null);

  const [isLoadingSessions, setIsLoadingSessions] = useState(false);
  const [isLoadingEngineers, setIsLoadingEngineers] = useState(false);
  const [isLoadingPhones, setIsLoadingPhones] = useState(false);
  const [isLoadingHook, setIsLoadingHook] = useState(false);

  const [showSessionModal, setShowSessionModal] = useState(false);
  const [showPhoneForm, setShowPhoneForm] = useState(false);
  const [editingPhoneId, setEditingPhoneId] = useState<string | null>(null);

  const [successMessage, setSuccessMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');
  const [isTestingMessage, setIsTestingMessage] = useState<string | null>(null);
  const [selectedSession, setSelectedSession] = useState<WhatsAppSession | undefined>(undefined);

  // Fetch all data on mount
  useEffect(() => {
    loadAllData();
  }, []);

  const loadAllData = async () => {
    try {
      await Promise.all([
        loadSessions(),
        loadEngineers(),
        loadEngineerPhones(),
        loadHookStatus(),
      ]);
    } catch (err) {
      setErrorMessage('Failed to load WhatsApp settings');
    }
  };

  const loadSessions = async () => {
    setIsLoadingSessions(true);
    try {
      const data = await apiService.getWASessions();
      setSessions(data);
    } catch (err) {
      console.error('Failed to load sessions:', err);
    } finally {
      setIsLoadingSessions(false);
    }
  };

  const loadEngineers = async () => {
    setIsLoadingEngineers(true);
    try {
      const data = await apiService.getEngineers(1, 100);
      setEngineers(data.data);
    } catch (err) {
      console.error('Failed to load engineers:', err);
    } finally {
      setIsLoadingEngineers(false);
    }
  };

  const loadEngineerPhones = async () => {
    setIsLoadingPhones(true);
    try {
      const data = await apiService.getEngineerPhones();
      setEngineerPhones(data);
    } catch (err) {
      console.error('Failed to load engineer phones:', err);
    } finally {
      setIsLoadingPhones(false);
    }
  };

  const loadHookStatus = async () => {
    setIsLoadingHook(true);
    try {
      const data = await apiService.getWAHookStatus();
      setHookStatus(data);
    } catch (err) {
      console.error('Failed to load webhook status:', err);
    } finally {
      setIsLoadingHook(false);
    }
  };

  const handleCreateSession = async (name: string): Promise<WhatsAppSession> => {
    const session = await apiService.createWASession(name);
    return session;
  };

  const handleSessionCreated = async (session: WhatsAppSession) => {
    if (sessions.some((s) => s.id === session.id)) {
      setSessions(sessions.map((s) => (s.id === session.id ? session : s)));
      setSuccessMessage('WhatsApp session updated successfully');
    } else {
      setSessions([...sessions, session]);
      setSuccessMessage('WhatsApp session created successfully');
    }
    setShowSessionModal(false);
    setSelectedSession(undefined);
    setTimeout(() => setSuccessMessage(''), 3000);
  };

  const handleDeleteSession = async (sessionId: string) => {
    if (!confirm('Are you sure you want to delete this session?')) return;

    try {
      await apiService.deleteSession(sessionId);
      setSessions(sessions.filter((s) => s.id !== sessionId));
      setSuccessMessage('Session deleted successfully');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : 'Failed to delete session');
    }
  };

  const handleAddPhone = async (data: {
    engineerId: string;
    phoneNumber: string;
    groupId?: string;
  }) => {
    try {
      const result = await apiService.assignPhoneToEngineer(
        data.engineerId,
        data.phoneNumber,
        data.groupId
      );
      setEngineerPhones([...engineerPhones, result]);
      setShowPhoneForm(false);
      setSuccessMessage('Engineer phone assigned successfully');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : 'Failed to assign phone');
    }
  };

  const handleUpdatePhone = async (data: {
    engineerId: string;
    phoneNumber: string;
    groupId?: string;
  }) => {
    if (!editingPhoneId) return;

    try {
      const result = await apiService.updateEngineerPhone(
        editingPhoneId,
        data.phoneNumber,
        data.groupId
      );
      setEngineerPhones(
        engineerPhones.map((p) => (p.id === editingPhoneId ? result : p))
      );
      setShowPhoneForm(false);
      setEditingPhoneId(null);
      setSuccessMessage('Engineer phone updated successfully');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : 'Failed to update phone');
    }
  };

  const handleDeletePhone = async (phoneId: string) => {
    if (!confirm('Are you sure you want to delete this phone assignment?')) return;

    try {
      await apiService.deleteEngineerPhone(phoneId);
      setEngineerPhones(engineerPhones.filter((p) => p.id !== phoneId));
      setSuccessMessage('Phone assignment deleted successfully');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : 'Failed to delete phone');
    }
  };

  const handleTestMessage = async (phoneId: string) => {
    setIsTestingMessage(phoneId);
    try {
      const phone = engineerPhones.find((p) => p.id === phoneId);
      if (!phone || !sessions.length) {
        throw new Error('No valid session or phone found');
      }

      const result = await apiService.testWAMessage(
        sessions[0].id,
        phone.phoneNumber,
        'Test message from AI-DESK'
      );

      if (result.success) {
        setSuccessMessage(`Test message sent to ${phone.phoneNumber}`);
      } else {
        setErrorMessage(result.message);
      }
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setErrorMessage(err instanceof Error ? err.message : 'Failed to send test message');
    } finally {
      setIsTestingMessage(null);
    }
  };

  const handleTestWebhook = async () => {
    setIsLoadingHook(true);
    try {
      await loadHookStatus();
      setSuccessMessage('Webhook status refreshed');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (err) {
      setErrorMessage('Failed to refresh webhook status');
    } finally {
      setIsLoadingHook(false);
    }
  };

  const editingPhone = editingPhoneId
    ? engineerPhones.find((p) => p.id === editingPhoneId)
    : null;

  return (
    <div className="space-y-8 pb-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900">WhatsApp Integration Settings</h1>
        <p className="mt-2 text-gray-600">Manage WhatsApp sessions, engineer phone numbers, and webhook integration</p>
      </div>

      {/* Success/Error Messages */}
      {successMessage && (
        <div className="p-4 bg-green-50 border border-green-200 rounded-lg flex gap-3">
          <Check className="w-5 h-5 text-green-600 flex-shrink-0 mt-0.5" />
          <p className="text-sm text-green-700">{successMessage}</p>
        </div>
      )}

      {errorMessage && (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg flex gap-3">
          <AlertCircle className="w-5 h-5 text-red-600 flex-shrink-0 mt-0.5" />
          <p className="text-sm text-red-700">{errorMessage}</p>
        </div>
      )}

      {/* Active Sessions Section */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Active Sessions</h2>
            <Button
              onClick={() => setShowSessionModal(true)}
              className="bg-green-600 hover:bg-green-700 text-white"
            >
              + Add WhatsApp Session
            </Button>
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Session Name</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Phone Number</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Status</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {isLoadingSessions ? (
                <tr>
                  <td colSpan={4} className="px-6 py-12 text-center">
                    <div className="inline-block w-6 h-6 border-3 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
                    <p className="mt-2 text-sm text-gray-600">Loading sessions...</p>
                  </td>
                </tr>
              ) : sessions.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-6 py-12 text-center text-gray-500">
                    No active sessions. Click "Add WhatsApp Session" to create one.
                  </td>
                </tr>
              ) : (
                sessions.map((session) => (
                  <tr key={session.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4 text-sm font-medium text-gray-900">{session.session_name}</td>
                    <td className="px-6 py-4 text-sm text-gray-600">{session.phoneNumber || '-'}</td>
                    <td className="px-6 py-4">
                      <SessionStatus status={session.status} phoneNumber={session.phoneNumber} />
                    </td>
                    <td className="px-6 py-4 text-sm flex gap-2">
                      {session.status.toLowerCase() === 'pending' && (
                        <Button
                          onClick={() => {
                            setSelectedSession(session);
                            setShowSessionModal(true);
                          }}
                          variant="outline"
                          size="sm"
                          className="text-green-600 hover:bg-green-50"
                        >
                          <Smartphone className="w-4 h-4 mr-1" />
                          Link Device
                        </Button>
                      )}
                      <Button
                        onClick={() => handleDeleteSession(session.id)}
                        variant="outline"
                        size="sm"
                        className="text-red-600 hover:bg-red-50"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Engineer Phone Numbers Section */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-gray-900">Engineer Phone Numbers</h2>
            <Button
              onClick={() => {
                setEditingPhoneId(null);
                setShowPhoneForm(true);
              }}
              className="bg-green-600 hover:bg-green-700 text-white"
            >
              + Assign Phone
            </Button>
          </div>
        </div>

        {showPhoneForm && (
          <div className="p-6 border-b border-gray-200 bg-gray-50">
            <h3 className="font-semibold text-gray-900 mb-4">
              {editingPhoneId ? 'Edit Phone Assignment' : 'Assign Phone to Engineer'}
            </h3>
            <EngineerPhoneForm
              engineers={engineers}
              isLoading={isLoadingEngineers}
              onSubmit={editingPhoneId ? handleUpdatePhone : handleAddPhone}
              onCancel={() => {
                setShowPhoneForm(false);
                setEditingPhoneId(null);
              }}
              initialData={
                editingPhone
                  ? {
                      engineerId: editingPhone.engineerId,
                      phoneNumber: editingPhone.phoneNumber,
                      groupId: editingPhone.groupId,
                    }
                  : undefined
              }
            />
          </div>
        )}

        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Engineer</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Phone Number</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Group ID</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Active</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-700 uppercase">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {isLoadingPhones ? (
                <tr>
                  <td colSpan={5} className="px-6 py-12 text-center">
                    <div className="inline-block w-6 h-6 border-3 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
                    <p className="mt-2 text-sm text-gray-600">Loading phones...</p>
                  </td>
                </tr>
              ) : engineerPhones.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-6 py-12 text-center text-gray-500">
                    No engineer phones assigned yet.
                  </td>
                </tr>
              ) : (
                engineerPhones.map((phone) => (
                  <tr key={phone.id} className="hover:bg-gray-50 transition-colors">
                    <td className="px-6 py-4 text-sm font-medium text-gray-900">{phone.engineerName}</td>
                    <td className="px-6 py-4 text-sm text-gray-600">{phone.phoneNumber}</td>
                    <td className="px-6 py-4 text-sm text-gray-600">{phone.groupId || '-'}</td>
                    <td className="px-6 py-4">
                      {phone.isActive ? (
                        <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                          <Check className="w-3 h-3" />
                          Active
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium bg-red-100 text-red-800">
                          <X className="w-3 h-3" />
                          Inactive
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 text-sm flex gap-2">
                      <Button
                        onClick={() => handleTestMessage(phone.id)}
                        variant="outline"
                        size="sm"
                        disabled={isTestingMessage === phone.id}
                        className="text-blue-600 hover:bg-blue-50"
                      >
                        <Phone className="w-4 h-4" />
                      </Button>
                      <Button
                        onClick={() => {
                          setEditingPhoneId(phone.id);
                          setShowPhoneForm(true);
                        }}
                        variant="outline"
                        size="sm"
                      >
                        Edit
                      </Button>
                      <Button
                        onClick={() => handleDeletePhone(phone.id)}
                        variant="outline"
                        size="sm"
                        className="text-red-600 hover:bg-red-50"
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Webhook Status Section */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-sm p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-gray-900">Webhook Status</h2>
          <Button
            onClick={handleTestWebhook}
            variant="outline"
            disabled={isLoadingHook}
          >
            <RefreshCw className={`w-4 h-4 ${isLoadingHook ? 'animate-spin' : ''}`} />
          </Button>
        </div>

        {hookStatus ? (
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              {hookStatus.status === 'connected' ? (
                <>
                  <Check className="w-5 h-5 text-green-600" />
                  <span className="text-sm font-medium text-green-700">Connected</span>
                </>
              ) : (
                <>
                  <AlertCircle className="w-5 h-5 text-red-600" />
                  <span className="text-sm font-medium text-red-700">Disconnected</span>
                </>
              )}
            </div>

            {hookStatus.lastMessageAt && (
              <div className="text-sm text-gray-600">
                Last message received: {new Date(hookStatus.lastMessageAt).toLocaleString()}
              </div>
            )}

            {hookStatus.errorLog && hookStatus.errorLog.length > 0 && (
              <div className="mt-4 p-4 bg-red-50 border border-red-200 rounded-md">
                <h3 className="text-sm font-semibold text-red-900 mb-2">Error Log</h3>
                <ul className="space-y-1">
                  {hookStatus.errorLog.slice(-5).map((error, idx) => (
                    <li key={idx} className="text-xs text-red-700">
                      {error}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        ) : (
          <div className="text-center py-6">
            <div className="inline-block w-6 h-6 border-3 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
            <p className="mt-2 text-sm text-gray-600">Loading status...</p>
          </div>
        )}
      </div>

      {/* Session Modal */}
      <WhatsAppSessionModal
        isOpen={showSessionModal}
        onClose={() => {
          setShowSessionModal(false);
          setSelectedSession(undefined);
        }}
        onSessionCreated={handleSessionCreated}
        onCreate={handleCreateSession}
        existingSession={selectedSession}
      />
    </div>
  );
};
