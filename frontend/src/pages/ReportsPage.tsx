import { useState, useEffect } from 'react';
import { Download, Mail, Trash2, Eye, ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from '@/components/ui/Button';
import { Select } from '@/components/ui/Select';
import { ReportGenerationForm } from '@/components/ReportGenerationForm';
import { ReportViewer } from '@/components/ReportViewer';
import { DownloadModal } from '@/components/DownloadModal';
import { ResendEmailModal } from '@/components/ResendEmailModal';
import { apiService } from '@/services/api';
import { Report, ReportData } from '@/types/reports';
import { Customer } from '@/types';
import { formatDistanceToNow } from 'date-fns';

export const ReportsPage: React.FC = () => {

  // State for reports list
  const [reports, setReports] = useState<Report[]>([]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [isLoadingReports, setIsLoadingReports] = useState(false);
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(false);
  const [error, setError] = useState('');
  const [totalReports, setTotalReports] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(10);

  // State for report generation
  const [isGenerating, setIsGenerating] = useState(false);
  const [generationError, setGenerationError] = useState('');

  // State for selected report
  const [selectedReport, setSelectedReport] = useState<ReportData | null>(null);
  const [isLoadingSelectedReport, setIsLoadingSelectedReport] = useState(false);

  // State for filters
  const [filterCustomer, setFilterCustomer] = useState('');
  const [filterMonth, setFilterMonth] = useState('');
  const [filterYear, setFilterYear] = useState('');

  // State for modals
  const [showDownloadModal, setShowDownloadModal] = useState(false);
  const [showResendModal, setShowResendModal] = useState(false);
  const [selectedReportForAction, setSelectedReportForAction] = useState<Report | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  // Fetch customers on mount
  useEffect(() => {
    const fetchCustomers = async () => {
      setIsLoadingCustomers(true);
      try {
        const response = await apiService.getCustomers(1, 100);
        setCustomers(response.data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load customers');
      } finally {
        setIsLoadingCustomers(false);
      }
    };

    fetchCustomers();
  }, []);

  // Fetch reports with filters
  useEffect(() => {
    const fetchReports = async () => {
      setIsLoadingReports(true);
      setError('');

      try {
        const filters = {
          page: currentPage,
          pageSize,
          ...(filterCustomer && { customerId: filterCustomer }),
          ...(filterMonth && { month: parseInt(filterMonth) }),
          ...(filterYear && { year: parseInt(filterYear) }),
        };

        const response = await apiService.getReports(filters);
        setReports(response.data);
        setTotalReports(response.total);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load reports');
      } finally {
        setIsLoadingReports(false);
      }
    };

    fetchReports();
  }, [currentPage, filterCustomer, filterMonth, filterYear, pageSize]);

  const handleGenerateReport = async (customerId: string, month: number, year: number) => {
    setIsGenerating(true);
    setGenerationError('');

    try {
      const report = await apiService.generateReport(customerId, month, year);
      setSelectedReport(report);
      setCurrentPage(1);
      setFilterCustomer('');
      setFilterMonth('');
      setFilterYear('');

      // Refresh reports list
      const response = await apiService.getReports({ page: 1, pageSize });
      setReports(response.data);
      setTotalReports(response.total);
    } catch (err) {
      setGenerationError(err instanceof Error ? err.message : 'Failed to generate report');
    } finally {
      setIsGenerating(false);
    }
  };

  const handleViewReport = async (report: Report) => {
    setIsLoadingSelectedReport(true);

    try {
      const reportData = await apiService.getReport(report.id);
      setSelectedReport(reportData);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load report');
    } finally {
      setIsLoadingSelectedReport(false);
    }
  };

  const handleDownloadReport = async (format: 'csv' | 'pdf') => {
    if (!selectedReportForAction) return;

    try {
      const blob = await apiService.downloadReport(selectedReportForAction.id, format);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `${selectedReportForAction.customerName}_${selectedReportForAction.month}_${selectedReportForAction.year}.${format}`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
      setShowDownloadModal(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to download report');
    }
  };

  const handleResendEmail = async (additionalEmails?: string[]) => {
    if (!selectedReportForAction) return;

    try {
      await apiService.resendReportEmail(selectedReportForAction.id, additionalEmails);
      setShowResendModal(false);

      // Refresh reports to update sentAt
      const response = await apiService.getReports({
        page: currentPage,
        pageSize,
        ...(filterCustomer && { customerId: filterCustomer }),
        ...(filterMonth && { month: parseInt(filterMonth) }),
        ...(filterYear && { year: parseInt(filterYear) }),
      });
      setReports(response.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to resend email');
    }
  };

  const handleDeleteReport = async (reportId: string) => {
    if (!confirm('Are you sure you want to delete this report?')) return;

    setIsDeleting(true);

    try {
      await apiService.deleteReport(reportId);

      // Refresh reports
      const response = await apiService.getReports({
        page: currentPage,
        pageSize,
        ...(filterCustomer && { customerId: filterCustomer }),
        ...(filterMonth && { month: parseInt(filterMonth) }),
        ...(filterYear && { year: parseInt(filterYear) }),
      });
      setReports(response.data);
      setTotalReports(response.total);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete report');
    } finally {
      setIsDeleting(false);
    }
  };

  const totalPages = Math.ceil(totalReports / pageSize);

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-semibold text-gray-900">Support Reports</h1>
        <p className="text-gray-600 mt-1">View analytics and performance metrics for your customers</p>
      </div>

      {/* Global Error */}
      {error && (
        <div className="p-4 bg-red-50 rounded-lg border border-red-200 text-sm text-red-700">
          {error}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Column: Generation Form and Reports List */}
        <div className="lg:col-span-2 space-y-6">
          {/* Report Generation Form */}
          <ReportGenerationForm
            customers={customers}
            onGenerate={handleGenerateReport}
            isLoading={isGenerating}
            error={generationError}
          />

          {/* Recent Reports */}
          <div className="bg-white rounded-lg border border-gray-200 shadow-xs overflow-hidden">
            <div className="p-6 border-b border-gray-200">
              <h3 className="text-lg font-semibold text-gray-900">Recent Reports</h3>
            </div>

            {/* Filters */}
            <div className="p-6 border-b border-gray-200 space-y-3">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                <Select
                  value={filterCustomer}
                  onChange={(e) => {
                    setFilterCustomer(e.target.value);
                    setCurrentPage(1);
                  }}
                  disabled={isLoadingReports}
                >
                  <option value="">All Customers</option>
                  {customers.map((customer) => (
                    <option key={customer.id} value={customer.id}>
                      {customer.name}
                    </option>
                  ))}
                </Select>

                <Select
                  value={filterMonth}
                  onChange={(e) => {
                    setFilterMonth(e.target.value);
                    setCurrentPage(1);
                  }}
                  disabled={isLoadingReports}
                >
                  <option value="">All Months</option>
                  {Array.from({ length: 12 }, (_, i) => ({
                    value: i + 1,
                    label: new Date(2024, i).toLocaleString('default', { month: 'long' }),
                  })).map((month) => (
                    <option key={month.value} value={month.value.toString()}>
                      {month.label}
                    </option>
                  ))}
                </Select>

                <Select
                  value={filterYear}
                  onChange={(e) => {
                    setFilterYear(e.target.value);
                    setCurrentPage(1);
                  }}
                  disabled={isLoadingReports}
                >
                  <option value="">All Years</option>
                  {Array.from({ length: 4 }, (_, i) => new Date().getFullYear() - 2 + i).map((year) => (
                    <option key={year} value={year.toString()}>
                      {year}
                    </option>
                  ))}
                </Select>
              </div>
            </div>

            {/* Reports Table */}
            <div className="overflow-x-auto">
              {isLoadingReports ? (
                <div className="p-6 text-center text-gray-500">
                  <div className="inline-block w-6 h-6 border-3 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
                  <p className="mt-2">Loading reports...</p>
                </div>
              ) : reports.length === 0 ? (
                <div className="p-6 text-center text-gray-500">
                  <p>No reports found</p>
                </div>
              ) : (
                <table className="w-full text-sm">
                  <thead className="bg-gray-50 border-b border-gray-200">
                    <tr>
                      <th className="px-6 py-3 text-left font-semibold text-gray-900">Customer</th>
                      <th className="px-6 py-3 text-left font-semibold text-gray-900">Period</th>
                      <th className="px-6 py-3 text-left font-semibold text-gray-900">Generated</th>
                      <th className="px-6 py-3 text-left font-semibold text-gray-900">Status</th>
                      <th className="px-6 py-3 text-right font-semibold text-gray-900">Actions</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-200">
                    {reports.map((report) => (
                      <tr key={report.id} className="hover:bg-gray-50">
                        <td className="px-6 py-4">
                          <p className="font-medium text-gray-900">{report.customerName}</p>
                        </td>
                        <td className="px-6 py-4">
                          <p className="text-gray-700">
                            {new Date(2024, report.month - 1).toLocaleString('default', { month: 'long' })} {report.year}
                          </p>
                        </td>
                        <td className="px-6 py-4">
                          <p className="text-gray-700">
                            {formatDistanceToNow(new Date(report.generatedAt || (report as any).generated_at), { addSuffix: true })}
                          </p>
                        </td>
                        <td className="px-6 py-4">
                          <span
                            className={`inline-block px-3 py-1 rounded-full text-xs font-semibold ${
                              report.sent
                                ? 'bg-green-50 text-green-700'
                                : 'bg-amber-50 text-amber-700'
                            }`}
                          >
                            {report.sent ? '✓ Sent' : '⏳ Not Sent'}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="flex justify-end gap-2">
                            <button
                              onClick={() => handleViewReport(report)}
                              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                              title="View Metrics"
                              disabled={isLoadingSelectedReport}
                            >
                              <Eye className="w-4 h-4 text-gray-600" />
                            </button>
                            <button
                              onClick={() => {
                                setSelectedReportForAction(report);
                                setShowDownloadModal(true);
                              }}
                              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                              title="Download"
                            >
                              <Download className="w-4 h-4 text-gray-600" />
                            </button>
                            <button
                              onClick={() => {
                                setSelectedReportForAction(report);
                                setShowResendModal(true);
                              }}
                              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
                              title="Resend Email"
                            >
                              <Mail className="w-4 h-4 text-gray-600" />
                            </button>
                            <button
                              onClick={() => handleDeleteReport(report.id)}
                              className="p-2 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
                              title="Delete"
                              disabled={isDeleting}
                            >
                              <Trash2 className="w-4 h-4 text-gray-600 hover:text-red-600" />
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-between p-6 border-t border-gray-200">
                <p className="text-sm text-gray-600">
                  Showing {(currentPage - 1) * pageSize + 1} to {Math.min(currentPage * pageSize, totalReports)} of{' '}
                  {totalReports} reports
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                    disabled={currentPage === 1}
                  >
                    <ChevronLeft className="w-4 h-4" />
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                    disabled={currentPage === totalPages}
                  >
                    <ChevronRight className="w-4 h-4" />
                  </Button>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Right Column: Report Viewer */}
        <div className="lg:col-span-1">
          {isLoadingSelectedReport ? (
            <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6 text-center">
              <div className="inline-block w-6 h-6 border-3 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
              <p className="mt-2 text-gray-500">Loading report...</p>
            </div>
          ) : selectedReport ? (
            <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6 max-h-[calc(100vh-200px)] overflow-y-auto">
              <ReportViewer report={selectedReport} />
            </div>
          ) : (
            <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6 text-center">
              <p className="text-gray-500">Select a report to view metrics</p>
            </div>
          )}
        </div>
      </div>

      {/* Modals */}
      <DownloadModal
        isOpen={showDownloadModal}
        onClose={() => {
          setShowDownloadModal(false);
          setSelectedReportForAction(null);
        }}
        onDownload={handleDownloadReport}
        csvSize={selectedReportForAction?.csvSize}
        pdfSize={selectedReportForAction?.pdfSize}
      />

      <ResendEmailModal
        isOpen={showResendModal}
        onClose={() => {
          setShowResendModal(false);
          setSelectedReportForAction(null);
        }}
        onResend={handleResendEmail}
        customerEmail={selectedReportForAction?.customerName}
      />
    </div>
  );
};
