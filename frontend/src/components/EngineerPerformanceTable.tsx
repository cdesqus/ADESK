import React, { useState } from 'react';
import { EngineerStat } from '@/types/reports';
import { ChevronUp, ChevronDown } from 'lucide-react';

interface EngineerPerformanceTableProps {
  engineers: EngineerStat[];
}

type SortField = 'name' | 'ticketsHandled' | 'avgTime' | 'resolutionRate';
type SortOrder = 'asc' | 'desc';

export const EngineerPerformanceTable: React.FC<EngineerPerformanceTableProps> = ({ engineers }) => {
  const [sortField, setSortField] = useState<SortField>('ticketsHandled');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('desc');
    }
  };

  const sortedEngineers = [...engineers].sort((a, b) => {
    let aVal: string | number;
    let bVal: string | number;

    switch (sortField) {
      case 'name':
        aVal = a.name;
        bVal = b.name;
        break;
      case 'ticketsHandled':
        aVal = a.ticketsHandled;
        bVal = b.ticketsHandled;
        break;
      case 'avgTime':
        aVal = a.avgTime;
        bVal = b.avgTime;
        break;
      case 'resolutionRate':
        aVal = a.resolutionRate;
        bVal = b.resolutionRate;
        break;
    }

    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return sortOrder === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
    }

    return sortOrder === 'asc' ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number);
  });

  const getResolutionRateColor = (rate: number) => {
    if (rate >= 90) return 'text-green-700 bg-green-50';
    if (rate >= 70) return 'text-amber-700 bg-amber-50';
    return 'text-red-700 bg-red-50';
  };

  const SortHeader: React.FC<{ label: string; field: SortField }> = ({ label, field }) => (
    <button
      onClick={() => handleSort(field)}
      className="flex items-center gap-1 hover:bg-gray-100 px-2 py-1 rounded transition-colors cursor-pointer font-semibold"
    >
      {label}
      {sortField === field && (
        sortOrder === 'asc' ? (
          <ChevronUp className="w-4 h-4" />
        ) : (
          <ChevronDown className="w-4 h-4" />
        )
      )}
    </button>
  );

  if (engineers.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6 text-center">
        <p className="text-gray-500">No engineer data available</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 shadow-xs overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-gray-50 border-b border-gray-200">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">
                <SortHeader label="Engineer" field="name" />
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">
                <SortHeader label="Tickets Handled" field="ticketsHandled" />
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">
                <SortHeader label="Avg Resolution Time" field="avgTime" />
              </th>
              <th className="px-6 py-3 text-left text-xs font-semibold text-gray-900">
                <SortHeader label="Resolution Rate" field="resolutionRate" />
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {sortedEngineers.map((engineer) => (
              <tr key={engineer.engineerId} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4">
                  <p className="text-sm font-medium text-gray-900">{engineer.name}</p>
                </td>
                <td className="px-6 py-4">
                  <p className="text-sm text-gray-700">{engineer.ticketsHandled}</p>
                </td>
                <td className="px-6 py-4">
                  <p className="text-sm text-gray-700">{engineer.avgTime.toFixed(1)}h</p>
                </td>
                <td className="px-6 py-4">
                  <div className={`inline-block px-3 py-1 rounded-full text-sm font-semibold ${getResolutionRateColor(engineer.resolutionRate)}`}>
                    {engineer.resolutionRate.toFixed(1)}%
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
