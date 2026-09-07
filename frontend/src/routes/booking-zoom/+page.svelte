<script lang="ts">
	import { onMount } from 'svelte';
	import { api, session, type ZoomBooking, type ZoomBookingInput, type ZoomResource, type ZoomResourceInput } from '$lib/api';
	import ToastNotification from '$lib/components/ToastNotification.svelte';
	import WeeklyZoomCalendar from '$lib/components/WeeklyZoomCalendar.svelte';
	import ZoomBookingForm from '$lib/components/ZoomBookingForm.svelte';
	import ZoomInformationPanel from '$lib/components/ZoomInformationPanel.svelte';
	import ZoomResourceForm from '$lib/components/ZoomResourceForm.svelte';
	import { addDays, bookingStatus, canEditZoomResources, startOfWeek } from '$lib/utils/booking-calendar';

	let weekStart = $state(startOfWeek(new Date()));
	let bookings = $state<ZoomBooking[]>([]);
	let resources = $state<ZoomResource[]>([]);
	let now = $state(new Date());
	let loading = $state(true);
	let busy = $state(false);
	let formOpen = $state(false);
	let editing = $state<ZoomBooking | null>(null);
	let selected = $state<ZoomBooking | null>(null);
	let confirmingDelete = $state(false);
	let editingResource = $state<ZoomResource | null>(null);
	let toast = $state({ visible: false, message: '', type: 'success' as 'success' | 'error' });
	const currentUserID = session()?.user.id;
	const canEditResources = canEditZoomResources(session()?.user.role);
	let weekLabel = $derived(`${weekStart.toLocaleDateString('id-ID', { day: 'numeric', month: 'short' })} – ${addDays(weekStart, 6).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' })}`);

	function notify(message: string, type: 'success' | 'error' = 'success') { toast = { visible: true, message, type }; }
	async function loadBookings() {
		loading = true;
		try {
			const end = addDays(weekStart, 7);
			bookings = (await api<{ bookings: ZoomBooking[] }>(`/zoom-bookings?start=${encodeURIComponent(weekStart.toISOString())}&end=${encodeURIComponent(end.toISOString())}`)).bookings ?? [];
		} catch (error) { notify(error instanceof Error ? error.message : 'Booking tidak dapat dimuat.', 'error'); }
		finally { loading = false; }
	}
	async function moveWeek(days: number) { weekStart = addDays(weekStart, days); await loadBookings(); }
	async function loadResources() {
		try { resources = (await api<{ resources: ZoomResource[] }>('/zoom-resources')).resources ?? []; }
		catch (error) { notify(error instanceof Error ? error.message : 'Informasi Zoom tidak dapat dimuat.', 'error'); }
	}
	async function save(input: ZoomBookingInput) {
		busy = true;
		try {
			await api(editing ? `/zoom-bookings/${editing.id}` : '/zoom-bookings', { method: editing ? 'PUT' : 'POST', body: JSON.stringify(input) });
			formOpen = false; editing = null;
			await loadBookings();
			notify('Booking Zoom berhasil disimpan.');
		} catch (error) { notify(error instanceof Error ? error.message : 'Booking tidak dapat disimpan.', 'error'); }
		finally { busy = false; }
	}
	async function saveResource(input: ZoomResourceInput) {
		if (!editingResource) return;
		busy = true;
		try {
			const { resource } = await api<{ resource: ZoomResource }>(`/zoom-resources/${editingResource.id}`, { method:'PUT', body:JSON.stringify(input) });
			resources = resources.map((item) => item.id === resource.id ? resource : item);
			bookings = bookings.map((booking) => booking.zoomResource.id === resource.id ? { ...booking, zoomResource:resource } : booking);
			editingResource = null; notify('Informasi Zoom berhasil diperbarui.');
		} catch (error) { notify(error instanceof Error ? error.message : 'Informasi Zoom tidak dapat diperbarui.', 'error'); }
		finally { busy = false; }
	}
	async function remove() {
		if (!selected) return;
		busy = true;
		try {
			await api(`/zoom-bookings/${selected.id}`, { method: 'DELETE' });
			selected = null; confirmingDelete = false;
			await loadBookings();
			notify('Booking Zoom berhasil dibatalkan.');
		} catch (error) { notify(error instanceof Error ? error.message : 'Booking tidak dapat dibatalkan.', 'error'); }
		finally { busy = false; }
	}
	const time = (value: string) => new Date(value).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
	const date = (value: string) => new Date(value).toLocaleDateString('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' });
	const duration = (booking: ZoomBooking) => Math.round((new Date(booking.endAt).getTime() - new Date(booking.startAt).getTime()) / 60000);
	const statusLabel = (booking: ZoomBooking) => ({ upcoming: 'Upcoming', ongoing: 'Ongoing', completed: 'Completed' })[bookingStatus(booking.startAt, booking.endAt, now)];

	onMount(() => {
		void Promise.all([loadBookings(), loadResources()]);
		const timer = setInterval(() => (now = new Date()), 30_000);
		return () => clearInterval(timer);
	});
