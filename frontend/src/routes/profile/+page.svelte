<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		goto('/my-profile', { replaceState: true });
	});
</script>

<svelte:head>
	<title>Redirecting to My Profile... | MyHIS</title>
</svelte:head>

<div style="padding: 40px; text-align: center; color: var(--his-muted);">
	<p>Mengalihkan ke halaman My Profile...</p>
</div>
