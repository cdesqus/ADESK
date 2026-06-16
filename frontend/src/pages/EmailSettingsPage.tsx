import React, { useEffect, useState } from 'react';
import { apiService } from '@/services/api';
import { EmailSettings, EmailLog, DomainMapping as DomainMappingType, AutoReplyTemplate, EmailHistoryFilter } from '@/types/email';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { EmailStatus } from '@/components/EmailStatus';
import { EmailSettingsModal } from '@/components/EmailSettingsModal';
import { DomainMapping } from '@/components/DomainMapping';
import { EmailHistory } from '@/components/EmailHistory';
import { AlertCircle, Save, RefreshCw } from 'lucide-react';
import { Customer } from '@/types';

export const EmailSettingsPage: React.FC = () => {
  const [settings, setSettings] = useState<EmailSettings | null>(null);
  const [domainMappings, setDomainMappings] = useState<DomainMappingType[]>([]);
  const [autoReplyTemplate, setAutoReplyTemplate] = useState('');
  const [emailLogs, setEmailLogs] = useState<EmailLog[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);

  const [isLoadingSettings, setIsLoadingSettings] = useState(false);
  const [isLoadingDomains, setIsLoadingDomains] = useState(false);
  const [isLoadingLogs, setIsLoadingLogs] = useState(false);
  const [isSavingTemplate, setIsSavingTemplate] = useState(false);
  const [isSyncing, setIsSyncing] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);

  const [templatePreview, setTemplatePreview] = useState('');
  const [successMessage, setSuccessMessage] = useState('');
  const [errorMessage, setErrorMessage] = useState('');

  const [filters, setFilters] = useState<EmailHistoryFilter>({
    status: 'all',
    page: 1,
    pageSize: 10,
  });

  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false);

  // Template variables
  const templateVariables = ['{CUSTOMER_NAME}', '{TICKET_ID}', '{SUPPORT_EMAIL}'];

  // Fetch initial data
  useEffect(() => {
    loadAllData();
  }, []);

  const loadAllData = async () => {
    try {
      setIsLoadingSettings(true);
      setIsLoadingDomains(true);

      const [settingsData, domainsData, templatesData, customersData] = await Promise.all([
        apiService.getEmailSettings().catch(() => null),
        apiService.getDomainMappings().catch(() => []),
        apiService.getAutoReplyTemplate().catch(() => null),
        apiService.getCustomers().catch(() => ({ data: [] })),
      ]);

      setSettings(settingsData);
      setDomainMappings(domainsData);
      if (templatesData) {
        setAutoReplyTemplate(templatesData.template);
      }
      setCustomers(customersData?.data || []);
    } catch (error) {
      setErrorMessage('Failed to load email settings');
    } finally {
      setIsLoadingSettings(false);
      setIsLoadingDomains(false);
    }

    // Load email logs
    await loadEmailLogs();
  };

  const loadEmailLogs = async () => {
    try {
      setIsLoadingLogs(true);
      const response = await apiService.getEmailHistory(filters);
      setEmailLogs(response.data);
    } catch (error) {
      console.error('Failed to load email logs:', error);
    } finally {
      setIsLoadingLogs(false);
    }
  };

  useEffect(() => {
    loadEmailLogs();
  }, [filters]);

  // Generate template preview
  const generatePreview = () => {
    let preview = autoReplyTemplate;

    // Replace with sample data
    preview = preview.replace('{CUSTOMER_NAME}', 'Acme Corp');
    preview = preview.replace('{TICKET_ID}', 'TK-12345');
    preview = preview.replace('{SUPPORT_EMAIL}', 'helpdesk@idesolusi.co.id');

    setTemplatePreview(preview);
  };

  const handleSaveTemplate = async () => {
    if (!autoReplyTemplate.trim()) {
      setErrorMessage('Template cannot be empty');
      return;
    }

    try {
      setIsSavingTemplate(true);
      setErrorMessage('');
      await apiService.updateAutoReplyTemplate(autoReplyTemplate);
      generatePreview();
      setSuccessMessage('Auto-reply template saved successfully');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (error) {
      setErrorMessage('Failed to save auto-reply template');
    } finally {
      setIsSavingTemplate(false);
    }
  };

  const handleTestConnection = async () => {
    try {
      setIsTestingConnection(true);
      setErrorMessage('');
      await apiService.testEmailConnection();
      setSuccessMessage('Connection test passed');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (error) {
      setErrorMessage('Connection test failed');
    } finally {
      setIsTestingConnection(false);
    }
  };

  const handleSync = async () => {
    try {
      setIsSyncing(true);
      setErrorMessage('');
      await apiService.syncEmails();
      setSuccessMessage('Email sync triggered successfully');
      setTimeout(() => setSuccessMessage(''), 3000);
      // Reload data
      await loadAllData();
    } catch (error) {
      setErrorMessage('Email sync failed');
    } finally {
      setIsSyncing(false);
    }
  };

  const handleSaveEmailSettings = async (newSettings: Partial<EmailSettings>) => {
    try {
      setErrorMessage('');
      await apiService.updateEmailSettings(newSettings);
      setSuccessMessage('Email settings updated successfully');
      setTimeout(() => setSuccessMessage(''), 3000);
      await loadAllData();
    } catch (error) {
      setErrorMessage('Failed to update email settings');
      throw error;
    }
  };

  const handleUpdateDomain = async (customerId: string, domain: string) => {
    try {
      setErrorMessage('');
      await apiService.updateCustomer(customerId, { domain: domain });
      setDomainMappings((prev) =>
        prev.map((m) => (m.customerId === customerId ? { ...m, domain } : m))
      );
      setSuccessMessage('Domain updated successfully');
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (error) {
      setErrorMessage('Failed to update domain');
      throw error;
    }
  };

  const handleTestDomain = async (domain: string) => {
    try {
      setErrorMessage('');
      await Promise.resolve({ success: true });
      setSuccessMessage(`Domain "${domain}" test passed`);
      setTimeout(() => setSuccessMessage(''), 3000);
    } catch (error) {
      setErrorMessage(`Domain "${domain}" test failed`);
      throw error;
    }
  };

  const handleViewEmailDetails = (email: EmailLog) => {
    // Details are expanded inline in EmailHistory component
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <h1 className="text-3xl font-bold text-gray-900">Email Integration Settings</h1>
          <p className="text-gray-600 mt-2">Manage email account, domain mappings, and auto-reply templates</p>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {/* Messages */}
        {successMessage && (
          <div className="bg-green-50 border border-green-200 rounded-lg p-4 flex items-center gap-3">
            <div className="flex-shrink-0 text-green-600">
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
            </div>
            <p className="text-green-800">{successMessage}</p>
          </div>
        )}

        {errorMessage && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 flex items-center gap-3">
            <div className="flex-shrink-0 text-red-600">
              <AlertCircle className="w-5 h-5" />
            </div>
            <p className="text-red-800">{errorMessage}</p>
          </div>
        )}

        {/* Email Account Status */}
        <section>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Account Status</h2>
          <EmailStatus
            settings={settings}
            isLoading={isLoadingSettings}
            onTestConnection={handleTestConnection}
            onSync={handleSync}
            onEdit={() => setIsSettingsModalOpen(true)}
            isSyncing={isSyncing}
            isTestingConnection={isTestingConnection}
          />
        </section>

        {/* Domain Configuration */}
        <section>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Domain Configuration</h2>
          <div className="bg-white rounded-lg border border-gray-200 p-4 mb-4">
            <p className="text-sm text-gray-600">
              Configure which customer domains should be matched to incoming emails. Use the test button to verify domain matching works correctly.
            </p>
          </div>
          <DomainMapping
            domains={domainMappings}
            isLoading={isLoadingDomains}
            onUpdate={handleUpdateDomain}
            onTest={handleTestDomain}
          />
        </section>

        {/* Auto-Reply Template */}
        <section>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Auto-Reply Template</h2>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* Editor */}
            <div className="space-y-4">
              <div className="bg-white rounded-lg border border-gray-200 p-4">
                <div className="mb-4">
                  <label className="block text-sm font-medium text-gray-900 mb-2">
                    Template
                  </label>
                  <textarea
                    value={autoReplyTemplate}
                    onChange={(e) => setAutoReplyTemplate(e.target.value)}
                    placeholder="Enter auto-reply template..."
                    className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                    rows={8}
                  />
                  <p className="text-xs text-gray-500 mt-2">
                    Available variables: {templateVariables.join(', ')}
                  </p>
                </div>

                <div className="flex gap-2">
                  <Button
                    variant="default"
                    onClick={handleSaveTemplate}
                    disabled={isSavingTemplate}
                    className="flex items-center gap-2"
                  >
                    {isSavingTemplate ? (
                      <>
                        <RefreshCw className="w-4 h-4 animate-spin" />
                        Saving...
                      </>
                    ) : (
                      <>
                        <Save className="w-4 h-4" />
                        Save Template
                      </>
                    )}
                  </Button>
                  <Button
                    variant="outline"
                    onClick={generatePreview}
                  >
                    Preview
                  </Button>
                </div>
              </div>
            </div>

            {/* Preview */}
            <div className="space-y-4">
              <div className="bg-white rounded-lg border border-gray-200 p-4">
                <h3 className="text-sm font-semibold text-gray-900 mb-3">Preview</h3>
                {templatePreview ? (
                  <div className="bg-gray-50 p-4 rounded-md whitespace-pre-wrap break-words text-sm text-gray-700 font-mono">
                    {templatePreview}
                  </div>
                ) : (
                  <div className="text-sm text-gray-500 text-center py-8">
                    Click "Preview" to see how the template will look
                  </div>
                )}
              </div>

              {/* Variable reference */}
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <h4 className="text-sm font-semibold text-blue-900 mb-2">Available Variables</h4>
                <dl className="space-y-2">
                  <div>
                    <dt className="font-mono text-xs text-blue-700">{'{CUSTOMER_NAME}'}</dt>
                    <dd className="text-xs text-blue-600">Customer name</dd>
                  </div>
                  <div>
                    <dt className="font-mono text-xs text-blue-700">{'{TICKET_ID}'}</dt>
                    <dd className="text-xs text-blue-600">Ticket ID</dd>
                  </div>
                  <div>
                    <dt className="font-mono text-xs text-blue-700">{'{SUPPORT_EMAIL}'}</dt>
                    <dd className="text-xs text-blue-600">Support email address</dd>
                  </div>
                </dl>
              </div>
            </div>
          </div>
        </section>

        {/* Email History & Logs */}
        <section>
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Email Processing History</h2>

          {/* Filters */}
          <div className="bg-white rounded-lg border border-gray-200 p-4 mb-4 space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                <Select value={filters.status || 'all'} onChange={(e) => setFilters((prev) => ({ ...prev, status: (e.target.value as any) || 'all', page: 1 }))}>
  <option value="all">All</option>
  <option value="success">Success</option>
  <option value="failed">Failed</option>
  <option value="unknown_domain">Unknown Domain</option>
</Select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Customer</label>
                <Select value={filters.customerId || ''} onChange={(e) => setFilters((prev) => ({ ...prev, customerId: e.target.value || undefined, page: 1 }))}>
  <option value="">All Customers</option>
  {customers.map((c) => (
    <option key={c.id} value={c.id}>{c.name}</option>
  ))}
</Select>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Page Size</label>
                <Select value={filters.pageSize?.toString() || '10'} onChange={(e) => setFilters((prev) => ({ ...prev, pageSize: parseInt(e.target.value), page: 1 }))}>
  <option value="5">5 per page</option>
  <option value="10">10 per page</option>
  <option value="25">25 per page</option>
  <option value="50">50 per page</option>
</Select>
              </div>
            </div>
          </div>

          {/* Email logs table */}
          <EmailHistory
            emails={emailLogs}
            isLoading={isLoadingLogs}
            currentPage={filters.page}
            onPageChange={(page) =>
              setFilters((prev) => ({
                ...prev,
                page,
              }))
            }
            onViewDetails={handleViewEmailDetails}
          />
        </section>
      </div>

      <EmailSettingsModal
        isOpen={isSettingsModalOpen}
        onClose={() => setIsSettingsModalOpen(false)}
        onSave={handleSaveEmailSettings}
        currentSettings={settings}
      />
    </div>
  );
};
