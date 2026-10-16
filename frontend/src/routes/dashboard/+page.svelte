<script lang="ts">
 import FeedbackButton from '$lib/components/FeedbackButton.svelte';
 import HermesChat from '$lib/components/HermesChat.svelte';
 import { currentUser } from '$lib/api';
 import { visibleFeatures } from '$lib/navigation';
 import NotificationBell from '$lib/components/NotificationBell.svelte';
 import LanguageSelector from '$lib/components/LanguageSelector.svelte';
 import { t as translate } from '$lib/i18n';
 const items=$derived(visibleFeatures($currentUser?.role));
</script>
<svelte:head><title>{$translate('Dashboard')} {$translate("| MyHIS")}</title><meta name="description" content={$translate('Dashboard kerja MyHIS.')}/></svelte:head>
<main class="dashboard-page">
 <header class="dashboard-header">
  <div><p class="dashboard-kicker">{$translate('Dashboard')}</p><h1>{$translate('Halo,')} <span>{$currentUser?.name ?? $translate('Nama Pengguna')}</span></h1><p class="dashboard-subtitle">{$translate('Pilih layanan untuk melanjutkan pekerjaan hari ini.')}</p></div>
  <div class="dashboard-actions"><LanguageSelector/><NotificationBell/></div>
 </header>
 <section class="dashboard-menu" aria-labelledby="menu-title">
  <div class="menu-heading"><div><p class="dashboard-kicker">{$translate('LAYANAN')}</p><h2 id="menu-title">{$translate('Menu utama')}</h2></div><p>{items.length} {$translate('layanan tersedia')}</p></div>
  <div class="menu-grid">{#each items as item (item.href)}
   <a class="menu-card" class:zoom-card={item.href==='/booking-zoom'} class:attendance-card={item.icon==='calendar'||item.icon==='clock'} class:progress-card={item.icon!=='calendar'&&item.icon!=='clock'} href={item.href}>
    <div class="menu-icon" aria-hidden="true"><svg viewBox="0 0 24 24">{#if item.icon==='users'||item.icon==='user'}<circle cx="12" cy="7" r="4"/><path d="M5 21v-2a7 7 0 0 1 14 0v2"/>{:else if item.icon==='calendar'||item.icon==='clock'}<rect x="4" y="5" width="16" height="15" rx="2"/><path d="M8 3v4M16 3v4M4 10h16M8 14h3M8 17h6"/>{:else if item.icon==='settings'}<circle cx="12" cy="12" r="4"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M5 19l2-2M17 7l2-2"/>{:else}<rect x="4" y="3" width="16" height="18" rx="2"/><path d="M8 8h8M8 12h8M8 16h5"/>{/if}</svg></div>
    <h3>{$translate(item.key)}</h3><p>{$translate(item.description)}</p>
   </a>
  {/each}</div>
 </section>
</main>
<FeedbackButton/>
<HermesChat/>
<style>.dashboard-page{padding-bottom:96px}.dashboard-actions{display:flex;align-items:center;gap:12px;flex-wrap:wrap}</style>
