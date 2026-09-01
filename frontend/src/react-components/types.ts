// ponytail: React TypeScript interfaces for Employee Profile Dashboard
export type EmploymentStatus =
  | 'Permanent Employee'
  | 'Contract Employee'
  | 'Probation'
  | 'Intern'
  | 'Freelancer'
  | 'Inactive';

export type UserRole = 'Employee' | 'HR / Admin';

export interface EmployeeProfile {
  id: string;
  name: string;
  jobTitle: string;
  division: string;
  currentSalary: number;
  residence: string;
  phoneNumber: string;
  email: string;
  employmentStatus: EmploymentStatus;
  avatarUrl: string;
  isActive: boolean;
}

export interface ProfileFormErrors {
  name?: string;
  division?: string;
  currentSalary?: string;
  residence?: string;
  phoneNumber?: string;
  email?: string;
  employmentStatus?: string;
}
