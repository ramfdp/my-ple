<script lang="ts">
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { api, APIError, currentUser, download, session, type Report, type ReportingPeriod, type WeeklyRoleGroup } from '$lib/api';
	import {subscribeChanges} from '$lib/realtime';
	import JobPicker from '$lib/components/JobPicker.svelte';
	import type { Job } from '$lib/api';
	import { canModifyReport, capitalizeReportDescription, groupReports } from '$lib/utils/report-table';

	const divisions = ['Project Engineer', 'AI Dev', 'Keuangan', 'Business Development', 'HR', 'PMO'];
	let selectedDivision = $state<number | null>(null);
	let reports = $state<Report[]>([]);
	let groups = $state<WeeklyRoleGroup[]>([]);
	let loading = $state(false);
	let error = $state('');
	let period = $state<ReportingPeriod | null>(null);
	let exporting = $state(false);
 let exportDialog:HTMLDialogElement;
 let exportError=$state('');
	let editing = $state<Report | null>(null);
	let deleting = $state<Report | null>(null);
	let editJob=$state<Job|null>(null), jobsBusy=$state(true), saving=$state(false);
	let editDescription = $state('');
	let latest=$state<Report|null>(null), remoteDeleted=$state(false);
	let pickerReport=$state<Report|null>(null);
	let loadGeneration=0;
	const tableRows = $derived(groupReports(groups));

	async function selectDivision(index: number, background=false) {
		const generation=++loadGeneration;
		selectedDivision = index;
		if(!background){reports=[];groups=[];loading=true}
		error = '';
		try {
			const response = await api<{ reports: Report[] | null; groups:WeeklyRoleGroup[] | null; period: ReportingPeriod }>(`/reports?primaryDivision=${encodeURIComponent(divisions[index])}`);
			if(generation!==loadGeneration)return;
			reports = response.reports ?? [];
			groups = response.groups ?? [];
			period = response.period;
			if(editing){
			 const server=reports.find(r=>r.id===editing?.id);
			 remoteDeleted=!server;
			 if(server&&server.version!==editing.version){latest=server}
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Laporan tidak dapat dimuat.';
		} finally {
			if(generation===loadGeneration)loading = false;
		}
	}

	async function exportReport(format:'pdf'|'docx') {
		if (selectedDivision === null || reports.length === 0 || exporting) return;
		exporting = true;
		exportError = '';
		try {
			const { blob, filename } = await download(`/reports.${format}?primaryDivision=${encodeURIComponent(divisions[selectedDivision])}`);
			const url = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = filename;
			link.click();
			URL.revokeObjectURL(url);
   exportDialog.close();
		} catch (err) {
			exportError = err instanceof Error ? err.message : 'Laporan tidak dapat diekspor.';
		} finally {
			exporting = false;
		}
	}

	async function saveEdit() {
		if (!editing || !editJob || jobsBusy || saving) return;
		saving=true;
		error = '';
		try {
			const patch:Record<string,unknown>={version:editing.version};
			if(editDescription!==editing.description)patch.description=editDescription;
			if(editJob.id&&editJob.id!==editing.jobId)Object.assign(patch,{jobId:editJob.id,workDivision:editJob.category,role:editJob.role,work:editJob.name});
			await api<{ report: Report }>(`/reports/${editing.id}`, { method: 'PATCH', body: JSON.stringify(patch) });
			editing = null;
			if (selectedDivision !== null) await selectDivision(selectedDivision);
		} catch (err) {
			if(err instanceof APIError&&err.data.code==='report_conflict')latest=err.data.report as Report;
			error = err instanceof Error ? err.message : 'Laporan tidak dapat diperbarui.';
		} finally { saving=false; }
	}

	async function deleteReport() {
		if (!deleting) return;
		error = '';
		try {
			await api<void>(`/reports/${deleting.id}`, { method: 'DELETE' });
			deleting = null;
			if (selectedDivision !== null) await selectDivision(selectedDivision);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Laporan tidak dapat dihapus.';
		}
	}

	onMount(() => {
		localStorage.removeItem('weekly-reports');
		const requested = new URLSearchParams(location.search).get('division');
		const index = divisions.indexOf(requested ?? '');
		if (index >= 0) void selectDivision(index);
		const stop=subscribeChanges(['reports.changed'],()=>{if(selectedDivision!==null)return selectDivision(selectedDivision,true)});
		return()=>{loadGeneration++;stop()};
	});
</script>

<svelte:head>
	<title>Weekly Progress | my HIS</title>
	<meta name="description" content="Pilih divisi dan lihat daftar pekerjaan mingguan." />
</svelte:head>

<svelte:window onfocus={() => { if (selectedDivision !== null) void selectDivision(selectedDivision,true); }} />

<main class="weekly-progress-page">
	<div class="weekly-progress-shell">
		<a class="back-link" href="/dashboard">← Kembali ke dashboard</a>
		<header class="weekly-header"><p class="weekly-kicker">WEEKLY PROGRESS</p><h1>Progres kerja per divisi.</h1><p>Pilih divisi terlebih dahulu untuk melihat daftar pekerjaan minggu ini.</p></header>
		<section class="division-selection" aria-labelledby="division-title">
			<a class="report-button" href="/weekly-progress/history">Weekly Report History</a>
			<h2 id="division-title">Pilih divisi</h2><p>Daftar ini menentukan pekerjaan yang akan ditampilkan.</p>
			<div class="division-grid">{#each divisions as division, index}<button class:selected={selectedDivision === index} class="division-button" type="button" aria-pressed={selectedDivision === index} onclick={() => void selectDivision(index)}>{division}<span aria-hidden="true">→</span></button>{/each}</div>
		</section>
		{#if selectedDivision !== null}
			<section class="task-list" aria-labelledby="task-title">
				{#if period?.readOnly}<p role="status" class="report-period">{period.status === 'finalized' ? 'Finalized' : 'Periode ditutup — arsip diproses'} — hanya baca. Laporan tetap tersedia selama Senin. Reset mulai Selasa 00:00 WIB setelah arsip dan PDF berhasil disimpan.</p>{/if}
				<div class="task-list-header"><div><h2 id="task-title">Pekerjaan yang sudah dilakukan</h2><p>Daftar pekerjaan pada minggu berjalan.</p>{#if period}<p class="report-period">Periode: {period.label}</p>{/if}</div><div class="task-list-actions"><p class="selected-division">{divisions[selectedDivision]}</p><div class="report-actions"><button class="report-button export-button" type="button" disabled={reports.length === 0 || exporting || loading} onclick={() => {exportError='';exportDialog.showModal()}}>Export</button>{#if !period?.readOnly}<a class="report-button" href={`/weekly-progress/new?division=${encodeURIComponent(divisions[selectedDivision])}`}>+ Input laporan</a>{/if}</div></div></div>
				<div class="work-table-wrap">
					<table class="work-table current-report-table" class:other-division-table={divisions[selectedDivision] !== 'Project Engineer'}>
						<thead><tr><th scope="col">Role</th><th scope="col">Kategori</th><th scope="col">No.</th><th scope="col">Pekerjaan yang dilakukan</th><th scope="col">Aksi</th></tr></thead>
						<tbody>
							{#if loading}<tr><td colspan="5">Memuat laporan…</td></tr>
							{:else if reports.length === 0}<tr><td colspan="5">Belum ada laporan pekerjaan untuk periode ini.</td></tr>
							{:else}{#each tableRows as row (row.type==='report'?row.report.id:row.key)}
								{#if row.type === 'work'}
									<tr class="work-group-row">
										{#if row.role}<td rowspan={row.roleRowSpan ?? 1} class="task-role">{row.role}</td>{/if}
										{#if row.category}<td rowspan={row.categoryRowSpan ?? 1} class="task-category">{row.category}</td>{/if}
										<td colspan="2" class="task-work-title" data-role={row.roleLabel} data-category={row.categoryLabel}><div class="job-heading"><span>{row.work}</span>{#if row.pics?.length}<small>PIC: {row.pics.map(pic=>pic.name+(pic.isActive?'':' (Nonaktif)')).join(', ')}</small>{/if}</div></td><td class="task-actions"></td>
									</tr>
								{:else}
									<tr><td class="task-number">{row.displayNumber}</td><td class="task-description">{row.report.description}</td><td class="task-actions">{#if !period?.readOnly && canModifyReport($currentUser, row.report.userId)}<div class="report-row-actions"><button class="edit-report" type="button" onclick={() => { latest=null;remoteDeleted=false;editing = row.report; pickerReport=row.report; editDescription = row.report.description; }}>Edit</button><button class="delete-report" type="button" onclick={() => (deleting = row.report)}>Hapus</button></div>{:else}<span class="no-actions">—</span>{/if}</td></tr>
								{/if}
							{/each}{/if}
						</tbody>
					</table>
				</div>
				{#if error}<p class="demo-note">{error}</p>{/if}
			</section>
		{/if}
	</div>
</main>

<dialog bind:this={exportDialog} class="report-delete-modal" aria-labelledby="export-title" oncancel={(e)=>{if(exporting)e.preventDefault()}}>
 <h2 id="export-title">Export Weekly Report</h2><p>Pilih format ekspor.</p>
 {#if exportError}<p role="alert">{exportError}</p>{/if}
 {#if exporting}<p role="status">Menyiapkan dokumen…</p>{/if}
 <div class="delete-modal-actions"><button class="report-button" disabled={exporting} onclick={()=>exportReport('pdf')}>Export PDF</button><button class="report-button" disabled={exporting} onclick={()=>exportReport('docx')}>Export Word</button><button class="drawer-cancel" disabled={exporting} onclick={()=>exportDialog.close()}>Batal</button></div>
</dialog>

{#if editing}
	<div class="report-drawer-layer" transition:fade={{ duration: 180 }}>
		<button class="report-drawer-backdrop" type="button" aria-label="Tutup edit laporan" onclick={() => (editing = null)}></button>
		<div class="report-drawer" role="dialog" aria-modal="true" aria-labelledby="edit-report-title" transition:fade={{ duration: 180 }}>
			<div class="report-drawer-header"><h2 id="edit-report-title">Edit laporan</h2><button class="drawer-close" type="button" aria-label="Tutup" onclick={() => (editing = null)}>×</button></div>
			<form class="report-form report-drawer-form" onsubmit={(event) => { event.preventDefault(); void saveEdit(); }}>
				{#key pickerReport}<JobPicker division={editing.primaryDivision} existing={pickerReport??editing} fixedDivision bind:selected={editJob} bind:busy={jobsBusy} />{/key}
				<label>Uraian pekerjaan yang dilakukan<textarea bind:value={editDescription} oninput={()=>editDescription=capitalizeReportDescription(editDescription)} rows="8" required></textarea></label>
				{#if latest}<section role="status"><p>Data server berubah (versi {latest.version}). Draf Anda tetap tersedia di atas.</p><p>Pekerjaan terbaru: {latest.work} · {latest.role} · {latest.workDivision}</p><p class="task-description">{latest.description}</p><button type="button" onclick={()=>{if(!latest||!editing)return;const descriptionChanged=editDescription!==editing.description;const jobChanged=editJob?.id!==editing.jobId; if(!descriptionChanged)editDescription=latest.description;if(!jobChanged)pickerReport=latest;editing=latest;latest=null;error=''}}>Sudah dibandingkan — gunakan versi terbaru sebagai dasar draf</button><button type="button" onclick={()=>{if(!latest)return;editDescription=latest.description;pickerReport=latest;editing=latest;latest=null;error=''}}>Gunakan data server</button></section>{/if}
				{#if remoteDeleted}<p role="alert">Laporan ini sudah dihapus atau diarsipkan. Draf Anda tetap tersedia untuk disalin.</p>{/if}
				{#if error}<p role="alert">{error}</p>{/if}
				<div class="drawer-form-actions"><button class="report-submit" type="submit" disabled={saving || jobsBusy || !editJob || !!latest || remoteDeleted || period?.readOnly}>{saving ? "Menyimpan…" : "Simpan perubahan"}</button><button class="drawer-cancel" type="button" onclick={() => (editing = null)}>Batal</button></div>
			</form>
		</div>
	</div>
{/if}
{#if deleting}
	<div class="report-modal-layer" transition:fade={{ duration: 180 }}>
		<button class="report-modal-backdrop" type="button" aria-label="Batal hapus laporan" onclick={() => (deleting = null)}></button>
		<div class="report-delete-modal" role="dialog" aria-modal="true" aria-labelledby="delete-report-title" transition:fade={{ duration: 180 }}>
			<h2 id="delete-report-title">Hapus laporan?</h2><p>Make sure Anda ingin menghapus laporan ini. Tindakan ini tidak dapat dibatalkan.</p>
			<div class="delete-modal-actions"><button class="delete-report-confirm" type="button" onclick={() => void deleteReport()}>Hapus laporan</button><button class="drawer-cancel" type="button" onclick={() => (deleting = null)}>Batal</button></div>
		</div>
	</div>
{/if}

<style>
dialog.report-delete-modal:not([open]){display:none}
dialog.report-delete-modal{position:fixed;border:0;max-height:calc(100dvh - 32px);overflow-y:auto}
dialog.report-delete-modal::backdrop{background:#152a4699}
.job-heading{display:flex;flex-wrap:wrap;justify-content:space-between;gap:4px 16px;overflow-wrap:anywhere}.job-heading span,.job-heading small{min-width:0;max-width:100%}.job-heading small{font-weight:400}
</style>
