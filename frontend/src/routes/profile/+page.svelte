<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		goto('/my-profile', { replaceState: true });
	});
</script>

<svelte:head>
	<title>Redirecting to My Profile... | my-ple</title>
</svelte:head>

<div style="padding: 40px; text-align: center; color: #64748b;">
	<p>Mengalihkan ke halaman My Profile...</p>
</div>
