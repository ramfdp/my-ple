<script lang="ts">
 import { onMount } from 'svelte';
 import { api, APIError, currentUser, type Job } from '$lib/api';
 import { locale, t as translate } from '$lib/i18n';
 import { billingStatuses, documentSuggestions, canUseBilling, type BillingStage, type BillingConfiguration, type BillingDocument, type BillingMilestone } from '$lib/billing-documents';
 import '$lib/styles/workspace.css';
 const authorized = $derived(canUseBilling($currentUser?.role));
 let stages=$state<BillingStage[]>([]), jobs=$state<Job[]>([]), loading=$state(true), busy=$state(false), error=$state(''), notice=$state('');
 let jobFilter=$state(''), statusFilter=$state(''), search=$state('');
 let dialog:HTMLDialogElement;
 let mode=$state<'configuration'|'status'|'notes'>('configuration'), stageId=$state(0), version=$state(0), documentId=$state(0), milestoneId=$state(0);
 let draftStatus=$state<string>('BELUM ADA'), draftNotes=$state(''), targetName=$state(''), modalError=$state(''), conflict=$state(false), confirming=$state(false);
 let config=$state<BillingConfiguration>({jobId:0,name:'',periodLabel:'',version:0,milestones:[]});
 let original=$state<BillingConfiguration|null>(null);
 const jobName=(id:number)=>jobs.find(j=>j.id===id)?.name??'';
 const matched=$derived(stages.filter(s=>(!jobFilter||s.jobId===Number(jobFilter))&&[jobName(s.jobId),s.name,s.periodLabel,...s.milestones.flatMap(m=>[m.name,...m.documents.map(d=>d.name)])].join(' ').toLocaleLowerCase().includes(search.trim().toLocaleLowerCase())));
 const visible=$derived(matched.filter(s=>!statusFilter||s.readiness===statusFilter));
 const ready=$derived(matched.filter(s=>s.readiness==='SIAP').length);
 const latest=$derived(stages.find(s=>s.id===stageId));
 const changes=$derived.by(()=>{
  const old=original?.milestones??[];
  return [
   ...old.filter(m=>!config.milestones.some(n=>n.id===m.id)).map(m=>'- '+m.name),
   ...old.flatMap(m=>m.documents.filter(d=>!config.milestones.some(n=>n.id===m.id&&n.documents.some(x=>x.id===d.id))).map(d=>'- '+m.name+' / '+d.name)),
   ...config.milestones.filter(m=>m.id===0).map(m=>'+ '+m.name),
   ...config.milestones.flatMap(m=>m.documents.filter(d=>d.id===0).map(d=>'+ '+m.name+' / '+d.name))
  ];
 });
 const validConfig=$derived(config.jobId>0&&!!config.name.trim()&&!!config.periodLabel.trim()&&config.milestones.length>0&&config.milestones.every(m=>m.name.trim()&&m.documents.length>0&&m.documents.every(d=>d.name.trim())&&new Set(m.documents.map(d=>d.name.trim().toLocaleLowerCase())).size===m.documents.length));
 const date=(value:string)=>new Intl.DateTimeFormat($locale,{timeZone:'Asia/Jakarta',dateStyle:'medium',timeStyle:'short'}).format(new Date(value));
 async function load(){
  loading=true;error='';
  try{const result=await api<{jobs:Job[];stages:BillingStage[]}>('/billing-documents');jobs=result.jobs;stages=result.stages;return true}
  catch(e){error=(e as Error).message;return false}finally{loading=false}
 }
 onMount(()=>{if(authorized)void load();else loading=false});
 function open(s:BillingStage|null,next:typeof mode){stageId=s?.id??0;version=s?.version??0;mode=next;modalError='';conflict=false;confirming=false;notice='';dialog.showModal()}
 function configure(s:BillingStage|null){
  config=s?{jobId:s.jobId,name:s.name,periodLabel:s.periodLabel,version:s.version,milestones:s.milestones.map(m=>({id:m.id,name:m.name,documents:m.documents.map(d=>({id:d.id,name:d.name}))}))}:{jobId:Number(jobFilter)||0,name:'',periodLabel:'',version:0,milestones:[{id:0,name:'',documents:[{id:0,name:''}]}]};
  original=s?structuredClone($state.snapshot(config)):null;open(s,'configuration');
 }
 function editStatus(s:BillingStage,d:BillingDocument){documentId=d.id;milestoneId=0;targetName=d.name;draftStatus=d.status;open(s,'status')}
 function editNotes(s:BillingStage,m?:BillingMilestone){documentId=0;milestoneId=m?.id??0;targetName=m?.name??s.name;draftNotes=m?.notes??s.notes;open(s,'notes')}
 async function save(e:SubmitEvent){
  e.preventDefault();if(busy||conflict)return;
  if(mode==='configuration'&&!validConfig)return;
  if(mode==='configuration'&&stageId&&changes.length&&!confirming){confirming=true;return}
  busy=true;modalError='';
  const path='/billing-documents/stages'+(stageId?'/'+stageId:'')+(mode==='configuration'&&stageId?'/configuration':'');
  const body=mode==='configuration'?{...config,version}:mode==='status'?{version,documentId,status:draftStatus}:{version,milestoneId,notes:draftNotes};
  try{
   const {stage}=await api<{stage:BillingStage}>(path,{method:mode==='configuration'?(stageId?'PUT':'POST'):'PATCH',body:JSON.stringify(body)});
   stages=[...stages.filter(s=>s.id!==stage.id),stage].sort((a,b)=>a.id-b.id);dialog.close();notice='Perubahan tersimpan.';
  }catch(e){modalError=(e as Error).message;if(e instanceof APIError&&e.status===409){conflict=true;await load()}}
  finally{busy=false}
 }
