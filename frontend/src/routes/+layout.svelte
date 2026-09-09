<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
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
	let auth = $derived(initializing ? 'loading' : $currentUser ? 'authenticated' : 'unauthenticated');
	onMount(async () => {
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
		page.url.pathname;
		if (!showSidebar || auth !== 'authenticated') return;
		return startNotifications();
	});
</script>

<svelte:head>
	<link rel="icon" type="image/webp" href="/harmoni%20insan%20phooto.webp" />
</svelte:head>

{#if auth === 'loading'}
	<p role="status" aria-live="polite">Memeriksa sesi…</p>
{:else if showSidebar && auth === 'authenticated'}
	<div class="feature-shell">
		<Sidebar bind:mobileOpen={mobileNavigationOpen} />
		<div class="feature-content" inert={mobileNavigationOpen}>{@render children()}</div>
	</div>
{:else if !showSidebar}
	{@render children()}
{/if}

<style>
	.feature-shell { display: flex; min-height: 100dvh; }
	.feature-content { min-width: 0; flex: 1; overflow-wrap: anywhere; }
	@media (max-width: 868px) { .feature-shell { flex-direction: column; } }
</style>
