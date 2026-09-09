import { get, writable } from 'svelte/store';

export const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';

export interface User {
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
 pics?:JobPIC[];
 jobId?: number;
	id: string; userId: string; primaryDivision: string; workDivision: string;
	role: string; work: string; number: number; description: string; completedAt: string;
}
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
  hostKey: string; id: number; name: string; code: string; description: string; meetingUrl: string; meetingId: string; createdAt: string; updatedAt: string 
}
export interface ZoomResourceInput { description: string; meetingUrl: string; meetingId: string; hostKey: string }
export interface Notification {
	id: string; type: 'zoom_booking_created'|'workspace_task'; actor: { id: string; name: string };
	workspace?: {title:string;workspaceId:number;message:string};
	entityType: string; entityId: string; createdAt: string; readAt: string | null;
	booking?: { id: string; zoomName: string; startAt: string; endAt: string };
}

export const currentUser = writable<User | null>(null);
export function session(): { user: User } | null { const user = get(currentUser); return user ? { user } : null; }
export function setAuthenticated(user: User | null) { currentUser.set(user); }

async function checked(response: Response, fallback: string) {
	if (response.status === 401) {
		setAuthenticated(null);
		if (typeof window !== 'undefined' && location.pathname !== '/') location.assign('/');
	}
	if (!response.ok) throw new Error((await response.json().catch(() => ({}))).error || fallback);
	return response;
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
	const response = await checked(await fetch(`${apiUrl}${path}`, { ...options, credentials: 'include', headers: { 'Content-Type': 'application/json', ...options.headers } }), 'Request failed.');
	return response.status === 204 ? undefined as T : response.json();
}

export async function upload<T>(path: string, body: FormData): Promise<T> {
	const response = await checked(await fetch(`${apiUrl}${path}`, { method: 'POST', body, credentials: 'include' }), 'Upload failed.');
	return response.json();
}

export async function download(path: string): Promise<{ blob: Blob; filename: string }> {
	const response = await checked(await fetch(`${apiUrl}${path}`, { credentials: 'include' }), 'Download failed.');
	const filename = response.headers.get('Content-Disposition')?.match(/filename="([^"]+)"/)?.[1] ?? 'Weekly_Report.pdf';
	return { blob: await response.blob(), filename };
}

export const assetUrl = (path: string) => path.startsWith('/uploads/') ? `${apiUrl}${path}` : path;
