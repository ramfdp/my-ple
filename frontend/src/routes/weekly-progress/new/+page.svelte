<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api, type ReportInput, type ReportingPeriod } from '$lib/api';

	const divisions = ['Project Engineer', 'AI Dev', 'Keuangan', 'Business Development', 'HR', 'PMO'];
	const categories: Record<string, string[]> = {'AI Dev':['Pengembangan','Pengujian Model','Deployment','Dokumentasi'],Keuangan:['Pembukuan','Pembayaran','Anggaran','Pelaporan'],'Business Development':['Prospek','Proposal','Kemitraan','Tindak Lanjut'],HR:['Kehadiran','Rekrutmen','Administrasi','Pengembangan SDM'],PMO:['Milestone','Rapat','Risiko','Pelaporan']};
	const qaJobs = ['MPS PIT','MPS SGR','MPS SGP','SHL JN','SOP Tata Kelola','Studi Daya Beli Gas Bumi','Review FS dan DED Pelabuhan Waipirit','Project Management Office','Reimagine Rumah IPS','Penataan Kawasan Stasiun Jember','Pengawasan Renovasi Gudang Besar JAKG','Pengawasan Instalasi Hidran BY Manggarai','Internal'];
	const projectEngineer: Record<string, Record<string, string[]>> = {
		PE:{'MK supervisi':['Penataan Kawasan Stasiun Jember','Pengawasan Renovasi Gudang Besar JAKG','Pengawasan Instalasi Hidran BY Manggarai'],'Non-MK supervisi':['MPS PIT','MPS SGR','MPS SGP','SHL JN','SOP Tata Kelola','Studi Daya Beli Gas Bumi','Review FS dan DED Pelabuhan Waipirit','Project Management Office','Reimagine Rumah IPS']},
		QA:{QA1:qaJobs,QA2:qaJobs},CIHPK:{JAKG:['JAKG Supervising Consultant (SPMK Date: March 12, 2026)'],'Manggarai Hydrant':['Supervising Consultant for the Manggarai Balai Yasa Hydrant Installation (SPMK Date: June 8, 2026)'],Jember:['Jember Station Area Redevelopment']}
	};
	let division = $state(divisions[0]), role = $state('PE'), category = $state(''), job = $state(''), description = $state(''), error = $state(''), submitting = $state(false);
	let mounted = $state(false), readOnly = $state(true), checking = $state(true);
	async function checkPeriod(selected: string) {
		checking = true;
		try {
			const response = await api<{ period: ReportingPeriod }>(`/reports?primaryDivision=${encodeURIComponent(selected)}`);
			if (division === selected) readOnly = response.period.readOnly;
		} catch (e) { readOnly = true; error = e instanceof Error ? e.message : 'Periode tidak dapat dimuat.'; }
		finally { if (division === selected) checking = false; }
	}
	$effect(() => { if (mounted) void checkPeriod(division); });
	$effect(() => {
		if (division !== 'Project Engineer') { role = ''; job = ''; if (!categories[division].includes(category)) category = categories[division][0]; return; }
		if (!projectEngineer[role]) role = 'PE';
		const availableCategories = Object.keys(projectEngineer[role]);
		if (!availableCategories.includes(category)) category = availableCategories[0];
		const jobs = projectEngineer[role][category] ?? [];
		if (!jobs.includes(job)) job = jobs.length === 1 ? jobs[0] : '';
	});
	function changeDivision(event: Event) {
		division = (event.currentTarget as HTMLSelectElement).value;
		role = division === 'Project Engineer' ? 'PE' : '';
		category = division === 'Project Engineer' ? 'MK supervisi' : categories[division][0];
		job = '';
	}
	onMount(() => { localStorage.removeItem('weekly-reports'); const requested = new URLSearchParams(location.search).get('division'); if (requested && divisions.includes(requested)) division = requested; mounted = true; });
	async function saveReport(event: SubmitEvent) {
		if (checking || readOnly) { event.preventDefault(); return; }
		event.preventDefault(); error = ''; submitting = true;
		try { await api<{report:unknown}>('/reports',{method:'POST',body:JSON.stringify({primaryDivision:division,workDivision:category,role,work:job,description:description.trim()} satisfies ReportInput)}); await goto(`/weekly-progress?division=${encodeURIComponent(division)}`); }
		catch (err) { error = err instanceof Error ? err.message : 'Laporan tidak dapat disimpan.'; }
		finally { submitting = false; }
	}
</script>

<svelte:head><title>Input Laporan | my HIS</title><meta name="description" content="Input laporan pekerjaan mingguan." /></svelte:head>
<svelte:window onfocus={() => { if (mounted) void checkPeriod(division); }} />
<main class="weekly-progress-page"><div class="weekly-progress-shell report-shell">
	<a class="back-link" href="/weekly-progress">← Kembali ke weekly progress</a>
	<header class="weekly-header"><p class="weekly-kicker">WEEKLY PROGRESS</p><h1>Input laporan pekerjaan.</h1><p>Isi pekerjaan yang sudah dilakukan agar tampil pada daftar progres.</p></header>
	<form class="report-form" onsubmit={saveReport}>
		{#if checking}<p role="status">Memeriksa periode laporan…</p>{:else if readOnly}<p role="status">Periode ditutup. Input baru dibuka Selasa 00:00 WIB setelah arsip periode sebelumnya aman. <a href="/weekly-progress/history">Lihat riwayat</a></p>{/if}
		<label>Main Division<select value={division} onchange={changeDivision}>{#each divisions as item}<option value={item}>{item}</option>{/each}</select></label>
		<label>Role{#if division === 'Project Engineer'}<select bind:value={role}>{#each Object.keys(projectEngineer) as item}<option value={item}>{item}</option>{/each}</select>{:else}<input value="Not configured" readonly />{/if}</label>
		<label>Job Category{#if division === 'Project Engineer'}<select bind:value={category}>{#each Object.keys(projectEngineer[role]) as item}<option value={item}>{item}</option>{/each}</select>{:else}<select bind:value={category}>{#each categories[division] as item}<option value={item}>{item}</option>{/each}</select>{/if}</label>
		<label>Job{#if division === 'Project Engineer'}<select bind:value={job} required><option value="" disabled>Select job</option>{#each projectEngineer[role][category] ?? [] as item}<option value={item}>{item}</option>{/each}</select>{:else}<input value="Not configured" readonly />{/if}</label>
		<label>Uraian pekerjaan yang dilakukan<textarea bind:value={description} rows="5" required maxlength="500"></textarea></label>
		{#if error}<p class="demo-note">{error}</p>{/if}<button class="report-submit" type="submit" disabled={submitting || readOnly || checking}>{submitting ? 'Menyimpan…' : 'Simpan laporan'}</button>
	</form>
</div></main>
