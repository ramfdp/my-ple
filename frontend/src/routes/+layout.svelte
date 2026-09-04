<script lang="ts">
	import { page } from '$app/state';
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
	$effect(() => {
		page.url.pathname;
		if (!showSidebar) return;
		return startNotifications();
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if showSidebar}
	<div class="feature-shell">
		<Sidebar />
		<div class="feature-content">{@render children()}</div>
	</div>
{:else}
	{@render children()}
{/if}

<style>
	/* ponytail: keep sidebar on side across desktop and mobile, scrolling naturally with page flow */
	.feature-shell { display: flex; min-height: 100vh; }
	.feature-content { min-width: 0; flex: 1; }
</style>
