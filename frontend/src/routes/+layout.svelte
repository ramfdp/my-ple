<script lang="ts">
 import { t as translate } from '$lib/i18n';
	import { restoreLanguage } from '$lib/i18n';
 import { page } from '$app/state';
	import { afterNavigate, goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api, currentUser, setAuthenticated, type User } from '$lib/api';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import { startNotifications } from '$lib/notifications';
	import '$lib/styles/base.css';
	import '$lib/styles/login.css';
	import '$lib/styles/dashboard.css';
	import '$lib/styles/weekly-progress.css';
	import '$lib/styles/booking-zoom.css';

	let { children } = $props();
	let mobileNavigationOpen = $state(false);
	let showSidebar = $derived(page.url.pathname !== '/');
	let initializing = $state(true);
	let backDepth = $state(0);
	afterNavigate(({ from, type, delta }) => {
		if (!from || from.url.pathname === '/') { backDepth = 0; return; }
		if (type === 'popstate') backDepth = Math.max(0, backDepth + (delta ?? 0));
		else if (type === 'link' || type === 'goto') backDepth++;
	});
	function back() {
		if (backDepth > 0) history.back();
		else void goto('/dashboard');
	}
	let auth = $derived(initializing ? 'loading' : $currentUser ? 'authenticated' : 'unauthenticated');
	onMount(async () => {
 restoreLanguage();
		localStorage.removeItem('session');
		try {
			const { user } = await api<{ user: User }>('/auth/me');
			setAuthenticated(user);
			if (page.url.pathname === '/') await goto('/dashboard');
		} catch {
			setAuthenticated(null);
			if (page.url.pathname !== '/') await goto('/');
		} finally {
			initializing = false;
		}
	});
	$effect(() => {
		if (!showSidebar || auth !== 'authenticated') return;
		return startNotifications();
	});
</script>

<svelte:head>
	<link rel="icon" type="image/webp" href="/harmoni%20insan%20phooto.webp" />
</svelte:head>

{#if auth === 'loading'}
	<p role="status" aria-live="polite">{$translate("Memeriksa sesi…")}</p>
{:else if showSidebar && auth === 'authenticated'}
	<div class="feature-shell">
		<Sidebar bind:mobileOpen={mobileNavigationOpen} />
		<div class="feature-content" inert={mobileNavigationOpen}><nav class="page-back"><button type="button" onclick={back} disabled={backDepth === 0 && page.url.pathname === '/dashboard'}>← {$translate('Kembali')}</button></nav>{@render children()}</div>
	</div>
{:else if !showSidebar}
	<nav class="page-back"><button type="button" disabled>← {$translate('Kembali')}</button></nav>
	{@render children()}
{/if}

<style>
	.page-back { padding: .75rem 1.5rem 0; }
	.page-back button { min-height: 40px; padding: .4rem .8rem; cursor: pointer; }
	.page-back button:disabled { opacity: .5; cursor: default; }
	@media print { .page-back { display: none; } }
	.feature-shell { display: flex; min-height: 100dvh; }
	.feature-content { min-width: 0; flex: 1; overflow-wrap: anywhere; }
	@media (max-width: 868px) { .feature-shell { flex-direction: column; } }
</style>
