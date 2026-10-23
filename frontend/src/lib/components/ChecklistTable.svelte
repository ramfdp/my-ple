<script lang="ts">
 import { t as translate } from '$lib/i18n';
 import { onMount, tick } from 'svelte';
 import { subscribeChanges } from '$lib/realtime';
 import { api, download, type Job } from '$lib/api';
 import { printTable } from '$lib/print-table';
 import type { DocumentChecklist, ChecklistSigner, ChecklistItem, ChecklistNewItem } from '$lib/document-checklist';
 let { job }: {job: Job} = $props();
 let checklists = $state<DocumentChecklist[]>([]), selectedId = $state(0);
 const checklist = $derived(checklists.find(c => c.id === selectedId) ?? null);
 const selectedIndex = $derived(checklists.findIndex(c => c.id === selectedId));
 let loading = $state(true), busy = $state(false), error = $state(''), notice = $state('');
 let dialog: HTMLDialogElement;
 let root: HTMLDivElement;
 let pending = false, disposed = false, generation = 0;
 const editingCell = () => root?.contains(document.activeElement) && document.activeElement?.matches('input,textarea');
 let digital = $state(false), internalCount = $state<number | undefined>(0), externalCount = $state<number | undefined>(0);
 let internal = $state<ChecklistSigner[]>([]), external = $state<ChecklistSigner[]>([]), configVersion = $state(0);
 let title = $state(''), editingId = $state(0), configuring = $state(false);
 type DraftItem = {clientId: string; parentKey: string; documentName: string};
 type ModalItem = {key: string; parentKey: string; number: string; depth: number; documentName: string; draft?: DraftItem};
 let draftItems = $state<DraftItem[]>([]), nextDraft = 0;
 let saving: Promise<boolean> | null = null;
 let exporting = $state(false);
 async function exportPDF(asDisplayed = false) {
  if (busy || exporting) return;
  const focused = document.activeElement;
  if (focused instanceof HTMLElement && root.contains(focused)) focused.blur();
  await tick();
  if (saving && !await saving) return;
  exporting = true; error = '';
  try {
   if (asDisplayed) printTable(root.querySelector('table')!, job.name);
   else await download(endpoint + '/pdf');
  }
  catch (e) { error = (e as Error).message; }
  finally { exporting = false; }
 }
 let modalError = $state('');
 const endpoint = $derived('/document-checklists/' + job.id);
 const tableEndpoint = $derived(endpoint + '/tables/' + selectedId);
 const active = $derived(checklist?.signers.filter(s => !s.archived) ?? []);
 const archived = $derived(checklist?.signers.filter(s => s.archived) ?? []);
 const groups = $derived(['internal', 'external'].map(type => ({type, label: type === 'internal' ? 'Paraf Hardcopy' : 'TTD Hardcopy', signers: active.filter(s => s.type === type)})).filter(g => g.signers.length));
 const valid = $derived(!!title.trim() && Number.isInteger(internalCount) && Number.isInteger(externalCount) && (internalCount ?? -1) >= 0 && (externalCount ?? -1) >= 0 && internal.length === internalCount && external.length === externalCount && [...internal, ...external].every(s => s.name.trim() && s.position.trim()) && draftItems.every(item => item.documentName.trim()));
 const modalItems = $derived.by(() => {
  const items: Omit<ModalItem, 'number' | 'depth'>[] = [
   ...(editingId ? checklist?.items ?? [] : []).map(item => ({key: 'i:'+item.id, parentKey: item.parentId ? 'i:'+item.parentId : '', documentName: item.documentName})),
   ...draftItems.map(draft => ({key: 'd:'+draft.clientId, parentKey: draft.parentKey, documentName: draft.documentName, draft}))
  ];
  const children = new Map<string, typeof items>();
  for (const item of items) children.set(item.parentKey, [...children.get(item.parentKey) ?? [], item]);
  const rows: ModalItem[] = [];
  function visit(parentKey = '', prefix = '', depth = 0) {
   (children.get(parentKey) ?? []).forEach((item, index) => {
    const number = prefix ? prefix+'.'+(index+1) : String(index+1);
    rows.push({...item, number, depth}); visit(item.key, number, depth+1);
   });
  }
  visit(); return rows;
 });
 async function load(background = false) {
  if(background && (busy || configuring || editingCell())) { pending = true; return; }
  const request = ++generation;
  pending = false;
  if(!background) loading = true;
  error = '';
  try {
   const result = (await api<{checklists: DocumentChecklist[]}>(endpoint)).checklists;
   if(disposed || request !== generation) return;
   if(background && (busy || configuring || editingCell())) { pending = true; return; }
   checklists = result;
   if (!result.some(c => c.id === selectedId)) selectedId = result[0]?.id ?? 0;
  }
  catch(e) { if(!disposed && request === generation) error = (e as Error).message; }
  finally { if(request === generation) loading = false; }
 }
 async function flush() { await tick(); if(!disposed && pending && !busy && !configuring && !editingCell()) await load(true); }
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
 function configure(create = false) {
  editingId = create ? 0 : selectedId; title = create ? '' : checklist?.title ?? '';
  digital = create ? false : checklist?.requireDigitalSignature ?? false; configVersion = create ? 0 : checklist?.version ?? 0;
  const signers = create ? [] : active;
  internal = signers.filter(s => s.type === 'internal').map(s => ({...s})); external = signers.filter(s => s.type === 'external').map(s => ({...s}));
  internalCount = internal.length; externalCount = external.length; draftItems = []; nextDraft = 0; modalError = ''; configuring = true; dialog.showModal();
 }
 function addDraft(parentKey = '') { draftItems = [...draftItems, {clientId: 'draft-'+(++nextDraft), parentKey, documentName: ''}]; }
 function removeDraft(clientId: string) {
  const removed = new Set(['d:'+clientId]);
  for (const item of draftItems) if (removed.has(item.parentKey)) removed.add('d:'+item.clientId);
  draftItems = draftItems.filter(item => !removed.has('d:'+item.clientId));
 }
 async function saveConfig(e: SubmitEvent) {
  e.preventDefault(); if (!valid || busy) return;
  const signers = [...internal, ...external];
  const removed = editingId && active.some(s => !signers.some(n => n.id === s.id && n.type === s.type));
  if (removed && !confirm($translate('Penandatangan yang dihapus akan diarsipkan beserta status lamanya. Lanjutkan?'))) return;
  busy = true; generation++; modalError = '';
  try {
   const newItems: ChecklistNewItem[] = draftItems.map(item => ({clientId: item.clientId, parentId: item.parentKey.startsWith('i:') ? Number(item.parentKey.slice(2)) : 0, parentClientId: item.parentKey.startsWith('d:') ? item.parentKey.slice(2) : '', documentName: item.documentName}));
   const saved = (await api<{checklist: DocumentChecklist}>(editingId ? endpoint + '/tables/' + editingId : endpoint, {method: editingId ? 'PUT' : 'POST', body: JSON.stringify({title, requireDigitalSignature: digital, version: configVersion, internalCount, externalCount, signers, newItems})})).checklist;
   replace(saved); selectedId = saved.id;
   dialog.close(); notice = 'Tabel checklist tersimpan.';
  } catch(e) { modalError = (e as Error).message; }
  finally { busy = false; void flush(); }
 }
 function replace(saved: DocumentChecklist) {
  const index = checklists.findIndex(c => c.id === saved.id);
  if(index < 0) checklists = [...checklists, saved]; else checklists[index] = saved;
 }
 function header(field: string, fallback: string) { return checklist?.headerLabels?.[field] || $translate(fallback); }
 async function editHeader(field: string, input: HTMLInputElement) {
  const old = field === 'title' ? checklist?.title ?? '' : header(field, '');
  const value = input.value.trim();
  if (old === value) return;
  if (!value || !input.reportValidity() || !await mutate('/headers', 'PATCH', {field, value})) input.value = old;
 }
 function finishHeader(e: KeyboardEvent) { if (e.key === 'Enter') { e.preventDefault(); (e.currentTarget as HTMLInputElement).blur(); } }
 async function navigate(offset: number) {
  if(busy || configuring) return;
  saving = null;
  const focused = document.activeElement;
  if(focused instanceof HTMLElement && root.contains(focused)) focused.blur();
  await tick();
  if(saving && !await saving) return;
  const next = checklists[selectedIndex + offset];
  if(next) { selectedId = next.id; notice = ''; }
 }
 function mutate(path: string, method: string, body?: object): Promise<boolean> {
  if(busy) return Promise.resolve(false);
  saving = performMutation(path, method, body);
  return saving;
 }
 async function performMutation(path: string, method: string, body?: object) {
  if (busy) return false;
  busy = true; generation++; error = ''; notice = '';
  try {
   replace((await api<{checklist: DocumentChecklist}>(tableEndpoint + path, {method, body: body ? JSON.stringify(body) : undefined})).checklist);
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
 async function editSigner(signer: ChecklistSigner, field: 'name' | 'position', input: HTMLTextAreaElement) {
  const old = signer[field], value = input.value.trim();
  const current = checklist;
  if (!current) { input.value = old; return; }
  if (old === value) return;
  if (!input.reportValidity()) { input.value = old; return; }
  const signers = active.map(current => current.id === signer.id ? {...current, [field]: value} : current);
  busy = true; generation++; error = ''; notice = '';
  try {
   replace((await api<{checklist: DocumentChecklist}>(tableEndpoint, {method: 'PUT', body: JSON.stringify({
    title: current.title, requireDigitalSignature: current.requireDigitalSignature, version: current.version,
    internalCount: signers.filter(s => s.type === 'internal').length,
    externalCount: signers.filter(s => s.type === 'external').length, signers
   })})).checklist);
   notice = 'Perubahan tersimpan.';
  } catch(e) { error = (e as Error).message; input.value = old; }
  finally { busy = false; void flush(); }
 }
 async function check(item: ChecklistItem, input: HTMLInputElement, signerId?: number) {
  if (!await mutate('/items/' + item.id, 'PATCH', {field: signerId ? 'signature' : 'digitalSigned', checked: input.checked, ...(signerId ? {signerId} : {})})) input.checked = signerId ? !!item.signatures[signerId] : item.digitalSigned;
 }
</script>
<div class="checklist" bind:this={root} onfocusout={() => void flush()}>
 {#if error}<p role="alert">{$translate(error)}</p><button disabled={busy} onclick={() => load()}>{$translate("Muat ulang checklist")}</button>{/if}
 {#if notice}<p role="status">{$translate(notice)}</p>{/if}
 {#if loading}<p role="status">{$translate("Memuat checklist...")}</p>
 {:else if !checklist && !error}<p>{$translate("Pekerjaan ini belum mempunyai checklist.")}</p><button class="ws-primary" onclick={() => configure(true)}>{$translate("+ Buat Tabel Checklist")}</button>
 {:else if checklist}
  <div class="table-controls">
   <button class="ws-primary" disabled={busy} onclick={() => configure(true)}>{$translate("+ Buat Tabel Checklist")}</button>
   <div class="table-navigation">
   <button disabled={busy || selectedIndex <= 0} aria-label={$translate("Tabel sebelumnya")} onpointerdown={e => e.preventDefault()} onclick={() => navigate(-1)}>&larr;</button>
   <button disabled={busy} onclick={() => configure()}>{$translate("Edit Tabel Checklist")}</button>
   <button disabled={busy || selectedIndex >= checklists.length - 1} aria-label={$translate("Tabel berikutnya")} onpointerdown={e => e.preventDefault()} onclick={() => navigate(1)}>&rarr;</button>
   </div>
   <span class="table-counter" aria-live="polite">{checklist.title} ({selectedIndex + 1} / {checklists.length})</span>
  </div>
  <p>{$translate("Perubahan sel disimpan setelah selesai mengisi.")}</p>
  <button type="button" disabled={busy || exporting} onpointerdown={e => e.preventDefault()} onclick={() => exportPDF()}>{$translate("Ekspor PDF pekerjaan")}</button>
  <button type="button" disabled={busy || exporting} onpointerdown={e => e.preventDefault()} onclick={() => exportPDF(true)}>{$translate('Cetak / Simpan PDF')}</button>
  <div class="table-scroll" role="region" aria-label={$translate('Checklist ')+job.name}>
   <table>
    <thead>
     <tr><th class="table-title" colspan={5 + active.length + (checklist.requireDigitalSignature ? 1 : 0)}><input class="header-input table-title-input" aria-label={$translate('Judul Tabel')} required maxlength="255" value={checklist.title} disabled={busy} onchange={e => editHeader('title', e.currentTarget)} onkeydown={finishHeader}/></th></tr>
     <tr><th rowspan={active.length ? 2 : 1} scope="col"><input class="header-input" aria-label={$translate('No.')} required maxlength="255" value={header('number', 'No.')} disabled={busy} onchange={e => editHeader('number', e.currentTarget)} onkeydown={finishHeader}/></th><th rowspan={active.length ? 2 : 1} scope="col"><input class="header-input" aria-label={$translate('Dokumen')} required maxlength="255" value={header('documentName', 'Dokumen')} disabled={busy} onchange={e => editHeader('documentName', e.currentTarget)} onkeydown={finishHeader}/></th>{#if checklist.requireDigitalSignature}<th rowspan={active.length ? 2 : 1} scope="col"><input class="header-input" aria-label={$translate('TTD Digital')} required maxlength="255" value={header('digitalSignature', 'TTD Digital')} disabled={busy} onchange={e => editHeader('digitalSignature', e.currentTarget)} onkeydown={finishHeader}/></th>{/if}{#if active.length}{#each groups as group}<th colspan={group.signers.length} scope="col"><input class="header-input" aria-label={$translate(group.label)} required maxlength="255" value={header(group.type === 'internal' ? 'internalGroup' : 'externalGroup', group.label)} disabled={busy} onchange={e => editHeader(group.type === 'internal' ? 'internalGroup' : 'externalGroup', e.currentTarget)} onkeydown={finishHeader}/></th>{/each}{/if}<th rowspan={active.length ? 2 : 1} scope="col"><input class="header-input" aria-label={$translate('Posisi Dokumen')} required maxlength="255" value={header('documentPosition', 'Posisi Dokumen')} disabled={busy} onchange={e => editHeader('documentPosition', e.currentTarget)} onkeydown={finishHeader}/></th><th rowspan={active.length ? 2 : 1} scope="col"><input class="header-input" aria-label={$translate('Tanggal Kirim Dokumen')} required maxlength="255" value={header('sentDate', 'Tanggal Kirim Dokumen')} disabled={busy} onchange={e => editHeader('sentDate', e.currentTarget)} onkeydown={finishHeader}/></th><th rowspan={active.length ? 2 : 1} scope="col"><input class="header-input" aria-label={$translate('Catatan')} required maxlength="255" value={header('notes', 'Catatan')} disabled={busy} onchange={e => editHeader('notes', e.currentTarget)} onkeydown={finishHeader}/></th></tr>
     {#if active.length}<tr>{#each groups as group}{#each group.signers as signer (signer.id)}<th class="signer-heading" scope="col"><textarea class="signer-position" aria-label={$translate('Jabatan')+' '+signer.name} required maxlength="255" rows="1" value={signer.position} disabled={busy} onchange={e => editSigner(signer,'position',e.currentTarget)}></textarea><textarea class="signer-name" aria-label={$translate('Nama')+' '+signer.name} required maxlength="255" rows="1" value={signer.name} disabled={busy} onchange={e => editSigner(signer,'name',e.currentTarget)}></textarea></th>{/each}{/each}</tr>{/if}
    </thead>
    <tbody>
     {#each checklist.items as item (item.id)}
      <tr><td>{item.number}</td><td><textarea aria-label={$translate('Dokumen ')+item.number} required maxlength="255" rows="1" value={item.documentName} disabled={busy} onchange={e => edit(item, 'documentName', e.currentTarget)}></textarea></td>
       {#if checklist.requireDigitalSignature}<td class="signature"><input type="checkbox" aria-label={$translate('TTD Digital ')+item.documentName} checked={item.digitalSigned} disabled={busy} onchange={e => check(item,e.currentTarget)}/></td>{/if}
       {#each groups as group}{#each group.signers as signer (signer.id)}<td class="signature"><input type="checkbox" aria-label={signer.name+' — '+item.documentName} checked={!!item.signatures[signer.id]} disabled={busy} onchange={e => check(item,e.currentTarget,signer.id)}/></td>{/each}{/each}
       <td><textarea aria-label={$translate('Posisi ')+item.documentName} maxlength="255" rows="1" value={item.documentPosition} disabled={busy} onchange={e => edit(item,'documentPosition',e.currentTarget)}></textarea></td>
       <td><input type="date" aria-label={$translate('Tanggal kirim ')+item.documentName} value={item.sentDate ?? ''} disabled={busy} onchange={e => edit(item,'sentDate',e.currentTarget)}/></td>
       <td><textarea aria-label={$translate('Catatan ')+item.documentName} maxlength="4000" value={item.notes} disabled={busy} onchange={e => edit(item,'notes',e.currentTarget)}></textarea></td>
      </tr>
     {:else}<tr><td colspan={5 + active.length + (checklist.requireDigitalSignature ? 1 : 0)}>{$translate("Belum ada dokumen. Tambahkan dokumen melalui Edit Tabel Checklist.")}</td></tr>{/each}
    </tbody>
   </table>
  </div>
  {#if archived.length}<details><summary>{$translate("Riwayat penandatangan yang diarsipkan (")}{archived.length})</summary>{#each archived as signer}<p><strong>{signer.position}</strong> — {signer.name} ({$translate(signer.type === 'internal' ? 'Internal' : 'External')})</p><ul>{#each checklist.items as item}<li>{item.documentName}: {item.signatures[signer.id] ? $translate('Sudah ditandatangani') : $translate('Belum ditandatangani')}</li>{/each}</ul>{/each}</details>{/if}
 {/if}
</div>
<dialog bind:this={dialog} aria-labelledby={'config-title-'+job.id} onclose={() => {configuring = false; void flush();}} oncancel={e => {if(busy)e.preventDefault();}}>
 <form onsubmit={saveConfig}>
  <h2 id={'config-title-'+job.id}>{editingId ? $translate('Edit Tabel Checklist') : $translate('Buat Tabel Checklist')} — {job.name}</h2>
  {#if modalError}<p role="alert">{$translate(modalError)}</p>{/if}
  <fieldset disabled={busy}>
   {#if !editingId}<label>{$translate("Judul Tabel")}<input required maxlength="255" bind:value={title}/></label>{/if}
   <label class="digital"><input type="checkbox" bind:checked={digital}/> {$translate("Butuh TTD Digital?")}</label>
   <label>{$translate("Paraf Hardcopy")}<input type="number" min="0" step="1" required bind:value={internalCount} oninput={e => {internalCount=e.currentTarget.valueAsNumber;resize('internal',internalCount);}}/></label>
   {#each internal as signer, i}<fieldset><legend>{$translate("Paraf Hardcopy")} {i+1}</legend><label>{$translate("Jabatan")}<input required maxlength="255" bind:value={signer.position}/></label><label>{$translate("Nama")}<input required maxlength="255" bind:value={signer.name}/></label></fieldset>{/each}
   <label>{$translate("Berapa banyak TTD Hardcopy?")}<input type="number" min="0" step="1" required bind:value={externalCount} oninput={e => {externalCount=e.currentTarget.valueAsNumber;resize('external',externalCount);}}/></label>
   {#each external as signer, i}<fieldset><legend>{$translate("TTD Hardcopy")} {i+1}</legend><label>{$translate("Jabatan")}<input required maxlength="255" bind:value={signer.position}/></label><label>{$translate("Nama")}<input required maxlength="255" bind:value={signer.name}/></label></fieldset>{/each}
   <fieldset class="modal-items"><legend>{$translate("Isi Dokumen")}</legend>
    <button type="button" class="add-root" onclick={() => addDraft()}>{$translate("+ Tambah Isi Dokumen")}</button>
    {#each modalItems as item (item.key)}
     <div class="modal-item" style:--depth={item.depth}>
      <strong>{item.number}</strong>
      {#if item.draft}<input aria-label={$translate('Dokumen ')+item.number} required maxlength="255" bind:value={item.draft.documentName}/><button type="button" aria-label={$translate('Hapus dokumen ')+item.number} onclick={() => removeDraft(item.draft!.clientId)}>{$translate("Hapus")}</button>
      {:else}<span>{item.documentName}</span>{/if}
      <button type="button" class="add-subpoint" aria-label={$translate('Tambah subpoint untuk ')+item.number} onclick={() => addDraft(item.key)}>{$translate("+ Tambah Subpoint")}</button>
     </div>
    {/each}
   </fieldset>
  </fieldset>
  <div class="ws-row"><button type="button" disabled={busy} onclick={() => dialog.close()}>{$translate("Batal")}</button><button class="ws-primary" disabled={busy || !valid}>{busy ? $translate('Menyimpan...') : $translate('Simpan')}</button></div>
 </form>
</dialog>
<style>
 .table-controls{display:grid;gap:.75rem;justify-items:center}.table-controls>.ws-primary{justify-self:start}.table-navigation{display:grid;grid-template-columns:44px minmax(0,1fr) 44px;gap:.5rem;width:min(100%,340px);align-items:stretch}.table-navigation button{min-width:0;min-height:44px;padding:.5rem;overflow-wrap:break-word}.table-counter{text-align:center;overflow-wrap:break-word;max-width:100%}.table-title{text-align:center;font-size:1.1rem}.checklist{padding:1rem;min-width:0}label{display:grid;gap:.35rem}input,textarea{font:inherit;padding:.5rem;border:1px solid var(--his-border);border-radius:6px}input:not([type=checkbox]),textarea{min-width:160px;max-width:100%;box-sizing:border-box}textarea{width:100%;min-height:42px;resize:vertical;field-sizing:content;overflow-wrap:anywhere;white-space:pre-wrap}.table-scroll{overflow-x:auto;max-width:100%;margin:1rem 0}table{border-collapse:collapse;width:100%}th,td{border:1px solid var(--his-border);padding:.55rem;vertical-align:middle}th{background:var(--his-surface);min-width:75px}.header-input{min-width:0!important;width:100%;text-align:center;background:transparent;border-color:transparent;font-weight:inherit}.header-input:focus{background:var(--his-background);border-color:var(--his-primary)}.table-title-input{font-size:1.1rem}.signer-heading{font-weight:400}.signer-heading textarea{display:block;min-width:7rem;text-align:center;background:transparent}.signer-heading .signer-position{font-weight:700}.signer-heading .signer-name{margin-top:.3rem}.signature{text-align:center}input[type=checkbox]{width:20px;height:20px;accent-color:var(--his-primary)}dialog{width:min(620px,calc(100vw - 32px));max-height:85dvh;box-sizing:border-box;overflow:auto;border:1px solid var(--his-border);border-radius:12px;padding:1.5rem}dialog::backdrop{background:#0006}fieldset{border:0;padding:0;margin:1rem 0;display:grid;gap:1rem}fieldset fieldset{border:1px solid var(--his-border);border-radius:8px;padding:1rem}.digital{display:flex;align-items:center}.modal-items{overflow-x:auto}.modal-item{display:grid;grid-template-columns:auto minmax(10rem,1fr) auto;gap:.5rem;align-items:center;margin-left:calc(var(--depth) * 1rem);min-width:22rem}.modal-item .add-subpoint{grid-column:2/-1;justify-self:start}.add-root{justify-self:start}details{margin-top:1rem}button{min-height:40px}button:disabled{opacity:.6}
</style>
