<script lang="ts">
 import { onMount, tick } from 'svelte';
 import { subscribeChanges } from '$lib/realtime';
 import { api, type Job } from '$lib/api';
 import type { DocumentChecklist, ChecklistSigner, ChecklistItem } from '$lib/document-checklist';
 let { job }: {job: Job} = $props();
 let checklist = $state<DocumentChecklist | null>(null), loading = $state(true), busy = $state(false), error = $state(''), notice = $state('');
 let dialog: HTMLDialogElement;
 let root: HTMLDivElement;
 let pending = false, disposed = false, generation = 0;
 const editingCell = () => root?.contains(document.activeElement) && document.activeElement?.matches('input,textarea');
 let digital = $state(false), internalCount = $state<number | undefined>(0), externalCount = $state<number | undefined>(0);
 let internal = $state<ChecklistSigner[]>([]), external = $state<ChecklistSigner[]>([]), configVersion = $state(0);
 let newDocument = $state(''), modalError = $state('');
 const endpoint = $derived('/document-checklists/' + job.id);
 const active = $derived(checklist?.signers.filter(s => !s.archived) ?? []);
 const archived = $derived(checklist?.signers.filter(s => s.archived) ?? []);
 const groups = $derived(['internal', 'external'].map(type => ({type, label: type === 'internal' ? 'Internal' : 'External', signers: active.filter(s => s.type === type)})).filter(g => g.signers.length));
 const valid = $derived(Number.isInteger(internalCount) && Number.isInteger(externalCount) && (internalCount ?? -1) >= 0 && (externalCount ?? -1) >= 0 && internal.length === internalCount && external.length === externalCount && [...internal, ...external].every(s => s.name.trim() && s.position.trim()));
 async function load(background = false) {
  if(background && (busy || editingCell())) { pending = true; return; }
  const request = ++generation;
  pending = false;
  if(!background) loading = true;
  error = '';
  try {
   const result = (await api<{checklist: DocumentChecklist | null}>(endpoint)).checklist;
   if(disposed || request !== generation) return;
   if(background && (busy || editingCell())) { pending = true; return; }
   checklist = result;
  }
  catch(e) { if(!disposed && request === generation) error = (e as Error).message; }
  finally { if(request === generation) loading = false; }
 }
 async function flush() { await tick(); if(!disposed && pending && !busy && !editingCell()) await load(true); }
 onMount(() => {
  void load();
  const stop = subscribeChanges(['document-checklists.changed'], () => load(true), change => !change.entityId || change.entityId === String(job.id));
  return () => { disposed = true; generation++; stop(); };
 });
 function resize(type: 'internal' | 'external', count: number | undefined) {
  if (!Number.isInteger(count) || count === undefined || count < 0) return;
  const old = type === 'internal' ? internal : external;
  const next = Array.from({length: count}, (_, i) => old[i] ?? {id: 0, type, name: '', position: '', displayOrder: i, archived: false});
  if (type === 'internal') internal = next; else external = next;
 }
 function configure() {
  digital = checklist?.requireDigitalSignature ?? false; configVersion = checklist?.version ?? 0;
  internal = active.filter(s => s.type === 'internal').map(s => ({...s})); external = active.filter(s => s.type === 'external').map(s => ({...s}));
  internalCount = internal.length; externalCount = external.length; modalError = ''; dialog.showModal();
 }
 async function saveConfig(e: SubmitEvent) {
  e.preventDefault(); if (!valid || busy) return;
  const signers = [...internal, ...external];
  const changed = active.some(s => !signers.some(n => n.id === s.id && n.type === s.type && n.name.trim() === s.name && n.position.trim() === s.position));
  if (changed && !confirm('Penandatangan yang dihapus atau diubah akan diarsipkan beserta status lamanya. Penandatangan pengganti dimulai tanpa centang. Lanjutkan?')) return;
  busy = true; generation++; modalError = '';
  try {
   checklist = (await api<{checklist: DocumentChecklist}>(endpoint, {method: configVersion ? 'PUT' : 'POST', body: JSON.stringify({requireDigitalSignature: digital, version: configVersion, internalCount, externalCount, signers})})).checklist;
   dialog.close(); notice = 'Struktur checklist tersimpan.';
  } catch(e) { modalError = (e as Error).message; }
  finally { busy = false; void flush(); }
 }
 async function mutate(path: string, method: string, body?: object) {
  if (busy) return false;
  busy = true; generation++; error = ''; notice = '';
  try {
   checklist = (await api<{checklist: DocumentChecklist}>(endpoint + path, {method, body: body ? JSON.stringify(body) : undefined})).checklist;
   notice = 'Perubahan tersimpan.'; return true;
  } catch(e) { error = (e as Error).message; return false; }
  finally { busy = false; void flush(); }
 }
 async function edit(item: ChecklistItem, field: 'documentName' | 'documentPosition' | 'sentDate' | 'notes', input: HTMLInputElement | HTMLTextAreaElement) {
  const old = item[field] ?? '';
  if (old === input.value) return;
  if (!input.reportValidity()) { input.value = old; return; }
  if (!await mutate('/items/' + item.id, 'PATCH', {field, value: input.value})) input.value = old;
 }
 async function check(item: ChecklistItem, input: HTMLInputElement, signerId?: number) {
  if (!await mutate('/items/' + item.id, 'PATCH', {field: signerId ? 'signature' : 'digitalSigned', checked: input.checked, ...(signerId ? {signerId} : {})})) input.checked = signerId ? !!item.signatures[signerId] : item.digitalSigned;
 }
