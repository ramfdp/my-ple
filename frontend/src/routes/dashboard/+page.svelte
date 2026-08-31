<script lang="ts">
	const userName = 'Nama Pengguna';
</script>

<svelte:head>
	<title>Dashboard | my-ple</title>
	<meta name="description" content="Dashboard kerja my-ple." />
</svelte:head>

<main class="dashboard-page">
	<header class="dashboard-header">
		<div>
			<p class="dashboard-kicker">DASHBOARD</p>
			<h1>Halo, <span>{userName}</span> <span aria-hidden="true">👋</span></h1>
			<p class="dashboard-subtitle">Pilih layanan untuk melanjutkan pekerjaan hari ini.</p>
		</div>
		<button class="notification-button" type="button" aria-label="Notifikasi">
			<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" /></svg>
			<span></span>
		</button>
	</header>

	<section class="dashboard-menu" aria-labelledby="menu-title">
		<div class="menu-heading">
			<div>
				<p class="dashboard-kicker">LAYANAN</p>
				<h2 id="menu-title">Menu utama</h2>
			</div>
			<p>4 layanan tersedia</p>
		</div>

		<div class="menu-grid">
			<article class="menu-card attendance-card">
				<div class="menu-icon" aria-hidden="true">
					<svg viewBox="0 0 24 24"><path d="M7 3v3M17 3v3M4 9h16M5 5h14a1 1 0 0 1 1 1v13a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1ZM8 13h3M8 17h5" /></svg>
				</div>
				<h3>Absensi</h3>
				<p>Catat kehadiran dan lihat riwayat kerja.</p>
			</article>

			<a class="menu-card progress-card" href="/weekly-progress">
				<div class="menu-icon" aria-hidden="true">
					<svg viewBox="0 0 24 24"><path d="M4 19V5M4 19h16M8 15v-4M12 15V8M16 15v-7" /></svg>
				</div>
				<h3>Weekly Progress</h3>
				<p>Ringkas progres dan prioritas mingguan.</p>
			</a>

			<article class="menu-card overtime-card">
				<div class="menu-icon" aria-hidden="true">
					<svg viewBox="0 0 24 24"><circle cx="12" cy="12" r="8" /><path d="M12 8v4l3 2M7 3v3M17 3v3" /></svg>
				</div>
				<h3>List Lembur</h3>
				<p>Kelola daftar dan pengajuan lembur.</p>
			</article>

			<article class="menu-card zoom-card">
				<div class="menu-icon" aria-hidden="true">
					<svg viewBox="0 0 24 24"><rect x="3" y="6" width="13" height="12" rx="2" /><path d="m16 10 5-3v10l-5-3" /></svg>
				</div>
				<h3>Booking Zoom</h3>
				<p>Jadwalkan ruang rapat virtual Anda.</p>
			</article>
		</div>
	</section>
</main>
