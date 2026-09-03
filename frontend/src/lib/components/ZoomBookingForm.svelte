<script lang="ts">
	import type { ZoomBooking, ZoomBookingInput, ZoomResource } from '$lib/api';
	import { localDate } from '$lib/utils/booking-calendar';

	let { booking = null, resources, busy = false, onclose, onsave }: {
		booking?: ZoomBooking | null; resources: ZoomResource[]; busy?: boolean; onclose: () => void; onsave: (input: ZoomBookingInput) => void;
	} = $props();
	let title = $state('');
	let description = $state('');
	let zoomResourceId = $state(0);
	let date = $state('');
	let startTime = $state('');
	let endTime = $state('');
	let error = $state('');
	$effect(() => {
		const start = booking ? new Date(booking.startAt) : new Date(Date.now() + 60 * 60 * 1000);
		title = booking?.title ?? ''; description = booking?.description ?? ''; zoomResourceId = booking?.zoomResource.id ?? resources[0]?.id ?? 0;
		date = localDate(start); startTime = start.toTimeString().slice(0, 5);
		endTime = (booking ? new Date(booking.endAt) : new Date(start.getTime() + 60 * 60 * 1000)).toTimeString().slice(0, 5);
	});

	function submit(event: SubmitEvent) {
		event.preventDefault();
		const start = new Date(`${date}T${startTime}:00`);
		const end = new Date(`${date}T${endTime}:00`);
		if (!title.trim()) { error = 'Judul meeting wajib diisi.'; return; }
		if (!zoomResourceId) { error = 'Zoom resource wajib dipilih.'; return; }
		if (!date || !startTime || !endTime || Number.isNaN(start.valueOf()) || Number.isNaN(end.valueOf())) { error = 'Tanggal dan waktu wajib diisi.'; return; }
		if (start >= end) { error = 'Waktu selesai harus setelah waktu mulai.'; return; }
		error = '';
		onsave({ title: title.trim(), description: description.trim(), zoomResourceId, startAt: start.toISOString(), endAt: end.toISOString() });
	}
</script>

<svelte:window onkeydown={(event) => event.key === 'Escape' && !busy && onclose()} />
<div class="zoom-modal-layer">
	<button class="zoom-modal-backdrop" type="button" aria-label="Tutup formulir" onclick={onclose}></button>
	<div class="zoom-modal" role="dialog" aria-modal="true" aria-labelledby="booking-form-title">
		<header><div><p class="zoom-kicker">BOOKING ZOOM</p><h2 id="booking-form-title">{booking ? 'Edit booking' : 'Buat booking baru'}</h2></div><button type="button" class="zoom-close" aria-label="Tutup" onclick={onclose}>×</button></header>
		<form onsubmit={submit}>
			<label>Judul meeting <span>*</span><input bind:value={title} required maxlength="200" /></label>
			<label>Deskripsi<textarea bind:value={description} maxlength="1000" rows="3" placeholder="Catatan singkat (opsional)"></textarea></label>
			<label>Zoom Resource <span>*</span><select bind:value={zoomResourceId} required><option value={0} disabled>Pilih Zoom</option>{#each resources as resource}<option value={resource.id}>{resource.name} — {resource.code}</option>{/each}</select></label>
			<div class="zoom-form-grid">
				<label>Tanggal <span>*</span><input type="date" bind:value={date} required /></label>
				<label>Mulai <span>*</span><input type="time" bind:value={startTime} required /></label>
				<label>Selesai <span>*</span><input type="time" bind:value={endTime} required /></label>
			</div>
			{#if error}<p class="zoom-form-error" role="alert">{error}</p>{/if}
			<footer><button type="button" class="zoom-secondary" onclick={onclose} disabled={busy}>Batal</button><button type="submit" class="zoom-primary" disabled={busy}>{busy ? 'Menyimpan…' : 'Simpan booking'}</button></footer>
		</form>
	</div>
</div>
