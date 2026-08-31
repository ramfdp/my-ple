<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	const workDivisions: Record<string, string[]> = {
		'Project Engineer': ['Quality Assurance 1', 'Quality Assurance 2', 'Pengawasan', 'Non-Pengawasan', 'CIHPK'],
		'AI Dev': ['Pengembangan', 'Pengujian Model', 'Deployment', 'Dokumentasi'],
		Keuangan: ['Pembukuan', 'Pembayaran', 'Anggaran', 'Pelaporan'],
		'Business Development': ['Prospek', 'Proposal', 'Kemitraan', 'Tindak Lanjut'],
		HR: ['Kehadiran', 'Rekrutmen', 'Administrasi', 'Pengembangan SDM'],
		PMO: ['Milestone', 'Rapat', 'Risiko', 'Pelaporan']
	};
	const divisions = Object.keys(workDivisions);

	let division = $state(divisions[0]);
	let category = $state(workDivisions[divisions[0]][0]);
	let work = $state('');
	let completedAt = $state('');
	let savedReports = $state<Record<string, string[][]>>({});
	let reportNumber = $derived(2 + (savedReports[division] ?? []).filter((report) => report[0] === category).length);

	$effect(() => {
		if (!workDivisions[division].includes(category)) category = workDivisions[division][0];
	});

	onMount(() => {
		const requestedDivision = new URLSearchParams(location.search).get('division');
		if (requestedDivision && divisions.includes(requestedDivision)) division = requestedDivision;
		savedReports = JSON.parse(localStorage.getItem('weekly-reports') ?? '{}');
	});

	function saveReport(event: SubmitEvent) {
		event.preventDefault();
		const reports = JSON.parse(localStorage.getItem('weekly-reports') ?? '{}');
		const timestamp = new Intl.DateTimeFormat('id-ID', { dateStyle: 'long', timeStyle: 'short' }).format(new Date(completedAt)).replace(' pukul', ',');
		reports[division] = [...(reports[division] ?? []), [category, work.trim(), timestamp]];
		localStorage.setItem('weekly-reports', JSON.stringify(reports));
		goto(`/weekly-progress?division=${encodeURIComponent(division)}`);
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
			<label>Divisi pekerjaan<select bind:value={category}>{#each workDivisions[division] as item}<option value={item}>{item}</option>{/each}</select></label>
			<label>Nomor laporan<input value={reportNumber} readonly aria-describedby="number-help" /><small id="number-help">Nomor dibuat otomatis agar tidak tumpang tindih.</small></label>
			<label>Pekerjaan yang dilakukan<textarea bind:value={work} rows="5" required maxlength="500" placeholder="Tuliskan pekerjaan yang sudah diselesaikan"></textarea></label>
			<label>Waktu dikerjakan<input bind:value={completedAt} type="datetime-local" required /></label>
			<button class="report-submit" type="submit">Simpan laporan</button>
		</form>
	</div>
</main>
