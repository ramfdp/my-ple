<script lang="ts">
 import { t as translate } from '$lib/i18n';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api, setAuthenticated, type User } from '$lib/api';
	let introducing = $state(true);
	onMount(() => {
		if (matchMedia('(prefers-reduced-motion: reduce)').matches) {
			introducing = false;
			return;
		}
		const timer = setTimeout(() => introducing = false, 1000);
		return () => clearTimeout(timer);
	});
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
	<title>{$translate("Masuk | MyHIS")}</title>
	<meta
		name="description"
		content={$translate("Masuk ke ruang kerja MyHIS untuk melanjutkan pekerjaan Anda.")}
	/>
</svelte:head>

{#if introducing}
	<div class="login-intro" role="status" aria-label="MyHIS">
		<img src="/harmoni%20insan%20phooto.webp" alt="MyHIS" width="188" height="176" />
	</div>
{:else}
<main class="login-page">
	<section class="login-panel" aria-labelledby="login-title">
		<div class="login-content">
			<img class="brand-logo" src="/harmoni%20insan%20phooto.webp" alt={$translate("MyHIS")} />

			<div class="login-heading">
				<p class="eyebrow">{$translate("MyHIS WORKSPACE")}</p>
				<h1 id="login-title">{$translate("Selamat datang kembali.")}</h1>
				<p>{$translate("Masuk untuk melanjutkan pekerjaan Anda.")}</p>
			</div>

			<form class="login-form" onsubmit={login}>
				<label for="username">{$translate("Username")}</label>
				<input id="username" name="username" autocomplete="username" placeholder={$translate("Masukkan username")} required />

				<label for="password">{$translate("Password")}</label>
				<input
					id="password"
					name="password"
					type="password"
					autocomplete="current-password"
					placeholder={$translate("Masukkan password")}
					required
				/>

				<button type="submit" disabled={submitting}>{$translate("Masuk")} <span aria-hidden="true">→</span></button>
			</form>

			{#if error}<p class="form-note" style="color:var(--his-danger)">{$translate(error)}</p>{/if}
		</div>
	</section>

	<aside class="showcase-panel" aria-label={$translate("Tentang MyHIS")}>
		<div class="grid-pattern" aria-hidden="true"></div>
		<div class="showcase-content">
			<p class="eyebrow">{$translate("WORK WITH CLARITY")}</p>
			<h2>{$translate("Satu tempat untuk menjaga pekerjaan tetap bergerak.")}</h2>
		</div>
	</aside>
</main>
{/if}
