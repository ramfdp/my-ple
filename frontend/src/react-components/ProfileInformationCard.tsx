import React from 'react';
import { Pencil } from 'lucide-react';
import type { EmployeeProfile, UserRole } from './types';
import { ProfileInformationItem } from './ProfileInformationItem';
import { EmployeeStatusBadge } from './EmployeeStatusBadge';

interface ProfileInformationCardProps {
  profile: EmployeeProfile;
  currentUserRole?: UserRole;
  onEdit?: () => void;
  className?: string;
}

export const formatRupiah = (amount: number): string => {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(amount);
};

export const ProfileInformationCard: React.FC<ProfileInformationCardProps> = ({
  profile,
  currentUserRole = 'Employee',
  onEdit,
  className = '',
}) => {
  const isHR = currentUserRole === 'HR / Admin';

  return (
    <section
      aria-labelledby="profile-info-heading"
      className={`bg-white rounded-2xl border border-slate-200 p-6 shadow-sm mt-6 ${className}`}
    >
      <div className="flex flex-col sm:flex-row sm:items-center justify-between pb-5 mb-5 border-b border-slate-100 gap-3">
        <div>
          <h2
            id="profile-info-heading"
            className="text-lg font-bold text-slate-900 tracking-tight"
          >
            Employee Information
          </h2>
          <p className="text-xs text-slate-500 mt-0.5">
            Internal company employee data and employment records
          </p>
        </div>

        <button
          type="button"
          onClick={onEdit}
          className="inline-flex items-center justify-center gap-1.5 px-3.5 py-2 rounded-xl bg-white border border-slate-200 text-slate-700 text-xs sm:text-sm font-semibold hover:bg-slate-50 hover:border-slate-300 hover:text-slate-900 transition-all shadow-xs"
        >
          <Pencil className="w-3.5 h-3.5 text-slate-500" />
          <span>Edit</span>
        </button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {/* Name */}
        <ProfileInformationItem
          icon="user"
          label="Name"
          value={profile.name}
        />

        {/* Division (RBAC restricted) */}
        <ProfileInformationItem
          icon="building"
          label="Division"
          value={profile.division}
          isLocked={!isHR}
          lockedTooltip="Only HR / Admin can modify division"
        />

        {/* Current Salary (RBAC restricted) */}
        <ProfileInformationItem
          icon="wallet"
          label="Current Salary"
          value={`${formatRupiah(profile.currentSalary)} / month`}
          isLocked={!isHR}
          lockedTooltip="Only HR / Admin can modify salary"
        />

        {/* Residence */}
        <ProfileInformationItem
          icon="map-pin"
          label="Residence"
          value={profile.residence}
        />

        {/* Phone Number */}
        <ProfileInformationItem
          icon="phone"
          label="Phone Number"
          value={profile.phoneNumber}
        />

        {/* Email */}
        <ProfileInformationItem
          icon="mail"
          label="Email"
          value={profile.email}
        />

        {/* Employment Status (RBAC restricted) */}
        <div className="col-span-1">
          <ProfileInformationItem
            icon="briefcase"
            label="Employment Status"
            isLocked={!isHR}
            lockedTooltip="Only HR / Admin can modify employment status"
          >
            <div className="mt-0.5">
              <EmployeeStatusBadge status={profile.employmentStatus} size="sm" />
            </div>
          </ProfileInformationItem>
        </div>
      </div>
    </section>
  );
};
