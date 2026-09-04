<script lang="ts">
	import { onMount } from 'svelte';
	import { fade, fly } from 'svelte/transition';
	import { api, currentUser, download, session, type Report, type ReportingPeriod } from '$lib/api';
	import { canModifyReport, groupReports } from '$lib/utils/report-table';

	const divisions = ['Project Engineer', 'AI Dev', 'Keuangan', 'Business Development', 'HR', 'PMO'];
	const workDivisions: Record<string, string[]> = {
		'Project Engineer': ['MK supervisi', 'Non-MK supervisi', 'QA1', 'QA2', 'JAKG', 'Manggarai Hydrant', 'Jember'],
		'AI Dev': ['Pengembangan', 'Pengujian Model', 'Deployment', 'Dokumentasi'], Keuangan: ['Pembukuan', 'Pembayaran', 'Anggaran', 'Pelaporan'],
		'Business Development': ['Prospek', 'Proposal', 'Kemitraan', 'Tindak Lanjut'], HR: ['Kehadiran', 'Rekrutmen', 'Administrasi', 'Pengembangan SDM'], PMO: ['Milestone', 'Rapat', 'Risiko', 'Pelaporan']
	};
	const qaJobs = ['MPS PIT','MPS SGR','MPS SGP','SHL JN','SOP Tata Kelola','Studi Daya Beli Gas Bumi','Review FS dan DED Pelabuhan Waipirit','Project Management Office','Reimagine Rumah IPS','Penataan Kawasan Stasiun Jember','Pengawasan Renovasi Gudang Besar JAKG','Pengawasan Instalasi Hidran BY Manggarai','Internal'];
	const projectEngineer: Record<string, Record<string, string[]>> = {PE:{'MK supervisi':['Penataan Kawasan Stasiun Jember','Pengawasan Renovasi Gudang Besar JAKG','Pengawasan Instalasi Hidran BY Manggarai'],'Non-MK supervisi':['MPS PIT','MPS SGR','MPS SGP','SHL JN','SOP Tata Kelola','Studi Daya Beli Gas Bumi','Review FS dan DED Pelabuhan Waipirit','Project Management Office','Reimagine Rumah IPS']},QA:{QA1:qaJobs,QA2:qaJobs},CIHPK:{JAKG:['JAKG Supervising Consultant (SPMK Date: March 12, 2026)'],'Manggarai Hydrant':['Supervising Consultant for the Manggarai Balai Yasa Hydrant Installation (SPMK Date: June 8, 2026)'],Jember:['Jember Station Area Redevelopment']}};
	let selectedDivision = $state<number | null>(null);
	let reports = $state<Report[]>([]);
	let loading = $state(false);
	let error = $state('');
	let period = $state<ReportingPeriod | null>(null);
	let exporting = $state(false);
	let rolloverTimer: ReturnType<typeof setTimeout> | undefined;
	let editing = $state<Report | null>(null);
	let deleting = $state<Report | null>(null);
	let editWorkDivision = $state('');
	let editRole = $state('');
	let editProjectWork = $state('');
	let editDescription = $state('');
	const tableRows = $derived(groupReports(reports));

	async function selectDivision(index: number) {
		selectedDivision = index;
		reports = [];
		loading = true;
		error = '';
		try {
			const response = await api<{ reports: Report[] | null; period: ReportingPeriod }>(`/reports?primaryDivision=${encodeURIComponent(divisions[index])}`);
			reports = response.reports ?? [];
			period = response.period;
			clearTimeout(rolloverTimer);
			rolloverTimer = setTimeout(() => void selectDivision(index), new Date(response.period.end).getTime() + 86_400_000 - Date.now() + 1_000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Laporan tidak dapat dimuat.';
		} finally {
			loading = false;
		}
	}

	async function exportPDF() {
		if (selectedDivision === null || reports.length === 0 || exporting) return;
		exporting = true;
		error = '';
		try {
			const { blob, filename } = await download(`/reports.pdf?primaryDivision=${encodeURIComponent(divisions[selectedDivision])}`);
			const url = URL.createObjectURL(blob);
			const link = document.createElement('a');
			link.href = url;
			link.download = filename;
			link.click();
			URL.revokeObjectURL(url);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Laporan tidak dapat diekspor.';
		} finally {
			exporting = false;
		}
	}

	async function saveEdit() {
		if (!editing) return;
		error = '';
		try {
			await api<{ report: Report }>(`/reports/${editing.id}`, { method: 'PUT', body: JSON.stringify({ primaryDivision: editing.primaryDivision, workDivision: editWorkDivision, role: editing.primaryDivision === 'Project Engineer' ? editRole : '', work: editing.primaryDivision === 'Project Engineer' ? editProjectWork : '', description: editDescription }) });
			editing = null;
			if (selectedDivision !== null) await selectDivision(selectedDivision);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Laporan tidak dapat diperbarui.';
		}
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
		return () => clearTimeout(rolloverTimer);
	});
</script>

<svelte:head>
	<title>Weekly Progress | my-ple</title>
	<meta name="description" content="Pilih divisi dan lihat daftar pekerjaan mingguan." />
</svelte:head>

