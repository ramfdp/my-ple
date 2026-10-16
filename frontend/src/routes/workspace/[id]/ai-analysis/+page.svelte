<script lang="ts">
 import {page} from '$app/state';
 import {afterNavigate} from '$app/navigation';
 import {api,APIError} from '$lib/api';
 import {isOneDriveLink} from '$lib/ai-planner';

 type Item={taskId:string;title:string;relatedUrl:string;summary:string;nextSteps:string[];decision:string;reviewerName?:string;version:number};
 type Batch={id:string;planDate:string;items:Item[];stale:boolean;finalized?:boolean;planCreated?:boolean};
 let batch=$state<Batch|null>(null),loading=$state(true),saving=$state(''),error=$state(''),notice=$state('');
 const wid=$derived(Number(page.params.id));
 async function load():Promise<boolean>{loading=true;error='';try{batch=(await api<{batch:Batch|null}>(`/ai-analyses/workspaces/${wid}`)).batch;return true}catch(e){batch=null;error=e instanceof APIError&&e.status===403?'Akses workspace ditolak.':e instanceof Error?e.message:'Gagal memuat analisis.';return false}finally{loading=false}}
 async function review(item:Item,decision:'APPROVED'|'REJECTED'){
  if(!batch)return;
  saving=item.taskId;error='';notice='';
  try{await api(`/ai-analyses/workspaces/${wid}/${batch.id}/tasks/${item.taskId}/review`,{method:'PATCH',body:JSON.stringify({decision,version:item.version})});if(await load())notice='Tinjauan tersimpan.';else error='Tinjauan tersimpan, tetapi gagal memuat data terbaru. Muat ulang.'}
  catch(e){if(e instanceof APIError&&e.status===409)await load();error=e instanceof APIError&&e.status===409?'Analisis atau pekerjaan berubah. Muat ulang.':e instanceof Error?e.message:'Gagal menyimpan tinjauan.'}
  finally{saving=''}
 }
 afterNavigate(()=>{void load()});
</script>

<svelte:head><title>Tinjauan analisis pekerjaan | My-PLE</title></svelte:head>
<main class="analysis">
 <a href={`/workspace/${wid}`}>Kembali ke Workspace</a>
 <h1>Tinjauan analisis pekerjaan</h1>
 <p>Setujui atau tolak setiap analisis. Rencana harian dibuat setelah semua analisis ditinjau.</p>
 {#if loading}<p role="status">Memuat analisis…</p>{/if}
 {#if error}<p role="alert">{error} <button onclick={()=>void load()}>Muat ulang</button></p>{/if}
 {#if notice}<p role="status">{notice}</p>{/if}
 {#if !loading&&batch}
  <h2>Rencana untuk {batch.planDate}</h2>
  {#if batch.stale}<p role="alert">Sumber pekerjaan berubah. Analisis ini hanya arsip; tinjauan dinonaktifkan.</p>{/if}
  {#each batch.items as item (item.taskId)}
   <article>
    <h3>{item.title||item.taskId}</h3>
    {#if !batch.stale&&isOneDriveLink(item.relatedUrl)}<p><a href={item.relatedUrl} target="_blank" rel="noopener noreferrer">Buka dokumen sumber</a></p>{/if}
    {#if item.decision==='UNAVAILABLE'}<p>Dokumen atau bukti tidak dapat dibaca. Pekerjaan ini dikecualikan.</p>
    {:else}
     <p>{item.summary}</p>
     <ol>{#each item.nextSteps as step}<li>{step}</li>{/each}</ol>
     <p>Status: {item.decision==='PENDING'?'Menunggu tinjauan':item.decision==='APPROVED'?'Disetujui':'Ditolak'}{item.reviewerName?` oleh ${item.reviewerName}`:''}</p>
     {#if item.decision==='PENDING'&&!batch.finalized&&!batch.stale}<div class="actions"><button disabled={saving!==''} onclick={()=>void review(item,'APPROVED')}>Setujui</button><button disabled={saving!==''} onclick={()=>void review(item,'REJECTED')}>Tolak</button></div>{/if}
    {/if}
   </article>
  {:else}<p>Tidak ada pekerjaan untuk dianalisis.</p>{/each}
  {#if batch.finalized}<p role="status">{batch.planCreated?'Analisis selesai. Lihat rencana pada AI Daily Job Planner.':'Tidak ada analisis disetujui; tidak ada rencana dibuat.'}</p>{:else if batch.items.every(item=>item.decision!=='PENDING')}<p role="status">Semua analisis ditinjau. Rencana dibuat oleh Worker pada pemeriksaan berikutnya.</p>{/if}
 {:else if !loading&&!error}<p>Belum ada analisis pekerjaan.</p>{/if}
</main>
<style>
 .analysis{max-width:900px;margin:auto;padding:1.5rem;color:var(--text-primary,#17243a)}article{border:1px solid #d7e0eb;border-radius:.8rem;padding:1rem;margin:1rem 0;background:white;overflow-wrap:anywhere}.actions{display:flex;gap:.7rem}button{font:inherit;padding:.55rem .8rem;cursor:pointer}button:disabled{opacity:.6;cursor:not-allowed}a{color:#174eab}
</style>
