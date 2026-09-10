<script lang="ts">
 import {onMount,untrack} from 'svelte';
 import {subscribeChanges} from '$lib/realtime';
 import {page} from '$app/state';
 import {api,currentUser} from '$lib/api';
 import NotificationBell from './NotificationBell.svelte';
 import {leaveTypes,leaveStatuses,type LeaveRequest,type LeaveInput} from '$lib/leave';
 import {labelDate} from '$lib/workspace';
 import '$lib/styles/workspace.css';
 let {review=false}:{review?:boolean}=$props();
 const authorized=$derived(!review||['Admin','Management'].includes($currentUser?.role??''));
 const endpoint=$derived('/leave-requests'+(review?'/approvals':''));
 let requests=$state<LeaveRequest[]>([]),loading=$state(true),busy=$state(false),error=$state(''),notice=$state('');
 let status=$state(''),kind=$state(''),offset=$state(0),selected=$state<LeaveRequest|null>(null),creating=$state(false),reviewerNote=$state('');
 let input=$state<LeaveInput>({requestId:'',requestType:'CUTI',startDate:'',endDate:'',reason:'',notes:''});
 let dialog=$state<HTMLDialogElement>(null!);
 let generation=0;
 const time=(s:string)=>new Date(s).toLocaleString('id-ID',{timeZone:'Asia/Jakarta'});
 async function refresh(){
  if(!authorized){loading=false;return}const sequence=++generation;loading=true;
  try{const result=await api<{requests:LeaveRequest[]}>(endpoint+'?'+new URLSearchParams({status,type:kind,offset:String(offset)}));if(sequence===generation)requests=result.requests}
  catch(e){if(sequence===generation)error=(e as Error).message}finally{if(sequence===generation)loading=false}
 }
 async function open(id:string){error='';try{const result=await api<{request:LeaveRequest}>(endpoint+'/'+id);selected=result.request;reviewerNote='';creating=false;dialog.showModal()}catch(e){error=(e as Error).message}}
 onMount(()=>{status=review?'PENDING':'';void refresh();let active=true;const stop=subscribeChanges(['leave.changed'],async()=>{await refresh();const id=selected?.id;if(id){const r=await api<{request:LeaveRequest}>(endpoint+'/'+id);if(active&&selected?.id===id)selected=r.request}});return()=>{active=false;generation++;stop()}});
 $effect(()=>{const id=page.url.searchParams.get('request');if(id&&authorized)untrack(()=>void open(id))});
 function create(){input={requestId:crypto.randomUUID(),requestType:'CUTI',startDate:'',endDate:'',reason:'',notes:''};error='';selected=null;creating=true;dialog.showModal()}
 function close(){if(busy)return;dialog.close();creating=false;selected=null;error=''}
 async function submit(e:SubmitEvent){
  e.preventDefault();if(busy)return;
  if(!Object.hasOwn(leaveTypes,input.requestType)||!input.startDate||!input.endDate||input.endDate<input.startDate||!input.reason.trim()){error='Periksa jenis, rentang tanggal, dan alasan pengajuan.';return}
  busy=true;error='';notice='';
  try{await api('/leave-requests',{method:'POST',body:JSON.stringify(input)});dialog.close();creating=false;notice='Pengajuan berhasil dikirim.';offset=0;await refresh()}
  catch(e){error=(e as Error).message}finally{busy=false}
 }
 async function decide(status:'APPROVED'|'REJECTED'){
  if(!selected||busy)return;busy=true;error='';notice='';const id=selected.id;
  try{const result=await api<{request:LeaveRequest}>(endpoint+'/'+id+'/decision',{method:'POST',body:JSON.stringify({status,reviewerNote})});selected=result.request;notice='Keputusan berhasil disimpan.';await refresh()}
  catch(e){error=(e as Error).message;try{selected=(await api<{request:LeaveRequest}>(endpoint+'/'+id)).request;await refresh()}catch{/* Preserve original error. */}}
  finally{busy=false}
 }
