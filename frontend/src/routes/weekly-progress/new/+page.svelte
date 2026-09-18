<script lang="ts">
 import { t as translate } from '$lib/i18n';
	import { onMount, tick } from 'svelte';
	import { subscribeChanges } from '$lib/realtime';
	import JobPicker from '$lib/components/JobPicker.svelte';
	import ToastNotification from '$lib/components/ToastNotification.svelte';
	import { api, type Job, type Report, type ReportInput, type ReportingPeriod } from '$lib/api';
	import { capitalizeReportDescription } from '$lib/utils/report-table';

	let division=$state('Project Engineer'), selected=$state<Job|null>(null), jobsBusy=$state(true), description=$state(''), error=$state(''), submitting=$state(false);
	let mounted = $state(false), readOnly = $state(true), checking = $state(true);
	let descriptionInput: HTMLTextAreaElement;
	let toast = $state({ visible: false, message: '' });
	async function checkPeriod(selected: string) {
		checking = true;
		try {
			const response = await api<{ period: ReportingPeriod }>(`/reports?primaryDivision=${encodeURIComponent(selected)}`);
			if (division === selected) readOnly = response.period.readOnly;
		} catch (e) { readOnly = true; error = e instanceof Error ? e.message : 'Periode tidak dapat dimuat.'; }
		finally { if (division === selected) checking = false; }
	}
	$effect(() => { if (mounted) void checkPeriod(division); });
	onMount(() => { const requested=new URLSearchParams(location.search).get('division'); if(requested)division=requested; mounted=true; return subscribeChanges(['reports.changed'], () => checkPeriod(division)); });
	async function saveReport(event: SubmitEvent) {
		if (checking || readOnly || jobsBusy || !selected || submitting) { event.preventDefault(); return; }
		event.preventDefault(); error = ''; submitting = true; toast.visible = false;
		let saved = false;
		try {
			await api<{report:Report}>('/reports',{method:'POST',body:JSON.stringify({primaryDivision:division,workDivision:selected.category,role:selected.role,work:selected.name,jobId:selected.id,description:description.trim()} satisfies ReportInput)});
			description = ''; saved = true;
			toast = { visible: true, message: 'Laporan berhasil ditambahkan' };
		}
		catch (err) { error = err instanceof Error ? err.message : 'Laporan tidak dapat disimpan.'; }
		finally { submitting = false; if (saved) { await tick(); descriptionInput?.focus(); } }
	}
</script>

<svelte:head><title>{$translate("Input Laporan | MyHIS")}</title><meta name="description" content={$translate("Input laporan pekerjaan mingguan.")} /></svelte:head>
<svelte:window onfocus={() => { if (mounted) void checkPeriod(division); }} />
<main class="weekly-progress-page"><div class="weekly-progress-shell report-shell">
	<a class="back-link" href="/dashboard">{$translate("← Kembali ke dashboard")}</a>
	<header class="weekly-header"><p class="weekly-kicker">{$translate("WEEKLY PROGRESS")}</p><h1>{$translate("Input laporan pekerjaan.")}</h1><p>{$translate("Isi pekerjaan yang sudah dilakukan agar tampil pada daftar progres.")}</p></header>
	<form class="report-form" onsubmit={saveReport}>
		{#if checking}<p role="status">{$translate("Memeriksa periode laporan…")}</p>{:else if readOnly}<p role="status">{$translate("Periode ditutup. Input baru dibuka Selasa 00:00 WIB setelah arsip periode sebelumnya aman.")} <a href="/weekly-progress/history">{$translate("Lihat riwayat")}</a></p>{/if}
		<JobPicker bind:division bind:selected bind:busy={jobsBusy} />
		<label>{$translate("Uraian pekerjaan yang dilakukan")}<textarea bind:this={descriptionInput} bind:value={description} oninput={()=>description=capitalizeReportDescription(description)} disabled={submitting} rows="8" required></textarea></label>
		{#if error}<p class="demo-note">{$translate(error)}</p>{/if}<button class="report-submit" type="submit" disabled={submitting || readOnly || checking || jobsBusy || !selected}>{submitting ? $translate('Menyimpan…') : $translate('Simpan laporan')}</button>
	</form>
</div></main>
<ToastNotification message={toast.message} visible={toast.visible} onclose={() => (toast.visible = false)} />
