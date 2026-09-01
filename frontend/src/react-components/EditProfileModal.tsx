import React, { useState, useEffect } from 'react';
import { X, Lock, Loader2 } from 'lucide-react';
import type { EmployeeProfile, EmploymentStatus, ProfileFormErrors, UserRole } from './types';
import { ProfileAvatar } from './ProfileAvatar';

interface EditProfileModalProps {
  isOpen: boolean;
  profile: EmployeeProfile;
  currentUserRole?: UserRole;
  onClose: () => void;
  onSave: (updated: EmployeeProfile) => Promise<void> | void;
}

const divisionOptions = [
  'Information Technology',
  'Software Engineering',
  'AI Dev',
  'Project Engineer',
  'Keuangan',
  'Business Development',
  'HR',
  'PMO',
];

const statusOptions: EmploymentStatus[] = [
  'Permanent Employee',
  'Contract Employee',
  'Probation',
  'Intern',
  'Freelancer',
  'Inactive',
];

export const EditProfileModal: React.FC<EditProfileModalProps> = ({
  isOpen,
  profile,
  currentUserRole = 'Employee',
  onClose,
  onSave,
}) => {
  const [formData, setFormData] = useState<EmployeeProfile>(profile);
  const [errors, setErrors] = useState<ProfileFormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  const isHR = currentUserRole === 'HR / Admin';

  useEffect(() => {
    if (isOpen) {
      setFormData({ ...profile });
      setErrors({});
    }
  }, [isOpen, profile]);

  if (!isOpen) return null;

  const validate = (): boolean => {
    const newErrors: ProfileFormErrors = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Full name is required.';
    }

    if (!formData.division.trim()) {
      newErrors.division = 'Division is required.';
    }

    if (formData.currentSalary <= 0 || isNaN(formData.currentSalary)) {
      newErrors.currentSalary = 'Current salary must be a positive number.';
    }

    if (!formData.residence.trim()) {
      newErrors.residence = 'Residence address is required.';
    }

    // Indonesian phone regex validation
    const cleanPhone = formData.phoneNumber.replace(/[\s-]/g, '');
    if (!formData.phoneNumber.trim()) {
      newErrors.phoneNumber = 'Phone number is required.';
    } else if (!/^(?:\+62|62|0)8[1-9]\d{7,10}$/.test(cleanPhone)) {
      newErrors.phoneNumber =
        'Valid Indonesian phone number required (e.g., +62 812-3456-7890).';
    }

    // Email regex validation
    if (!formData.email.trim()) {
      newErrors.email = 'Email address is required.';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Please enter a valid email address.';
    }

    if (!formData.employmentStatus) {
      newErrors.employmentStatus = 'Employment status is required.';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) return;

    setIsSubmitting(true);
    try {
      await onSave(formData);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-fadeIn"
      onClick={onClose}
      role="presentation"
    >
      <div
        className="w-full max-w-2xl bg-white rounded-2xl shadow-xl border border-slate-200 overflow-hidden max-h-[90vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby="modal-headline"
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-100">
          <div>
            <h2 id="modal-headline" className="text-lg font-bold text-slate-900">
              Edit Employee Profile
            </h2>
            <p className="text-xs text-slate-500">
              Update employee details and organizational information
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close modal"
            className="w-8 h-8 rounded-lg bg-slate-100 text-slate-500 hover:bg-slate-200 hover:text-slate-900 flex items-center justify-center transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Non-HR Notice */}
        {!isHR && (
          <div className="mx-6 mt-4 p-3 rounded-xl bg-slate-50 border border-slate-200 flex items-center gap-2.5 text-xs text-slate-600">
            <Lock className="w-4 h-4 text-teal-600 flex-shrink-0" />
            <span>
              You are logged in as <strong>Employee</strong>. Division, Salary, and
              Status fields are restricted to HR/Admin.
            </span>
          </div>
        )}

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto p-6 space-y-5" noValidate>
          {/* Avatar edit */}
          <div className="flex items-center gap-4 pb-4 border-b border-slate-100">
            <ProfileAvatar
              src={formData.avatarUrl}
              alt={formData.name}
              size={64}
              borderWidth={3}
            />
            <div className="flex-1">
              <label htmlFor="modal-avatar-url" className="block text-xs font-semibold text-slate-700 mb-1">
                Profile Photo URL
              </label>
              <input
                id="modal-avatar-url"
                type="url"
                value={formData.avatarUrl}
                onChange={(e) => setFormData({ ...formData, avatarUrl: e.target.value })}
                placeholder="https://images.unsplash.com/..."
                className="w-full text-xs sm:text-sm px-3.5 py-2 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-teal-500/20 focus:border-teal-500 transition-all"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {/* Name */}
            <div>
              <label htmlFor="modal-name" className="block text-xs font-semibold text-slate-700 mb-1">
                Full Name <span className="text-rose-500">*</span>
              </label>
              <input
                id="modal-name"
                type="text"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className={`w-full text-sm px-3.5 py-2 rounded-xl border ${
                  errors.name ? 'border-rose-500 ring-1 ring-rose-500/20' : 'border-slate-300'
                } focus:outline-none focus:ring-2 focus:ring-teal-500/20 focus:border-teal-500 transition-all`}
              />
              {errors.name && <p className="text-xs text-rose-500 mt-1">{errors.name}</p>}
            </div>

            {/* Job Title */}
            <div>
              <label htmlFor="modal-job-title" className="block text-xs font-semibold text-slate-700 mb-1">
                Job Title
              </label>
              <input
                id="modal-job-title"
                type="text"
                value={formData.jobTitle}
                onChange={(e) => setFormData({ ...formData, jobTitle: e.target.value })}
                className="w-full text-sm px-3.5 py-2 rounded-xl border border-slate-300 focus:outline-none focus:ring-2 focus:ring-teal-500/20 focus:border-teal-500 transition-all"
              />
            </div>

            {/* Division (HR Only) */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label htmlFor="modal-division" className="text-xs font-semibold text-slate-700">
                  Division <span className="text-rose-500">*</span>
                </label>
                {!isHR && (
                  <span className="text-[10px] font-semibold text-slate-400 bg-slate-100 px-1.5 py-0.5 rounded">
                    HR Only
                  </span>
                )}
              </div>
              <select
                id="modal-division"
                value={formData.division}
                disabled={!isHR}
                onChange={(e) => setFormData({ ...formData, division: e.target.value })}
                className={`w-full text-sm px-3.5 py-2 rounded-xl border ${
                  !isHR
                    ? 'bg-slate-50 text-slate-400 border-slate-200 cursor-not-allowed'
                    : 'border-slate-300 focus:ring-teal-500/20 focus:border-teal-500'
                } ${errors.division ? 'border-rose-500' : ''} transition-all`}
              >
                {divisionOptions.map((div) => (
                  <option key={div} value={div}>
                    {div}
                  </option>
                ))}
              </select>
              {errors.division && (
                <p className="text-xs text-rose-500 mt-1">{errors.division}</p>
              )}
            </div>

            {/* Salary (HR Only) */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label htmlFor="modal-salary" className="text-xs font-semibold text-slate-700">
                  Current Salary (IDR) <span className="text-rose-500">*</span>
                </label>
                {!isHR && (
                  <span className="text-[10px] font-semibold text-slate-400 bg-slate-100 px-1.5 py-0.5 rounded">
                    HR Only
                  </span>
                )}
              </div>
              <input
                id="modal-salary"
                type="number"
                value={formData.currentSalary}
                disabled={!isHR}
                step="100000"
                min="1"
                onChange={(e) =>
                  setFormData({ ...formData, currentSalary: Number(e.target.value) })
                }
                className={`w-full text-sm px-3.5 py-2 rounded-xl border ${
                  !isHR
                    ? 'bg-slate-50 text-slate-400 border-slate-200 cursor-not-allowed'
                    : 'border-slate-300 focus:ring-teal-500/20 focus:border-teal-500'
                } ${errors.currentSalary ? 'border-rose-500' : ''} transition-all`}
              />
              {errors.currentSalary && (
                <p className="text-xs text-rose-500 mt-1">{errors.currentSalary}</p>
              )}
            </div>

            {/* Residence */}
            <div>
              <label htmlFor="modal-residence" className="block text-xs font-semibold text-slate-700 mb-1">
                Residence <span className="text-rose-500">*</span>
              </label>
              <input
                id="modal-residence"
                type="text"
                value={formData.residence}
                onChange={(e) => setFormData({ ...formData, residence: e.target.value })}
                className={`w-full text-sm px-3.5 py-2 rounded-xl border ${
                  errors.residence ? 'border-rose-500' : 'border-slate-300'
                } focus:outline-none focus:ring-2 focus:ring-teal-500/20 focus:border-teal-500 transition-all`}
              />
              {errors.residence && (
                <p className="text-xs text-rose-500 mt-1">{errors.residence}</p>
              )}
            </div>

            {/* Phone Number */}
            <div>
              <label htmlFor="modal-phone" className="block text-xs font-semibold text-slate-700 mb-1">
                Phone Number <span className="text-rose-500">*</span>
              </label>
              <input
                id="modal-phone"
                type="tel"
                value={formData.phoneNumber}
                onChange={(e) => setFormData({ ...formData, phoneNumber: e.target.value })}
                placeholder="+62 812-3456-7890"
                className={`w-full text-sm px-3.5 py-2 rounded-xl border ${
                  errors.phoneNumber ? 'border-rose-500' : 'border-slate-300'
                } focus:outline-none focus:ring-2 focus:ring-teal-500/20 focus:border-teal-500 transition-all`}
              />
              {errors.phoneNumber && (
                <p className="text-xs text-rose-500 mt-1">{errors.phoneNumber}</p>
              )}
            </div>

            {/* Email */}
            <div>
              <label htmlFor="modal-email" className="block text-xs font-semibold text-slate-700 mb-1">
                Email Address <span className="text-rose-500">*</span>
              </label>
              <input
                id="modal-email"
                type="email"
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                className={`w-full text-sm px-3.5 py-2 rounded-xl border ${
                  errors.email ? 'border-rose-500' : 'border-slate-300'
                } focus:outline-none focus:ring-2 focus:ring-teal-500/20 focus:border-teal-500 transition-all`}
              />
              {errors.email && (
                <p className="text-xs text-rose-500 mt-1">{errors.email}</p>
              )}
            </div>

            {/* Employment Status (HR Only) */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label htmlFor="modal-status" className="text-xs font-semibold text-slate-700">
                  Employment Status <span className="text-rose-500">*</span>
                </label>
                {!isHR && (
                  <span className="text-[10px] font-semibold text-slate-400 bg-slate-100 px-1.5 py-0.5 rounded">
                    HR Only
                  </span>
                )}
              </div>
              <select
                id="modal-status"
                value={formData.employmentStatus}
                disabled={!isHR}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    employmentStatus: e.target.value as EmploymentStatus,
                  })
                }
                className={`w-full text-sm px-3.5 py-2 rounded-xl border ${
                  !isHR
                    ? 'bg-slate-50 text-slate-400 border-slate-200 cursor-not-allowed'
                    : 'border-slate-300 focus:ring-teal-500/20 focus:border-teal-500'
                } ${errors.employmentStatus ? 'border-rose-500' : ''} transition-all`}
              >
                {statusOptions.map((st) => (
                  <option key={st} value={st}>
                    {st}
                  </option>
                ))}
              </select>
              {errors.employmentStatus && (
                <p className="text-xs text-rose-500 mt-1">{errors.employmentStatus}</p>
              )}
            </div>
          </div>

          {/* Footer Actions */}
          <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-4 py-2 rounded-xl border border-slate-300 text-slate-700 font-semibold text-sm hover:bg-slate-50 transition-all"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="inline-flex items-center gap-2 px-5 py-2 rounded-xl bg-teal-600 hover:bg-teal-700 text-white font-semibold text-sm shadow-sm transition-all disabled:opacity-60"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Saving...</span>
                </>
              ) : (
                <span>Save Changes</span>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
