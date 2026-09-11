<script lang="ts">
 import { onMount } from 'svelte';
 import { api } from '$lib/api';
 import {subscribeChanges} from '$lib/realtime';
 import '$lib/styles/workspace.css';
 let workspaces=$state<{id:number;name:string;memberCount:number;activeTaskCount:number}[]>([]), loading=$state(true), error=$state('');
 async function load(){loading=true;error='';try{workspaces=(await api<{workspaces:typeof workspaces}>('/workspaces')).workspaces}catch(e){error=(e as Error).message}finally{loading=false}}
 onMount(()=>{void load();return subscribeChanges(['workspace.changed'],load)});
</script>
<svelte:head><title>Team Workspace | MyHIS</title></svelte:head>
<main class="workspace-page">
 <a class="back-link" href="/dashboard">← Kembali ke Dashboard</a>
 <header><p class="dashboard-kicker">TEAM WORKSPACE</p><h1>Pilih Workspace Divisi</h1><p>Catatan, timeline, dan koordinasi pekerjaan tim.</p></header>
 {#if loading}<p role="status">Memuat workspace…</p>{:else if error}<p role="alert">{error}</p><button onclick={load}>Coba lagi</button>
 {:else if !workspaces.length}<div class="ws-empty">Belum ada workspace yang dapat Anda akses. Hubungi administrator untuk memeriksa peran akun.</div>
 {:else}<div class="ws-directory">{#each workspaces as workspace}<a class="ws-panel" href={'/workspace/'+workspace.id}><h2>{workspace.name}</h2><p>{workspace.memberCount} anggota · {workspace.activeTaskCount} pekerjaan aktif</p><strong>Buka Workspace →</strong></a>{/each}</div>{/if}
</main>
