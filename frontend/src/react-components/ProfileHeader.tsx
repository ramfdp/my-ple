import React from 'react';
import { Pencil } from 'lucide-react';
import type { EmployeeProfile, UserRole } from './types';
import { ProfileAvatar } from './ProfileAvatar';
import { EmployeeStatusBadge } from './EmployeeStatusBadge';

interface ProfileHeaderProps {
  profile: EmployeeProfile;
  currentUserRole?: UserRole;
  onEdit?: () => void;
  onRoleChange?: (role: UserRole) => void;
  className?: string;
}

export const ProfileHeader: React.FC<ProfileHeaderProps> = ({
  profile,
  currentUserRole = 'Employee',
  onEdit,
  onRoleChange,
  className = '',
}) => {
  return (
    <header
      className={`bg-white rounded-2xl border border-slate-200 overflow-hidden shadow-sm ${className}`}
    >
      {/* Gradient Banner */}
      <div
        className="h-36 relative flex justify-end items-start p-4"
        style={{
          background: 'linear-gradient(90deg, #99F6E4 0%, #D1FAE5 50%, #ECFCCB 100%)',
        }}
      >
        {/* Role Switcher Pill for Testing RBAC */}
        {onRoleChange && (
          <div className="relative z-10 inline-flex items-center bg-white/90 backdrop-blur-md border border-white/60 p-1 rounded-full shadow-xs gap-1">
            <span className="hidden sm:inline text-[11px] font-bold text-teal-800 uppercase px-2">
              Viewing Role:
            </span>
            <button
              type="button"
              onClick={() => onRoleChange('Employee')}
              className={`text-xs font-semibold px-3 py-1 rounded-full transition-all ${
                currentUserRole === 'Employee'
                  ? 'bg-teal-700 text-white shadow-xs'
                  : 'text-slate-600 hover:text-slate-900'
              }`}
            >
              Employee
            </button>
            <button
              type="button"
              onClick={() => onRoleChange('HR / Admin')}
              className={`text-xs font-semibold px-3 py-1 rounded-full transition-all ${
                currentUserRole === 'HR / Admin'
                  ? 'bg-teal-700 text-white shadow-xs'
                  : 'text-slate-600 hover:text-slate-900'
              }`}
            >
              HR / Admin
            </button>
          </div>
        )}
      </div>

      {/* Header Content with Overlapping Avatar */}
      <div className="px-6 pb-6 pt-0 flex flex-col sm:flex-row sm:items-end justify-between gap-4 relative">
        <div className="flex flex-col sm:flex-row sm:items-end gap-5">
          <div className="-mt-14 relative z-20">
            <ProfileAvatar
              src={profile.avatarUrl}
              alt={profile.name}
              size={104}
              borderWidth={4}
              editable={true}
              onEdit={onEdit}
            />
          </div>

          <div className="pb-1 flex flex-col gap-1.5">
            <div className="flex items-center gap-3 flex-wrap">
              <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight leading-tight">
                {profile.name}
              </h1>
              <EmployeeStatusBadge
                status={profile.isActive ? 'Active Employee' : 'Inactive'}
                size="sm"
              />
            </div>

            <p className="text-sm font-medium text-slate-500 flex items-center gap-2 flex-wrap">
              <span className="font-semibold text-slate-700">{profile.jobTitle}</span>
              <span className="text-slate-300">•</span>
              <span className="text-teal-700 bg-teal-50 px-2 py-0.5 rounded-md font-semibold text-xs">
                {profile.division}
              </span>
            </p>
          </div>
        </div>

        <div className="pb-1 w-full sm:w-auto">
          <button
            type="button"
            onClick={onEdit}
            className="w-full sm:w-auto inline-flex items-center justify-center gap-2 bg-white text-slate-800 border border-slate-300 hover:border-slate-400 hover:bg-slate-50 px-4 py-2.5 rounded-xl text-sm font-semibold shadow-xs transition-all hover:-translate-y-0.5"
          >
            <Pencil className="w-4 h-4 text-teal-600" />
            <span>Edit Profile</span>
          </button>
        </div>
      </div>
    </header>
  );
};
