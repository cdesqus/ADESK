import React from 'react';
import { Mail, CheckCircle2, AlertCircle, RefreshCw } from 'lucide-react';
import { EmailSettings } from '@/types/email';
import { Button } from '@/components/ui/Button';
import { formatDistanceToNow } from 'date-fns';

interface EmailStatusProps {
  settings: EmailSettings | null;
  isLoading?: boolean;
  onTestConnection?: () => void;
  onSync?: () => void;
  onEdit?: () => void;
  isSyncing?: boolean;
  isTestingConnection?: boolean;
}

export const EmailStatus: React.FC<EmailStatusProps> = ({
  settings,
  isLoading,
  onTestConnection,
  onSync,
  onEdit,
  isSyncing,
  isTestingConnection,
}) => {
  const isConnected = settings?.status === 'connected';
  const lastSyncDate = settings?.lastSync ? new Date(settings.lastSync) : null;
  const lastSyncText = lastSyncDate
    ? formatDistanceToNow(lastSyncDate, { addSuffix: true })
    : 'Never';

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 bg-gray-200 rounded-full animate-pulse" />
          <div className="flex-1">
            <div className="h-4 w-48 bg-gray-200 rounded animate-pulse mb-2" />
            <div className="h-3 w-32 bg-gray-100 rounded animate-pulse" />
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-6">
      <div className="space-y-4">
        <div className="flex items-start justify-between">
          <div className="flex items-start gap-4">
            <div className={`mt-1 p-2 rounded-lg ${isConnected ? 'bg-green-50' : 'bg-red-50'}`}>
              {isConnected ? (
                <CheckCircle2 className="w-6 h-6 text-green-600" />
              ) : (
                <AlertCircle className="w-6 h-6 text-red-600" />
              )}
            </div>
            <div>
              <h3 className="text-sm font-semibold text-gray-900">Email Account</h3>
              <p className="text-lg font-medium text-gray-900 mt-1">
                {settings?.username || 'Not configured'}
              </p>
              <p className="text-sm text-gray-600 mt-2">
                Status:{' '}
                <span
                  className={`font-medium ${isConnected ? 'text-green-600' : 'text-red-600'}`}
                >
                  {isConnected ? 'Connected' : 'Disconnected'}
                </span>
              </p>
              <p className="text-sm text-gray-600 mt-1">
                Last sync: <span className="font-medium text-gray-900">{lastSyncText}</span>
              </p>
            </div>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={onEdit}
              className="flex items-center gap-2"
            >
              <Mail className="w-4 h-4" />
              Edit Settings
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={onTestConnection}
              disabled={isTestingConnection || !isConnected}
              className="flex items-center gap-2"
            >
              {isTestingConnection ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  Testing...
                </>
              ) : (
                <>
                  <RefreshCw className="w-4 h-4" />
                  Test Connection
                </>
              )}
            </Button>
            <Button
              variant="default"
              size="sm"
              onClick={onSync}
              disabled={isSyncing || !isConnected}
              className="flex items-center gap-2"
            >
              {isSyncing ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  Syncing...
                </>
              ) : (
                <>
                  <RefreshCw className="w-4 h-4" />
                  Manual Sync
                </>
              )}
            </Button>
          </div>
        </div>

        <div className="pt-4 border-t border-gray-100">
          <p className="text-xs text-gray-500">
            Polling interval: <span className="font-medium">{settings?.pollingInterval || 'N/A'}</span>
          </p>
        </div>
      </div>
    </div>
  );
};