</script>
<div class="checklist" bind:this={root} onfocusout={() => void flush()}>
 {#if error}<p role="alert">{error}</p><button disabled={busy} onclick={() => load()}>Muat ulang checklist</button>{/if}
 {#if notice}<p role="status">{notice}</p>{/if}
 {#if loading}<p role="status">Memuat checklist...</p>
 {:else if !checklist && !error}<p>Pekerjaan ini belum mempunyai checklist.</p><button class="ws-primary" onclick={configure}>+ Buat Tabel Checklist</button>
 {:else if checklist}
  <div class="ws-row"><button disabled={busy} onclick={configure}>Edit Struktur Checklist</button><span>Perubahan sel disimpan setelah selesai mengisi.</span></div>
  <form class="add-document" onsubmit={async e => {e.preventDefault(); if (newDocument.trim() && await mutate('/items', 'POST', {field: 'documentName', value: newDocument})) newDocument = '';}}>
   <label>Dokumen baru<input aria-label="Dokumen baru" required maxlength="255" bind:value={newDocument} disabled={busy}/></label><button class="ws-primary" disabled={busy || !newDocument.trim()}>+ Tambah Dokumen</button>
  </form>
  <div class="table-scroll" role="region" aria-label={'Checklist '+job.name}>
   <table>
    <thead>
     <tr><th rowspan={active.length ? 3 : 1} scope="col">No</th><th rowspan={active.length ? 3 : 1} scope="col">Dokumen</th>{#if checklist.requireDigitalSignature}<th rowspan={active.length ? 3 : 1} scope="col">TTD Digital</th>{/if}{#if active.length}<th colspan={active.length} scope="colgroup">TTD Basah</th>{/if}<th rowspan={active.length ? 3 : 1} scope="col">Posisi Dokumen</th><th rowspan={active.length ? 3 : 1} scope="col">Tanggal Kirim Dokumen</th><th rowspan={active.length ? 3 : 1} scope="col">Catatan</th><th rowspan={active.length ? 3 : 1} scope="col">Aksi</th></tr>
     {#if active.length}<tr>{#each groups as group}<th colspan={group.signers.length} scope="colgroup">{group.label}</th>{/each}</tr><tr>{#each groups as group}{#each group.signers as signer (signer.id)}<th scope="col">{signer.name}<small>{signer.position}</small></th>{/each}{/each}</tr>{/if}
    </thead>
    <tbody>
     {#each checklist.items as item, index (item.id)}
      <tr><td>{index + 1}</td><td><input aria-label={'Dokumen '+(index+1)} required maxlength="255" value={item.documentName} disabled={busy} onchange={e => edit(item, 'documentName', e.currentTarget)}/></td>
       {#if checklist.requireDigitalSignature}<td class="signature"><input type="checkbox" aria-label={'TTD Digital '+item.documentName} checked={item.digitalSigned} disabled={busy} onchange={e => check(item,e.currentTarget)}/></td>{/if}
       {#each groups as group}{#each group.signers as signer (signer.id)}<td class="signature"><input type="checkbox" aria-label={signer.name+' — '+item.documentName} checked={!!item.signatures[signer.id]} disabled={busy} onchange={e => check(item,e.currentTarget,signer.id)}/></td>{/each}{/each}
       <td><input aria-label={'Posisi '+item.documentName} maxlength="255" value={item.documentPosition} disabled={busy} onchange={e => edit(item,'documentPosition',e.currentTarget)}/></td>
       <td><input type="date" aria-label={'Tanggal kirim '+item.documentName} value={item.sentDate ?? ''} disabled={busy} onchange={e => edit(item,'sentDate',e.currentTarget)}/></td>
       <td><textarea aria-label={'Catatan '+item.documentName} maxlength="4000" value={item.notes} disabled={busy} onchange={e => edit(item,'notes',e.currentTarget)}></textarea></td>
       <td><button disabled={busy} aria-label={'Hapus '+item.documentName} onclick={() => {if(confirm('Hapus dokumen '+item.documentName+' dari checklist?')) void mutate('/items/'+item.id,'DELETE');}}>Hapus</button></td>
      </tr>
     {:else}<tr><td colspan={6 + active.length + (checklist.requireDigitalSignature ? 1 : 0)}>Belum ada dokumen. Tambahkan dokumen pertama di atas.</td></tr>{/each}
    </tbody>
   </table>
  </div>
  {#if archived.length}<details><summary>Riwayat penandatangan yang diarsipkan ({archived.length})</summary>{#each archived as signer}<p><strong>{signer.name}</strong> — {signer.position} ({signer.type})</p><ul>{#each checklist.items as item}<li>{item.documentName}: {item.signatures[signer.id] ? 'Sudah ditandatangani' : 'Belum ditandatangani'}</li>{/each}</ul>{/each}</details>{/if}
 {/if}
</div>
<dialog bind:this={dialog} aria-labelledby={'config-title-'+job.id} oncancel={e => {if(busy)e.preventDefault();}}>
 <form onsubmit={saveConfig}>
  <h2 id={'config-title-'+job.id}>Struktur Checklist — {job.name}</h2>
  {#if modalError}<p role="alert">{modalError}</p>{/if}
  <fieldset disabled={busy}>
   <label class="digital"><input type="checkbox" bind:checked={digital}/> Butuh TTD Digital?</label>
   <label>Berapa banyak TTD Internal?<input type="number" min="0" step="1" required bind:value={internalCount} oninput={e => {internalCount=e.currentTarget.valueAsNumber;resize('internal',internalCount);}}/></label>
   {#each internal as signer, i}<fieldset><legend>TTD Internal {i+1}</legend><label>Nama<input required maxlength="255" bind:value={signer.name}/></label><label>Sebagai<input required maxlength="255" bind:value={signer.position}/></label></fieldset>{/each}
   <label>Berapa banyak TTD External?<input type="number" min="0" step="1" required bind:value={externalCount} oninput={e => {externalCount=e.currentTarget.valueAsNumber;resize('external',externalCount);}}/></label>
   {#each external as signer, i}<fieldset><legend>TTD External {i+1}</legend><label>Nama<input required maxlength="255" bind:value={signer.name}/></label><label>Sebagai<input required maxlength="255" bind:value={signer.position}/></label></fieldset>{/each}
  </fieldset>
  <p>Perubahan identitas penandatangan membuat kolom baru. Status lama tetap tersedia di riwayat arsip.</p>
  <div class="ws-row"><button type="button" disabled={busy} onclick={() => dialog.close()}>Batal</button><button class="ws-primary" disabled={busy || !valid}>{busy ? 'Menyimpan...' : 'Simpan'}</button></div>
 </form>
</dialog>
<style>
 .checklist{padding:1rem;min-width:0}.add-document{display:flex;align-items:end;gap:.75rem;margin:1rem 0;flex-wrap:wrap}label{display:grid;gap:.35rem}input,textarea{font:inherit;padding:.5rem;border:1px solid var(--his-border);border-radius:6px}input:not([type=checkbox]),textarea{min-width:160px;max-width:100%;box-sizing:border-box}textarea{min-height:42px;resize:vertical}.table-scroll{overflow-x:auto;max-width:100%;margin:1rem 0}table{border-collapse:collapse;width:100%}th,td{border:1px solid var(--his-border);padding:.55rem;vertical-align:middle}th{background:var(--his-surface);min-width:75px}th small{display:block;font-weight:400;margin-top:.3rem}.signature{text-align:center}input[type=checkbox]{width:20px;height:20px;accent-color:var(--his-primary)}dialog{width:min(620px,calc(100vw - 32px));max-height:85dvh;box-sizing:border-box;overflow:auto;border:1px solid var(--his-border);border-radius:12px;padding:1.5rem}dialog::backdrop{background:#0006}fieldset{border:0;padding:0;margin:1rem 0;display:grid;gap:1rem}fieldset fieldset{border:1px solid var(--his-border);border-radius:8px;padding:1rem}.digital{display:flex;align-items:center}details{margin-top:1rem}button{min-height:40px}button:disabled{opacity:.6}
</style>
