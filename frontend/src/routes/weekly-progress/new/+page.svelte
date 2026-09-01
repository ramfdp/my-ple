<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	const divisions = ['Project Engineer', 'AI Dev', 'Keuangan', 'Business Development', 'HR', 'PMO'];
	const roleCategories: Record<string, Record<string, string[]>> = {
		PE: {
			'MK / supervisi': ['Penataan Kawasan Stasiun Jember', 'Pengawasan Renovasi Gudang Besar JAKG', 'Pengawasan Instalasi Hidran BY Manggarai'],
			'Non MK / Non Supervisi': ['MPS PIT', 'MPS SGR', 'MPS SGP', 'SHL JN', 'SOP Tata Kelola', 'Studi Daya Beli Gas Bumi', 'Review FS dan DED Pelabuhan Waipirit', 'Project Management Office', 'Reimagine Rumah IPS']
		},
		'Quality Assurance': {
			'QA 1': ['MPS PIT', 'MPS SGR', 'MPS SGP', 'SHL JN', 'SOP Tata Kelola', 'Studi Daya Beli Gas Bumi', 'Review FS dan DED Pelabuhan Waipirit', 'Project Management Office', 'Reimagine Rumah IPS', 'Penataan Kawasan Stasiun Jember', 'Pengawasan Renovasi Gudang Besar JAKG', 'Pengawasan Instalasi Hidran BY Manggarai', 'internal'],
			'QA 2': ['MPS PIT', 'MPS SGR', 'MPS SGP', 'SHL JN', 'SOP Tata Kelola', 'Studi Daya Beli Gas Bumi', 'Review FS dan DED Pelabuhan Waipirit', 'Project Management Office', 'Reimagine Rumah IPS', 'Penataan Kawasan Stasiun Jember', 'Pengawasan Renovasi Gudang Besar JAKG', 'Pengawasan Instalasi Hidran BY Manggarai', 'internal']
		},
		CIHPK: {
			JAKG: ['Konsultan Pengawas JAKG (Tanggal SPMK: 12 Maret 2026)'],
			'Hydrant Manggarai': ['Konsultan Pengawas Instalasi Hydrant Balai Yasa Manggarai (Tanggal SPMK: 08 Juni 2026)'],
			Jember: ['Penataan Kawasan Stasiun Jember']
		}
	};
	const roles = Object.keys(roleCategories);
	const initialRole = roles[0];
	const initialCategory = Object.keys(roleCategories[initialRole])[0];
	type SavedReport = [role: string, category: string, sequence: number, work: string];

	let division = $state(divisions[0]);
	let role = $state(initialRole);
	let category = $state(initialCategory);
	let project = $state(roleCategories[initialRole][initialCategory][0]);
	let work = $state('');
	let savedReports = $state<Record<string, SavedReport[]>>({});
	let reportNumber = $derived(1 + (savedReports[division] ?? []).filter((report) => report[1] === category).length);

	$effect(() => {
		const categories = Object.keys(roleCategories[role]);
		const selectedCategory = categories.includes(category) ? category : categories[0];
		if (category !== selectedCategory) category = selectedCategory;
		const projects = roleCategories[role][selectedCategory];
		if (!projects.includes(project)) project = projects[0];
	});

	onMount(() => {
		const requestedDivision = new URLSearchParams(location.search).get('division');
		if (requestedDivision && divisions.includes(requestedDivision)) division = requestedDivision;
		savedReports = JSON.parse(localStorage.getItem('weekly-reports') ?? '{}');
	});

	function saveReport(event: SubmitEvent) {
		event.preventDefault();
		const reports = JSON.parse(localStorage.getItem('weekly-reports') ?? '{}');
		reports[division] = [...(reports[division] ?? []), [role, category, reportNumber, work.trim()]];
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
			<label>Role<select bind:value={role}>{#each roles as item}<option value={item}>{item}</option>{/each}</select></label>
			<label>Kategori<select bind:value={category}>{#each Object.keys(roleCategories[role]) as item}<option value={item}>{item}</option>{/each}</select></label>
			<label>Pekerjaan<select bind:value={project}>{#each roleCategories[role][category] as item}<option value={item}>{item}</option>{/each}</select></label>
			<label>Nomor laporan<input value={reportNumber} readonly aria-describedby="number-help" /><small id="number-help">Nomor dibuat otomatis agar tidak tumpang tindih.</small></label>
			<label>Uraian pekerjaan yang dilakukan<textarea bind:value={work} rows="5" required maxlength="500" placeholder="Tuliskan pekerjaan yang sudah diselesaikan"></textarea></label>
			<button class="report-submit" type="submit">Simpan laporan</button>
		</form>
	</div>
</main>
