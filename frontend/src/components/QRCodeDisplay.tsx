import React, { useState } from 'react';
import { Copy, RefreshCw, AlertCircle } from 'lucide-react';
import { Button } from '@/components/ui/Button';

interface QRCodeDisplayProps {
  qrCode: string | null;
  isLoading: boolean;
  error?: string;
  onRefresh?: () => void;
}

export const QRCodeDisplay: React.FC<QRCodeDisplayProps> = ({
  qrCode,
  isLoading,
  error,
  onRefresh,
}) => {
  const [copySuccess, setCopySuccess] = useState(false);

  const handleCopy = () => {
    if (qrCode) {
      navigator.clipboard.writeText(qrCode);
      setCopySuccess(true);
      setTimeout(() => setCopySuccess(false), 2000);
    }
  };

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center p-8">
        <div className="w-12 h-12 border-4 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
        <p className="mt-4 text-gray-600 text-sm">Generating QR code...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center p-8 bg-red-50 rounded-lg border border-red-200">
        <AlertCircle className="w-8 h-8 text-red-600 mb-2" />
        <p className="text-red-700 text-sm font-medium">{error}</p>
        {onRefresh && (
          <Button onClick={onRefresh} variant="outline" size="sm" className="mt-4">
            <RefreshCw className="w-3 h-3 mr-2" />
            Try again
          </Button>
        )}
      </div>
    );
  }

  if (!qrCode) {
    return null;
  }

  return (
    <div className="flex flex-col items-center gap-4">
      <div className="bg-white p-4 rounded-lg border border-gray-200">
        <img
          src={qrCode}
          alt="WhatsApp QR Code"
          className="w-64 h-64 object-contain"
        />
      </div>

      <p className="text-center text-sm text-gray-600 max-w-sm">
        Scan this QR code with WhatsApp on your phone to connect this session.
      </p>

      <div className="flex gap-2">
        <Button
          onClick={handleCopy}
          variant="outline"
          size="sm"
        >
          <Copy className="w-3 h-3 mr-2" />
          {copySuccess ? 'Copied!' : 'Copy'}
        </Button>
        {onRefresh && (
          <Button
            onClick={onRefresh}
            variant="outline"
            size="sm"
          >
            <RefreshCw className="w-3 h-3 mr-2" />
            Refresh
          </Button>
        )}
      </div>
    </div>
  );
};
