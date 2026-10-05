<script lang="ts">
 import { onMount, tick } from 'svelte';
 import { api, APIError, currentUser, type Job } from '$lib/api';
 import { locale, t as translate } from '$lib/i18n';
 import { billingStatuses, documentSuggestions, canUseBilling, type BillingStage, type BillingConfiguration, type BillingDocument, type BillingMilestone } from '$lib/billing-documents';
 import '$lib/styles/workspace.css';
 import { subscribeChanges } from '$lib/realtime';
 import { printTable } from '$lib/print-table';
 let billingTable = $state<HTMLTableElement>();
 const authorized = $derived(canUseBilling($currentUser?.role));
 let stages=$state<BillingStage[]>([]), jobs=$state<Job[]>([]), loading=$state(true), busy=$state(false), error=$state(''), notice=$state('');
 let jobFilter=$state(''), statusFilter=$state(''), search=$state('');
 let dialog:HTMLDialogElement;
 let pending=false, disposed=false, generation=0;
 const editing=()=>busy||dialog?.open||document.activeElement?.matches('.status-select');
 async function flush(){await tick();if(pending&&!disposed&&!editing())await load(true)}
 let mode=$state<'configuration'|'notes'>('configuration'), stageId=$state(0), version=$state(0), milestoneId=$state(0);
 let draftNotes=$state(''), targetName=$state(''), modalError=$state(''), conflict=$state(false), confirming=$state(false);
 let config=$state<BillingConfiguration>({jobId:0,name:'',periodLabel:'',version:0,documents:[],milestones:[]});
 let original=$state<BillingConfiguration|null>(null);
 const jobName=(id:number)=>jobs.find(j=>j.id===id)?.name??'';
 const matched=$derived(stages.filter(s=>(!jobFilter||s.jobId===Number(jobFilter))&&[jobName(s.jobId),s.name,s.periodLabel,...(s.documents??[]).map(d=>d.name),...s.milestones.flatMap(m=>[m.name,...m.documents.map(d=>d.name)])].join(' ').toLocaleLowerCase().includes(search.trim().toLocaleLowerCase())));
 const visible=$derived(matched.filter(s=>!statusFilter||s.readiness===statusFilter));
 const ready=$derived(matched.filter(s=>s.readiness==='SIAP').length);
 const latest=$derived(stages.find(s=>s.id===stageId));
 const changes=$derived.by(()=>{
  const old=original?.milestones??[];
  return [
   ...(original?.documents??[]).filter(d=>!config.documents.some(n=>n.id===d.id)).map(d=>'- '+d.name),
   ...config.documents.filter(d=>d.id===0).map(d=>'+ '+d.name),
   ...old.filter(m=>!config.milestones.some(n=>n.id===m.id)).map(m=>'- '+m.name),
   ...old.flatMap(m=>m.documents.filter(d=>!config.milestones.some(n=>n.id===m.id&&n.documents.some(x=>x.id===d.id))).map(d=>'- '+m.name+' / '+d.name)),
   ...config.milestones.filter(m=>m.id===0).map(m=>'+ '+m.name),
   ...config.milestones.flatMap(m=>m.documents.filter(d=>d.id===0).map(d=>'+ '+m.name+' / '+d.name))
  ];
 });
 const key=(name:string)=>name.trim().toLowerCase();
 const columns=$derived.by(()=>{
  const names=new Map(documentSuggestions.map(name=>[key(name),name]));
  for(const stage of visible)for(const doc of [...(stage.documents??[]),...stage.milestones.flatMap(m=>m.documents)])if(!names.has(key(doc.name)))names.set(key(doc.name),doc.name);
  return [...names.values()];
 });
 const findDocument=(documents:BillingDocument[],name:string)=>documents.find(d=>key(d.name)===key(name));
 const validDocuments=(docs:{name:string}[])=>docs.every(d=>d.name.trim())&&new Set(docs.map(d=>key(d.name))).size===docs.length;
 const validConfig=$derived(config.jobId>0&&!!config.name.trim()&&validDocuments(config.documents)&&config.milestones.length>0&&config.milestones.every(m=>m.name.trim()&&(m.documents.length>0||config.documents.length>0)&&validDocuments(m.documents)&&m.documents.every(d=>!config.documents.some(shared=>key(shared.name)===key(d.name)))));
 const today=new Intl.DateTimeFormat('id-ID',{timeZone:'Asia/Jakarta',dateStyle:'long'}).format(new Date());
 function configuration(s:BillingStage):BillingConfiguration{return {jobId:s.jobId,name:s.name,periodLabel:s.periodLabel,version:s.version,documents:(s.documents??[]).map(d=>({id:d.id,name:d.name})),milestones:s.milestones.map(m=>({id:m.id,name:m.name,documents:m.documents.map(d=>({id:d.id,name:d.name}))}))}}
 async function changeStatus(s:BillingStage,d:BillingDocument,element:HTMLSelectElement){
  const status=element.value;
  element.value=d.status;
  if(busy||status===d.status)return;
  busy=true;generation++;error='';notice='Menyimpan...';
  try{
   const {stage}=await api<{stage:BillingStage}>('/billing-documents/stages/'+s.id,{method:'PATCH',body:JSON.stringify({version:s.version,documentId:d.id,status})});
   stages=stages.map(item=>item.id===stage.id?stage:item);notice='Perubahan tersimpan.';
  }catch(e){
   notice='';
   if(e instanceof APIError&&e.status===409){await load();error='Data penagihan sudah berubah. Periksa status terbaru, lalu pilih kembali perubahan Anda.'}
   else error=(e as Error).message;
  }finally{busy=false;void flush()}
 }
 const date=(value:string)=>new Intl.DateTimeFormat($locale,{timeZone:'Asia/Jakarta',dateStyle:'medium',timeStyle:'short'}).format(new Date(value));
 async function load(background=false){
  if(background&&editing()){pending=true;return false}
  const request=++generation;pending=false;
  if(!background)loading=true;
  error='';
  try{
   const result=await api<{jobs:Job[];stages:BillingStage[]}>('/billing-documents');
   if(disposed||request!==generation)return false;
   if(background&&editing()){pending=true;return false}
   jobs=result.jobs;stages=result.stages;return true;
  }catch(e){if(!disposed&&request===generation)error=(e as Error).message;return false}
  finally{if(request===generation)loading=false}
 }
 onMount(()=>{
  if(!authorized){loading=false;return}
  void load();
  const stop=subscribeChanges(['billing-documents.changed','jobs.changed'],()=>load(true).then(()=>{}));
  return()=>{disposed=true;generation++;stop()};
 });
 function open(s:BillingStage|null,next:typeof mode){stageId=s?.id??0;version=s?.version??0;mode=next;modalError='';conflict=false;confirming=false;notice='';dialog.showModal()}
 function configure(s:BillingStage|null){
  config=s?configuration(s):{jobId:Number(jobFilter)||0,name:'',periodLabel:'',version:0,documents:[],milestones:[{id:0,name:'',documents:[{id:0,name:''}]}]};
  original=s?structuredClone($state.snapshot(config)):null;open(s,'configuration');
 }
 function editNotes(s:BillingStage,m?:BillingMilestone){milestoneId=m?.id??0;targetName=m?.name??s.name;draftNotes=m?.notes??s.notes;open(s,'notes')}
 async function save(e:SubmitEvent){
  e.preventDefault();if(busy||conflict)return;
  if(mode==='configuration'&&!validConfig)return;
  if(mode==='configuration'&&stageId&&changes.length&&!confirming){confirming=true;return}
  busy=true;generation++;modalError='';
  const path='/billing-documents/stages'+(stageId?'/'+stageId:'')+(mode==='configuration'&&stageId?'/configuration':'');
  const body=mode==='configuration'?{...config,version}:{version,milestoneId,notes:draftNotes};
  try{
   const {stage}=await api<{stage:BillingStage}>(path,{method:mode==='configuration'?(stageId?'PUT':'POST'):'PATCH',body:JSON.stringify(body)});
   stages=[...stages.filter(s=>s.id!==stage.id),stage].sort((a,b)=>a.id-b.id);dialog.close();notice='Perubahan tersimpan.';
  }catch(e){modalError=(e as Error).message;if(e instanceof APIError&&e.status===409){conflict=true;await load()}}
  finally{busy=false;void flush()}
 }
