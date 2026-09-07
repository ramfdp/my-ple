import React, { useState } from 'react';
import {
  LayoutDashboard,
  Users,
  User,
  CalendarCheck,
  Clock,
  Wallet,
  FileText,
  Settings,
  LogOut,
  Menu,
  X,
} from 'lucide-react';

interface SidebarProps {
  activeItem?: string;
  onNavigate?: (href: string) => void;
  className?: string;
}

const menuItems = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Employees', href: '/employees', icon: Users },
  { name: 'My Profile', href: '/my-profile', icon: User },
  { name: 'Attendance', href: '/attendance', icon: CalendarCheck },
  { name: 'Leave', href: '/leave', icon: Clock },
  { name: 'Payroll', href: '/payroll', icon: Wallet },
  { name: 'Documents', href: '/documents', icon: FileText },
  { name: 'Settings', href: '/settings', icon: Settings },
];

export const Sidebar: React.FC<SidebarProps> = ({
  activeItem = 'My Profile',
  onNavigate,
  className = '',
}) => {
  const [mobileOpen, setMobileOpen] = useState(false);

  return (
    <>
      {/* Mobile Top Header */}
      <div className="lg:hidden sticky top-0 left-0 right-0 h-14 bg-white border-b border-slate-200 px-4 flex items-center justify-between z-40">
        <button
          type="button"
          onClick={() => setMobileOpen(!mobileOpen)}
          aria-label="Toggle menu"
          className="p-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50"
        >
          {mobileOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
        </button>
        <div className="flex items-center gap-2">
          <img
            src="/harmoni%20insan%20phooto.webp"
            alt="my HIS"
            className="w-7 h-7 object-contain"
          />
          <span className="font-bold text-slate-900 tracking-tight">my HIS</span>
        </div>
      </div>

      {/* Mobile Backdrop */}
      {mobileOpen && (
        <div
          className="lg:hidden fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-40"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Main Sidebar */}
      <aside
        className={`fixed lg:sticky top-0 left-0 bottom-0 h-screen w-64 bg-white border-r border-slate-200 flex flex-col z-50 transition-transform duration-200 ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
        } ${className}`}
      >
        {/* Company Brand */}
        <div className="p-5 border-b border-slate-100 flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-teal-50 border border-teal-100 flex items-center justify-center p-1">
            <img
              src="/harmoni%20insan%20phooto.webp"
              alt="my HIS"
              className="w-full h-full object-contain"
            />
          </div>
          <div>
            <h1 className="font-bold text-slate-900 text-base leading-tight">my HIS</h1>
            <span className="text-[11px] font-semibold text-teal-700 tracking-wide uppercase">
              HR Workspace
            </span>
          </div>
        </div>

        {/* Nav List */}
        <nav className="flex-1 p-3.5 space-y-1 overflow-y-auto">
          {menuItems.map((item) => {
            const Icon = item.icon;
            const isActive = item.name === activeItem;

            return (
              <a
                key={item.name}
                href={item.href}
                onClick={(e) => {
                  if (onNavigate) {
                    e.preventDefault();
                    onNavigate(item.href);
                  }
                  setMobileOpen(false);
                }}
                className={`flex items-center gap-3 px-3.5 py-2.5 rounded-xl text-sm font-semibold transition-all ${
                  isActive
                    ? 'bg-teal-600 text-white shadow-sm shadow-teal-600/30'
                    : 'text-slate-600 hover:text-slate-900 hover:bg-slate-50'
                }`}
              >
                <Icon className="w-4 h-4 flex-shrink-0" />
                <span>{item.name}</span>
              </a>
            );
          })}
        </nav>

        {/* Footer */}
        <div className="p-3 border-t border-slate-100">
          <a
            href="/"
            className="flex items-center gap-2.5 px-3.5 py-2.5 rounded-xl text-sm font-semibold text-rose-600 hover:bg-rose-50 transition-colors"
          >
            <LogOut className="w-4 h-4" />
            <span>Log out</span>
          </a>
        </div>
      </aside>
    </>
  );
};
