export type BookingStatus = 'upcoming' | 'ongoing' | 'completed';
export const canEditZoomResources = (role?: string) => ['admin', 'management'].includes(role?.toLowerCase() ?? '');

export function startOfWeek(date: Date): Date {
	const result = new Date(date);
	result.setHours(0, 0, 0, 0);
	result.setDate(result.getDate() - ((result.getDay() + 6) % 7));
	return result;
}

export function addDays(date: Date, days: number): Date {
	const result = new Date(date);
	result.setDate(result.getDate() + days);
	return result;
}

export function weekDays(monday: Date): Date[] {
	return Array.from({ length: 7 }, (_, index) => addDays(monday, index));
}

export function bookingStatus(startAt: string, endAt: string, now: Date): BookingStatus {
	if (now < new Date(startAt)) return 'upcoming';
	if (now < new Date(endAt)) return 'ongoing';
	return 'completed';
}

export function localDate(date: Date): string {
	const year = date.getFullYear();
	const month = String(date.getMonth() + 1).padStart(2, '0');
	const day = String(date.getDate()).padStart(2, '0');
	return `${year}-${month}-${day}`;
}
