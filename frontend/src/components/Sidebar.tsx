import { Link, useLocation } from 'react-router-dom';
import { LayoutDashboard, Ticket, Users, Wrench, BarChart3, Mail, MessageCircle, X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface SidebarProps {
  isOpen?: boolean;
  onClose?: () => void;
}

const navItems = [
  { href: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { href: '/tickets', icon: Ticket, label: 'Tickets' },
  { href: '/customers', icon: Users, label: 'Customers' },
  { href: '/engineers', icon: Wrench, label: 'Engineers' },
  { href: '/reports', icon: BarChart3, label: 'Reports' },
];

const adminItems = [
  { href: '/settings/email', icon: Mail, label: 'Email Settings' },
  { href: '/settings/whatsapp', icon: MessageCircle, label: 'WhatsApp Settings' },
];

export const Sidebar: React.FC<SidebarProps> = ({ isOpen = true, onClose }) => {
  const location = useLocation();

  return (
    <>
      {/* Mobile Overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-30 md:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed left-0 top-16 h-[calc(100vh-64px)] w-64 bg-white border-r border-gray-200 transition-transform duration-200 ease-in-out z-40 md:sticky md:translate-x-0",
          isOpen ? 'translate-x-0' : '-translate-x-full'
        )}
      >
        <nav className="p-4 space-y-2">
          {/* Close button for mobile */}
          <button
            onClick={onClose}
            className="md:hidden w-full flex justify-end p-2"
            aria-label="Close menu"
          >
            <X className="w-5 h-5 text-gray-700" />
          </button>

          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname === item.href;

            return (
              <Link
                key={item.href}
                to={item.href}
                className={cn(
                  "flex items-center gap-3 px-4 py-2.5 rounded-md transition-colors text-sm font-medium",
                  isActive
                    ? 'bg-primary-50 text-primary-700 border-l-2 border-primary-700'
                    : 'text-gray-700 hover:bg-gray-100'
                )}
                onClick={onClose}
              >
                <Icon className="w-4 h-4" />
                {item.label}
              </Link>
            );
          })}

          {/* Admin section divider */}
          {adminItems.length > 0 && (
            <>
              <div className="my-4 border-t border-gray-200" />
              <p className="px-4 py-2 text-xs font-semibold text-gray-600 uppercase">Admin</p>

              {adminItems.map((item) => {
                const Icon = item.icon;
                const isActive = location.pathname === item.href;

                return (
                  <Link
                    key={item.href}
                    to={item.href}
                    className={cn(
                      "flex items-center gap-3 px-4 py-2.5 rounded-md transition-colors text-sm font-medium",
                      isActive
                        ? 'bg-primary-50 text-primary-700 border-l-2 border-primary-700'
                        : 'text-gray-700 hover:bg-gray-100'
                    )}
                    onClick={onClose}
                  >
                    <Icon className="w-4 h-4" />
                    {item.label}
                  </Link>
                );
              })}
            </>
          )}
        </nav>
      </aside>
    </>
  );
};
