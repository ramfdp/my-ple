<script lang="ts">
	import type { ZoomResource } from '$lib/api';
	let { resources, canEdit = false, onedit }: { resources: ZoomResource[]; canEdit?: boolean; onedit: (resource: ZoomResource) => void } = $props();
</script>

<aside class="zoom-info-panel" aria-labelledby="zoom-info-title">
	<header><p class="zoom-kicker">RESOURCE</p><h2 id="zoom-info-title">Zoom Information</h2></header>
	{#each resources as resource}
		<section class="zoom-info-item">
			<div class="zoom-info-heading"><div><h3>{resource.name}</h3><span>{resource.code}</span></div>{#if canEdit}<button type="button" onclick={() => onedit(resource)}>Edit</button>{/if}</div>
			<dl>
				<div><dt>Meeting ID</dt><dd>{resource.meetingId || 'Belum dikonfigurasi'}</dd></div>
				<div><dt>Meeting URL</dt><dd>{#if resource.meetingUrl}<a href={resource.meetingUrl} target="_blank" rel="noreferrer">Buka link Zoom</a>{:else}Belum dikonfigurasi{/if}</dd></div>
				<div><dt>Deskripsi</dt><dd>{resource.description || 'Belum dikonfigurasi'}</dd></div>
				<div><dt>Host key</dt><dd>{resource.hostKey || 'Belum dikonfigurasi'}</dd></div>
			</dl>
		</section>
	{/each}
</aside>