</script>
<svelte:head><title>{review?'Persetujuan Izin & Cuti':'Izin & Cuti'} | my HIS</title></svelte:head>
<main class="workspace-page leave-page">
 <header class="ws-row"><div><p class="dashboard-kicker">PENGAJUAN KARYAWAN</p><h1>{review?'Persetujuan Izin & Cuti':'Izin & Cuti'}</h1><p>{review?'Tinjau pengajuan dan riwayat keputusan.':'Ajukan izin atau cuti dan pantau persetujuannya.'}</p></div><NotificationBell/></header>
 {#if !authorized}<p role="alert">Halaman persetujuan hanya tersedia untuk Admin dan Management.</p>
 {:else}
 <div class="ws-row"><h2>{review?'Daftar Pengajuan':'Pengajuan Saya'}</h2>{#if !review}<button class="ws-primary" onclick={create}>+ Ajukan Izin / Cuti</button>{/if}</div>
 <form class="ws-filters" onsubmit={(e)=>{e.preventDefault();error='';offset=0;void refresh()}}>
 <label>Status<select bind:value={status}><option value="">Semua status</option>{#each Object.entries(leaveStatuses) as [value,label]}<option {value}>{label}</option>{/each}</select></label>
 <label>Jenis<select bind:value={kind}><option value="">Semua jenis</option>{#each Object.entries(leaveTypes) as [value,label]}<option {value}>{label}</option>{/each}</select></label><button disabled={loading}>Terapkan</button>
 </form>
 {#if error&&!dialog?.open}<p role="alert">{error}</p>{/if}{#if notice}<p role="status">{notice}</p>{/if}
 {#if loading}<p role="status">Memuat pengajuan...</p>{/if}
 <div class="leave-cards">{#each requests as request (request.id)}
 <article class="ws-panel leave-card"><div class="ws-row"><h3>{review?request.employeeName:leaveTypes[request.requestType]}</h3><span class="leave-status" class:approved={request.status==='APPROVED'} class:rejected={request.status==='REJECTED'}>{leaveStatuses[request.status]}</span></div>
 {#if review}<p>{leaveTypes[request.requestType]}{request.division?' / '+request.division:''}</p>{/if}
 <p>{labelDate(request.startDate)} &ndash; {labelDate(request.endDate)}</p><p class="leave-reason">{request.reason}</p><small>Diajukan: {time(request.createdAt)} WIB</small>
 <button class="leave-open" onclick={()=>open(request.id)}>Lihat pengajuan</button></article>
 {:else}{#if !loading}<p class="ws-empty">{review&&status==='PENDING'?'Tidak ada pengajuan yang menunggu persetujuan.':review?'Tidak ada pengajuan sesuai filter.':'Belum ada pengajuan izin atau cuti.'}</p>{/if}{/each}</div>
 <div class="ws-row ws-pagination"><span>Halaman {Math.floor(offset/50)+1}</span><div class="leave-actions"><button disabled={loading||offset===0} onclick={()=>{offset-=50;void refresh()}}>Sebelumnya</button><button disabled={loading||requests.length<50} onclick={()=>{offset+=50;void refresh()}}>Berikutnya</button></div></div>
 {/if}
 <dialog bind:this={dialog} class="ws-dialog leave-dialog" oncancel={(e)=>{e.preventDefault();close()}} aria-labelledby="leave-title">
 <header class="ws-row"><h2 id="leave-title">{creating?'Ajukan Izin / Cuti':'Detail Pengajuan'}</h2><button disabled={busy} aria-label="Tutup pengajuan" onclick={close}>&times;</button></header>
 {#if error}<p role="alert">{error}</p>{/if}
 {#if creating}
 <form onsubmit={submit}><fieldset class="ws-form" disabled={busy}>
 <label class="ws-full">Jenis Pengajuan<select bind:value={input.requestType} required>{#each Object.entries(leaveTypes) as [value,label]}<option {value}>{label}</option>{/each}</select></label>
 <label>Tanggal Mulai<input type="date" bind:value={input.startDate} required/></label><label>Tanggal Selesai<input type="date" min={input.startDate||undefined} bind:value={input.endDate} required/></label>
 <label class="ws-full">Tujuan / Alasan<textarea rows="3" maxlength="4000" bind:value={input.reason} required></textarea></label>
 <label class="ws-full">Catatan Tambahan (opsional)<textarea rows="3" maxlength="4000" bind:value={input.notes}></textarea></label>
 </fieldset><div class="leave-actions"><button type="button" disabled={busy} onclick={close}>Batalkan</button><button class="ws-primary" disabled={busy||!input.reason.trim()}>{busy?'Mengirim...':'Ajukan'}</button></div></form>
 {:else if selected}
 <dl><dt>Nama</dt><dd>{selected.employeeName}</dd>{#if selected.division}<dt>Divisi</dt><dd>{selected.division}</dd>{/if}<dt>Jenis</dt><dd>{leaveTypes[selected.requestType]}</dd><dt>Tanggal</dt><dd>{labelDate(selected.startDate)} &ndash; {labelDate(selected.endDate)}</dd><dt>Tujuan / Alasan</dt><dd>{selected.reason}</dd><dt>Catatan</dt><dd>{selected.notes||'-'}</dd><dt>Diajukan</dt><dd>{time(selected.createdAt)} WIB</dd><dt>Status</dt><dd>{leaveStatuses[selected.status]}</dd>
 {#if selected.reviewedAt}<dt>Reviewer</dt><dd>{selected.reviewerName}</dd><dt>Tanggal Keputusan</dt><dd>{time(selected.reviewedAt)} WIB</dd><dt>Catatan Reviewer</dt><dd>{selected.reviewerNote||'-'}</dd>{/if}</dl>
 {#if review&&selected.status==='PENDING'}
 {#if selected.userId===$currentUser?.id}<p>Pengajuan Anda harus ditinjau oleh Admin atau Management lain.</p>
 {:else}<form onsubmit={(e)=>{e.preventDefault();void decide('APPROVED')}}><label>Catatan Reviewer (opsional)<textarea rows="3" maxlength="4000" bind:value={reviewerNote} disabled={busy}></textarea></label><div class="leave-actions"><button type="button" disabled={busy} onclick={()=>decide('REJECTED')}>Tolak</button><button class="ws-primary" disabled={busy}>{busy?'Menyimpan...':'Setujui'}</button></div></form>{/if}
 {/if}
 {/if}
 </dialog>
</main>
<style>
 .leave-page{max-width:1400px}.leave-cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,340px),1fr));gap:16px}.leave-card{display:flex;flex-direction:column;gap:10px}.leave-card h3,.leave-card p{margin:0}.leave-card small{color:#64748b}.leave-open{align-self:flex-start;margin-top:auto}.leave-status{background:#fff0d0;color:#855814;border-radius:6px;padding:6px 8px;font-size:13px}.leave-status.approved{background:#ddf5e5;color:#216941}.leave-status.rejected{background:#fee2e2;color:#a52626}.leave-actions{display:flex;gap:12px;flex-wrap:wrap;margin-top:16px}.leave-page button{min-height:44px;max-width:100%}.leave-dialog{position:fixed;inset:0;margin:auto;overflow-wrap:anywhere}.leave-dialog h2{margin:0}.leave-dialog dt{font-weight:600;margin-top:14px}.leave-dialog dd{margin:6px 0;white-space:pre-wrap}.leave-dialog textarea{min-height:96px}.leave-reason{white-space:pre-wrap;display:-webkit-box;-webkit-box-orient:vertical;-webkit-line-clamp:3;line-clamp:3;overflow:hidden}
</style>
