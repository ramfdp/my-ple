import React, { useState } from 'react';
import { RotateCw, CheckCircle2, AlertCircle } from 'lucide-react';
import type { EmployeeProfile, UserRole } from './types';
import { Sidebar } from './Sidebar';
import { ProfileHeader } from './ProfileHeader';
import { ProfileInformationCard } from './ProfileInformationCard';
import { EditProfileModal } from './EditProfileModal';

const initialEmployeeData: EmployeeProfile = {
  id: 'EMP-2024-0089',
  name: 'Andi Pratama',
  jobTitle: 'Software Engineering',
  division: 'Information Technology',
  currentSalary: 12500000,
  residence: 'South Jakarta',
  phoneNumber: '+62 812-3456-7890',
  email: 'andi.pratama@company.com',
  employmentStatus: 'Permanent Employee',
  avatarUrl:
    'https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&q=80&w=400',
  isActive: true,
};

export const EmployeeProfilePage: React.FC = () => {
  const [employee, setEmployee] = useState<EmployeeProfile>(initialEmployeeData);
  const [currentUserRole, setCurrentUserRole] = useState<UserRole>('Employee');
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [toast, setToast] = useState<{
    visible: boolean;
    message: string;
    type: 'success' | 'error' | 'info';
  }>({
    visible: false,
    message: '',
    type: 'success',
  });

  const showToast = (
    message: string,
    type: 'success' | 'error' | 'info' = 'success'
  ) => {
    setToast({ visible: true, message, type });
    setTimeout(() => {
      setToast((prev) => ({ ...prev, visible: false }));
    }, 3500);
  };

  const handleSaveProfile = async (updated: EmployeeProfile) => {
    // ponytail: simulate realistic network response
    await new Promise((resolve) => setTimeout(resolve, 400));
    setEmployee(updated);
    setIsEditModalOpen(false);
    showToast('Employee profile updated successfully.', 'success');
  };

  const handleRefresh = () => {
    setIsLoading(true);
    setTimeout(() => {
      setIsLoading(false);
    }, 700);
  };

  return (
    <div className="flex min-h-screen bg-[#F8FAFC]">
      {/* Sidebar */}
      <Sidebar activeItem="My Profile" />

      {/* Main Content Area */}
      <main className="flex-1 min-w-0 p-4 sm:p-7 lg:p-9">
        <div className="max-w-5xl mx-auto space-y-6">
          {/* Breadcrumb & Refresh Bar */}
          <div className="flex items-center justify-between gap-4">
            <nav className="flex items-center gap-2 text-xs sm:text-sm font-medium text-slate-500">
              <a href="/dashboard" className="hover:text-teal-700 transition-colors">
                Dashboard
              </a>
              <span className="text-slate-300">/</span>
              <span className="font-semibold text-slate-900">My Profile</span>
            </nav>

            <button
              type="button"
              onClick={handleRefresh}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-slate-200 bg-white text-xs font-semibold text-slate-600 hover:bg-slate-50 hover:text-slate-900 transition-all shadow-xs"
            >
              <RotateCw className="w-3.5 h-3.5" />
              <span>Refresh</span>
            </button>
          </div>

          {isLoading ? (
            /* Skeleton Loading State */
            <div className="space-y-6 animate-pulse" aria-busy="true">
              <div className="bg-white rounded-2xl border border-slate-200 overflow-hidden">
                <div className="h-36 bg-slate-200" />
                <div className="p-6 pt-0 flex items-end gap-5">
                  <div className="-mt-14 w-24 h-24 rounded-full bg-slate-300 border-4 border-white" />
                  <div className="space-y-2 pb-2">
                    <div className="h-6 w-48 bg-slate-200 rounded-md" />
                    <div className="h-4 w-32 bg-slate-100 rounded-md" />
                  </div>
                </div>
              </div>

              <div className="bg-white rounded-2xl border border-slate-200 p-6 space-y-4">
                <div className="h-5 w-40 bg-slate-200 rounded-md" />
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  {[...Array(6)].map((_, i) => (
                    <div key={i} className="h-16 bg-slate-100 rounded-xl" />
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <>
              {/* Profile Header */}
              <ProfileHeader
                profile={employee}
                currentUserRole={currentUserRole}
                onEdit={() => setIsEditModalOpen(true)}
                onRoleChange={(role) => {
                  setCurrentUserRole(role);
                  showToast(`Role switched to ${role}`, 'info');
                }}
              />

              {/* Employee Information Card */}
              <ProfileInformationCard
                profile={employee}
                currentUserRole={currentUserRole}
                onEdit={() => setIsEditModalOpen(true)}
              />
            </>
          )}
        </div>
      </main>

      {/* Edit Profile Modal */}
      <EditProfileModal
        isOpen={isEditModalOpen}
        profile={employee}
        currentUserRole={currentUserRole}
        onClose={() => setIsEditModalOpen(false)}
        onSave={handleSaveProfile}
      />

      {/* Toast Notification */}
      {toast.visible && (
        <div
          role="status"
          className="fixed bottom-6 right-6 z-50 flex items-center gap-3 px-4 py-3 bg-white rounded-xl shadow-lg border border-slate-200 border-l-4 border-l-teal-500 animate-slideUp max-w-sm"
        >
          {toast.type === 'error' ? (
            <AlertCircle className="w-5 h-5 text-rose-500 flex-shrink-0" />
          ) : (
            <CheckCircle2 className="w-5 h-5 text-teal-600 flex-shrink-0" />
          )}
          <p className="text-xs sm:text-sm font-semibold text-slate-900 m-0">
            {toast.message}
          </p>
        </div>
      )}
    </div>
  );
};

export default EmployeeProfilePage;
