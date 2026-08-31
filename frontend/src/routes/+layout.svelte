<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import '$lib/styles/base.css';
	import '$lib/styles/brand-logo.css';
	import '$lib/styles/login.css';
	import '$lib/styles/dashboard.css';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{@render children()}
