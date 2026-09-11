<script lang="ts">
 import JobPICEditor from '$lib/components/JobPICEditor.svelte';
 import { onMount } from 'svelte';
 import {subscribeChanges} from '$lib/realtime';
 import { api, currentUser, type Job, type JobMasters } from '$lib/api';
 let allowed=$derived(!!$currentUser);
 let masters=$state<JobMasters>({divisions:[],roles:[],categories:[]});
 let divisionId=$state(0),roleId=$state<number|null>(null),categoryId=$state<number|null>(null);
 let jobs=$state<Job[]>([]),loading=$state(true),busy=$state(false),error=$state(''),success=$state(''),search=$state('');
 let editing=$state<Job|null>(null),showForm=$state(false),name=$state(''),description=$state(''),active=$state(true),sortOrder=$state(0),deleting=$state<Job|null>(null);
 let roles=$derived(masters.roles.filter(r=>r.divisionId===divisionId));
 let categories=$derived(masters.categories.filter(c=>c.divisionId===divisionId && c.roleId===roleId));
 let visible=$derived(jobs.filter(j=>j.name.toLocaleLowerCase().includes(search.toLocaleLowerCase())));
 let contextLocked=$derived(busy || (showForm && !!editing?.referenced));
 function resetCategory(){categoryId=masters.categories.find(c=>c.divisionId===divisionId && c.roleId===roleId)?.id ?? null;}
 function resetRole(){roleId=masters.roles.find(r=>r.divisionId===divisionId)?.id ?? null;resetCategory();}
 let request=0;
 async function load(background=false){
  const version=++request; if(!background){loading=true;jobs=[]}error='';
  try{const response=await api<{jobs:Job[]}>(`/jobs?division_id=${divisionId}&role_id=${roleId ?? 0}&job_category_id=${categoryId ?? 0}`);if(version===request)jobs=response.jobs;}
  catch(e){if(version===request)error=e instanceof Error?e.message:'Pekerjaan tidak dapat dimuat.';}
  finally{if(version===request)loading=false;}
 }
 async function initialize(){
  try{masters=await api<JobMasters>('/jobs/masters');divisionId=masters.divisions[0]?.id ?? 0;resetRole();}
  catch(e){error=e instanceof Error?e.message:'Pilihan tidak dapat dimuat.';loading=false;}
 }
 onMount(()=>{if(allowed)void initialize();const stop=subscribeChanges(['jobs.changed'],()=>{if(allowed&&divisionId)return load(true)});return()=>{request++;stop()}});
 $effect(()=>{divisionId;roleId;categoryId;if(divisionId && allowed)void load();});
 function openForm(job:Job|null){editing=job;name=job?.name ?? '';description=job?.description ?? '';active=job?.isActive ?? true;sortOrder=job?.sortOrder ?? jobs.length;showForm=true;error='';success='';}
 async function save(event:SubmitEvent){
  event.preventDefault();if(busy)return;busy=true;error='';success='';
  try{await api(editing?`/jobs/${editing.id}`:'/jobs',{method:editing?'PUT':'POST',body:JSON.stringify({divisionId,roleId,jobCategoryId:categoryId,name:name.trim(),description,isActive:active,sortOrder})});showForm=false;editing=null;await load();success='Pekerjaan berhasil disimpan.';}
  catch(e){error=e instanceof Error?e.message:'Pekerjaan tidak dapat disimpan.';}finally{busy=false;}
 }
 async function toggle(job:Job){
  busy=true;error='';success='';
  try{const response=await api<{job:Job}>(`/jobs/${job.id}/status`,{method:'PATCH',body:JSON.stringify({isActive:!job.isActive})});jobs=jobs.map(j=>j.id===job.id?response.job:j);success=`${job.name}: ${response.job.isActive?'Aktif':'Nonaktif'}.`;}
  catch(e){error=e instanceof Error?e.message:'Status tidak dapat diubah.';}finally{busy=false;}
 }
 async function remove(){
  if(!deleting||busy)return;busy=true;error='';success='';
  try{await api(`/jobs/${deleting.id}`,{method:'DELETE'});deleting=null;await load();success='Pekerjaan berhasil dihapus.';}
  catch(e){error=e instanceof Error?e.message:'Pekerjaan tidak dapat dihapus.';}finally{busy=false;}
 }
 let dialog:HTMLDialogElement;
 $effect(()=>{if(deleting)dialog?.showModal();else dialog?.close();});
</script>

