<script lang="ts">
	import { t as translate } from '$lib/i18n';
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
	<title>{$translate("Masuk | MyHIS")}</title>
	<meta
		name="description"
		content={$translate("Masuk ke ruang kerja MyHIS untuk melanjutkan pekerjaan Anda.")}
	/>
</svelte:head>

<main class="login-page">
	<img class="login-art" src="/login-folds.webp" alt="" width="1720" height="1000" fetchpriority="high" />
	<header class="login-brand">
		<img src="/harmoni%20insan%20phooto.webp" width="64" height="64" alt="" />
		<span>MyHIS</span>
	</header>
	<section class="login-body" aria-labelledby="login-title">
		<div class="login-heading">
			<h1 id="login-title">{$translate("Masuk")}</h1>
			<p>{$translate("Masuk untuk melanjutkan pekerjaan Anda.")}</p>
		</div>
		<form class="login-form" onsubmit={login} aria-busy={submitting}>
			<label for="username">{$translate("Username")}</label>
			<input id="username" name="username" autocomplete="username" required />

			<label for="password">{$translate("Password")}</label>
			<input id="password" name="password" type="password" autocomplete="current-password" required />

			{#if error}<p class="login-error" role="alert">{$translate(error)}</p>{/if}
			<button type="submit" disabled={submitting}>
				{$translate(submitting ? "Sedang masuk…" : "Masuk")}
			</button>
		</form>
	</section>
</main>
