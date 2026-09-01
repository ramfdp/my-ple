<script lang="ts">
	import { onMount } from 'svelte';

	type Task = [role: string, category: string, sequence: number, work: string];
	type LegacyTask = [string, string];
	type Division = { name: string; tasks: LegacyTask[] };

	const divisions: Division[] = [
		{ name: 'Project Engineer', tasks: [['Quality Assurance 1', 'Memperbarui progres pekerjaan lapangan.'], ['Pengawasan', 'Meninjau gambar kerja dan kebutuhan material.'], ['Non-Pengawasan', 'Menyusun laporan koordinasi teknis mingguan.'], ['CIHPK', 'Memperbarui dokumentasi pekerjaan proyek.']] },
		{ name: 'AI Dev', tasks: [['Pengembangan', 'Mengembangkan fitur otomatisasi berbasis AI.'], ['Pengujian Model', 'Menguji akurasi keluaran model AI.'], ['Deployment', 'Memantau performa layanan AI.'], ['Dokumentasi', 'Memperbarui dokumentasi implementasi AI.']] },
		{ name: 'Keuangan', tasks: [['Pembukuan', 'Merekap transaksi dan dokumen pendukung.'], ['Pembayaran', 'Meninjau tagihan serta jadwal pembayaran.'], ['Anggaran', 'Memperbarui proyeksi arus kas mingguan.'], ['Pelaporan', 'Memeriksa kelengkapan laporan biaya proyek.']] },
		{ name: 'Business Development', tasks: [['Prospek', 'Memperbarui daftar prospek dan peluang kerja sama.'], ['Proposal', 'Menyiapkan bahan proposal atau presentasi.'], ['Kemitraan', 'Menindaklanjuti komunikasi dengan calon mitra.'], ['Tindak Lanjut', 'Menyusun ringkasan peluang proyek baru.']] },
		{ name: 'HR', tasks: [['Kehadiran', 'Meninjau data kehadiran dan kebutuhan tim.'], ['Rekrutmen', 'Melakukan koordinasi proses rekrutmen.'], ['Administrasi', 'Memperbarui administrasi karyawan.'], ['Pengembangan SDM', 'Menyusun rekap kebutuhan personel proyek.']] },
		{ name: 'PMO', tasks: [['Milestone', 'Memantau capaian milestone setiap proyek.'], ['Rapat', 'Menyusun catatan rapat dan tindak lanjut.'], ['Risiko', 'Memperbarui daftar risiko serta isu proyek.'], ['Pelaporan', 'Mengonsolidasikan laporan kemajuan proyek.']] }
	];

	let selectedDivision = $state<number | null>(null);
	let savedTasks = $state<Record<string, Task[]>>({});

	function loadSavedTasks(): Record<string, Task[]> {
		const parsed: unknown = JSON.parse(localStorage.getItem('weekly-reports') ?? '{}');
		if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
		return Object.fromEntries(Object.entries(parsed).map(([division, reports]) => [division, Array.isArray(reports) ? reports.flatMap((report, index) => {
			if (!Array.isArray(report)) return [];
			if (report.length === 3) return [[division, String(report[0]), index + 2, String(report[1])] as Task];
			return [[String(report[0]), String(report[1]), Number(report[2]) || index + 1, String(report[3])] as Task];
		}) : []]));
	}

	onMount(() => {
		try {
			savedTasks = loadSavedTasks();
			const requestedDivision = new URLSearchParams(location.search).get('division');
			const index = divisions.findIndex((division) => division.name === requestedDivision);
			if (index >= 0) selectedDivision = index;
		} catch {
			localStorage.removeItem('weekly-reports');
		}
	});
</script>

<svelte:head>
	<title>Weekly Progress | my-ple</title>
	<meta name="description" content="Pilih divisi dan lihat daftar pekerjaan mingguan." />
</svelte:head>

<main class="weekly-progress-page">
	<div class="weekly-progress-shell">
		<a class="back-link" href="/dashboard">← Kembali ke dashboard</a>
		<header class="weekly-header">
			<p class="weekly-kicker">WEEKLY PROGRESS</p>
			<h1>Progres kerja per divisi.</h1>
			<p>Pilih divisi terlebih dahulu untuk melihat daftar pekerjaan minggu ini.</p>
		</header>

		<section class="division-selection" aria-labelledby="division-title">
			<h2 id="division-title">Pilih divisi</h2>
			<p>Daftar ini menentukan pekerjaan yang akan ditampilkan.</p>
			<div class="division-grid">
				{#each divisions as division, index}
					<button class:selected={selectedDivision === index} class="division-button" type="button" aria-pressed={selectedDivision === index} onclick={() => (selectedDivision = index)}>
						{division.name}<span aria-hidden="true">→</span>
					</button>
				{/each}
			</div>
		</section>

		{#if selectedDivision !== null}
			{@const division = divisions[selectedDivision]}
			{@const tasks = [...division.tasks.map(([category, work]) => [division.name, category, 1, work] as Task), ...(savedTasks[division.name] ?? [])]}
			<section class="task-list" aria-labelledby="task-title">
				<div class="task-list-header">
					<div><h2 id="task-title">Pekerjaan yang sudah dilakukan</h2><p>Daftar pekerjaan pada minggu berjalan.</p></div>
					<div class="task-list-actions"><p class="selected-division">{division.name}</p><a class="report-button" href={`/weekly-progress/new?division=${encodeURIComponent(division.name)}`}>+ Input laporan</a></div>
				</div>
				<div class="work-table-wrap">
					<table class="work-table">
						<thead><tr><th scope="col">Role</th><th scope="col">Category</th><th scope="col">Number</th><th scope="col">Work Performed</th></tr></thead>
						<tbody>{#each tasks as task}<tr><td class="task-role">{task[0]}</td><td>{task[1]}</td><td class="task-number">{task[2]}</td><td>{task[3]}</td></tr>{/each}</tbody>
					</table>
				</div>
				<p class="demo-note">Data contoh — akan dihubungkan ke laporan pengguna setelah database tersedia.</p>
			</section>
		{/if}
	</div>
</main>