<svelte:head><title>Pengaturan Pekerjaan | MyHIS</title></svelte:head>
<main class="weekly-progress-page"><div class="weekly-progress-shell">
 <header class="weekly-header"><p class="weekly-kicker">SETTINGS</p><h1>Pengaturan Pekerjaan</h1><p>Kelola pekerjaan per divisi, role, dan kategori.</p></header>
 {#if !allowed}<p role="alert">Silakan masuk untuk mengelola pekerjaan.</p>{:else}
 <section class="task-list">
 <div class="report-form filters">
 <label>Divisi<select bind:value={divisionId} disabled={contextLocked} onchange={resetRole}>{#each masters.divisions as d}<option value={d.id}>{d.name}</option>{/each}</select></label>
 {#if roles.length}<label>Role<select bind:value={roleId} disabled={contextLocked} onchange={resetCategory}><option value={null}>Tanpa role</option>{#each roles as r}<option value={r.id}>{r.name}</option>{/each}</select></label>{/if}
 {#if categories.length}<label>Kategori<select bind:value={categoryId} disabled={contextLocked}><option value={null}>Tanpa kategori</option>{#each categories as c}<option value={c.id}>{c.name}</option>{/each}</select></label>{/if}
 <label>Cari pekerjaan<input type="search" bind:value={search} placeholder="Nama pekerjaan" /></label>
 </div>
 {#if error}<p role="alert">{error} {#if !showForm && !deleting}<button type="button" onclick={()=>void (divisionId?load():initialize())}>Coba lagi</button>{/if}</p>{/if}
 {#if success}<p role="status">{success}</p>{/if}
 {#if allowed}
 {#if showForm}
 <form class="report-form job-form" onsubmit={save}>
  <h2>{editing?'Edit pekerjaan':'Tambah pekerjaan'}</h2><p>Divisi, role, dan kategori mengikuti pilihan di atas.{editing?.referenced?' Konteks pekerjaan terkunci karena sudah digunakan.':''}</p>
  <label>Nama pekerjaan<input bind:value={name} required maxlength="255" disabled={busy} /></label>
  <label>Deskripsi<textarea bind:value={description} maxlength="2000" disabled={busy}></textarea></label>
  <label>Urutan<input type="number" bind:value={sortOrder} min="0" step="1" required disabled={busy} /></label>
  <label>Status<select bind:value={active} disabled={busy}><option value={true}>Aktif</option><option value={false}>Nonaktif</option></select></label>
  <div class="actions"><button class="report-submit" disabled={busy || !divisionId}>{busy?'Menyimpan…':'Simpan pekerjaan'}</button><button class="drawer-cancel" type="button" disabled={busy} onclick={()=>{showForm=false;editing=null;}}>Batal</button></div>
 </form>
 {:else}<button class="report-button" type="button" disabled={busy || loading || !divisionId} onclick={()=>openForm(null)}>+ Tambah Pekerjaan</button>{/if}
 {/if}
 {#if loading}<p role="status">Memuat pekerjaan…</p>{:else if !visible.length}<p role="status">{search?'Tidak ada pekerjaan yang cocok.':'Belum ada pekerjaan pada kategori ini.'}</p>{:else}
 <div class="work-table-wrap"><table class="work-table"><thead><tr><th>Pekerjaan</th><th>Divisi</th><th>Role</th><th>Kategori</th><th>Status</th>{#if allowed}<th>Aksi</th>{/if}</tr></thead><tbody>
 {#each visible as job (job.id)}<tr><td>{job.name}<JobPICEditor jobId={job.id} pics={job.pics ?? []} onchange={(pics)=>job.pics=pics} /></td><td>{job.division}</td><td>{job.role || '—'}</td><td>{job.category || '—'}</td><td>{#if allowed}<button type="button" role="switch" aria-checked={job.isActive} aria-label={`Status ${job.name}`} disabled={busy} onclick={()=>void toggle(job)}>{job.isActive?'Aktif':'Nonaktif'}</button>{:else}{job.isActive?'Aktif':'Nonaktif'}{/if}</td>{#if allowed}<td><div class="actions"><button class="edit-report" type="button" disabled={busy} onclick={()=>openForm(job)}>Edit</button><button class="delete-report" type="button" disabled={busy} onclick={()=>{error='';deleting=job;}}>Hapus</button></div></td>{/if}</tr>{/each}
 </tbody></table></div>{/if}
 </section>{/if}
</div></main>
<dialog bind:this={dialog} class="report-delete-modal" oncancel={(event)=>{if(busy)event.preventDefault();else deleting=null;}} aria-labelledby="delete-job-title">
 <h2 id="delete-job-title">Hapus “{deleting?.name}”?</h2><p>Pekerjaan yang sudah digunakan pada laporan tidak dapat dihapus. Nonaktifkan pekerjaan tersebut sebagai gantinya.</p>
 {#if error}<p role="alert">{error}</p>{/if}<div class="delete-modal-actions"><button class="delete-report-confirm" type="button" disabled={busy} onclick={()=>void remove()}>{busy?'Menghapus…':'Hapus pekerjaan'}</button><button class="drawer-cancel" type="button" disabled={busy} onclick={()=>deleting=null}>Batal</button></div>
</dialog>
<style>
 .filters{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,180px),1fr));gap:16px;margin-bottom:24px}.actions{display:flex;gap:8px;flex-wrap:wrap}.job-form{margin:24px 0;padding:20px;background:var(--his-surface,#f6f8fa);border-radius:12px}.work-table-wrap{margin-top:20px}td{overflow-wrap:anywhere}dialog{position:fixed;inset:0;margin:auto;transform:none;border:0;max-width:min(480px,calc(100vw - 32px));max-height:90dvh;overflow:auto}dialog::backdrop{background:#0006}
</style>
