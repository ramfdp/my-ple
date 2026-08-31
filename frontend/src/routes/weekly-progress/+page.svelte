<script lang="ts">
	const divisions = [
		{ name: 'Project Engineer', tasks: [['Quality Assurance', 'Memperbarui progres pekerjaan lapangan.', '25 Agustus 2026, 09.15'], ['Pengawasan', 'Meninjau gambar kerja dan kebutuhan material.', '27 Agustus 2026, 13.40'], ['Non-Pengawasan', 'Menyusun laporan koordinasi teknis mingguan.', '28 Agustus 2026, 16.20'], ['CIHPK', 'Memperbarui dokumentasi pekerjaan proyek.', '29 Agustus 2026, 10.10']] },
		{ name: 'AI Dev', tasks: [['Quality Assurance', 'Menguji akurasi keluaran model AI.', '25 Agustus 2026, 09.30'], ['Pengawasan', 'Memantau performa layanan AI.', '27 Agustus 2026, 13.15'], ['Non-Pengawasan', 'Mengembangkan fitur otomatisasi berbasis AI.', '28 Agustus 2026, 15.40'], ['CIHPK', 'Memperbarui dokumentasi implementasi AI.', '29 Agustus 2026, 10.30']] },
		{ name: 'Keuangan', tasks: [['Quality Assurance', 'Merekap transaksi dan dokumen pendukung.', '25 Agustus 2026, 10.05'], ['Pengawasan', 'Meninjau tagihan serta jadwal pembayaran.', '27 Agustus 2026, 14.10'], ['Non-Pengawasan', 'Memperbarui proyeksi arus kas mingguan.', '28 Agustus 2026, 15.45'], ['CIHPK', 'Memeriksa kelengkapan laporan biaya proyek.', '29 Agustus 2026, 11.20']] },
		{ name: 'Business Development', tasks: [['Quality Assurance', 'Memperbarui daftar prospek dan peluang kerja sama.', '25 Agustus 2026, 11.30'], ['Pengawasan', 'Menyiapkan bahan proposal atau presentasi.', '27 Agustus 2026, 09.50'], ['Non-Pengawasan', 'Menindaklanjuti komunikasi dengan calon mitra.', '28 Agustus 2026, 14.25'], ['CIHPK', 'Menyusun ringkasan peluang proyek baru.', '29 Agustus 2026, 13.15']] },
		{ name: 'HR', tasks: [['Quality Assurance', 'Meninjau data kehadiran dan kebutuhan tim.', '25 Agustus 2026, 08.45'], ['Pengawasan', 'Melakukan koordinasi proses rekrutmen.', '27 Agustus 2026, 11.15'], ['Non-Pengawasan', 'Memperbarui administrasi karyawan.', '28 Agustus 2026, 13.30'], ['CIHPK', 'Menyusun rekap kebutuhan personel proyek.', '29 Agustus 2026, 14.00']] },
		{ name: 'PMO', tasks: [['Quality Assurance', 'Memantau capaian milestone setiap proyek.', '25 Agustus 2026, 09.35'], ['Pengawasan', 'Menyusun catatan rapat dan tindak lanjut.', '27 Agustus 2026, 15.00'], ['Non-Pengawasan', 'Memperbarui daftar risiko serta isu proyek.', '28 Agustus 2026, 16.40'], ['CIHPK', 'Mengonsolidasikan laporan kemajuan proyek.', '29 Agustus 2026, 15.30']] }
	];

	let selectedDivision = $state<number | null>(null);
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
			<section class="task-list" aria-labelledby="task-title">
				<div class="task-list-header">
					<div><h2 id="task-title">Pekerjaan yang sudah dilakukan</h2><p>Daftar pekerjaan pada minggu berjalan.</p></div>
					<p class="selected-division">{division.name}</p>
				</div>
				<div class="work-table-wrap">
					<table class="work-table">
						<thead><tr><th scope="col">Divisi</th><th scope="col">Pekerjaan yang dilakukan</th><th scope="col">Waktu dikerjakan</th></tr></thead>
						<tbody>{#each division.tasks as task}<tr><td class="task-division">{task[0]}</td><td>{task[1]}</td><td class="task-time">{task[2]} WIB</td></tr>{/each}</tbody>
					</table>
				</div>
				<p class="demo-note">Data contoh — akan dihubungkan ke laporan pengguna setelah database tersedia.</p>
			</section>
		{/if}
	</div>
</main>
