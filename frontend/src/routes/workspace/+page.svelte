<script lang="ts">
 import { t as translate } from '$lib/i18n';
 import { onMount } from 'svelte';
 import { api } from '$lib/api';
 import {subscribeChanges} from '$lib/realtime';
 import '$lib/styles/workspace.css';
 let workspaces=$state<{id:number;name:string;memberCount:number;activeTaskCount:number}[]>([]), loading=$state(true), error=$state('');
 async function load(background=false){if(!background)loading=true;error='';try{workspaces=(await api<{workspaces:typeof workspaces}>('/workspaces')).workspaces}catch(e){error=(e as Error).message}finally{loading=false}}
 onMount(()=>{void load();return subscribeChanges(['workspace.changed'],()=>load(true))});
</script>
<svelte:head><title>{$translate("Team Workspace | MyHIS")}</title></svelte:head>
<main class="workspace-page">
 <a class="back-link" href="/dashboard">{$translate("← Kembali ke Dashboard")}</a>
 <header><p class="dashboard-kicker">{$translate("TEAM WORKSPACE")}</p><h1>{$translate("Pilih Workspace Divisi")}</h1><p>{$translate("Catatan, timeline, dan koordinasi pekerjaan tim.")}</p></header>
 {#if error && workspaces.length}<p role="alert">{$translate(error)}</p>{/if}
 {#if loading}<p role="status">{$translate("Memuat workspace…")}</p>{:else if error && !workspaces.length}<p role="alert">{$translate(error)}</p><button onclick={() => load()}>{$translate("Coba lagi")}</button>
 {:else if !workspaces.length}<div class="ws-empty">{$translate("Belum ada workspace yang dapat Anda akses. Hubungi administrator untuk memeriksa peran akun.")}</div>
 {:else}<div class="ws-directory">{#each workspaces as workspace}<a class="ws-panel" href={'/workspace/'+workspace.id}><h2>{workspace.name}</h2><p>{workspace.memberCount} {$translate("anggota ·")} {workspace.activeTaskCount} {$translate("pekerjaan aktif")}</p><strong>{$translate("Buka Workspace →")}</strong></a>{/each}</div>{/if}
</main>
