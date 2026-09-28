<script lang="ts">
	import { t as translate } from '$lib/i18n';
	import { onMount } from 'svelte';
	import type { ZoomResource, ZoomResourceInput } from '$lib/api';
	let { resource, busy = false, onclose, onsave }: { resource: ZoomResource; busy?: boolean; onclose: () => void; onsave: (input: ZoomResourceInput) => void } = $props();
	let meetingId = $state(''), hostKey = $state(''), passcode = $state(''), meetingUrl = $state(''), description = $state(''), error = $state('');
	onMount(() => { meetingId = resource.meetingId; hostKey = resource.hostKey; passcode = resource.passcode ?? ''; meetingUrl = resource.meetingUrl; description = resource.description; });
	function submit(event: SubmitEvent) {
		event.preventDefault();
		if (meetingUrl && !meetingUrl.startsWith('https://')) { error = 'Meeting URL harus menggunakan HTTPS.'; return; }
		error = ''; onsave({ meetingId: meetingId.trim(), hostKey: hostKey.trim(), passcode: passcode.trim(), meetingUrl: meetingUrl.trim(), description: description.trim() });
	}
</script>

<svelte:window onkeydown={(event) => event.key === 'Escape' && !busy && onclose()} />
<div class="zoom-modal-layer"><button class="zoom-modal-backdrop" type="button" aria-label={$translate("Tutup formulir")} onclick={onclose}></button><div class="zoom-modal" role="dialog" aria-modal="true" aria-labelledby="resource-form-title">
	<header><div><p class="zoom-kicker">{resource.code}</p><h2 id="resource-form-title">{$translate("Edit")} {resource.name}</h2></div><button type="button" class="zoom-close" aria-label={$translate("Tutup")} onclick={onclose}>×</button></header>
	<form onsubmit={submit}>
		<label>{$translate("Meeting ID")}<input bind:value={meetingId} maxlength="100" placeholder={$translate("Belum dikonfigurasi")} /></label>
		<label>{$translate("Passcode")}<input type="text" bind:value={passcode} maxlength="100" autocomplete="off" placeholder={$translate("Belum dikonfigurasi")} /></label>
		<label>{$translate("Host key")}<input type="text" bind:value={hostKey} maxlength="100" placeholder={$translate("Belum dikonfigurasi")} /></label>
		<label>{$translate("Meeting URL")}<input type="url" bind:value={meetingUrl} maxlength="2048" placeholder="https://zoom.us/j/..." /></label>
		<label>{$translate("Deskripsi")}<textarea bind:value={description} maxlength="1000" rows="4" placeholder={$translate("Informasi penggunaan resource")}></textarea></label>
		{#if error}<p class="zoom-form-error" role="alert">{$translate(error)}</p>{/if}
		<footer><button class="zoom-secondary" type="button" disabled={busy} onclick={onclose}>{$translate("Batal")}</button><button class="zoom-primary" type="submit" disabled={busy}>{busy ? $translate('Menyimpan…') : $translate('Simpan informasi')}</button></footer>
	</form>
</div></div>
