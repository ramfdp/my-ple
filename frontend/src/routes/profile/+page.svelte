<script lang="ts">
 import { t as translate } from '$lib/i18n';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		goto('/my-profile', { replaceState: true });
	});
</script>

<svelte:head>
	<title>{$translate("Redirecting to My Profile... | MyHIS")}</title>
</svelte:head>

<div style="padding: 40px; text-align: center; color: var(--his-muted);">
	<p>{$translate("Mengalihkan ke halaman My Profile...")}</p>
</div>