</script>

{#snippet documentName(doc:{name:string},label:string)}
 <div class="document-name-editor">
  <label>{label}<input required maxlength="255" bind:value={doc.name}/></label>
  <label>{$translate('Pilihan dokumen')}<select aria-label={$translate('Pilihan dokumen')+' '+label} value={doc.name} onchange={e=>{doc.name=e.currentTarget.value;confirming=false}}>
   <option value="" disabled>{$translate('Pilih dokumen')}</option>
   {#if doc.name&&!documentSuggestions.includes(doc.name)}<option value={doc.name}>{doc.name}</option>{/if}
   {#each documentSuggestions as name}<option value={name}>{name}</option>{/each}
  </select></label>
 </div>
{/snippet}

<svelte:head><title>{$translate('Dokumen Penagihan')}</title></svelte:head>
<main class="workspace-page billing-page" onfocusout={()=>void flush()}>
 <header class="ws-row"><div><h1>{$translate('REKAP KESIAPAN DOKUMEN TAGIHAN')}</h1><p>{$translate('Rekap kesiapan dokumen penagihan per pekerjaan.')}</p></div>{#if authorized}<button class="ws-primary" disabled={loading||busy||!jobs.some(j=>j.isActive)} onclick={()=>configure(null)}>{$translate('Tambah tahap penagihan')}</button>{/if}</header>
 {#if !authorized}<p role="alert">{$translate('Akses penagihan ditolak.')}</p>{:else}
 {#if error}<p role="alert">{$translate(error)}</p><button onclick={()=>load()}>{$translate('Muat ulang')}</button>{/if}
 {#if notice}<p role="status">{$translate(notice)}</p>{/if}
 <div class="ws-filters">
  <label>{$translate('Pekerjaan')}<select aria-label={$translate('Pekerjaan')} bind:value={jobFilter}><option value="">{$translate('Semua pekerjaan')}</option>{#each jobs as j}<option value={String(j.id)}>{[j.name,j.division,j.role,j.category].filter(Boolean).join(' / ')}{j.isActive?'':' ('+$translate('Nonaktif')+')'}</option>{/each}</select></label>
  <label>{$translate('Kesiapan tahap')}<select aria-label={$translate('Kesiapan tahap')} bind:value={statusFilter}><option value="">{$translate('Semua status')}</option><option value="SIAP">{$translate('SIAP')}</option><option value="BELUM SIAP">{$translate('BELUM SIAP')}</option></select></label>
  <label>{$translate('Cari penagihan')}<input type="search" bind:value={search}/></label>
 </div>
 <div class="ws-metrics"><section class="ws-panel"><span>{$translate('Tahap SIAP')}</span><strong>{ready}</strong></section><section class="ws-panel"><span>{$translate('Tahap BELUM SIAP')}</span><strong>{matched.length-ready}</strong></section></div>
 {#if loading}<p role="status">{$translate('Memuat penagihan...')}</p>{:else}
 <p class="data-date">{$translate('Data per')}: {today}</p>
 <button class="ws-primary" disabled={busy||!visible.length} onclick={()=>{if(billingTable)printTable(billingTable,$translate('REKAP KESIAPAN DOKUMEN TAGIHAN')+' — '+today)}}>{$translate('Cetak / Simpan PDF')}</button>
 {#if visible.length}
 <div class="billing-scroll" role="region" aria-label={$translate('Rekap kesiapan dokumen tagihan')} aria-busy={busy}>
 <table class="billing-table" bind:this={billingTable}>
  <caption class="sr-only">{$translate('Rekap kesiapan dokumen tagihan')}</caption>
  <thead><tr><th colspan="3" scope="colgroup">{$translate('PEKERJAAN')}</th><th colspan={columns.length} scope="colgroup">{$translate('DOKUMEN')}</th><th rowspan="2" scope="col">{$translate('STATUS')}</th><th rowspan="2" scope="col">{$translate('CATATAN')}</th></tr>
  <tr><th scope="col">{$translate('Pekerjaan')}</th><th scope="col">{$translate('Tahap')}</th><th scope="col">{$translate('Bulan / sub-tahap')}</th>{#each columns as name}<th scope="col" class="document-heading">{name}</th>{/each}</tr></thead>
  {#each jobs.filter(j=>visible.some(s=>s.jobId===j.id)) as job (job.id)}
   {@const jobStages=visible.filter(s=>s.jobId===job.id)}
   <tbody>
   {#each jobStages as stage, si (stage.id)}
    {#each stage.milestones.length?stage.milestones:[null] as milestone, mi (milestone?.id??0)}
     <tr data-stage-id={stage.id}>
      {#if si===0&&mi===0}<th class="job-cell" scope="rowgroup" rowspan={jobStages.reduce((n,s)=>n+Math.max(1,s.milestones.length),0)}>{job.name}</th>{/if}
      {#if mi===0}<td rowspan={Math.max(1,stage.milestones.length)} class="stage-cell"><strong>{stage.name}</strong>{#if stage.periodLabel}<small>{stage.periodLabel}</small>{/if}<button disabled={busy} onclick={()=>configure(stage)}>{$translate('Edit requirement')}</button><small title={date(stage.updatedAt)}>{$translate('Diperbarui (WIB)')}: {date(stage.updatedAt)}</small></td>{/if}
      <th scope="row" class="milestone-cell">{milestone?.name??$translate('Requirement belum dikonfigurasi.')}{#if milestone}<button class="note-button" disabled={busy} title={milestone.notes||$translate('Catatan sub-tahap')} aria-label={$translate('Catatan sub-tahap')+' '+milestone.name} onclick={()=>editNotes(stage,milestone)}>{$translate('Catatan sub-tahap')}{milestone.notes?' *':''}</button>{/if}</th>
      {#each columns as name}
       {@const shared=findDocument(stage.documents??[],name)}
       {@const doc=shared??findDocument(milestone?.documents??[],name)}
       {#if !shared||mi===0}<td class="document-cell" rowspan={shared?Math.max(1,stage.milestones.length):1} class:unused={!doc}>
        {#if doc}<select class="status-select" data-status={doc.status} value={doc.status} disabled={busy} aria-label={$translate('Status dokumen')+' '+job.name+' / '+stage.name+' / '+(shared?$translate('Seluruh tahap'):milestone?.name)+' / '+name} title={date(doc.updatedAt)} onchange={e=>changeStatus(stage,doc,e.currentTarget)}>{#each billingStatuses as status}<option value={status}>{status}</option>{/each}</select>{:else}<span class="sr-only">{$translate('Tidak diwajibkan')}</span>{/if}
       </td>{/if}
      {/each}
      {#if mi===0}<td rowspan={Math.max(1,stage.milestones.length)} class="readiness" class:ready={stage.readiness==='SIAP'}>{$translate(stage.readiness)}</td><td rowspan={Math.max(1,stage.milestones.length)} class="notes-cell"><p class="notes">{stage.notes||$translate('Belum ada catatan.')}</p><button disabled={busy} onclick={()=>editNotes(stage)}>{$translate('Catatan tahap')}</button></td>{/if}
     </tr>
    {/each}
   {/each}
   </tbody>
  {/each}
 </table>
 </div>
 {:else}<p class="ws-empty">{$translate('Belum ada tahap penagihan yang cocok.')}</p>{/if}
 {/if}{/if}
</main>

<dialog bind:this={dialog} class="ws-dialog billing-dialog" onclose={()=>void flush()} oncancel={e=>{if(busy)e.preventDefault()}} aria-labelledby="billing-editor-title">
 <form onsubmit={save}>
  <header class="billing-editor-header"><h2 id="billing-editor-title">{$translate(mode==='configuration'?(stageId?'Edit requirement':'Tambah tahap penagihan'):'Edit catatan')}</h2><button class="billing-close" type="button" aria-label={$translate('Tutup')} disabled={busy} onclick={()=>dialog.close()}>&times;</button></header>
  {#if modalError}<p role="alert">{$translate(modalError)}</p>{/if}
  {#if conflict}<section class="conflict"><p>{$translate('Draf dipertahankan. Bandingkan dengan data terbaru sebelum mencoba lagi.')}</p>
   {#if latest}<h3>{latest.name} — {latest.periodLabel}</h3><p>{latest.notes}</p><ul>{#each latest.documents??[] as d}<li>{d.name}: {$translate(d.status)}</li>{/each}</ul>{#each latest.milestones as m}<h4>{m.name} — {$translate(m.readiness)}</h4><p>{m.notes}</p><ul>{#each m.documents as d}<li>{d.name}: {$translate(d.status)}</li>{/each}</ul>{/each}
   <button type="button" disabled={busy||latest.version<=version} onclick={()=>{if(mode==='configuration')original=configuration(latest!);version=latest!.version;conflict=false;confirming=false;modalError=''}}>{$translate('Gunakan versi terbaru untuk draf ini')}</button>{/if}
  </section>{/if}
  <fieldset disabled={busy}>
  {#if mode==='configuration'}
   <label>{$translate('Pekerjaan')}<select aria-label={$translate('Pekerjaan')} required disabled={!!stageId} bind:value={config.jobId}><option value={0}>{$translate('Pilih pekerjaan')}</option>{#each jobs.filter(j=>j.isActive||j.id===config.jobId) as j}<option value={j.id}>{[j.name,j.division,j.role,j.category].filter(Boolean).join(' / ')}</option>{/each}</select></label>
   <label>{$translate('Nama tahap')}<input required maxlength="255" bind:value={config.name}/></label>
   <p>{$translate('Pilih hanya dokumen wajib. Tanda / tidak dihitung sebagai alternatif otomatis.')}</p>
   {#each config.milestones as milestone, i}
   <fieldset class="requirement">
    <legend>{$translate('Status Dokumen per Sub-Tahap')} {i+1}</legend>
    <label>{$translate('Nama sub-tahap')}<input required maxlength="255" bind:value={milestone.name}/></label>
    {#each milestone.documents as doc, j}<div class="requirement-row">{@render documentName(doc,$translate('Dokumen wajib')+' '+(j+1))}<button type="button" onclick={()=>{milestone.documents=milestone.documents.filter((_,n)=>n!==j);confirming=false}}>{$translate('Hapus dokumen')}</button></div>{/each}
    <div class="ws-row"><button type="button" onclick={()=>{milestone.documents=[...milestone.documents,{id:0,name:''}];confirming=false}}>{$translate('Tambah dokumen wajib')}</button><button type="button" onclick={()=>{config.milestones=config.milestones.filter((_,n)=>n!==i);confirming=false}}>{$translate('Hapus sub-tahap')}</button></div>
   </fieldset>{/each}
   <button type="button" onclick={()=>{config.milestones=[...config.milestones,{id:0,name:'',documents:[{id:0,name:''}]}];confirming=false}}>{$translate('Tambah sub-tahap')}</button>
   <fieldset class="requirement"><legend>{$translate('Status Dokumen per Tahap')}</legend>
    <p>{$translate('Satu status berlaku untuk seluruh bulan dalam tahap ini. Jangan tambahkan dokumen yang sama pada setiap bulan.')}</p>
    {#each config.documents as doc, i}<div class="requirement-row">{@render documentName(doc,$translate('Dokumen tahap')+' '+(i+1))}<button type="button" onclick={()=>{config.documents=config.documents.filter((_,n)=>n!==i);confirming=false}}>{$translate('Hapus dokumen')}</button></div>{/each}
    <button type="button" onclick={()=>{config.documents=[...config.documents,{id:0,name:''}];confirming=false}}>{$translate('Tambah dokumen tahap')}</button>
   </fieldset>

   {#if !validConfig}<p>{$translate('Isi pekerjaan, nama tahap, dan minimal satu sub-tahap. Setiap sub-tahap memerlukan dokumen sendiri atau dokumen per tahap. Nama dokumen harus unik dan tidak boleh digunakan pada kedua cakupan.')}</p>{/if}
   {#if confirming}<section role="status" class="conflict"><p>{$translate('Konfirmasi perubahan requirement. Penghapusan dapat membuat tahap menjadi SIAP. Dokumen baru dimulai BELUM ADA.')}</p><ul>{#each changes as change}<li>{change}</li>{/each}</ul></section>{/if}
  {:else}<p>{targetName}</p><label>{$translate('Catatan')}<textarea maxlength="4000" rows="5" bind:value={draftNotes}></textarea></label>{/if}
  </fieldset>
  <div class="ws-row"><button type="button" disabled={busy} onclick={()=>dialog.close()}>{$translate('Batal')}</button><button class="ws-primary" disabled={busy||conflict||(mode==='configuration'&&!validConfig)}>{$translate(busy?'Menyimpan...':confirming?'Konfirmasi dan simpan':'Simpan')}</button></div>
 </form>
</dialog>
<style>
 .billing-editor-header{display:flex;align-items:flex-start;gap:1rem;justify-content:space-between}.billing-editor-header h2{margin:0;min-width:0}.billing-close{flex:0 0 44px;font-size:1.5rem!important}.document-name-editor{flex:1;min-width:0;display:grid;gap:.5rem;width:100%}
 .billing-page{min-width:0;max-width:100%}.billing-page h1{font-size:clamp(1.05rem,2vw,1.5rem);overflow-wrap: break-word}.billing-page>header{background:#a9c5ef;color:#102438;padding:1rem;border-radius:8px}.ws-metrics{grid-template-columns:repeat(2,minmax(0,1fr))}.billing-page .ws-metrics .ws-panel{padding:.75rem;margin-top:0}.ws-metrics strong{display:block}.document-heading,.document-cell{min-width:140px}.data-date{font-size:.85rem}.billing-scroll{position:relative;max-width:100%;overflow:auto;border:1px solid #64748b;border-radius:6px}.billing-scroll:focus-visible{outline:3px solid var(--his-primary);outline-offset:3px}.billing-table{border-collapse:separate;border-spacing:0;min-width:1850px;width:100%;font-size:.78rem;background:var(--his-surface,#fff)}.billing-table th,.billing-table td{border-right:1px solid #64748b;border-bottom:1px solid #64748b;padding:.4rem;vertical-align:middle}.billing-table thead th{background:#fff200;color:#17202a;text-align:center;font-weight:700}.billing-table tbody th{font-weight:500;text-align:left}.billing-table tbody+tbody tr:first-child>*{border-top:2px solid #334155}.billing-table button{font:inherit;min-height:44px;border:1px solid var(--his-border);border-radius:5px;background:var(--his-surface,#fff);color:var(--his-primary);padding:.4rem .5rem;cursor:pointer}.billing-table button:disabled{opacity:.6;cursor:wait}.billing-table small{display:block;font-size:.7rem;margin:.4rem 0;color:var(--his-muted)}.job-cell{min-width:90px;max-width:150px;overflow-wrap: break-word}.stage-cell{min-width:130px;max-width:180px;overflow-wrap: break-word}.milestone-cell{min-width:170px;max-width:220px;overflow-wrap: break-word}.note-button{display:block;margin-top:.25rem}.notes-cell{min-width:240px;max-width:320px}.notes{white-space:pre-wrap;overflow-wrap: break-word}.billing-table .unused{background:#d0d0d0}.readiness{background:#f4cdca;color:#70201b;min-width:100px;text-align:center;font-weight:600}.readiness.ready{background:#b5dfcf;color:#124b39}.status-select{width:100%;min-width:136px;min-height:44px;border:1px solid transparent;border-radius:8px;padding:.35rem .25rem;font:inherit;font-weight:600;cursor:pointer}.status-select:focus-visible{outline:3px solid #2563eb;outline-offset:2px}.status-select[data-status="BELUM ADA"]{background:#b91c1c;color:white}.status-select[data-status="PEMBUATAN"]{background:#ffc7aa;color:#713414}.status-select[data-status="KURANG LENGKAP"]{background:#ffe7a3;color:#63470a}.status-select[data-status="SIRKULIR TTD"]{background:#d2edbd;color:#24562b}.status-select[data-status="OK"]{background:#11784f;color:white}.status-select option{background:white;color:#17202a}.status-select:disabled{opacity:.65;cursor:wait}.sr-only{position:absolute;width:1px;height:1px;padding:0;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
 .billing-dialog{font:inherit}.billing-dialog fieldset{min-width:0;border:0;padding:0;margin:1rem 0;display:grid;gap:.8rem}.billing-dialog label{display:grid;gap:.35rem}.billing-dialog input,.billing-dialog select,.billing-dialog textarea{box-sizing:border-box;width:100%;min-width:0;font:inherit;padding:.65rem;border:1px solid var(--his-border);border-radius:6px}.billing-dialog button{font:inherit;min-height:44px;padding:.5rem .7rem;border:1px solid var(--his-border);border-radius:6px;cursor:pointer}.billing-dialog .ws-primary{background:var(--his-primary);color:white}.billing-dialog .requirement,.conflict{border:1px solid var(--his-border);padding:1rem;border-radius:8px}.requirement-row{display:flex;align-items:end;gap:.5rem}.requirement-row label{flex:1;min-width:0}.billing-dialog [role=alert]{color:var(--his-danger)}.billing-dialog h3,.billing-dialog h4,.conflict li{overflow-wrap: break-word}.billing-dialog button:disabled{opacity:.5;cursor:default}@media(max-width:464px){.ws-filters{display:grid;grid-template-columns:minmax(0,1fr)}.requirement-row{flex-direction:column;align-items:stretch}.billing-dialog .requirement{padding:.65rem}}
</style>
