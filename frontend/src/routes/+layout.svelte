<script lang="ts">
	import { page } from '$app/state';
	import favicon from '$lib/assets/favicon.svg';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import '$lib/styles/base.css';
	import '$lib/styles/login.css';
	import '$lib/styles/dashboard.css';
	import '$lib/styles/weekly-progress.css';

	let { children } = $props();
	let showSidebar = $derived(page.url.pathname !== '/');
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
	.feature-shell { display: flex; min-height: 100vh; }
	.feature-content { min-width: 0; flex: 1; }
	@media (max-width: 700px) { .feature-shell { display: block; } }
</style>
