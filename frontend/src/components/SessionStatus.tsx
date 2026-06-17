import React from 'react';
import { CheckCircle2, AlertCircle, Circle } from 'lucide-react';

interface SessionStatusProps {
  status: string;
  phoneNumber?: string;
}

export const SessionStatus: React.FC<SessionStatusProps> = ({ status, phoneNumber }) => {
  const getStatusConfig = () => {
    switch (status.toLowerCase()) {
      case 'active':
      case 'connected':
      case 'working':
        return {
          icon: CheckCircle2,
          text: 'Active',
          bgColor: 'bg-green-50',
          textColor: 'text-green-700',
          badgeColor: 'bg-green-100 text-green-800',
        };
      case 'pending':
      case 'scan_qr_code':
        return {
          icon: AlertCircle,
          text: 'Awaiting scan',
          bgColor: 'bg-yellow-50',
          textColor: 'text-yellow-700',
          badgeColor: 'bg-yellow-100 text-yellow-800',
        };
      case 'disconnected':
        return {
          icon: Circle,
          text: 'Inactive',
          bgColor: 'bg-red-50',
          textColor: 'text-red-700',
          badgeColor: 'bg-red-100 text-red-800',
        };
      default:
        return {
          icon: Circle,
          text: 'Unknown',
          bgColor: 'bg-gray-50',
          textColor: 'text-gray-700',
          badgeColor: 'bg-gray-100 text-gray-800',
        };
    }
  };

  const config = getStatusConfig();
  const Icon = config.icon;

  return (
    <div className="flex flex-col gap-1">
      <div className="flex items-center gap-2">
        <Icon className="w-4 h-4" />
        <span className={`inline-block px-3 py-1 rounded-full text-xs font-medium ${config.badgeColor}`}>
          {config.text}
        </span>
      </div>
      {phoneNumber && <p className="text-xs text-gray-600">{phoneNumber}</p>}
    </div>
  );
};
