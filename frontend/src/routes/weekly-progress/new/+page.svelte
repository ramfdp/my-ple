<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import JobPicker from '$lib/components/JobPicker.svelte';
	import { api, type Job, type ReportInput, type ReportingPeriod } from '$lib/api';

	let division=$state('Project Engineer'), selected=$state<Job|null>(null), jobsBusy=$state(true), description=$state(''), error=$state(''), submitting=$state(false);
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
	onMount(() => { const requested=new URLSearchParams(location.search).get('division'); if(requested)division=requested; mounted=true; });
	async function saveReport(event: SubmitEvent) {
		if (checking || readOnly || jobsBusy || !selected || submitting) { event.preventDefault(); return; }
		event.preventDefault(); error = ''; submitting = true;
		try { await api<{report:unknown}>('/reports',{method:'POST',body:JSON.stringify({primaryDivision:division,workDivision:selected!.category,role:selected!.role,work:selected!.name,jobId:selected!.id,description:description.trim()} satisfies ReportInput)}); await goto(`/weekly-progress?division=${encodeURIComponent(division)}`); }
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
		<JobPicker bind:division bind:selected bind:busy={jobsBusy} />
		<label>Uraian pekerjaan yang dilakukan<textarea bind:value={description} rows="5" required maxlength="500"></textarea></label>
		{#if error}<p class="demo-note">{error}</p>{/if}<button class="report-submit" type="submit" disabled={submitting || readOnly || checking || jobsBusy || !selected}>{submitting ? 'Menyimpan…' : 'Simpan laporan'}</button>
	</form>
</div></main>
