<script lang="ts">
	import BrandLogo from '$lib/components/BrandLogo.svelte';
</script>

<svelte:head>
	<title>Masuk | my-ple</title>
	<meta
		name="description"
		content="Masuk ke ruang kerja my-ple untuk melanjutkan pekerjaan Anda."
	/>
</svelte:head>

<main class="login-page">
	<section class="login-panel" aria-labelledby="login-title">
		<div class="login-content">
			<BrandLogo />

			<div class="login-heading">
				<p class="eyebrow">MY-PLE WORKSPACE</p>
				<h1 id="login-title">Selamat datang kembali.</h1>
				<p>Masuk untuk melanjutkan pekerjaan Anda.</p>
			</div>

			<form class="login-form" onsubmit={(event) => event.preventDefault()}>
				<label for="username">Username</label>
				<input id="username" name="username" autocomplete="username" placeholder="Masukkan username" required />

				<label for="password">Password</label>
				<input
					id="password"
					name="password"
					type="password"
					autocomplete="current-password"
					placeholder="Masukkan password"
					required
				/>

				<button type="submit">Masuk <span aria-hidden="true">→</span></button>
			</form>

			<p class="form-note">Akses akun akan dihubungkan setelah layanan autentikasi tersedia.</p>
		</div>
	</section>

	<aside class="showcase-panel" aria-label="Tentang my-ple">
		<div class="grid-pattern" aria-hidden="true"></div>
		<div class="showcase-content">
			<p class="eyebrow">WORK WITH CLARITY</p>
			<h2>Satu tempat untuk menjaga pekerjaan tetap bergerak.</h2>
		</div>
	</aside>
</main>
