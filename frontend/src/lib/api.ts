import { errorMessages } from './messages';
import { setLanguage, hasMessage } from './i18n';
import { get, writable } from 'svelte/store';

export const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';

export interface User {
 language: 'id' | 'en';
	id: string; name: string; email: string; username: string; roleId: number; role: string;
	jobTitle: string; salary: number; residence: string; phoneNumber: string;
	employmentStatus: string; avatarUrl: string; isActive: boolean; createdAt: string; updatedAt: string;
}
export interface Role { id: number; name: string }
export interface JobPIC { userId:string; name:string; isActive:boolean; username?:string }
export interface Job { pics?:JobPIC[]; id: number; divisionId: number; roleId: number | null; jobCategoryId: number | null; name: string; description: string; isActive: boolean; sortOrder: number; division: string; role: string; category: string; referenced: boolean }
export interface JobMaster { id: number; name: string; divisionId?: number; roleId: number | null }
export interface JobMasters { divisions: JobMaster[]; roles: JobMaster[]; categories: JobMaster[] }
export interface Report {
 version:number;
 pics?:JobPIC[];
 jobId?: number;
	id: string; userId: string; primaryDivision: string; workDivision: string;
	role: string; work: string; number: number; description: string; completedAt: string;
}
export interface WeeklyProgressGroup { number:number; content:string; report:Report }
export interface WeeklyJobGroup { key:string; name:string; pics?:JobPIC[]; progress:WeeklyProgressGroup[] }
export interface WeeklyCategoryGroup { name:string; jobs:WeeklyJobGroup[] }
export interface WeeklyRoleGroup { name:string; categories:WeeklyCategoryGroup[] }
export interface ReportingPeriod { start: string; end: string; label: string; readOnly: boolean; status?: 'active' | 'pending' | 'failed' | 'finalized' }
export interface WeeklyArchive {
	id: string; division: string; weekStart: string; weekEnd: string; status: 'pending' | 'finalized' | 'failed';
	finalizedAt: string | null; pdfGeneratedAt: string | null; filename: string;
	items?: (Report & { employeeName: string; employeeRole: string })[];
}
export interface ReportInput { jobId?: number; primaryDivision: string; workDivision: string; role: string; work: string; description: string }
export interface ZoomBooking {
	id: string; userId: string; createdBy: string; title: string; description: string; zoomResource: ZoomResource;
	startAt: string; endAt: string; createdAt: string; updatedAt: string;
}
export interface ZoomBookingInput { title: string; description: string; zoomResourceId: number; startAt: string; endAt: string }
export interface ZoomResource {
  passcode: string; hostKey: string; id: number; name: string; code: string; description: string; meetingUrl: string; meetingId: string; createdAt: string; updatedAt: string
}
export interface ZoomResourceInput { description: string; meetingUrl: string; meetingId: string; hostKey: string; passcode: string }
export interface Notification {
 leave?: import('./leave').LeaveRequest & {title:string;message:string;path:string};
	overtime?: {sheetId:string;employeeName:string;period:string;status:string;title:string;message:string;path:string};
	id: string; type: 'zoom_booking_created'|'workspace_task'|'work_item_assigned'|'leave_request_created'|'leave_request_approved'|'leave_request_rejected'|'overtime_submitted'|'overtime_approved'|'overtime_rejected'; actor: { id: string; name: string };
	workspace?: {title:string;workspaceId:number;message:string;projectName?:string;startDate?:string;dueDate?:string;priority?:string};
	entityType: string; entityId: string; createdAt: string; readAt: string | null;
	booking?: { id: string; zoomName: string; startAt: string; endAt: string; description?: string };
}

export const currentUser = writable<User | null>(null);
export function session(): { user: User } | null { const user = get(currentUser); return user ? { user } : null; }
export function setAuthenticated(user: User | null) { currentUser.set(user); if(user) setLanguage(user.language); }

export class APIError extends Error {
 constructor(message:string, public status:number, public data:Record<string,unknown>){super(message)}
}
async function checked(response: Response, fallback: string) {
	if (response.status === 401) {
		setAuthenticated(null);
		if (typeof window !== 'undefined' && location.pathname !== '/') location.assign('/');
	}
	if (!response.ok) { const data=await response.json().catch(()=>({})); throw new APIError((Object.hasOwn(errorMessages,data.messageCode) ? errorMessages[data.messageCode] : undefined) || (hasMessage(data.error) ? data.error : hasMessage(fallback) ? fallback : 'Terjadi kesalahan. Coba lagi.'),response.status,data); }
	return response;
}

async function fetchResponse(url: string, options: RequestInit) {
 try { return await fetch(url,options); }
 catch { throw new APIError('Request failed.',0,{}); }
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
	const response = await checked(await fetchResponse(`${apiUrl}${path}`, { ...options, credentials: 'include', headers: { 'Content-Type': 'application/json', ...options.headers } }), 'Request failed.');
	return response.status === 204 ? undefined as T : response.json();
}

export async function upload<T>(path: string, body: FormData): Promise<T> {
	const response = await checked(await fetchResponse(`${apiUrl}${path}`, { method: 'POST', body, credentials: 'include' }), 'Upload failed.');
	return response.json();
}

export async function download(path: string): Promise<void> {
	const response = await checked(await fetchResponse(`${apiUrl}${path}`, { credentials: 'include' }), 'Download failed.');
	const filename = response.headers.get('Content-Disposition')?.match(/filename="([^"]+)"/)?.[1] ?? 'Weekly_Report.pdf';
	const url = URL.createObjectURL(await response.blob());
	const link = document.createElement('a');
	link.href = url;
	link.download = filename;
	try { link.click(); } finally { URL.revokeObjectURL(url); }
}

export const assetUrl = (path: string) => path.startsWith('/uploads/') ? `${apiUrl}${path}` : path;
