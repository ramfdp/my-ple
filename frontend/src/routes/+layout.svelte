<script lang="ts">
 import { t as translate } from '$lib/i18n';
	import { restoreLanguage } from '$lib/i18n';
 import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { goBack } from '$lib/navigation';
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
	let pathname = $derived(page.url.pathname.replace(/\/$/, '') || '/');
	let content = $state<HTMLDivElement>();
	// These routes already render their own dashboard link, including workspace details.
	let hasPageBack = $derived(
		['/booking-zoom', '/weekly-progress', '/weekly-progress/new', '/weekly-progress/history', '/workspace'].includes(pathname)
		|| /^\/workspace\/[^/]+$/.test(pathname)
	);
	$effect(() => {
		pathname;
		if (!content || matchMedia('(prefers-reduced-motion: reduce)').matches) return;
		const animation = content.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 180, easing: 'ease-out' });
		return () => animation.cancel();
	});
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
		<div class="feature-content" bind:this={content} inert={mobileNavigationOpen}>
			{#if pathname !== '/dashboard' && !hasPageBack}
				<nav class="page-back"><a class="back-link" href="/dashboard" onclick={goBack}>← {$translate('Kembali')}</a></nav>
			{/if}
			{@render children()}
		</div>
	</div>
{:else if !showSidebar}
	<div class="login-content-frame" bind:this={content}>{@render children()}</div>
{/if}

<style>
	.page-back { padding: var(--his-page-padding) var(--his-page-padding) 0; }
	@media print { .page-back { display: none; } }
	.feature-shell { display: flex; min-height: 100dvh; }
	.feature-content { min-width: 0; flex: 1; overflow-wrap: break-word; }
	@media (max-width: 868px) { .feature-shell { flex-direction: column; } }
</style>
