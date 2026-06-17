import React from 'react';
import { ReportData } from '@/types/reports';
import { ReportCharts } from '@/components/ReportCharts';
import { EngineerPerformanceTable } from '@/components/EngineerPerformanceTable';
import { formatDistanceToNow } from 'date-fns';

interface ReportViewerProps {
  report: ReportData;
}

const KPICard: React.FC<{
  label: string;
  value: string | number;
  subtext?: string;
  color?: 'green' | 'red' | 'neutral';
}> = ({ label, value, subtext, color = 'neutral' }) => {
  const colorClasses = {
    green: 'text-green-700',
    red: 'text-red-700',
    neutral: 'text-gray-900',
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6">
      <p className="text-sm text-gray-600 mb-2">{label}</p>
      <p className={`text-3xl font-semibold ${colorClasses[color]}`}>{value}</p>
      {subtext && <p className="text-xs text-gray-500 mt-2">{subtext}</p>}
    </div>
  );
};

export const ReportViewer: React.FC<ReportViewerProps> = ({ report }) => {
  const metrics = report.metrics;
  const resolvedPercentage = metrics.totalTickets > 0
    ? ((metrics.resolvedTickets / metrics.totalTickets) * 100).toFixed(1)
    : '0';

  const getMetricColor = (value: number, threshold: number): 'green' | 'red' | 'neutral' => {
    if (value >= threshold) return 'green';
    if (value < threshold * 0.7) return 'red';
    return 'neutral';
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h2 className="text-2xl font-semibold text-gray-900">{report.customerName}</h2>
        <p className="text-gray-600 mt-1">{report.month} Report</p>
        <p className="text-xs text-gray-500 mt-2">
          Generated {formatDistanceToNow(new Date(report.generatedAt || (report as any).generated_at), { addSuffix: true })}
        </p>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <KPICard label="Total Tickets" value={metrics.totalTickets} />
        <KPICard
          label="Resolved"
          value={`${metrics.resolvedTickets} (${resolvedPercentage}%)`}
          color={getMetricColor(parseFloat(resolvedPercentage), 80)}
        />
        <KPICard
          label="Avg Resolution Time"
          value={`${metrics.averageResolutionTime.toFixed(1)}h`}
          color={getMetricColor(100 - metrics.averageResolutionTime, 50)}
        />
        <KPICard
          label="SLA Compliance"
          value={`${metrics.slaCompliance.toFixed(1)}%`}
          color={getMetricColor(metrics.slaCompliance, 95)}
        />
      </div>

      {/* Status Summary */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-blue-50 rounded-lg p-4 border border-blue-200">
          <p className="text-xs text-blue-600 font-medium">Open</p>
          <p className="text-2xl font-semibold text-blue-900">{metrics.openTickets}</p>
        </div>
        <div className="bg-amber-50 rounded-lg p-4 border border-amber-200">
          <p className="text-xs text-amber-600 font-medium">In Progress</p>
          <p className="text-2xl font-semibold text-amber-900">{metrics.inProgressTickets}</p>
        </div>
        <div className="bg-purple-50 rounded-lg p-4 border border-purple-200">
          <p className="text-xs text-purple-600 font-medium">Waiting</p>
          <p className="text-2xl font-semibold text-purple-900">
            {(metrics.totalTickets - metrics.openTickets - metrics.inProgressTickets - metrics.resolvedTickets)}
          </p>
        </div>
        <div className="bg-green-50 rounded-lg p-4 border border-green-200">
          <p className="text-xs text-green-600 font-medium">Resolved</p>
          <p className="text-2xl font-semibold text-green-900">{metrics.resolvedTickets}</p>
        </div>
      </div>

      {/* Charts */}
      <ReportCharts metrics={metrics} />

      {/* Engineer Performance */}
      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Engineer Performance</h3>
        <EngineerPerformanceTable engineers={metrics.engineerStats} />
      </div>
    </div>
  );
};
