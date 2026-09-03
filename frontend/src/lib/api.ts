const apiUrl = import.meta.env.VITE_API_URL || 'http://localhost:3000';

export interface User {
	id: string; name: string; email: string; username: string; roleId: number; role: string;
	jobTitle: string; salary: number; residence: string; phoneNumber: string;
	employmentStatus: string; avatarUrl: string; isActive: boolean; createdAt: string; updatedAt: string;
}
export interface Role { id: number; name: string }
export interface Report {
	id: string; userId: string; primaryDivision: string; workDivision: string;
	role: string; work: string; number: number; description: string; completedAt: string;
}
export interface ReportingPeriod { start: string; end: string; label: string }
export interface ReportInput { primaryDivision: string; workDivision: string; role: string; work: string; description: string }
export interface ZoomBooking {
	id: string; userId: string; createdBy: string; title: string; description: string; zoomResource: ZoomResource;
	startAt: string; endAt: string; createdAt: string; updatedAt: string;
}
export interface ZoomBookingInput { title: string; description: string; zoomResourceId: number; startAt: string; endAt: string }
export interface ZoomResource {
  hostKey: string; id: number; name: string; code: string; description: string; meetingUrl: string; meetingId: string; createdAt: string; updatedAt: string 
}
export interface ZoomResourceInput { description: string; meetingUrl: string; meetingId: string; hostKey: string }

export function session(): { token: string; user: User } | null {
	if (typeof window === 'undefined') return null;
	const value = localStorage.getItem('session');
	return value ? JSON.parse(value) : null;
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
	const current = session();
	const response = await fetch(`${apiUrl}${path}`, { ...options, headers: { 'Content-Type': 'application/json', ...(current ? { Authorization: `Bearer ${current.token}` } : {}), ...options.headers } });
	if (!response.ok) throw new Error((await response.json().catch(() => ({}))).error || 'Request failed.');
	return response.status === 204 ? undefined as T : response.json();
}

export async function upload<T>(path: string, body: FormData): Promise<T> {
	const current = session();
	const response = await fetch(`${apiUrl}${path}`, { method: 'POST', body, headers: current ? { Authorization: `Bearer ${current.token}` } : {} });
	if (!response.ok) throw new Error((await response.json().catch(() => ({}))).error || 'Upload failed.');
	return response.json();
}

export async function download(path: string): Promise<{ blob: Blob; filename: string }> {
	const current = session();
	const response = await fetch(`${apiUrl}${path}`, { headers: current ? { Authorization: `Bearer ${current.token}` } : {} });
	if (!response.ok) throw new Error((await response.json().catch(() => ({}))).error || 'Download failed.');
	const filename = response.headers.get('Content-Disposition')?.match(/filename="([^"]+)"/)?.[1] ?? 'Weekly_Report.pdf';
	return { blob: await response.blob(), filename };
}

export const assetUrl = (path: string) => path.startsWith('/uploads/') ? `${apiUrl}${path}` : path;
