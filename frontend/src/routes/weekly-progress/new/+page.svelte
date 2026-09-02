<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api, type ReportInput } from '$lib/api';

	const divisions = ['Project Engineer', 'AI Dev', 'Keuangan', 'Business Development', 'HR', 'PMO'];
	const workDivisions: Record<string, string[]> = {
		'Project Engineer': ['Quality Assurance 1', 'Quality Assurance 2', 'Pengawasan', 'Non-Pengawasan', 'CIHPK'],
		'AI Dev': ['Pengembangan', 'Pengujian Model', 'Deployment', 'Dokumentasi'],
		Keuangan: ['Pembukuan', 'Pembayaran', 'Anggaran', 'Pelaporan'],
		'Business Development': ['Prospek', 'Proposal', 'Kemitraan', 'Tindak Lanjut'],
		HR: ['Kehadiran', 'Rekrutmen', 'Administrasi', 'Pengembangan SDM'],
		PMO: ['Milestone', 'Rapat', 'Risiko', 'Pelaporan']
	};

	let division = $state(divisions[0]);
	let workDivision = $state(workDivisions[divisions[0]][0]);
	let work = $state('');
	let error = $state('');
	let submitting = $state(false);

	$effect(() => {
		if (!workDivisions[division].includes(workDivision)) workDivision = workDivisions[division][0];
	});

	onMount(() => {
		localStorage.removeItem('weekly-reports');
		const requestedDivision = new URLSearchParams(location.search).get('division');
		if (requestedDivision && divisions.includes(requestedDivision)) division = requestedDivision;
	});

	async function saveReport(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		submitting = true;
		try {
			await api<{ report: unknown }>('/reports', {
				method: 'POST',
				body: JSON.stringify({ primaryDivision: division, workDivision, description: work.trim() } satisfies ReportInput)
			});
			await goto(`/weekly-progress?division=${encodeURIComponent(division)}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Laporan tidak dapat disimpan.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Input Laporan | my-ple</title>
	<meta name="description" content="Input laporan pekerjaan mingguan." />
</svelte:head>

<main class="weekly-progress-page">
	<div class="weekly-progress-shell report-shell">
		<a class="back-link" href="/weekly-progress">← Kembali ke weekly progress</a>
		<header class="weekly-header">
			<p class="weekly-kicker">WEEKLY PROGRESS</p>
			<h1>Input laporan pekerjaan.</h1>
			<p>Isi pekerjaan yang sudah dilakukan agar tampil pada daftar progres.</p>
		</header>

		<form class="report-form" onsubmit={saveReport}>
			<label>Divisi utama<select bind:value={division}>{#each divisions as item}<option value={item}>{item}</option>{/each}</select></label>
			<label>Kategori pekerjaan<select bind:value={workDivision}>{#each workDivisions[division] as item}<option value={item}>{item}</option>{/each}</select></label>
			<label>Nomor laporan<input value="Dibuat saat disimpan" readonly aria-describedby="number-help" /><small id="number-help">Nomor dibuat oleh database agar tidak tumpang tindih.</small></label>
			<label>Uraian pekerjaan yang dilakukan<textarea bind:value={work} rows="5" required maxlength="500" placeholder="Tuliskan pekerjaan yang sudah diselesaikan"></textarea></label>
			{#if error}<p class="demo-note">{error}</p>{/if}
			<button class="report-submit" type="submit" disabled={submitting}>{submitting ? 'Menyimpan…' : 'Simpan laporan'}</button>
		</form>
	</div>
</main>
