const apiUrl = 'http://localhost:3000';

export interface User {
	id: string; name: string; email: string; username: string; roleId: number; role: string;
	jobTitle: string; salary: number; residence: string; phoneNumber: string;
	employmentStatus: string; avatarUrl: string; isActive: boolean; createdAt: string; updatedAt: string;
}
export interface Role { id: number; name: string }

export function session(): { token: string; user: User } | null {
	if (typeof localStorage === 'undefined') return null;
	const value = localStorage.getItem('session');
	return value ? JSON.parse(value) : null;
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
	const current = session();
	const response = await fetch(`${apiUrl}${path}`, { ...options, headers: { 'Content-Type': 'application/json', ...(current ? { Authorization: `Bearer ${current.token}` } : {}), ...options.headers } });
	if (!response.ok) throw new Error((await response.json().catch(() => ({}))).error || 'Request failed.');
	return response.status === 204 ? undefined as T : response.json();
}
