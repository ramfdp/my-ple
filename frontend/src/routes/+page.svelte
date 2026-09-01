<script lang="ts">
	import { goto } from '$app/navigation';
	import { api } from '$lib/api';
	let error = $state('');
	async function login(event: SubmitEvent) {
		event.preventDefault(); error = '';
		const data = new FormData(event.currentTarget as HTMLFormElement);
		try { const result = await api<{ token: string; user: object }>('/auth/login', { method: 'POST', body: JSON.stringify({ username: data.get('username'), password: data.get('password') }) }); localStorage.setItem('session', JSON.stringify(result)); goto('/dashboard'); } catch (e) { error = e instanceof Error ? e.message : 'Login failed.'; }
	}
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
			<img class="brand-logo" src="../../Logo_PLE-removebg-preview-300x233.png" alt="my-ple" />

			<div class="login-heading">
				<p class="eyebrow">MY-PLE WORKSPACE</p>
				<h1 id="login-title">Selamat datang kembali.</h1>
				<p>Masuk untuk melanjutkan pekerjaan Anda.</p>
			</div>

			<form class="login-form" onsubmit={login}>
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

			{#if error}<p class="form-note" style="color:#b91c1c">{error}</p>{/if}
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