</script>

<svelte:head><title>Booking Zoom | my HIS</title><meta name="description" content="Jadwalkan dan kelola meeting Zoom." /></svelte:head>
<main class="booking-page"><div class="booking-shell">
	<a class="booking-back" href="/dashboard">← Kembali ke dashboard</a>
	<header class="booking-header"><div><p class="zoom-kicker">BOOKING ZOOM</p><h1>Booking Zoom</h1><p>Jadwalkan dan kelola meeting Zoom Anda.</p></div><button class="zoom-primary" type="button" onclick={() => { editing = null; formOpen = true; }}>+ Buat Booking</button></header>
	<div class="booking-workspace"><section class="schedule-card" aria-labelledby="week-title">
		<div class="schedule-toolbar"><button type="button" class="week-button" onclick={() => void moveWeek(-7)} aria-label="Minggu sebelumnya">←</button><div><p>JADWAL MINGGUAN</p><h2 id="week-title">{weekLabel}</h2></div><div class="week-actions"><button type="button" class="today-button" onclick={() => { weekStart = startOfWeek(new Date()); void loadBookings(); }}>Hari ini</button><button type="button" class="week-button" onclick={() => void moveWeek(7)} aria-label="Minggu berikutnya">→</button></div></div>
		{#if loading}<div class="calendar-state" aria-live="polite">Memuat booking…</div>
		{:else if resources.length}{#if bookings.length === 0}<div class="empty-booking"><strong>Belum ada booking Zoom minggu ini.</strong><button type="button" onclick={() => (formOpen = true)}>Buat booking pertama</button></div>{/if}<WeeklyZoomCalendar monday={weekStart} {bookings} {resources} {now} onselect={(booking) => (selected = booking)} />{/if}
	</section><ZoomInformationPanel {resources} canEdit={canEditResources} onedit={(resource) => (editingResource = resource)} /></div>
</div></main>

{#if formOpen}<ZoomBookingForm booking={editing} {resources} {busy} onclose={() => { if (!busy) { formOpen = false; editing = null; } }} onsave={(input) => void save(input)} />{/if}
{#if editingResource}<ZoomResourceForm resource={editingResource} {busy} onclose={() => { if (!busy) editingResource = null; }} onsave={(input) => void saveResource(input)} />{/if}
{#if selected}
	<div class="zoom-modal-layer"><button class="zoom-modal-backdrop" type="button" aria-label="Tutup detail" onclick={() => (selected = null)}></button><div class="zoom-modal detail-modal" role="dialog" aria-modal="true" aria-labelledby="booking-detail-title">
		<header><div><span class={`status-pill ${bookingStatus(selected.startAt, selected.endAt, now)}`}>{statusLabel(selected)}</span><h2 id="booking-detail-title">{selected.title}</h2></div><button type="button" class="zoom-close" aria-label="Tutup" onclick={() => (selected = null)}>×</button></header>
		<dl><div><dt>Zoom</dt><dd>{selected.zoomResource.name} — {selected.zoomResource.code}</dd></div><div><dt>Tanggal</dt><dd>{date(selected.startAt)}</dd></div><div><dt>Waktu</dt><dd>{time(selected.startAt)} – {time(selected.endAt)} ({duration(selected)} menit)</dd></div><div><dt>Dibuat oleh</dt><dd>{selected.createdBy}</dd></div>{#if selected.description}<div><dt>Deskripsi</dt><dd>{selected.description}</dd></div>{/if}</dl>
		<footer>{#if selected.zoomResource.meetingUrl}<a class="zoom-primary" href={selected.zoomResource.meetingUrl} target="_blank" rel="noreferrer">Join Zoom</a>{/if}{#if selected.userId === currentUserID && bookingStatus(selected.startAt, selected.endAt, now) === 'upcoming'}<button class="zoom-secondary" type="button" onclick={() => { editing = selected; selected = null; formOpen = true; }}>Edit</button><button class="zoom-danger" type="button" onclick={() => (confirmingDelete = true)}>Batalkan</button>{/if}</footer>
		{#if confirmingDelete}<div class="delete-confirm"><p>Yakin ingin membatalkan booking Zoom ini?</p><div><button class="zoom-secondary" type="button" onclick={() => (confirmingDelete = false)}>Tidak</button><button class="zoom-danger" type="button" disabled={busy} onclick={() => void remove()}>{busy ? 'Membatalkan…' : 'Ya, batalkan'}</button></div></div>{/if}
	</div></div>
{/if}
<ToastNotification message={toast.message} type={toast.type} visible={toast.visible} onclose={() => (toast.visible = false)} />
