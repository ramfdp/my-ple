<script lang="ts">
	import { onMount } from 'svelte';
	import { api, session, type Report } from '$lib/api';

	const divisions = ['Project Engineer', 'AI Dev', 'Keuangan', 'Business Development', 'HR', 'PMO'];
	const workDivisions: Record<string, string[]> = {
		'Project Engineer': ['MK supervisi', 'Non-MK supervisi', 'QA 1', 'QA 2', 'JAKG', 'Manggarai Hydrant', 'Jember'],
		'AI Dev': ['Pengembangan', 'Pengujian Model', 'Deployment', 'Dokumentasi'], Keuangan: ['Pembukuan', 'Pembayaran', 'Anggaran', 'Pelaporan'],
		'Business Development': ['Prospek', 'Proposal', 'Kemitraan', 'Tindak Lanjut'], HR: ['Kehadiran', 'Rekrutmen', 'Administrasi', 'Pengembangan SDM'], PMO: ['Milestone', 'Rapat', 'Risiko', 'Pelaporan']
	};
	const qaJobs = ['MPS PIT','MPS SGR','MPS SGP','SHL JN','SOP Tata Kelola','Studi Daya Beli Gas Bumi','Review FS dan DED Pelabuhan Waipirit','Project Management Office','Reimagine Rumah IPS','Penataan Kawasan Stasiun Jember','Pengawasan Renovasi Gudang Besar JAKG','Pengawasan Instalasi Hidran BY Manggarai','Internal'];
	const projectEngineer: Record<string, Record<string, string[]>> = {PE:{'MK supervisi':['Penataan Kawasan Stasiun Jember','Pengawasan Renovasi Gudang Besar JAKG','Pengawasan Instalasi Hidran BY Manggarai'],'Non-MK supervisi':['MPS PIT','MPS SGR','MPS SGP','SHL JN','SOP Tata Kelola','Studi Daya Beli Gas Bumi','Review FS dan DED Pelabuhan Waipirit','Project Management Office','Reimagine Rumah IPS']},'QA 1':{'QA 1':qaJobs},'QA 2':{'QA 2':qaJobs},CIHPK:{JAKG:['JAKG Supervising Consultant (SPMK Date: March 12, 2026)'],'Manggarai Hydrant':['Supervising Consultant for the Manggarai Balai Yasa Hydrant Installation (SPMK Date: June 8, 2026)'],Jember:['Jember Station Area Redevelopment']}};
	let selectedDivision = $state<number | null>(null);
	let reports = $state<Report[]>([]);
	let loading = $state(false);
	let error = $state('');
	let editing = $state<Report | null>(null);
	let editWorkDivision = $state('');
	let editRole = $state('');
	let editProjectWork = $state('');
	let editDescription = $state('');

	async function selectDivision(index: number) {
		selectedDivision = index;
		reports = [];
		loading = true;
		error = '';
		try {
			reports = (await api<{ reports: Report[] | null }>(`/reports?primaryDivision=${encodeURIComponent(divisions[index])}`)).reports ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Laporan tidak dapat dimuat.';
		} finally {
			loading = false;
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

	async function deleteReport(report: Report) {
		if (!confirm(`Hapus laporan #${report.number}?`)) return;
		error = '';
		try {
			await api<void>(`/reports/${report.id}`, { method: 'DELETE' });
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
				<div class="task-list-header"><div><h2 id="task-title">Pekerjaan yang sudah dilakukan</h2><p>Daftar pekerjaan pada minggu berjalan.</p></div><div class="task-list-actions"><p class="selected-division">{divisions[selectedDivision]}</p><a class="report-button" href={`/weekly-progress/new?division=${encodeURIComponent(divisions[selectedDivision])}`}>+ Input laporan</a></div></div>
				<div class="work-table-wrap"><table class="work-table"><thead><tr><th scope="col">Division</th><th scope="col">Category</th><th scope="col">Role</th><th scope="col">Work</th><th scope="col">Number</th><th scope="col">Work Performed</th><th scope="col">Actions</th></tr></thead><tbody>{#if loading}<tr><td colspan="7">Memuat laporan…</td></tr>{:else if reports.length === 0}<tr><td colspan="7">No report available</td></tr>{:else}{#each reports as report}<tr><td class="task-role">{report.primaryDivision}</td><td>{report.workDivision}</td><td>{report.role || '—'}</td><td>{report.work || '—'}</td><td class="task-number">{report.number}</td><td>{report.description}</td><td>{#if session()?.user.id === report.userId}<button type="button" onclick={() => { editing = report; editWorkDivision = report.workDivision; editRole = report.role; editProjectWork = report.work; editDescription = report.description; }}>Edit</button> <button type="button" onclick={() => void deleteReport(report)}>Delete</button>{/if}</td></tr>{/each}{/if}</tbody></table></div>
				{#if error}<p class="demo-note">{error}</p>{/if}
				{#if editing}<form class="report-form" onsubmit={(event) => { event.preventDefault(); void saveEdit(); }}>{#if editing.primaryDivision === 'Project Engineer'}<label>Role<select bind:value={editRole} onchange={() => { editWorkDivision = Object.keys(projectEngineer[editRole])[0]; editProjectWork = projectEngineer[editRole][editWorkDivision][0] ?? ''; }}>{#each Object.keys(projectEngineer) as item}<option value={item}>{item}</option>{/each}</select></label><label>Job Category<select bind:value={editWorkDivision} onchange={() => { editProjectWork = projectEngineer[editRole][editWorkDivision][0] ?? ''; }}>{#each Object.keys(projectEngineer[editRole]) as item}<option value={item}>{item}</option>{/each}</select></label><label>Job<select bind:value={editProjectWork} required>{#each projectEngineer[editRole][editWorkDivision] ?? [] as item}<option value={item}>{item}</option>{/each}</select></label>{:else}<label>Job Category<select bind:value={editWorkDivision}>{#each workDivisions[editing.primaryDivision] as item}<option value={item}>{item}</option>{/each}</select></label>{/if}<label>Uraian pekerjaan yang dilakukan<textarea bind:value={editDescription} rows="5" required maxlength="500"></textarea></label><button class="report-submit" type="submit">Simpan perubahan</button><button type="button" onclick={() => (editing = null)}>Batal</button></form>{/if}
			</section>
		{/if}
	</div>
</main>
