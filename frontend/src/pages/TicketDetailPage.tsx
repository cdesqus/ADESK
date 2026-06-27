import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { Ticket, TicketComment, TicketUpdate, TicketStatus } from '@/types';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { Select } from '@/components/ui/Select';
import { apiService } from '@/services/api';
import { ArrowLeft, Send, AlertCircle, MessageSquare, Mail } from 'lucide-react';
import { formatDistanceToNow, format } from 'date-fns';
import ReactQuill from 'react-quill-new';
import 'react-quill-new/dist/quill.snow.css';

const statusColors: Record<TicketStatus, { bg: string; text: string }> = {
  open: { bg: 'bg-blue-50', text: 'text-blue-800' },
  in_progress: { bg: 'bg-amber-50', text: 'text-amber-800' },
  waiting_customer: { bg: 'bg-purple-50', text: 'text-purple-800' },
  resolved: { bg: 'bg-green-50', text: 'text-green-800' },
  closed: { bg: 'bg-gray-100', text: 'text-gray-800' },
};

export const TicketDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [comments, setComments] = useState<TicketComment[]>([]);
  const [updates, setUpdates] = useState<TicketUpdate[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [commentText, setCommentText] = useState('');
  const [isSubmittingComment, setIsSubmittingComment] = useState(false);
  const [activeTab, setActiveTab] = useState<'internal' | 'email'>('internal');
  const [newStatus, setNewStatus] = useState<TicketStatus | ''>('');

  useEffect(() => {
    const loadTicket = async () => {
      try {
        setIsLoading(true);
        setError(null);
        if (!id) throw new Error('Ticket ID is required');

        const [ticketData, commentsData, updatesData] = await Promise.all([
          apiService.getTicket(id),
          apiService.getTicketComments(id),
          apiService.getTicketUpdates(id),
        ]);

        setTicket(ticketData);
        setComments(commentsData);
        setUpdates(updatesData);
        setNewStatus(ticketData.status);
      } catch (err) {
        const errorMsg = err instanceof Error ? err.message : 'Failed to load ticket';
        setError(errorMsg);
      } finally {
        setIsLoading(false);
      }
    };

    loadTicket();
  }, [id]);

  const handleAddComment = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!commentText.trim() || !id) return;

    try {
      setIsSubmittingComment(true);
      const isEmail = activeTab === 'email';
      const newComment = await apiService.addComment(id, commentText, isEmail);
      setComments([...comments, newComment]);
      setCommentText('');
      setActiveTab('internal');

      // Auto-update status locally if it was an email reply
      if (isEmail && ticket) {
        setTicket({ ...ticket, status: 'waiting_customer' });
        setNewStatus('waiting_customer');
      }
    } catch (err) {
      console.error('Failed to add comment:', err);
    } finally {
      setIsSubmittingComment(false);
    }
  };

  const handleStatusChange = async (status: TicketStatus) => {
    if (!id || !ticket) return;

    try {
      const updated = await apiService.updateTicket(id, { status });
      setTicket(updated);
      setNewStatus(updated.status);
    } catch (err) {
      console.error('Failed to update status:', err);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <div className="inline-block w-8 h-8 border-4 border-gray-300 border-t-primary-700 rounded-full animate-spin"></div>
          <p className="mt-4 text-gray-600">Loading ticket...</p>
        </div>
      </div>
    );
  }

  if (error || !ticket) {
    return (
      <div className="max-w-4xl mx-auto">
        <Link to="/dashboard" className="flex items-center gap-2 text-primary-700 hover:text-primary-800 mb-6">
          <ArrowLeft className="w-4 h-4" />
          Back to Dashboard
        </Link>
        <div className="rounded-lg bg-red-50 p-6 border border-red-200 flex items-start gap-4">
          <AlertCircle className="w-6 h-6 text-red-600 flex-shrink-0 mt-0.5" />
          <div>
            <h3 className="text-lg font-semibold text-red-900">Error loading ticket</h3>
            <p className="text-red-700 mt-1">{error || 'Ticket not found'}</p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto">
      {/* Header */}
      <Link to="/dashboard" className="flex items-center gap-2 text-primary-700 hover:text-primary-800 mb-6">
        <ArrowLeft className="w-4 h-4" />
        Back to Dashboard
      </Link>

      <div className="bg-white rounded-lg border border-gray-200 shadow-xs p-6 md:p-8 mb-6">
        {/* Title Section */}
        <div className="mb-6">
          <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-4 mb-4">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">{ticket.title}</h1>
              <p className="text-sm font-medium text-gray-600 mb-2">Ticket Number: #{ticket.ticket_number || ticket.id}</p>
            </div>
            <div>
              <span className={`inline-flex px-3 py-1.5 rounded-full text-sm font-medium ${(statusColors[ticket.status.toLowerCase() as TicketStatus] || statusColors.open).bg} ${(statusColors[ticket.status.toLowerCase() as TicketStatus] || statusColors.open).text}`}>
                {ticket.status.replace('_', ' ')}
              </span>
            </div>
          </div>
        </div>

        <div className="border-t border-gray-200 pt-6">
          {/* Details Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8">
            <div>
              <h3 className="text-sm font-semibold text-gray-900 mb-4">Ticket Information</h3>
              <div className="space-y-4">
                <div>
                  <p className="text-xs text-gray-600 uppercase tracking-wide">Customer & Contact</p>
                  <p className="text-sm font-medium text-gray-900 mt-0.5">
                    {ticket.customer?.name || 'N/A'}
                  </p>
                  {(ticket.email_from || ticket.whatsapp_from) && (
                    <p className="text-xs text-gray-500 mt-1">
                      {ticket.email_from || ticket.whatsapp_from}
                    </p>
                  )}
                </div>
                <div>
                  <p className="text-xs text-gray-600 uppercase tracking-wide">Priority</p>
                  <p className="text-sm font-medium text-gray-900 mt-0.5 capitalize">
                    {ticket.priority}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-gray-600 uppercase tracking-wide">Created</p>
                  <p className="text-sm font-medium text-gray-900 mt-0.5">
                    {format(new Date(ticket.createdAt || (ticket as any).created_at), 'MMM d, yyyy HH:mm')}
                  </p>
                </div>
              </div>
            </div>

            <div>
              <h3 className="text-sm font-semibold text-gray-900 mb-4">Actions</h3>
              <div className="space-y-4">
                <div>
                  <label className="block text-xs text-gray-600 uppercase tracking-wide mb-2">
                    Change Status
                  </label>
                  <Select
                    value={newStatus}
                    onChange={(e) => handleStatusChange(e.target.value as TicketStatus)}
                    className="text-sm"
                  >
                    <option value="open">Open</option>
                    <option value="in_progress">In Progress</option>
                    <option value="waiting_customer">Waiting Customer</option>
                    <option value="resolved">Resolved</option>
                    <option value="closed">Closed</option>
                  </Select>
                </div>
                {ticket.assignedEngineer && (
                  <div>
                    <p className="text-xs text-gray-600 uppercase tracking-wide">Assigned To</p>
                    <p className="text-sm font-medium text-gray-900 mt-1">
                      {ticket.assignedEngineer.name}
                    </p>
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Description */}
          <div className="border-t border-gray-200 pt-6">
            <h3 className="text-sm font-semibold text-gray-900 mb-3">Description</h3>
            <div className="prose prose-sm max-w-none">
              <p className="text-gray-700 whitespace-pre-wrap">{ticket.description}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Comments Section */}
      <div className="bg-white rounded-lg border border-gray-200 shadow-xs overflow-hidden">
        <div className="p-6 md:p-8 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900 mb-6">Comments & History</h2>

          {/* Comments List */}
          <div className="space-y-4 mb-6">
            {comments.length === 0 ? (
              <p className="text-sm text-gray-600 py-4 text-center">No comments yet</p>
            ) : (
              comments.map((comment) => (
                <div key={comment.id} className="flex gap-4 py-4 border-b border-gray-100 last:border-0">
                  <div className="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center flex-shrink-0 text-sm font-semibold text-primary-700">
                    {comment.user?.name?.charAt(0).toUpperCase()}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center justify-between mb-1">
                      <p className="text-sm font-medium text-gray-900">
                        {comment.user?.name || 'Unknown User'}
                      </p>
                      <p className="text-xs text-gray-500">
                        {formatDistanceToNow(new Date(comment.createdAt || (comment as any).created_at), { addSuffix: true })}
                      </p>
                    </div>
                    <div 
                      className="text-sm text-gray-700 whitespace-pre-wrap prose prose-sm max-w-none"
                      dangerouslySetInnerHTML={{ __html: comment.content }}
                    />
                  </div>
                </div>
              ))
            )}
          </div>

          {/* Tabs for Add Comment / Reply */}
          {ticket?.status !== 'closed' && (
            <div className="mt-8 border rounded-lg border-gray-200 overflow-hidden">
              <div className="flex border-b border-gray-200 bg-gray-50">
                <button
                  type="button"
                  onClick={() => setActiveTab('internal')}
                  className={`flex items-center gap-2 px-6 py-3 text-sm font-medium transition-colors ${
                    activeTab === 'internal'
                      ? 'bg-white text-primary-700 border-t-2 border-t-primary-600'
                      : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100 border-t-2 border-t-transparent'
                  }`}
                >
                  <MessageSquare className="w-4 h-4" />
                  Internal Comment
                </button>
                <button
                  type="button"
                  onClick={() => setActiveTab('email')}
                  className={`flex items-center gap-2 px-6 py-3 text-sm font-medium transition-colors ${
                    activeTab === 'email'
                      ? 'bg-white text-primary-700 border-t-2 border-t-primary-600'
                      : 'text-gray-600 hover:text-gray-900 hover:bg-gray-100 border-t-2 border-t-transparent'
                  }`}
                >
                  <Mail className="w-4 h-4" />
                  Reply to Customer
                </button>
              </div>

              <div className="p-4 bg-white">
                {activeTab === 'internal' ? (
                  <form onSubmit={handleAddComment} className="flex gap-3">
                    <Input
                      placeholder="Add an internal note..."
                      value={commentText}
                      onChange={(e) => setCommentText(e.target.value)}
                      disabled={isSubmittingComment}
                      className="flex-1"
                    />
                    <Button
                      type="submit"
                      disabled={!commentText.trim() || isSubmittingComment}
                    >
                      <Send className="w-4 h-4 mr-2" />
                      Add Note
                    </Button>
                  </form>
                ) : (
                  <form onSubmit={handleAddComment} className="flex flex-col gap-4">
                    <div className="mb-2">
                      <ReactQuill 
                        theme="snow" 
                        value={commentText} 
                        onChange={setCommentText}
                        readOnly={isSubmittingComment}
                        placeholder="Write your email reply here..."
                        className="h-48 mb-12" // mb-12 gives space for the quill toolbar which is absolute sometimes, or just for the bottom
                      />
                    </div>
                    <div className="flex justify-end pt-2 mt-4 border-t border-gray-100">
                      <Button
                        type="submit"
                        disabled={!commentText.trim() || commentText === '<p><br></p>' || isSubmittingComment}
                      >
                        <Send className="w-4 h-4 mr-2" />
                        Send Email Reply
                      </Button>
                    </div>
                  </form>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Updates Timeline */}
        {updates.length > 0 && (
          <div className="p-6 md:p-8 bg-gray-50">
            <h3 className="text-sm font-semibold text-gray-900 mb-4">Activity Timeline</h3>
            <div className="space-y-3">
              {updates.map((update) => (
                <div key={update.id} className="flex gap-4 text-sm">
                  <div className="w-1.5 h-1.5 rounded-full bg-primary-700 flex-shrink-0 mt-1.5" />
                  <div className="flex-1">
                    <p className="text-gray-900">
                      <span className="font-medium">{update.user.name}</span>
                      {update.type === 'status_change' && ` changes status to closed, ${update.newValue}`}
                      {update.type === 'priority_change' && ` changed priority from ${update.oldValue} to ${update.newValue}`}
                      {update.type === 'assignment' && ` assigned to ${update.newValue}`}
                      {update.type === 'comment' && ' added a comment'}
                    </p>
                    <p className="text-gray-600 text-xs mt-0.5">
                      {formatDistanceToNow(new Date(update.createdAt || (update as any).created_at), { addSuffix: true })}
                    </p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
