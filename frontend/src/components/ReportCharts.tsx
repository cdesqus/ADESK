import React from 'react';
import { ReportMetrics } from '@/types/reports';

interface ReportChartsProps {
  metrics: ReportMetrics;
}

const COLORS = {
  open: '#3b82f6',
  in_progress: '#f59e0b',
  resolved: '#10b981',
  waiting_customer: '#8b5cf6',
  closed: '#6b7280',
  low: '#10b981',
  medium: '#3b82f6',
  high: '#f59e0b',
  urgent: '#ef4444',
};

const PieChart: React.FC<{ data: Record<string, number>; title: string; colors?: Record<string, string> }> = ({
  data,
  title,
  colors = {},
}) => {
  const total = Object.values(data).reduce((sum, val) => sum + val, 0);
  if (total === 0) return <div className="text-center text-gray-500">No data available</div>;

  const entries = Object.entries(data);
  let currentAngle = 0;

  return (
    <div className="w-full h-64">
      <h3 className="text-sm font-semibold text-gray-900 mb-4">{title}</h3>
      <svg viewBox="0 0 200 200" className="w-full h-full">
        {entries.map(([label, value], index) => {
          const sliceAngle = (value / total) * 360;
          const startAngle = currentAngle;
          const endAngle = currentAngle + sliceAngle;
          const color = colors[label] || Object.values(COLORS)[index % Object.values(COLORS).length];

          const startRad = (startAngle * Math.PI) / 180;
          const endRad = (endAngle * Math.PI) / 180;

          const x1 = 100 + 70 * Math.cos(startRad);
          const y1 = 100 + 70 * Math.sin(startRad);
          const x2 = 100 + 70 * Math.cos(endRad);
          const y2 = 100 + 70 * Math.sin(endRad);

          const largeArc = sliceAngle > 180 ? 1 : 0;

          const pathData = [
            `M 100 100`,
            `L ${x1} ${y1}`,
            `A 70 70 0 ${largeArc} 1 ${x2} ${y2}`,
            `Z`,
          ].join(' ');

          currentAngle = endAngle;

          return (
            <path key={label} d={pathData} fill={color} stroke="white" strokeWidth="2" />
          );
        })}
      </svg>
      <div className="mt-4 grid grid-cols-2 gap-2 text-xs">
        {entries.map(([label, value]) => (
          <div key={label} className="flex items-center gap-2">
            <div
              className="w-3 h-3 rounded-full"
              style={{ backgroundColor: colors[label] || COLORS[label as keyof typeof COLORS] }}
            />
            <span className="text-gray-700">
              {label}: {value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
};

const BarChart: React.FC<{ data: Record<string, number>; title: string; colors?: Record<string, string> }> = ({
  data,
  title,
  colors = {},
}) => {
  const entries = Object.entries(data);
  const max = Math.max(...Object.values(data), 1);

  return (
    <div className="w-full h-64">
      <h3 className="text-sm font-semibold text-gray-900 mb-4">{title}</h3>
      <div className="flex items-end justify-around h-48 gap-2">
        {entries.map(([label, value], index) => {
          const height = (value / max) * 100;
          const color = colors[label] || Object.values(COLORS)[index % Object.values(COLORS).length];

          return (
            <div key={label} className="flex flex-col items-center gap-1 flex-1">
              <div className="w-full flex items-end justify-center h-40">
                <div
                  className="w-full rounded-t"
                  style={{
                    height: `${height}%`,
                    backgroundColor: color,
                    minHeight: '4px',
                  }}
                />
              </div>
              <span className="text-xs text-gray-700 text-center truncate w-full">{label}</span>
              <span className="text-xs font-semibold text-gray-900">{value}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
};

export const ReportCharts: React.FC<ReportChartsProps> = ({ metrics }) => {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6">
        <PieChart data={metrics.byStatus} title="Status Distribution" colors={COLORS} />
      </div>

      <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6">
        <BarChart data={metrics.byPriority} title="Priority Distribution" colors={COLORS} />
      </div>

      <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6">
        <BarChart data={metrics.bySource} title="Source Distribution" colors={COLORS} />
      </div>
    </div>
  );
};
