<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api, setAuthenticated, type User } from '$lib/api';
	import favicon from '$lib/assets/favicon.svg';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import { startNotifications } from '$lib/notifications';
	import '$lib/styles/base.css';
	import '$lib/styles/login.css';
	import '$lib/styles/dashboard.css';
	import '$lib/styles/weekly-progress.css';
	import '$lib/styles/booking-zoom.css';

	let { children } = $props();
	let showSidebar = $derived(page.url.pathname !== '/');
	let auth = $state<'loading' | 'authenticated' | 'unauthenticated'>('loading');
	onMount(async () => {
		localStorage.removeItem('session');
		try {
			const { user } = await api<{ user: User }>('/auth/me');
			setAuthenticated(user); auth = 'authenticated';
			if (page.url.pathname === '/') await goto('/dashboard');
		} catch {
			setAuthenticated(null); auth = 'unauthenticated';
			if (page.url.pathname !== '/') await goto('/');
		}
	});
	$effect(() => {
		page.url.pathname;
		if (!showSidebar || auth !== 'authenticated') return;
		return startNotifications();
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if showSidebar && auth === 'authenticated'}
	<div class="feature-shell">
		<Sidebar />
		<div class="feature-content">{@render children()}</div>
	</div>
{:else if !showSidebar}
	{@render children()}
{/if}

<style>
	/* ponytail: keep sidebar on side across desktop and mobile, scrolling naturally with page flow */
	.feature-shell { display: flex; min-height: 100vh; }
	.feature-content { min-width: 0; flex: 1; }
</style>
