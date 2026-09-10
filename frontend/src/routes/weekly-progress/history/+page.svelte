<script lang="ts">
	import { onMount } from 'svelte';
	import {subscribeChanges} from '$lib/realtime';
	import { api, download, type WeeklyArchive } from '$lib/api';
	let division = $state(''), date = $state(''), page = $state(1);
	let archives = $state<WeeklyArchive[]>([]), detail = $state<WeeklyArchive | null>(null);
	let loading = $state(true), hasMore = $state(false), error = $state(''), downloading = $state('');
	let generation=0;
	const divisions = ['Project Engineer', 'AI Dev', 'Keuangan', 'Business Development', 'HR', 'PMO'];
	const formatDate = (value: string) => new Intl.DateTimeFormat('id-ID', { timeZone: 'Asia/Jakarta', day: 'numeric', month: 'short', year: 'numeric' }).format(new Date(value));
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
			const { blob, filename } = await download(`/reports/history/${id}/pdf`);
			const url = URL.createObjectURL(blob), link = document.createElement('a');
			link.href = url; link.download = filename; link.click(); URL.revokeObjectURL(url);
		} catch (e) { error = e instanceof Error ? e.message : 'PDF tidak dapat diunduh.'; }
		finally { downloading = ''; }
	}
	onMount(() => { void load();const stop=subscribeChanges(['reports.changed'],()=>load(page,true));return()=>{generation++;stop()}; });
</script>

<svelte:head><title>Weekly Report History | my HIS</title></svelte:head>
<main class="weekly-progress-page"><div class="weekly-progress-shell">
	<a class="back-link" href="/weekly-progress">← Weekly Progress</a>
	<header class="weekly-header"><p class="weekly-kicker">WEEKLY REPORT</p><h1>Weekly Report History</h1><p>Arsip permanen laporan mingguan. Semua tanggal menggunakan WIB.</p></header>
	<form class="report-form" onsubmit={(event) => { event.preventDefault(); void load(); }}>
		<label>Divisi<select bind:value={division}><option value="">Semua divisi</option>{#each divisions as item}<option>{item}</option>{/each}</select></label>
		<label>Tanggal dalam periode<input type="date" bind:value={date} /></label>
		<button class="report-submit" disabled={loading}>Tampilkan</button>
	</form>
	{#if error}<p role="alert" class="demo-note">{error}</p>{/if}
	<section class="task-list" aria-label="Daftar arsip" aria-busy={loading}>
		{#if loading}<p role="status">Memuat riwayat…</p>{:else}
		<div class="work-table-wrap"><table class="work-table"><thead><tr><th scope="col">Divisi</th><th scope="col">Periode</th><th scope="col">Finalisasi (WIB)</th><th scope="col">Status</th><th scope="col">PDF / Detail</th></tr></thead><tbody>
			{#each archives as archive (archive.id)}<tr>
				<td>{archive.division}</td><td>{period(archive)}</td><td>{archive.finalizedAt ? new Intl.DateTimeFormat('id-ID', { timeZone: 'Asia/Jakarta', dateStyle: 'medium', timeStyle: 'short' }).format(new Date(archive.finalizedAt)) : '—'}</td>
				<td>{archive.status === 'finalized' ? 'Finalized' : archive.status === 'failed' ? 'PDF gagal — menunggu percobaan ulang' : 'Diproses'}</td>
				<td><div class="history-row-actions"><button class="report-button" type="button" onclick={() => void openDetail(archive.id)}>Detail</button><button class="report-button" type="button" disabled={archive.status !== 'finalized' || !!downloading} onclick={() => void savePDF(archive.id)}>{downloading === archive.id ? 'Mengunduh…' : archive.pdfGeneratedAt ? 'Download PDF' : 'PDF belum tersedia'}</button></div></td>
			</tr>{:else}<tr><td colspan="5">Belum ada arsip untuk filter ini.</td></tr>{/each}
		</tbody></table></div>
		<nav class="report-actions" aria-label="Halaman riwayat"><button class="report-button" disabled={page === 1} onclick={() => void load(page - 1)}>Sebelumnya</button><span>Halaman {page}</span><button class="report-button" disabled={!hasMore} onclick={() => void load(page + 1)}>Berikutnya</button></nav>
		{/if}
	</section>
	{#if detail}<section class="task-list" aria-label="Isi arsip">
		<h2>{detail.division} — {period(detail)}</h2><p>Snapshot saat finalisasi; perubahan data saat ini tidak mengubah arsip ini.</p>
		<button class="report-button" onclick={() => (detail = null)}>Tutup detail</button>
		<div class="work-table-wrap"><table class="work-table"><thead><tr><th scope="col">Pegawai</th><th scope="col">Role</th><th scope="col">Kategori</th><th scope="col">Pekerjaan</th><th scope="col">Laporan</th></tr></thead><tbody>
			{#each detail.items ?? [] as item}<tr><td>{item.employeeName}</td><td>{item.role || '—'}</td><td>{item.workDivision}</td><td>{item.work || '—'}</td><td>{item.description}</td></tr>{/each}
		</tbody></table></div>
	</section>{/if}
</div></main>
