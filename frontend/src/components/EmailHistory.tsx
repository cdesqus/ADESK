import React, { useState } from 'react';
import { EmailLog } from '@/types/email';
import { Button } from '@/components/ui/Button';
import { Eye, ChevronLeft, ChevronRight, AlertCircle, CheckCircle2 } from 'lucide-react';
import { formatDistanceToNow } from 'date-fns';

interface EmailHistoryProps {
  emails: EmailLog[];
  isLoading?: boolean;
  currentPage?: number;
  totalPages?: number;
  onPageChange?: (page: number) => void;
  onViewDetails?: (email: EmailLog) => void;
}

const statusConfig = {
  success: { icon: CheckCircle2, color: 'text-green-600', bg: 'bg-green-50', label: 'Success' },
  failed: { icon: AlertCircle, color: 'text-red-600', bg: 'bg-red-50', label: 'Failed' },
  unknown_domain: { icon: AlertCircle, color: 'text-yellow-600', bg: 'bg-yellow-50', label: 'Unknown Domain' },
};

export const EmailHistory: React.FC<EmailHistoryProps> = ({
  emails,
  isLoading,
  currentPage = 1,
  totalPages = 1,
  onPageChange,
  onViewDetails,
}) => {
  const [expandedId, setExpandedId] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="p-6">
          <div className="space-y-3">
            {[...Array(5)].map((_, i) => (
              <div key={i} className="h-16 bg-gray-100 rounded animate-pulse" />
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
                From
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                Subject
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                Status
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                Ticket
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider">
                Date
              </th>
              <th className="px-6 py-3 text-right text-xs font-semibold text-gray-700 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {emails.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-6 py-8 text-center">
                  <p className="text-gray-500 text-sm">No emails found</p>
                </td>
              </tr>
            ) : (
              emails.map((email) => {
                const config = statusConfig[email.status];
                const StatusIcon = config.icon;

                return (
                  <React.Fragment key={email.id}>
                    <tr className="hover:bg-gray-50">
                      <td className="px-6 py-4">
                        <p className="text-sm text-gray-900 font-medium truncate">{email.from}</p>
                      </td>
                      <td className="px-6 py-4">
                        <p className="text-sm text-gray-700 truncate">{email.subject}</p>
                      </td>
                      <td className="px-6 py-4">
                        <div className={`inline-flex items-center gap-2 px-2.5 py-1 rounded-full text-xs font-medium ${config.bg}`}>
                          <StatusIcon className={`w-3.5 h-3.5 ${config.color}`} />
                          <span className={config.color}>{config.label}</span>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        {email.ticketId ? (
                          <a
                            href={`/tickets/${email.ticketId}`}
                            className="text-sm font-mono text-primary-600 hover:underline"
                          >
                            {email.ticketId}
                          </a>
                        ) : (
                          <span className="text-sm text-gray-400">-</span>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        <p className="text-sm text-gray-600">
                          {formatDistanceToNow(new Date(email.createdAt), { addSuffix: true })}
                        </p>
                      </td>
                      <td className="px-6 py-4 text-right">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => {
                            setExpandedId(expandedId === email.id ? null : email.id);
                            if (onViewDetails && expandedId !== email.id) {
                              onViewDetails(email);
                            }
                          }}
                          className="flex items-center gap-2 mx-auto"
                        >
                          <Eye className="w-4 h-4" />
                          <span className="hidden sm:inline">Details</span>
                        </Button>
                      </td>
                    </tr>
                    {expandedId === email.id && (
                      <tr className="bg-gray-50">
                        <td colSpan={6} className="px-6 py-4">
                          <div className="space-y-3">
                            <div>
                              <p className="text-xs font-semibold text-gray-600 uppercase">From</p>
                              <p className="text-sm text-gray-900 mt-1">{email.from}</p>
                            </div>
                            <div>
                              <p className="text-xs font-semibold text-gray-600 uppercase">Subject</p>
                              <p className="text-sm text-gray-900 mt-1">{email.subject}</p>
                            </div>
                            {email.body && (
                              <div>
                                <p className="text-xs font-semibold text-gray-600 uppercase">Body</p>
                                <div className="text-sm text-gray-700 mt-1 max-h-48 overflow-y-auto bg-white p-3 rounded border border-gray-200 whitespace-pre-wrap break-words">
                                  {email.body}
                                </div>
                              </div>
                            )}
                            <div className="grid grid-cols-2 gap-4 pt-2">
                              <div>
                                <p className="text-xs font-semibold text-gray-600 uppercase">Domain</p>
                                <p className="text-sm text-gray-900 mt-1 font-mono">{email.domainMatched}</p>
                              </div>
                              <div>
                                <p className="text-xs font-semibold text-gray-600 uppercase">Customer</p>
                                <p className="text-sm text-gray-900 mt-1">{email.customerId}</p>
                              </div>
                            </div>
                            {email.error && (
                              <div className="bg-red-50 border border-red-200 rounded p-3">
                                <p className="text-xs font-semibold text-red-600 uppercase">Error</p>
                                <p className="text-sm text-red-700 mt-1">{email.error}</p>
                              </div>
                            )}
                          </div>
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="px-6 py-4 border-t border-gray-200 flex items-center justify-between">
          <p className="text-sm text-gray-600">
            Page {currentPage} of {totalPages}
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => onPageChange?.(currentPage - 1)}
              disabled={currentPage === 1}
              className="flex items-center gap-2"
            >
              <ChevronLeft className="w-4 h-4" />
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => onPageChange?.(currentPage + 1)}
              disabled={currentPage === totalPages}
              className="flex items-center gap-2"
            >
              Next
              <ChevronRight className="w-4 h-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
};
