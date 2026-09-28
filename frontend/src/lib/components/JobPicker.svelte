<script lang="ts">
 import { t as translate } from '$lib/i18n';
 import { onMount } from 'svelte';
 import { subscribeChanges } from '$lib/realtime';
 import { api, type Job, type JobMasters, type Report } from '$lib/api';
 let { division = $bindable(''), selected = $bindable<Job | null>(null), busy = $bindable(true), existing = null, fixedDivision = false }: { division?: string; selected?: Job | null; busy?: boolean; existing?: Report | null; fixedDivision?: boolean } = $props();
 let masters = $state<JobMasters>({divisions:[],roles:[],categories:[]});
 let divisionId = $derived(masters.divisions.find(d => d.name === division)?.id);
 let roleId = $state<number | null>(null), categoryId = $state<number | null>(null);
 let jobs = $state<Job[]>([]), error = $state(''), loaded = $state(false);
 let roles = $derived(masters.roles.filter(r => r.divisionId === divisionId));
 let selectedRole = $derived(roles.find(r => r.id === roleId));
 let categories = $derived(selectedRole?.name === 'CIHPK' ? [] : masters.categories.filter(c => c.divisionId === divisionId && c.roleId === roleId));
 function resetRole() {
  roleId = masters.roles.find(r => r.divisionId === divisionId && r.name === existing?.role)?.id ?? roles[0]?.id ?? null;
  resetCategory();
 }
 function resetCategory() {
	if(selectedRole?.name==='CIHPK'){categoryId=null;return;}
  const available = masters.categories.filter(c => c.divisionId === divisionId && c.roleId === roleId);
  categoryId = available.find(c => c.name === existing?.workDivision)?.id ?? available[0]?.id ?? null;
 }
 let masterRequest = 0;
 async function loadMasters(preserve = false) {
  const request = ++masterRequest;
  error=''; if(!preserve)busy=true;
  try { const result=await api<JobMasters>('/jobs/masters'); if(request!==masterRequest)return; masters=result; if (!masters.divisions.some(d=>d.name===division)) division=masters.divisions[0]?.name ?? ''; if(!preserve)resetRole(); loaded=true; }
  catch(e){ if(request===masterRequest){busy=false; error=e instanceof Error?e.message:'Pilihan tidak dapat dimuat.';} }
 }
 onMount(()=>{void loadMasters();const stop=subscribeChanges(['jobs.changed'],()=>loadMasters(loaded));return()=>{masterRequest++;stop();};});
 let lastFilters = '';
 $effect(()=>{
  masters;
  if(!loaded)return;
  if(!divisionId){busy=false;selected=null;jobs=[];return;}
  const d=divisionId,r=roleId,c=categoryId;
  const key=JSON.stringify([d,r,c]), sameFilters=key===lastFilters;
  lastFilters=key;
  let cancelled=false;
  busy=!sameFilters; if(!sameFilters){selected=null;jobs=[];} error='';
  const categoryFilter=selectedRole?.name==='CIHPK'?'':`&job_category_id=${c ?? 0}`;
  api<{jobs:Job[]}>(`/jobs?division_id=${d}&role_id=${r ?? 0}${categoryFilter}&active=true`).then(response=>{
   if(cancelled)return;
   jobs=response.jobs;
   const previous=sameFilters?selected:null;
   if(previous?.isActive){selected=jobs.find(j=>j.id===previous.id)??null;return;}
   const roleName=masters.roles.find(x=>x.id===r)?.name ?? '';
   const categoryName=masters.categories.find(x=>x.id===c)?.name ?? '';
   if(existing && division===existing.primaryDivision && roleName===existing.role && (roleName==='CIHPK'||categoryName===existing.workDivision)){
    const current=jobs.find(j=>existing.jobId ? j.id===existing.jobId : j.name===existing.work);
    if(current) selected=current;
    else {
     const historical:Job={id:existing.jobId ?? 0,divisionId:d,roleId:r,jobCategoryId:c,name:existing.work || existing.workDivision,description:'',isActive:false,sortOrder:0,division,role:existing.role,category:existing.workDivision,referenced:true};
     jobs=[historical,...jobs]; selected=historical;
    }
   } else if(jobs.length===1) selected=jobs[0];
  }).catch(e=>{if(!cancelled)error=e instanceof Error?e.message:'Pekerjaan tidak dapat dimuat.';}).finally(()=>{if(!cancelled)busy=false;});
  return ()=>{cancelled=true;};
 });
</script>

<label>{$translate("Divisi")}<select bind:value={division} disabled={fixedDivision || !loaded} onchange={resetRole}>{#each masters.divisions as d}<option value={d.name}>{d.name}</option>{/each}</select></label>
{#if roles.length}<label>{$translate("Role")}<select bind:value={roleId} onchange={resetCategory}><option value={null}>{$translate("Tanpa role")}</option>{#each roles as r}<option value={r.id}>{$translate(r.name)}</option>{/each}</select></label>{/if}
{#if selectedRole?.name === 'CIHPK'}<label>{$translate("Kategori")}<input value="MK/Supervisi" readonly /></label>{/if}
{#if categories.length}<label>{$translate("Kategori")}<select bind:value={categoryId}><option value={null}>{$translate("Tanpa kategori")}</option>{#each categories as c}<option value={c.id}>{c.name}</option>{/each}</select></label>{/if}
<label>{$translate("Pekerjaan")}<select value={selected?.id ?? ''} disabled={busy} required onchange={(e)=>selected=jobs.find(j=>j.id===Number(e.currentTarget.value)) ?? null}><option value="" disabled>{$translate("Pilih pekerjaan")}</option>{#each jobs as job}<option value={job.id}>{job.name}{!job.isActive ? $translate(' (tersimpan — nonaktif/legacy)') : ''}</option>{/each}</select></label>
{#if busy && !error}<p role="status">{$translate("Memuat pekerjaan…")}</p>{:else if error}<p role="alert">{$translate(error)} <button type="button" onclick={()=>{loaded=false;void loadMasters();}}>{$translate("Coba lagi")}</button></p>{:else if !jobs.length}<p role="status">{$translate("Belum ada pekerjaan pada kategori ini.")}</p>{/if}
