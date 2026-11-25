<script lang="ts">
 import { desktopState, enableDesktopPush, disableDesktopPush } from '$lib/desktop-push';
 import { t } from '$lib/i18n';
 const labels = {
  loading:'Memeriksa notifikasi desktop…', on:'Nonaktifkan notifikasi desktop', off:'Aktifkan notifikasi desktop',
  blocked:'Notifikasi desktop diblokir. Ubah izin di pengaturan situs.',
  unsupported:'Browser ini tidak mendukung notifikasi desktop.',
  unconfigured:'Notifikasi desktop belum tersedia.', error:'Coba lagi mengaktifkan notifikasi desktop',
  disableError:'Coba lagi menonaktifkan notifikasi desktop'
 };
</script>

<div class="desktop-control" aria-live="polite">
 {#if $desktopState === 'on' || $desktopState === 'disableError'}
  <button type="button" onclick={disableDesktopPush}>{$t(labels[$desktopState])}</button>
 {:else if $desktopState === 'off' || $desktopState === 'error'}
  <button type="button" onclick={enableDesktopPush}>{$t(labels[$desktopState])}</button>
 {:else}
  <small>{$t(labels[$desktopState])}</small>
 {/if}
</div>

<style>
 .desktop-control { padding: .5rem 1rem; }
 button { font: inherit; cursor: pointer; }
</style>

