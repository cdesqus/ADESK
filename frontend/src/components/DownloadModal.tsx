import React, { useState } from 'react';
import { X, Download } from 'lucide-react';
import { Button } from '@/components/ui/Button';

interface DownloadModalProps {
  isOpen: boolean;
  onClose: () => void;
  onDownload: (format: 'csv' | 'pdf') => Promise<void>;
  csvSize?: number;
  pdfSize?: number;
  isLoading?: boolean;
}

const formatFileSize = (bytes?: number): string => {
  if (!bytes) return 'Unknown';
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

export const DownloadModal: React.FC<DownloadModalProps> = ({
  isOpen,
  onClose,
  onDownload,
  csvSize,
  pdfSize,
  isLoading = false,
}) => {
  const [format, setFormat] = useState<'csv' | 'pdf'>('csv');
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState('');

  const handleDownload = async () => {
    setError('');
    setDownloading(true);

    try {
      await onDownload(format);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Download failed');
    } finally {
      setDownloading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-lg shadow-lg max-w-md w-full">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Download Report</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            aria-label="Close"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4">
          {/* Format Selection */}
          <div className="space-y-3">
            {/* CSV Option */}
            <label className="flex items-start gap-3 p-4 border-2 rounded-lg cursor-pointer transition-colors"
              style={{
                borderColor: format === 'csv' ? '#3b82f6' : '#e5e7eb',
                backgroundColor: format === 'csv' ? '#eff6ff' : '#f9fafb',
              }}
            >
              <input
                type="radio"
                value="csv"
                checked={format === 'csv'}
                onChange={(e) => setFormat(e.target.value as 'csv' | 'pdf')}
                className="mt-1"
              />
              <div className="flex-1">
                <p className="font-medium text-gray-900">CSV Format</p>
                <p className="text-sm text-gray-600">Spreadsheet compatible</p>
                {csvSize && <p className="text-xs text-gray-500 mt-1">{formatFileSize(csvSize)}</p>}
              </div>
            </label>

            {/* PDF Option */}
            <label className="flex items-start gap-3 p-4 border-2 rounded-lg cursor-pointer transition-colors"
              style={{
                borderColor: format === 'pdf' ? '#3b82f6' : '#e5e7eb',
                backgroundColor: format === 'pdf' ? '#eff6ff' : '#f9fafb',
              }}
            >
              <input
                type="radio"
                value="pdf"
                checked={format === 'pdf'}
                onChange={(e) => setFormat(e.target.value as 'csv' | 'pdf')}
                className="mt-1"
              />
              <div className="flex-1">
                <p className="font-medium text-gray-900">PDF Format</p>
                <p className="text-sm text-gray-600">Professional printable document</p>
                {pdfSize && <p className="text-xs text-gray-500 mt-1">{formatFileSize(pdfSize)}</p>}
              </div>
            </label>
          </div>

          {/* Error Message */}
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
              {error}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex gap-3 p-6 border-t border-gray-200">
          <Button
            variant="outline"
            onClick={onClose}
            className="flex-1"
            disabled={downloading || isLoading}
          >
            Cancel
          </Button>
          <Button
            onClick={handleDownload}
            className="flex-1"
            disabled={downloading || isLoading}
          >
            {downloading ? (
              <span className="flex items-center gap-2">
                <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                Downloading...
              </span>
            ) : (
              <span className="flex items-center gap-2">
                <Download className="w-4 h-4" />
                Download {format.toUpperCase()}
              </span>
            )}
          </Button>
        </div>
      </div>
    </div>
  );
};
