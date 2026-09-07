<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, setAuthenticated, type User } from '$lib/api';
	let error = $state('');
	let submitting = $state(false);
	async function login(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;
		error = ''; submitting = true;
		const data = new FormData(event.currentTarget as HTMLFormElement);
		try {
			await api('/auth/login', { method: 'POST', body: JSON.stringify({ username: data.get('username'), password: data.get('password') }) });
			const { user } = await api<{ user: User }>('/auth/me');
			setAuthenticated(user);
			await goto('/dashboard');
		} catch (e) {
			error = e instanceof Error ? e.message : 'Login failed.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Masuk | my HIS</title>
	<meta
		name="description"
		content="Masuk ke ruang kerja my HIS untuk melanjutkan pekerjaan Anda."
	/>
</svelte:head>

<main class="login-page">
	<section class="login-panel" aria-labelledby="login-title">
		<div class="login-content">
			<img class="brand-logo" src="/harmoni%20insan%20phooto.webp" alt="my HIS" />

			<div class="login-heading">
				<p class="eyebrow">my HIS WORKSPACE</p>
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

				<button type="submit" disabled={submitting}>Masuk <span aria-hidden="true">→</span></button>
			</form>

			{#if error}<p class="form-note" style="color:var(--his-danger)">{error}</p>{/if}
		</div>
	</section>

	<aside class="showcase-panel" aria-label="Tentang my HIS">
		<div class="grid-pattern" aria-hidden="true"></div>
		<div class="showcase-content">
			<p class="eyebrow">WORK WITH CLARITY</p>
			<h2>Satu tempat untuk menjaga pekerjaan tetap bergerak.</h2>
		</div>
	</aside>
</main>