</script>

<svelte:head><title>{$translate('Dokumen Penagihan')}</title></svelte:head>
<main class="workspace-page billing-page">
 <header class="ws-row"><div><h1>{$translate('Dokumen Penagihan')}</h1><p>{$translate('Rekap kesiapan dokumen penagihan per pekerjaan.')}</p></div>{#if authorized}<button class="ws-primary" disabled={loading||busy||!jobs.some(j=>j.isActive)} onclick={()=>configure(null)}>{$translate('Tambah tahap penagihan')}</button>{/if}</header>
 {#if !authorized}<p role="alert">{$translate('Akses penagihan ditolak.')}</p>{:else}
 {#if error}<p role="alert">{$translate(error)}</p><button onclick={load}>{$translate('Muat ulang')}</button>{/if}
 {#if notice}<p role="status">{$translate(notice)}</p>{/if}
 <div class="ws-filters">
  <label>{$translate('Pekerjaan')}<select aria-label={$translate('Pekerjaan')} bind:value={jobFilter}><option value="">{$translate('Semua pekerjaan')}</option>{#each jobs as j}<option value={String(j.id)}>{[j.name,j.division,j.role,j.category].filter(Boolean).join(' / ')}{j.isActive?'':' ('+$translate('Nonaktif')+')'}</option>{/each}</select></label>
  <label>{$translate('Kesiapan tahap')}<select aria-label={$translate('Kesiapan tahap')} bind:value={statusFilter}><option value="">{$translate('Semua status')}</option><option value="SIAP">{$translate('SIAP')}</option><option value="BELUM SIAP">{$translate('BELUM SIAP')}</option></select></label>
  <label>{$translate('Cari penagihan')}<input type="search" bind:value={search}/></label>
 </div>
 <div class="ws-metrics"><section class="ws-panel"><span>{$translate('Tahap SIAP')}</span><strong>{ready}</strong></section><section class="ws-panel"><span>{$translate('Tahap BELUM SIAP')}</span><strong>{matched.length-ready}</strong></section></div>
 {#if loading}<p role="status">{$translate('Memuat penagihan...')}</p>{:else}
 {#each jobs.filter(j=>visible.some(s=>s.jobId===j.id)) as job (job.id)}
 <section class="billing-job"><h2>{job.name}</h2><p>{[job.division,job.role,job.category].filter(Boolean).join(' / ')}</p>
 {#each visible.filter(s=>s.jobId===job.id) as stage (stage.id)}
 <article class="ws-panel stage">
  <div class="ws-row"><h3>{stage.name} <small>{stage.periodLabel}</small></h3><span class:ready={stage.readiness==='SIAP'} class="readiness">{$translate(stage.readiness)}</span><button onclick={()=>configure(stage)}>{$translate('Edit requirement')}</button></div>
  <p class="timestamp">{$translate('Diperbarui (WIB)')}: {date(stage.updatedAt)}</p>
  <p class="notes">{stage.notes||$translate('Belum ada catatan.')}</p><button onclick={()=>editNotes(stage)}>{$translate('Catatan tahap')}</button>
  {#each stage.milestones as milestone (milestone.id)}
  <section class="milestone">
   <div class="ws-row"><h4>{milestone.name}</h4><span class:ready={milestone.readiness==='SIAP'} class="readiness">{$translate(milestone.readiness)}</span></div>
   <p class="timestamp">{$translate('Diperbarui (WIB)')}: {date(milestone.updatedAt)}</p>
   {#if !milestone.documents.length}<p>{$translate('Requirement belum dikonfigurasi.')}</p>{/if}
   <ul class="documents">{#each milestone.documents as doc (doc.id)}<li><div><strong>{doc.name}</strong><small>{date(doc.updatedAt)} {$translate('WIB')}</small></div><span class="document-status">{$translate(doc.status)}</span><button aria-label={$translate('Ubah status')+' '+doc.name} onclick={()=>editStatus(stage,doc)}>{$translate('Ubah status')}</button></li>{/each}</ul>
   <p class="notes">{milestone.notes||$translate('Belum ada catatan.')}</p><button onclick={()=>editNotes(stage,milestone)}>{$translate('Catatan milestone')}</button>
  </section>
  {/each}
  {#if !stage.milestones.length}<p>{$translate('Requirement belum dikonfigurasi.')}</p>{/if}
 </article>{/each}</section>
 {:else}<p class="ws-empty">{$translate('Belum ada tahap penagihan yang cocok.')}</p>{/each}
 {/if}{/if}
</main>

<dialog bind:this={dialog} class="ws-dialog billing-dialog" oncancel={e=>{if(busy)e.preventDefault()}} aria-labelledby="billing-editor-title">
 <form onsubmit={save}>
  <h2 id="billing-editor-title">{$translate(mode==='configuration'?(stageId?'Edit requirement':'Tambah tahap penagihan'):mode==='status'?'Ubah status':'Edit catatan')}</h2>
  {#if modalError}<p role="alert">{$translate(modalError)}</p>{/if}
  {#if conflict}<section class="conflict"><p>{$translate('Draf dipertahankan. Bandingkan dengan data terbaru sebelum mencoba lagi.')}</p>
   {#if latest}<h3>{latest.name} — {latest.periodLabel}</h3><p>{latest.notes}</p>{#each latest.milestones as m}<h4>{m.name} — {$translate(m.readiness)}</h4><p>{m.notes}</p><ul>{#each m.documents as d}<li>{d.name}: {$translate(d.status)}</li>{/each}</ul>{/each}
   <button type="button" disabled={busy||latest.version<=version} onclick={()=>{if(mode==='configuration')original={jobId:latest!.jobId,name:latest!.name,periodLabel:latest!.periodLabel,version:latest!.version,milestones:latest!.milestones.map(m=>({id:m.id,name:m.name,documents:m.documents.map(d=>({id:d.id,name:d.name}))}))};version=latest!.version;conflict=false;confirming=false;modalError=''}}>{$translate('Gunakan versi terbaru untuk draf ini')}</button>{/if}
  </section>{/if}
  <fieldset disabled={busy}>
  {#if mode==='configuration'}
   <label>{$translate('Pekerjaan')}<select aria-label={$translate('Pekerjaan')} required disabled={!!stageId} bind:value={config.jobId}><option value={0}>{$translate('Pilih pekerjaan')}</option>{#each jobs.filter(j=>j.isActive||j.id===config.jobId) as j}<option value={j.id}>{[j.name,j.division,j.role,j.category].filter(Boolean).join(' / ')}</option>{/each}</select></label>
   <label>{$translate('Nama tahap')}<input required maxlength="255" bind:value={config.name}/></label>
   <label>{$translate('Periode penagihan')}<input required maxlength="255" bind:value={config.periodLabel}/></label>
   <p>{$translate('Pilih hanya dokumen wajib. Tanda / tidak dihitung sebagai alternatif otomatis.')}</p>
   {#each config.milestones as milestone, i}
   <fieldset class="requirement">
    <legend>{$translate('Milestone')} {i+1}</legend>
    <label>{$translate('Nama milestone')}<input required maxlength="255" bind:value={milestone.name}/></label>
    {#each milestone.documents as doc, j}<div class="requirement-row"><label>{$translate('Dokumen wajib')} {j+1}<input required maxlength="255" list="billing-document-names" bind:value={doc.name}/></label><button type="button" onclick={()=>{milestone.documents=milestone.documents.filter((_,n)=>n!==j);confirming=false}}>{$translate('Hapus dokumen')}</button></div>{/each}
    <div class="ws-row"><button type="button" onclick={()=>{milestone.documents=[...milestone.documents,{id:0,name:''}];confirming=false}}>{$translate('Tambah dokumen wajib')}</button><button type="button" onclick={()=>{config.milestones=config.milestones.filter((_,n)=>n!==i);confirming=false}}>{$translate('Hapus milestone')}</button></div>
   </fieldset>{/each}
   <button type="button" onclick={()=>{config.milestones=[...config.milestones,{id:0,name:'',documents:[{id:0,name:''}]}];confirming=false}}>{$translate('Tambah milestone')}</button>
   <datalist id="billing-document-names">{#each documentSuggestions as name}<option value={name}></option>{/each}</datalist>
   {#if !validConfig}<p>{$translate('Isi pekerjaan, tahap, periode, minimal satu milestone dan dokumen wajib. Nama dokumen dalam milestone harus unik.')}</p>{/if}
   {#if confirming}<section role="status" class="conflict"><p>{$translate('Konfirmasi perubahan requirement. Penghapusan dapat membuat tahap menjadi SIAP. Dokumen baru dimulai BELUM ADA.')}</p><ul>{#each changes as change}<li>{change}</li>{/each}</ul></section>{/if}
  {:else if mode==='status'}<p>{targetName}</p><label>{$translate('Status dokumen')}<select aria-label={$translate('Status dokumen')} bind:value={draftStatus}>{#each billingStatuses as status}<option value={status}>{$translate(status)}</option>{/each}</select></label>
  {:else}<p>{targetName}</p><label>{$translate('Catatan')}<textarea maxlength="4000" rows="5" bind:value={draftNotes}></textarea></label>{/if}
  </fieldset>
  <div class="ws-row"><button type="button" disabled={busy} onclick={()=>dialog.close()}>{$translate('Batal')}</button><button class="ws-primary" disabled={busy||conflict||(mode==='configuration'&&!validConfig)}>{$translate(busy?'Menyimpan...':confirming?'Konfirmasi dan simpan':'Simpan')}</button></div>
 </form>
</dialog>
<style>
 .ws-metrics strong{display:block}.stage{margin-top:1rem}.stage h3,.milestone h4{margin:.25rem 0;overflow-wrap:anywhere}.stage h3 small{display:block;font-weight:400;margin-top:.35rem}.billing-job{margin-top:2rem}.readiness{display:inline-block;background:var(--his-danger-soft);color:var(--his-danger);border:1px solid var(--his-danger-border);padding:.35rem .65rem;border-radius:6px;font-size:.8rem;font-weight:700;white-space:nowrap}.readiness.ready{background:var(--his-primary-soft);color:var(--his-primary);border-color:var(--his-primary)}.milestone{border-top:1px solid var(--his-border);margin-top:1.25rem;padding-top:1rem}.timestamp,.documents small{font-size:.75rem;color:var(--his-muted)}.notes{white-space:pre-wrap;overflow-wrap:anywhere}.documents{list-style:none;padding:0}.documents li{display:grid;grid-template-columns:minmax(0,1fr) auto auto;align-items:center;gap:.75rem;padding:.75rem 0;border-bottom:1px solid var(--his-border)}.documents small{display:block;margin-top:.3rem}.document-status{font-size:.8rem}.billing-dialog{font:inherit}.billing-dialog fieldset{min-width:0;border:0;padding:0;margin:1rem 0;display:grid;gap:.8rem}.billing-dialog label{display:grid;gap:.35rem}.billing-dialog input,.billing-dialog select,.billing-dialog textarea{box-sizing:border-box;width:100%;min-width:0;font:inherit;padding:.65rem;border:1px solid var(--his-border);border-radius:6px}.billing-dialog button{font:inherit;min-height:44px;padding:.5rem .7rem;border:1px solid var(--his-border);border-radius:6px;cursor:pointer}.billing-dialog .ws-primary{background:var(--his-primary);color:white}.billing-dialog .requirement,.conflict{border:1px solid var(--his-border);padding:1rem;border-radius:8px}.requirement-row{display:flex;align-items:end;gap:.5rem}.requirement-row label{flex:1;min-width:0}.billing-dialog [role=alert]{color:var(--his-danger)}.billing-dialog h3,.billing-dialog h4,.conflict li{overflow-wrap:anywhere}.billing-dialog button:disabled{opacity:.5;cursor:default}@media(max-width:464px){.documents li{grid-template-columns:minmax(0,1fr)}.documents button{justify-self:start}.ws-filters{display:grid;grid-template-columns:minmax(0,1fr)}.requirement-row{flex-direction:column;align-items:stretch}.stage{padding:14px}.billing-dialog .requirement{padding:.65rem}}
</style>
