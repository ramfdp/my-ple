<script lang="ts">
 import { onMount } from 'svelte';
 import { api, currentUser, type Job } from '$lib/api';
 import { canUseDocumentChecklist } from '$lib/document-checklist';
 import ChecklistTable from '$lib/components/ChecklistTable.svelte';
 import '$lib/styles/workspace.css';
 const authorized = $derived(canUseDocumentChecklist($currentUser?.role));
 let jobs = $state<Job[]>([]), loading = $state(true), error = $state('');
 let expanded = $state<Record<number, boolean>>({});
 async function load() {
  if (!authorized) { loading = false; return; }
  loading = true; error = '';
  try { jobs = (await api<{jobs: Job[]}>('/document-checklists')).jobs; }
  catch (e) { error = (e as Error).message; }
  finally { loading = false; }
 }
 onMount(load);
</script>
<svelte:head><title>Checklist Dokumen Pekerjaan | MyHIS</title></svelte:head>
<main class="workspace-page">
 <header><p class="dashboard-kicker">DOKUMEN PEKERJAAN</p><h1>Checklist Dokumen Pekerjaan</h1><p>Pantau dokumen dan kelengkapan tanda tangan setiap pekerjaan.</p></header>
 {#if !authorized}<p role="alert">Akses ditolak. Fitur ini tersedia untuk Project Engineer, Keuangan, Admin, dan Management.</p>
 {:else if loading}<p role="status">Memuat pekerjaan...</p>
 {:else if error}<p role="alert">{error}</p><button onclick={load}>Coba lagi</button>
 {:else if !jobs.length}<p>Belum ada pekerjaan.</p>
 {:else}
  {#each jobs as job (job.id)}
   <section class="ws-panel job">
    <button class="job-heading" aria-expanded={!!expanded[job.id]} aria-controls={'checklist-'+job.id} onclick={() => expanded[job.id] = !expanded[job.id]}>
     <span><strong>{job.name}</strong><small>{[job.division, job.role, job.category].filter(Boolean).join(' / ')}{job.isActive ? '' : ' · Nonaktif'}</small></span><span aria-hidden="true">{expanded[job.id] ? '−' : '+'}</span>
    </button>
    <div id={'checklist-'+job.id} hidden={!expanded[job.id]}>{#if expanded[job.id]}<ChecklistTable {job}/>{/if}</div>
   </section>
  {/each}
 {/if}
</main>
<style>
 .job{margin-top:1rem;padding:0;min-width:0}.job-heading{display:flex;justify-content:space-between;align-items:center;gap:1rem;width:100%;text-align:left;padding:1rem;border:0;background:transparent}.job-heading small{display:block;font-weight:400;color:var(--his-muted);margin-top:.3rem}.job-heading strong{font-size:1rem}
</style>