<main class="weekly-progress-page">
	<div class="weekly-progress-shell">
		<a class="back-link" href="/dashboard">← Kembali ke dashboard</a>
		<header class="weekly-header"><p class="weekly-kicker">WEEKLY PROGRESS</p><h1>Progres kerja per divisi.</h1><p>Pilih divisi terlebih dahulu untuk melihat daftar pekerjaan minggu ini.</p></header>
		<section class="division-selection" aria-labelledby="division-title">
			<h2 id="division-title">Pilih divisi</h2><p>Daftar ini menentukan pekerjaan yang akan ditampilkan.</p>
			<div class="division-grid">{#each divisions as division, index}<button class:selected={selectedDivision === index} class="division-button" type="button" aria-pressed={selectedDivision === index} onclick={() => void selectDivision(index)}>{division}<span aria-hidden="true">→</span></button>{/each}</div>
		</section>
		{#if selectedDivision !== null}
			<section class="task-list" aria-labelledby="task-title">
				<div class="task-list-header"><div><h2 id="task-title">Pekerjaan yang sudah dilakukan</h2><p>Daftar pekerjaan pada minggu berjalan.</p>{#if period}<p class="report-period">Periode: {period.label}</p>{/if}</div><div class="task-list-actions"><p class="selected-division">{divisions[selectedDivision]}</p><div class="report-actions"><button class="report-button export-button" type="button" disabled={reports.length === 0 || exporting || loading} onclick={() => void exportPDF()}>{exporting ? 'Mengekspor…' : 'Export PDF'}</button><a class="report-button" href={`/weekly-progress/new?division=${encodeURIComponent(divisions[selectedDivision])}`}>+ Input laporan</a></div></div></div>
				<div class="work-table-wrap">
					<table class="work-table">
						<thead><tr><th scope="col">Role</th><th scope="col">Kategori</th><th scope="col">No.</th><th scope="col">Pekerjaan yang dilakukan</th><th scope="col">Aksi</th></tr></thead>
						<tbody>
							{#if loading}<tr><td colspan="5">Memuat laporan…</td></tr>
							{:else if reports.length === 0}<tr><td colspan="5">Belum ada laporan pekerjaan untuk periode ini.</td></tr>
							{:else}{#each tableRows as row}
								{#if row.type === 'work'}
									<tr class="work-group-row">
										{#if row.role}<td rowspan={row.roleRowSpan ?? 1} class="task-role">{row.role}</td>{/if}
										{#if row.category}<td rowspan={row.categoryRowSpan ?? 1} class="task-category">{row.category}</td>{/if}
										<td colspan="3" class="task-work-title">{row.work}</td>
									</tr>
								{:else}
									<tr><td class="task-number">{row.displayNumber}</td><td class="task-description">{row.report.description}</td><td class="task-actions">{#if canModifyReport($currentUser, row.report.userId)}<div class="report-row-actions"><button class="edit-report" type="button" onclick={() => { editing = row.report; editWorkDivision = row.report.workDivision; editRole = row.report.role; editProjectWork = row.report.work; editDescription = row.report.description; }}>Edit</button><button class="delete-report" type="button" onclick={() => (deleting = row.report)}>Hapus</button></div>{:else}<span class="no-actions">—</span>{/if}</td></tr>
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

{#if editing}
	<div class="report-drawer-layer" transition:fade={{ duration: 180 }}>
		<button class="report-drawer-backdrop" type="button" aria-label="Tutup edit laporan" onclick={() => (editing = null)}></button>
		<div class="report-drawer" role="dialog" aria-modal="true" aria-labelledby="edit-report-title" transition:fly={{ x: 32, duration: 240, opacity: 0 }}>
			<div class="report-drawer-header"><h2 id="edit-report-title">Edit laporan</h2><button class="drawer-close" type="button" aria-label="Tutup" onclick={() => (editing = null)}>×</button></div>
			<form class="report-form report-drawer-form" onsubmit={(event) => { event.preventDefault(); void saveEdit(); }}>
				{#if editing.primaryDivision === 'Project Engineer'}
					<label>Role<select bind:value={editRole} onchange={() => { editWorkDivision = Object.keys(projectEngineer[editRole])[0]; editProjectWork = projectEngineer[editRole][editWorkDivision][0] ?? ''; }}>{#each Object.keys(projectEngineer) as item}<option value={item}>{item}</option>{/each}</select></label>
					<label>Job Category<select bind:value={editWorkDivision} onchange={() => { editProjectWork = projectEngineer[editRole][editWorkDivision][0] ?? ''; }}>{#each Object.keys(projectEngineer[editRole]) as item}<option value={item}>{item}</option>{/each}</select></label>
					<label>Job<select bind:value={editProjectWork} required>{#each projectEngineer[editRole][editWorkDivision] ?? [] as item}<option value={item}>{item}</option>{/each}</select></label>
				{:else}
					<label>Job Category<select bind:value={editWorkDivision}>{#each workDivisions[editing.primaryDivision] as item}<option value={item}>{item}</option>{/each}</select></label>
				{/if}
				<label>Uraian pekerjaan yang dilakukan<textarea bind:value={editDescription} rows="5" required maxlength="500"></textarea></label>
				<div class="drawer-form-actions"><button class="report-submit" type="submit">Simpan perubahan</button><button class="drawer-cancel" type="button" onclick={() => (editing = null)}>Batal</button></div>
			</form>
		</div>
	</div>
{/if}
{#if deleting}
	<div class="report-modal-layer" transition:fade={{ duration: 180 }}>
		<button class="report-modal-backdrop" type="button" aria-label="Batal hapus laporan" onclick={() => (deleting = null)}></button>
		<div class="report-delete-modal" role="dialog" aria-modal="true" aria-labelledby="delete-report-title" transition:fly={{ y: 18, duration: 220, opacity: 0 }}>
			<h2 id="delete-report-title">Hapus laporan?</h2><p>Make sure Anda ingin menghapus laporan ini. Tindakan ini tidak dapat dibatalkan.</p>
			<div class="delete-modal-actions"><button class="delete-confirm" type="button" onclick={() => void deleteReport()}>Hapus laporan</button><button class="drawer-cancel" type="button" onclick={() => (deleting = null)}>Batal</button></div>
		</div>
	</div>
{/if}
