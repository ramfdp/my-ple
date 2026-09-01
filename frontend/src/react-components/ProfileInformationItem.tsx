import React from 'react';
import {
  User,
  Building2,
  Wallet,
  MapPin,
  Phone,
  Mail,
  Briefcase,
  Lock,
} from 'lucide-react';

export type IconType =
  | 'user'
  | 'building'
  | 'wallet'
  | 'map-pin'
  | 'phone'
  | 'mail'
  | 'briefcase';

interface ProfileInformationItemProps {
  icon: IconType;
  label: string;
  value?: string | number | React.ReactNode;
  isLocked?: boolean;
  lockedTooltip?: string;
  children?: React.ReactNode;
  className?: string;
}

const iconMap: Record<IconType, React.FC<{ className?: string }>> = {
  user: User,
  building: Building2,
  wallet: Wallet,
  'map-pin': MapPin,
  phone: Phone,
  mail: Mail,
  briefcase: Briefcase,
};

export const ProfileInformationItem: React.FC<ProfileInformationItemProps> = ({
  icon,
  label,
  value,
  isLocked = false,
  lockedTooltip = 'HR/Admin only',
  children,
  className = '',
}) => {
  const IconComponent = iconMap[icon] || User;

  return (
    <div
      className={`flex items-start gap-3.5 p-3.5 rounded-xl bg-white border border-slate-100 hover:border-slate-200 transition-all hover:bg-slate-50/50 ${className}`}
    >
      <div className="w-9 h-9 rounded-lg bg-teal-50 border border-teal-100 text-teal-700 flex items-center justify-center flex-shrink-0">
        <IconComponent className="w-4 h-4" />
      </div>

      <div className="flex flex-col gap-1 min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-slate-500">{label}</span>
          {isLocked && (
            <span
              title={lockedTooltip}
              className="inline-flex items-center gap-1 text-[10px] font-semibold text-slate-400 bg-slate-100 px-1.5 py-0.5 rounded"
            >
              <Lock className="w-2.5 h-2.5" />
              HR Only
            </span>
          )}
        </div>

        <div className="text-sm font-semibold text-slate-900 break-words">
          {value !== undefined && value !== null ? value : children}
        </div>
      </div>
    </div>
  );
};
