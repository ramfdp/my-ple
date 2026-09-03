<script lang="ts">
	import type { ZoomBooking, ZoomResource } from '$lib/api';
	import { bookingStatus, localDate, weekDays } from '$lib/utils/booking-calendar';

	let { monday, bookings, resources, now, onselect }: { monday: Date; bookings: ZoomBooking[]; resources: ZoomResource[]; now: Date; onselect: (booking: ZoomBooking) => void } = $props();
	const hours = Array.from({ length: 24 }, (_, hour) => hour);
	let days = $derived(weekDays(monday));
	const time = (value: string) => new Date(value).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
	const laneBookings = (day: Date, resourceId: number) => bookings.filter((booking) => booking.zoomResource.id === resourceId && localDate(new Date(booking.startAt)) === localDate(day));
	const top = (booking: ZoomBooking) => { const value = new Date(booking.startAt); return (value.getHours() * 60 + value.getMinutes()) * 0.8; };
	const height = (booking: ZoomBooking) => Math.max(30, (new Date(booking.endAt).getTime() - new Date(booking.startAt).getTime()) / 75000);
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex: scrollable calendar must be keyboard-focusable -->
<div class="calendar-scroll" role="region" tabindex="0" aria-label="Kalender booking Zoom mingguan">
	<div class="zoom-calendar">
		<div class="calendar-corner">LOKAL</div>
		{#each days as day, dayIndex}<div class:today={localDate(day) === localDate(now)} class="calendar-day-head" style={`grid-column:${2 + dayIndex * resources.length}/span ${resources.length}`}><strong>{day.toLocaleDateString('id-ID', { weekday: 'short' })}</strong><span>{day.toLocaleDateString('id-ID', { day: 'numeric', month: 'short' })}</span></div>{/each}
		{#each days as day, dayIndex}{#each resources as resource, resourceIndex}<div class="calendar-resource-head" style={`grid-column:${2 + dayIndex * resources.length + resourceIndex}`}><strong>{resource.name}</strong><span>{resource.code}</span></div>{/each}{/each}
		<div class="time-axis">{#each hours as hour}<span style={`top:${hour * 48}px`}>{String(hour).padStart(2, '0')}:00</span>{/each}</div>
		{#each days as day, dayIndex}{#each resources as resource, resourceIndex}
			<div class:today={localDate(day) === localDate(now)} class="calendar-lane" style={`grid-column:${2 + dayIndex * resources.length + resourceIndex}`}>
				{#each hours as hour}<div class="hour-line" style={`top:${hour * 48}px`}></div>{/each}
				{#if localDate(day) === localDate(now)}<div class="now-line" style={`top:${(now.getHours() * 60 + now.getMinutes()) * 0.8}px`}>{#if resourceIndex === 0}<span>{time(now.toISOString())}</span>{/if}</div>{/if}
				{#each laneBookings(day, resource.id) as booking}<button class={`calendar-booking ${bookingStatus(booking.startAt, booking.endAt, now)}`} style={`top:${top(booking)}px;height:${height(booking)}px`} type="button" onclick={() => onselect(booking)} title={booking.title}><strong>{booking.title}</strong><span>{time(booking.startAt)}–{time(booking.endAt)}</span><small>{booking.createdBy}</small></button>{/each}
			</div>
		{/each}{/each}
	</div>
</div>
