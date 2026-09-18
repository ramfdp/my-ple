<script lang="ts">
 import { t as translate } from '$lib/i18n';
 import JobPICEditor from '$lib/components/JobPICEditor.svelte';
 import { onMount } from 'svelte';
 import {subscribeChanges} from '$lib/realtime';
 import { api, currentUser, type Job, type JobMasters } from '$lib/api';
 let allowed=$derived(!!$currentUser);
 let masters=$state<JobMasters>({divisions:[],roles:[],categories:[]});
 let divisionId=$state(0),roleId=$state<number|null>(null),categoryId=$state<number|null>(null);
 let jobs=$state<Job[]>([]),loading=$state(true),busy=$state(false),error=$state(''),success=$state(''),search=$state('');
 let showCategoryForm=$state(false),categoryName=$state('');
 let isCIHPK=$derived(masters.roles.find(r=>r.id===roleId)?.name==='CIHPK');
 let editing=$state<Job|null>(null),showForm=$state(false),name=$state(''),description=$state(''),active=$state(true),sortOrder=$state(0),deleting=$state<Job|null>(null);
 let roles=$derived(masters.roles.filter(r=>r.divisionId===divisionId));
 let categories=$derived(masters.categories.filter(c=>c.divisionId===divisionId && c.roleId===roleId));
 let visible=$derived(jobs.filter(j=>j.name.toLocaleLowerCase().includes(search.toLocaleLowerCase())));
 let contextLocked=$derived(busy || showCategoryForm || (showForm && !!editing?.referenced));
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
 onMount(()=>{if(allowed)void initialize();const stop=subscribeChanges(['jobs.changed'],async()=>{if(allowed&&divisionId){masters=await api<JobMasters>('/jobs/masters');await load(true)}});return()=>{request++;stop()}});
 $effect(()=>{divisionId;roleId;categoryId;if(divisionId && allowed)void load();});
 async function saveCategory(event:SubmitEvent){
  event.preventDefault();if(busy || isCIHPK)return;busy=true;error='';success='';
  try{
   const result=await api<{category:JobMasters['categories'][number]}>('/jobs/categories',{method:'POST',body:JSON.stringify({divisionId,roleId,name:categoryName.trim()})});
   masters={...masters,categories:[...masters.categories.filter(c=>c.id!==result.category.id),result.category]};
   categoryId=result.category.id;showCategoryForm=false;categoryName='';success='Kategori berhasil ditambahkan.';
  }catch(e){error=e instanceof Error?e.message:'Kategori tidak dapat disimpan.';}finally{busy=false;}
 }
 function openForm(job:Job|null){editing=job;name=job?.name ?? '';description=job?.description ?? '';active=job?.isActive ?? true;sortOrder=job?.sortOrder ?? jobs.length;showForm=true;error='';success='';}
 async function save(event:SubmitEvent){
  event.preventDefault();if(busy)return;busy=true;error='';success='';
  try{await api(editing?`/jobs/${editing.id}`:'/jobs',{method:editing?'PUT':'POST',body:JSON.stringify({divisionId,roleId,jobCategoryId:categoryId,name:name.trim(),description,isActive:active,sortOrder})});showForm=false;editing=null;await load();success='Pekerjaan berhasil disimpan.';}
  catch(e){error=e instanceof Error?e.message:'Pekerjaan tidak dapat disimpan.';}finally{busy=false;}
 }
 async function toggle(job:Job){
  busy=true;error='';success='';
  try{const response=await api<{job:Job}>(`/jobs/${job.id}/status`,{method:'PATCH',body:JSON.stringify({isActive:!job.isActive})});jobs=jobs.map(j=>j.id===job.id?response.job:j);success='Perubahan tersimpan.';}
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

<svelte:head><title>{$translate("Pengaturan Pekerjaan | MyHIS")}</title></svelte:head>
<main class="weekly-progress-page"><div class="weekly-progress-shell">
 <header class="weekly-header"><p class="weekly-kicker">{$translate("SETTINGS")}</p><h1>{$translate("Pengaturan Pekerjaan")}</h1><p>{$translate("Kelola pekerjaan per divisi, role, dan kategori.")}</p></header>
 {#if !allowed}<p role="alert">{$translate("Silakan masuk untuk mengelola pekerjaan.")}</p>{:else}
 <section class="task-list">
 <div class="report-form filters">
 <label>{$translate("Divisi")}<select bind:value={divisionId} disabled={contextLocked} onchange={resetRole}>{#each masters.divisions as d}<option value={d.id}>{d.name}</option>{/each}</select></label>
 {#if roles.length}<label>{$translate("Role")}<select bind:value={roleId} disabled={contextLocked} onchange={resetCategory}><option value={null}>{$translate("Tanpa role")}</option>{#each roles as r}<option value={r.id}>{$translate(r.name)}</option>{/each}</select></label>{/if}
 {#if categories.length}<label>{$translate("Kategori")}<select bind:value={categoryId} disabled={contextLocked}><option value={null}>{$translate("Tanpa kategori")}</option>{#each categories as c}<option value={c.id}>{c.name}</option>{/each}</select></label>{/if}
 <label>{$translate("Cari pekerjaan")}<input type="search" bind:value={search} placeholder={$translate("Nama pekerjaan")} /></label>
 </div>
 {#if error}<p role="alert">{$translate(error)} {#if !showForm && !deleting}<button type="button" onclick={()=>void (divisionId?load():initialize())}>{$translate("Coba lagi")}</button>{/if}</p>{/if}
 {#if success}<p role="status">{$translate(success)}</p>{/if}
 {#if showCategoryForm}
 <form class="report-form job-form" onsubmit={saveCategory}>
  <h2>{$translate("Tambah Kategori")}</h2>
  <label>{$translate("Nama kategori")}<input bind:value={categoryName} required maxlength="100" disabled={busy} /></label>
  <div class="actions"><button class="report-submit" disabled={busy}>{$translate("Simpan kategori")}</button><button class="drawer-cancel" type="button" disabled={busy} onclick={()=>showCategoryForm=false}>{$translate("Batal")}</button></div>
 </form>
 {:else if !showForm}
 <button class="report-button" type="button" disabled={busy || loading || !divisionId || isCIHPK} onclick={()=>{showCategoryForm=true;categoryName='';error='';success='';}}>{$translate("Tambah Kategori")}</button>
 {#if isCIHPK}<p>{$translate("CIHPK tetap memakai kategori MK/Supervisi.")}</p>{/if}
 {/if}
 {#if showForm}
 <form class="report-form job-form" onsubmit={save}>
  <h2>{editing?$translate('Edit pekerjaan'):$translate('Tambah pekerjaan')}</h2><p>{$translate("Divisi, role, dan kategori mengikuti pilihan di atas.")}{editing?.referenced?$translate(' Konteks pekerjaan terkunci karena sudah digunakan.'):''}</p>
  <label>{$translate("Nama pekerjaan")}<input bind:value={name} required maxlength="255" disabled={busy} /></label>
  <label>{$translate("Deskripsi")}<textarea bind:value={description} maxlength="2000" disabled={busy}></textarea></label>
  <label>{$translate("Urutan")}<input type="number" bind:value={sortOrder} min="0" step="1" required disabled={busy} /></label>
  <label>{$translate("Status")}<select bind:value={active} disabled={busy}><option value={true}>{$translate("Aktif")}</option><option value={false}>{$translate("Nonaktif")}</option></select></label>
  <div class="actions"><button class="report-submit" disabled={busy || !divisionId}>{busy?$translate('Menyimpan…'):$translate('Simpan pekerjaan')}</button><button class="drawer-cancel" type="button" disabled={busy} onclick={()=>{showForm=false;editing=null;}}>{$translate("Batal")}</button></div>
 </form>
 {:else}<button class="report-button" type="button" disabled={busy || loading || !divisionId || showCategoryForm} onclick={()=>openForm(null)}>{$translate("+ Tambah Pekerjaan")}</button>{/if}
 {#if loading}<p role="status">{$translate("Memuat pekerjaan…")}</p>{:else if !visible.length}<p role="status">{search?$translate('Tidak ada pekerjaan yang cocok.'):$translate('Belum ada pekerjaan pada kategori ini.')}</p>{:else}
 <div class="work-table-wrap"><table class="work-table"><thead><tr><th>{$translate("Pekerjaan")}</th><th>{$translate("Divisi")}</th><th>{$translate("Role")}</th><th>{$translate("Kategori")}</th><th>{$translate("Status")}</th><th>{$translate("Aksi")}</th></tr></thead><tbody>
 {#each visible as job (job.id)}<tr><td>{job.name}<JobPICEditor jobId={job.id} pics={job.pics ?? []} onchange={(pics)=>job.pics=pics} /></td><td>{job.division}</td><td>{job.role || '—'}</td><td>{job.category || '—'}</td><td><button type="button" role="switch" aria-checked={job.isActive} aria-label={`Status ${job.name}`} disabled={busy} onclick={()=>void toggle(job)}>{job.isActive?$translate('Aktif'):$translate('Nonaktif')}</button></td><td><div class="actions"><button class="edit-report" type="button" disabled={busy || showCategoryForm} onclick={()=>openForm(job)}>{$translate("Edit")}</button><button class="delete-report" type="button" disabled={busy} onclick={()=>{error='';deleting=job;}}>{$translate("Hapus")}</button></div></td></tr>{/each}
 </tbody></table></div>{/if}
 </section>{/if}
</div></main>
<dialog bind:this={dialog} class="report-delete-modal" oncancel={(event)=>{if(busy)event.preventDefault();else deleting=null;}} aria-labelledby="delete-job-title">
 <h2 id="delete-job-title">{$translate("Hapus “")}{deleting?.name}”?</h2><p>{$translate("Pekerjaan yang sudah digunakan pada laporan tidak dapat dihapus. Nonaktifkan pekerjaan tersebut sebagai gantinya.")}</p>
 {#if error}<p role="alert">{$translate(error)}</p>{/if}<div class="delete-modal-actions"><button class="delete-report-confirm" type="button" disabled={busy} onclick={()=>void remove()}>{busy?$translate('Menghapus…'):$translate('Hapus pekerjaan')}</button><button class="drawer-cancel" type="button" disabled={busy} onclick={()=>deleting=null}>{$translate("Batal")}</button></div>
</dialog>
<style>
 .filters{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,180px),1fr));gap:16px;margin-bottom:24px}.actions{display:flex;gap:8px;flex-wrap:wrap}.job-form{margin:24px 0;padding:20px;background:var(--his-surface,#f6f8fa);border-radius:12px}.work-table-wrap{margin-top:20px}td{overflow-wrap:anywhere}dialog{position:fixed;inset:0;margin:auto;transform:none;border:0;max-width:min(480px,calc(100vw - 32px));max-height:90dvh;overflow:auto}dialog::backdrop{background:#0006}
</style>
