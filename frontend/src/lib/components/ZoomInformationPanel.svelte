<script lang="ts">
 import { t as translate } from '$lib/i18n';
	import { onDestroy } from 'svelte';
	import type { ZoomResource } from '$lib/api';
	let { resources, canEdit = false, onedit }: { resources: ZoomResource[]; canEdit?: boolean; onedit: (resource: ZoomResource) => void } = $props();
	let feedback = $state<Record<number, string>>({});
	const timers = new Map<number, ReturnType<typeof setTimeout>>();
	onDestroy(() => timers.forEach(clearTimeout));
	async function copy(resource: ZoomResource) {
		clearTimeout(timers.get(resource.id));
		try {
			if (navigator.clipboard?.writeText) await navigator.clipboard.writeText(resource.meetingUrl);
			else {
				const focused = document.activeElement;
				const field = document.createElement('textarea');
				field.value = resource.meetingUrl;
				field.style.cssText = 'position:fixed;opacity:0;inset:0;pointer-events:none';
				document.body.append(field);
				try { field.select(); if (!document.execCommand('copy')) throw new Error('copy'); }
				finally { field.remove(); if (focused instanceof HTMLElement) focused.focus(); }
			}
			feedback[resource.id] = 'Tersalin!';
		} catch { feedback[resource.id] = 'Link tidak dapat disalin.'; }
		timers.set(resource.id, setTimeout(() => { delete feedback[resource.id]; timers.delete(resource.id); }, 2000));
	}
</script>

<aside class="zoom-info-panel" aria-labelledby="zoom-info-title">
	<header><p class="zoom-kicker">{$translate("RESOURCE")}</p><h2 id="zoom-info-title">{$translate("Zoom Information")}</h2></header>
	{#each resources as resource}
		<section class="zoom-info-item">
			<div class="zoom-info-heading"><div><h3>{resource.name}</h3><span>{resource.code}</span></div>{#if canEdit}<button type="button" onclick={() => onedit(resource)}>{$translate("Edit")}</button>{/if}</div>
			<dl>
				<div><dt>{$translate("Meeting ID")}</dt><dd>{resource.meetingId || $translate('Belum dikonfigurasi')}</dd></div>
				<div><dt>{$translate("Passcode")}</dt><dd>{resource.passcode || $translate('Belum dikonfigurasi')}</dd></div>
				<div><dt>{$translate("Host key")}</dt><dd>{resource.hostKey || $translate('Belum dikonfigurasi')}</dd></div>
				<div><dt>{$translate("Meeting URL")}</dt><dd>{#if resource.meetingUrl}<a href={resource.meetingUrl} target="_blank" rel="noreferrer">{resource.meetingUrl}</a><div class="copy-actions"><button type="button" aria-label={$translate('Salin link ') + resource.name} onclick={() => void copy(resource)}>{$translate("Salin")}</button><span role="status">{$translate(feedback[resource.id] || '')}</span></div>{:else}{$translate("Belum dikonfigurasi")}{/if}</dd></div>
				<div><dt>{$translate("Deskripsi")}</dt><dd>{resource.description || $translate('Belum dikonfigurasi')}</dd></div>
			</dl>
		</section>
	{/each}
</aside>

<style>
	dd { min-width: 0; overflow-wrap: anywhere; }
	.copy-actions { display: flex !important; flex-wrap: wrap; align-items: center; gap: 8px !important; margin-top: 6px; }
	.copy-actions button { min-height: 44px; padding: 6px 12px; border: 1px solid var(--his-border); border-radius: 6px; background: var(--his-primary-soft); color: var(--his-primary); font: inherit; }
	.copy-actions span { font-size: .75rem; }
</style>
