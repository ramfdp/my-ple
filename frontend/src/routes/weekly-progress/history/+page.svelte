<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
	import { onMount } from 'svelte';
	import {subscribeChanges} from '$lib/realtime';
	import { api, currentUser, download, type WeeklyArchive } from '$lib/api';
	let division = $state(''), date = $state(''), page = $state(1);
	let archives = $state<WeeklyArchive[]>([]), detail = $state<WeeklyArchive | null>(null);
	let loading = $state(true), hasMore = $state(false), error = $state(''), downloading = $state('');
	let generation=0;
	const allDivisions = ['Project Engineer', 'AI Dev', 'Keuangan', 'Business Development', 'HR', 'PMO'];
	const roleDivision:Record<string,string>={'Project Engineer':'Project Engineer','IT':'AI Dev','Finance':'Keuangan','Business Development':'Business Development','HR':'HR','PMO':'PMO'};
	const divisions=$derived(['Admin','Management'].includes($currentUser?.role??'')?allDivisions:allDivisions.filter(d=>d===roleDivision[$currentUser?.role??'']));
	const formatDate = (value: string) => new Intl.DateTimeFormat($locale, { timeZone: 'Asia/Jakarta', day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(value));
	const period = (a: WeeklyArchive) => `${formatDate(a.weekStart)} – ${formatDate(new Date(new Date(a.weekEnd).getTime() - 1).toISOString())}`;
	async function load(next = 1, background=false) {
		const sequence=++generation;
		if(!background){loading = true; error = ''; detail = null;}
		try {
			const response = await api<{ archives: WeeklyArchive[]; hasMore: boolean }>(`/reports/history?${new URLSearchParams({ division, date, page: String(next) })}`);
			if(sequence===generation){archives = response.archives; hasMore = response.hasMore; page = next;}
		} catch (e) { if(sequence===generation)error = e instanceof Error ? e.message : 'Riwayat tidak dapat dimuat.'; }
		finally { if(sequence===generation)loading = false; }
	}
	async function openDetail(id: string) {
		error = ''; detail = null;
		try { detail = (await api<{ archive: WeeklyArchive }>(`/reports/history/${id}`)).archive; }
		catch (e) { error = e instanceof Error ? e.message : 'Arsip tidak dapat dimuat.'; }
	}
	async function savePDF(id: string) {
		downloading = id; error = '';
		try {
			await download(`/reports/history/${id}/pdf`);
		} catch (e) { error = e instanceof Error ? e.message : 'PDF tidak dapat diunduh.'; }
		finally { downloading = ''; }
	}
	onMount(() => { void load();const stop=subscribeChanges(['reports.changed'],()=>load(page,true));return()=>{generation++;stop()}; });
</script>

<svelte:head><title>{$translate("Weekly Report History | MyHIS")}</title></svelte:head>
<main class="weekly-progress-page"><div class="weekly-progress-shell">
	<a class="back-link" href="/weekly-progress">{$translate("← Weekly Progress")}</a>
	<header class="weekly-header"><p class="weekly-kicker">{$translate("WEEKLY REPORT")}</p><h1>{$translate("Weekly Report History")}</h1><p>{$translate("Arsip permanen laporan mingguan. Semua tanggal menggunakan WIB.")}</p></header>
	<form class="report-form" onsubmit={(event) => { event.preventDefault(); void load(); }}>
		<label>{$translate("Divisi")}<select bind:value={division}>{#if divisions.length>1}<option value="">{$translate("Semua divisi")}</option>{/if}{#each divisions as item}<option>{$translate(item)}</option>{/each}</select></label>
		<label>{$translate("Tanggal dalam periode")}<input type="date" bind:value={date} /></label>
		<button class="report-submit" disabled={loading}>{$translate("Tampilkan")}</button>
	</form>
	{#if error}<p role="alert" class="demo-note">{$translate(error)}</p>{/if}
	<section class="task-list" aria-label={$translate("Daftar arsip")} aria-busy={loading}>
		{#if loading}<p role="status">{$translate("Memuat riwayat…")}</p>{:else}
		<div class="work-table-wrap"><table class="work-table"><thead><tr><th scope="col">{$translate("Divisi")}</th><th scope="col">{$translate("Periode")}</th><th scope="col">{$translate("Finalisasi (WIB)")}</th><th scope="col">{$translate("Status")}</th><th scope="col">{$translate("PDF / Detail")}</th></tr></thead><tbody>
			{#each archives as archive (archive.id)}<tr>
				<td>{$translate(archive.division)}</td><td>{period(archive)}</td><td>{archive.finalizedAt ? new Intl.DateTimeFormat($locale, { timeZone: 'Asia/Jakarta', dateStyle: 'medium', timeStyle: 'short' }).format(new Date(archive.finalizedAt)) : '—'}</td>
				<td>{archive.status === 'finalized' ? $translate('Finalized') : archive.status === 'failed' ? $translate('PDF gagal — menunggu percobaan ulang') : $translate('Diproses')}</td>
				<td><div class="history-row-actions"><button class="report-button" type="button" onclick={() => void openDetail(archive.id)}>{$translate("Detail")}</button><button class="report-button" type="button" disabled={archive.status !== 'finalized' || !!downloading} onclick={() => void savePDF(archive.id)}>{downloading === archive.id ? $translate('Mengunduh…') : archive.pdfGeneratedAt ? $translate('Download PDF') : $translate('PDF belum tersedia')}</button></div></td>
			</tr>{:else}<tr><td colspan="5">{$translate("Belum ada arsip untuk filter ini.")}</td></tr>{/each}
		</tbody></table></div>
		<nav class="report-actions" aria-label={$translate("Halaman riwayat")}><button class="report-button" disabled={page === 1} onclick={() => void load(page - 1)}>{$translate("Sebelumnya")}</button><span>{$translate("Halaman")} {page}</span><button class="report-button" disabled={!hasMore} onclick={() => void load(page + 1)}>{$translate("Berikutnya")}</button></nav>
		{/if}
	</section>
	{#if detail}<section class="task-list" aria-label={$translate("Isi arsip")}>
		<h2>{detail.division} — {period(detail)}</h2><p>{$translate("Snapshot saat finalisasi; perubahan data saat ini tidak mengubah arsip ini.")}</p>
		<button class="report-button" onclick={() => (detail = null)}>{$translate("Tutup detail")}</button>
		<div class="work-table-wrap"><table class="work-table"><thead><tr><th scope="col">{$translate("Pegawai")}</th><th scope="col">{$translate("Role")}</th><th scope="col">{$translate("Kategori")}</th><th scope="col">{$translate("Pekerjaan")}</th><th scope="col">{$translate("Laporan")}</th></tr></thead><tbody>
			{#each detail.items ?? [] as item}<tr><td>{item.employeeName}</td><td>{item.role || '—'}</td><td>{item.workDivision}</td><td>{item.work || '—'}</td><td>{item.description}</td></tr>{/each}
		</tbody></table></div>
	</section>{/if}
</div></main>
