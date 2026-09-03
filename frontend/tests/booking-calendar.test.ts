import assert from 'node:assert/strict';
import test from 'node:test';
import { bookingStatus, canEditZoomResources, startOfWeek, weekDays } from '../src/lib/utils/booking-calendar.ts';

test('week starts on Monday and contains seven days', () => {
	const monday = startOfWeek(new Date(2026, 8, 3, 12));
	assert.equal(monday.getDay(), 1);
	assert.equal(monday.getDate(), 31);
	assert.deepEqual(weekDays(monday).map((day) => day.getDate()), [31, 1, 2, 3, 4, 5, 6]);
});

test('only Admin and Management can edit Zoom resources', () => {
	assert.equal(canEditZoomResources('Admin'), true);
	assert.equal(canEditZoomResources('management'), true);
	assert.equal(canEditZoomResources('HR'), false);
	assert.equal(canEditZoomResources('Project Engineer'), false);
});

test('booking status changes exactly at start and end', () => {
	const start = '2026-09-05T09:00:00.000Z';
	const end = '2026-09-05T10:00:00.000Z';
	assert.equal(bookingStatus(start, end, new Date('2026-09-05T08:59:59.000Z')), 'upcoming');
	assert.equal(bookingStatus(start, end, new Date(start)), 'ongoing');
	assert.equal(bookingStatus(start, end, new Date(end)), 'completed');
});
