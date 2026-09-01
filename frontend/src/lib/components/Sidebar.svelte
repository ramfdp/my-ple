<script lang="ts">
	let open = $state(true);
</script>

<aside class:compact={!open} class="sidebar" aria-label="Navigasi utama">
	<button class="sidebar-toggle" type="button" onclick={() => (open = !open)} aria-expanded={open} aria-label="Tampilkan atau sembunyikan sidebar">•••</button>
	{#if open}
		<a class="sidebar-brand" href="/dashboard">my-ple</a>
		<nav>
			<details open>
				<summary>Dashboard <span aria-hidden="true">⌄</span></summary>
				<a href="/dashboard">Beranda</a>
				<a href="/overtime">List Lembur</a>
				<a href="/zoom-booking">Booking Zoom</a>
				<a href="/attendance">Absensi</a>
				<a href="/weekly-progress">Weekly Progress</a>
			</details>
			<a href="/dashboard#profile">Profile</a>
			<a class="logout-link" href="/">Log out</a>
		</nav>
	{/if}
</aside>

<style>
	/* ponytail: sticky sidebar follows viewport during scroll, avoiding scroll-back trips */
	.sidebar { position: sticky; top: 0; width: 240px; height: 100dvh; padding: 20px; background: #102a43; color: #d8e7e2; transition: width 180ms ease; flex-shrink: 0; overflow-y: auto; }
	.sidebar.compact { width: 64px; padding: 20px 12px; }
	.sidebar-toggle { width: 40px; height: 36px; border: 0; border-radius: 10px; background: rgb(255 255 255 / 10%); color: #fff; font-weight: 800; letter-spacing: 2px; }
	.sidebar-brand { display: block; margin: 28px 12px 36px; color: #fff; font-family: 'Plus Jakarta Sans', sans-serif; font-size: 1.25rem; font-weight: 800; letter-spacing: -0.05em; text-decoration: none; }
	nav, details { display: grid; gap: 6px; }
	summary, nav > a, details a { display: flex; align-items: center; justify-content: space-between; padding: 11px 12px; border-radius: 10px; color: inherit; font-size: 0.9rem; font-weight: 700; text-decoration: none; }
	summary { cursor: pointer; }
	details a { padding-left: 24px; color: #9bb5b0; font-size: 0.82rem; }
	summary:hover, nav > a:hover, details a:hover { background: rgb(255 255 255 / 10%); color: #fff; }
	.logout-link { margin-top: 20px; color: #ffb7c2; }
</style>
