<script lang="ts">
 import { onMount } from 'svelte';
 import { api, currentUser, type Job } from '$lib/api';
 import { subscribeChanges } from '$lib/realtime';
 import { referenceLessons, type LessonLearned } from '$lib/lessons-learned';
 import '$lib/styles/workspace.css';

 const authorized = $derived(['project engineer','admin','management'].includes(($currentUser?.role ?? '').toLowerCase()));
 let jobs = $state<Job[]>([]), lessons = $state<LessonLearned[]>([]);
 let loading = $state(true), busy = $state(false), error = $state(''), notice = $state('');
 let jobFilter = $state(0), kindFilter = $state(''), search = $state(''), showReference = $state(false);
 let dialog: HTMLDialogElement;
 let draft = $state<LessonLearned>(blank());
 let generation = 0;
 const filtered = $derived(lessons.filter(l => (!jobFilter || l.jobId === jobFilter) && (!kindFilter || l.kind === kindFilter) && `${l.title} ${l.location} ${l.internalFactor} ${l.externalFactor} ${l.impact} ${l.action}`.toLocaleLowerCase().includes(search.trim().toLocaleLowerCase())));
 const jobName = (id:number) => jobs.find(j => j.id === id)?.name ?? '';
 function blank():LessonLearned { return {id:0,jobId:jobFilter || 0,kind:'failure',foundOn:'',title:'',location:'',internalFactor:'',externalFactor:'',impact:'',action:'',status:'open',notes:'',resolvedOn:'',version:0}; }
 async function load(background = false) {
  const request = ++generation;
  if (!authorized) { loading = false; return; }
  if (!background) loading = true;
  error = '';
  try {
   const result = await api<{jobs:Job[];lessons:LessonLearned[]}>('/lessons-learned');
   if (request === generation) { jobs = result.jobs; lessons = result.lessons; }
  } catch (e) { if (request === generation) error = (e as Error).message; }
  finally { if (request === generation) loading = false; }
 }
 onMount(() => {
  void load();
  const stop = subscribeChanges(['lessons-learned.changed','jobs.changed'], () => { if (!dialog?.open && !busy) void load(true); });
  return () => { generation++; stop(); };
 });
 function open(lesson?:LessonLearned) {
  error = ''; notice = '';
  draft = lesson ? structuredClone($state.snapshot(lesson)) : blank();
  dialog.showModal();
 }
 async function save(event:SubmitEvent) {
  event.preventDefault();
  if (busy) return;
  busy = true; error = '';
  const input = $state.snapshot(draft);
  try {
   const {lesson} = await api<{lesson:LessonLearned}>(`/lessons-learned${input.id ? '/'+input.id : ''}`, {method:input.id ? 'PUT' : 'POST',body:JSON.stringify(input)});
   lessons = [...lessons.filter(item => item.id !== lesson.id),lesson].sort((a,b) => a.jobId-b.jobId || a.id-b.id);
   dialog.close(); notice = 'Evaluasi tersimpan.';
  } catch (e) { error = (e as Error).message; }
  finally { busy = false; }
 }
 async function remove(lesson:LessonLearned) {
  if (busy || !confirm(`Hapus temuan “${lesson.title}”?`)) return;
  busy = true; error = '';
  try {
   await api<void>(`/lessons-learned/${lesson.id}?version=${lesson.version}`, {method:'DELETE'});
   lessons = lessons.filter(item => item.id !== lesson.id); notice = 'Evaluasi dihapus.';
  } catch (e) { error = (e as Error).message; }
  finally { busy = false; }
 }
</script>

<svelte:head><title>Evaluasi & Lesson Learned | MyHIS</title></svelte:head>
<main class="workspace-page lessons-page">
 <header>
  <p class="dashboard-kicker">PROJECT ENGINEER</p>
  <h1>Evaluasi & Lesson Learned</h1>
  <p>Catat kegagalan, keberhasilan, faktor penyebab, dampak, dan tindak lanjut untuk setiap pekerjaan.</p>
 </header>
 {#if !authorized}<p role="alert">Akses ditolak. Fitur ini tersedia untuk Project Engineer, Admin, dan Management.</p>
 {:else if loading}<p role="status">Memuat evaluasi...</p>
 {:else}
  {#if error}<p role="alert" class="message error">{error}</p>{/if}
  {#if notice}<p role="status" class="message">{notice}</p>{/if}
  <div class="toolbar">
   <label>Pekerjaan<select bind:value={jobFilter}><option value={0}>Semua pekerjaan</option>{#each jobs as job}<option value={job.id}>{job.name}</option>{/each}</select></label>
   <label>Jenis<select bind:value={kindFilter}><option value="">Semua temuan</option><option value="failure">Kegagalan / isu</option><option value="success">Keberhasilan</option></select></label>
   <label>Cari<input type="search" bind:value={search} placeholder="Judul, faktor, dampak..." /></label>
   <button class="primary" onclick={() => open()} disabled={!jobs.length}>+ Tambah temuan</button>
  </div>
  {#if !jobs.length}<p class="empty">Belum ada pekerjaan Project Engineer.</p>
  {:else if !filtered.length}<p class="empty">Belum ada temuan untuk pilihan ini.</p>
  {:else}
   <div class="lesson-list">
    {#each filtered as lesson (lesson.id)}
     <article class="ws-panel lesson">
      <div class="lesson-top"><span class:failure={lesson.kind === 'failure'} class="tag">{lesson.kind === 'failure' ? 'Kegagalan / isu' : 'Keberhasilan'}</span><span>{jobName(lesson.jobId)}</span>{#if lesson.foundOn}<time datetime={lesson.foundOn}>{lesson.foundOn}</time>{/if}</div>
      <h2>{lesson.title}</h2>
      {#if lesson.location}<p><strong>Lokasi:</strong> {lesson.location}</p>{/if}
      <div class="details">
       {#if lesson.internalFactor}<p><strong>{lesson.kind === 'failure' ? 'Penyebab internal' : 'Pendukung internal'}</strong>{lesson.internalFactor}</p>{/if}
       {#if lesson.externalFactor}<p><strong>{lesson.kind === 'failure' ? 'Penyebab eksternal' : 'Pendukung eksternal'}</strong>{lesson.externalFactor}</p>{/if}
       {#if lesson.impact}<p><strong>Dampak / analisis</strong>{lesson.impact}</p>{/if}
       {#if lesson.action}<p><strong>{lesson.kind === 'failure' ? 'Tindak lanjut' : 'Strategi'}</strong>{lesson.action}</p>{/if}
      </div>
      {#if lesson.notes}<p><strong>Keterangan:</strong> {lesson.notes}</p>{/if}
      <div class="lesson-footer">{#if lesson.kind === 'failure'}<span>Status: {lesson.status === 'open' ? 'Terbuka' : lesson.status === 'in_progress' ? 'Dalam proses' : `Selesai ${lesson.resolvedOn}`}</span>{/if}<div><button onclick={() => open(lesson)}>Edit</button><button class="danger" disabled={busy} onclick={() => remove(lesson)}>Hapus</button></div></div>
     </article>
    {/each}
   </div>
  {/if}
  <section class="reference ws-panel">
   <div class="reference-heading"><div><h2>Rekap pembelajaran dari spreadsheet</h2><p>13 temuan gabungan. Gunakan sebagai acuan; catatan proyek di atas disimpan terpisah.</p></div><button aria-expanded={showReference} onclick={() => showReference = !showReference}>{showReference ? 'Sembunyikan' : 'Lihat rekap'}</button></div>
   {#if showReference}<div class="reference-list">{#each referenceLessons as item}<article><span class:failure={item.kind === 'failure'} class="tag">{item.kind === 'failure' ? 'Kegagalan' : 'Keberhasilan'}</span><h3>{item.title}</h3><p>{item.summary}</p><p><strong>Sumber:</strong> {item.source}</p><p><strong>Tindak lanjut:</strong> {item.action}</p></article>{/each}</div><a href="https://docs.google.com/spreadsheets/d/16jkdK2DqlPGyVWOhfH71mPWU-7RznQiG1CzF1kmmUlI/edit?gid=78634903#gid=78634903" target="_blank" rel="noopener noreferrer">Buka sumber spreadsheet</a>{/if}
  </section>
 {/if}
</main>

<dialog bind:this={dialog} aria-labelledby="lesson-dialog-title">
 <form onsubmit={save}>
  <div class="dialog-top"><h2 id="lesson-dialog-title">{draft.id ? 'Edit temuan' : 'Tambah temuan'}</h2><button type="button" onclick={() => dialog.close()} aria-label="Tutup">×</button></div>
  {#if error}<p role="alert" class="message error">{error}</p>{/if}
  <div class="form-grid">
   <label>Pekerjaan<select required bind:value={draft.jobId}><option value={0} disabled>Pilih pekerjaan</option>{#each jobs as job}<option value={job.id}>{job.name}</option>{/each}</select></label>
   <label>Jenis<select bind:value={draft.kind} onchange={() => { draft.status='open'; draft.resolvedOn=''; }}><option value="failure">Kegagalan / isu</option><option value="success">Keberhasilan</option></select></label>
   <label>Tanggal temuan<input type="date" bind:value={draft.foundOn} /></label>
   <label>Lokasi (opsional)<input maxlength="4000" bind:value={draft.location} /></label>
   <label class="wide">{draft.kind === 'failure' ? 'Uraian masalah' : 'Uraian keberhasilan'}<textarea required maxlength="2000" rows="2" bind:value={draft.title}></textarea></label>
   <label>Faktor internal<textarea maxlength="4000" rows="3" bind:value={draft.internalFactor}></textarea></label>
   <label>Faktor eksternal<textarea maxlength="4000" rows="3" bind:value={draft.externalFactor}></textarea></label>
   <label class="wide">Dampak / analisis<textarea maxlength="4000" rows="3" bind:value={draft.impact}></textarea></label>
   <label class="wide">{draft.kind === 'failure' ? 'Tindak lanjut' : 'Strategi'}<textarea maxlength="4000" rows="3" bind:value={draft.action}></textarea></label>
   {#if draft.kind === 'failure'}<label>Status<select bind:value={draft.status} onchange={() => { if (draft.status !== 'resolved') draft.resolvedOn=''; }}><option value="open">Terbuka</option><option value="in_progress">Dalam proses</option><option value="resolved">Selesai</option></select></label>{#if draft.status === 'resolved'}<label>Tanggal penyelesaian<input required type="date" bind:value={draft.resolvedOn} /></label>{/if}{/if}
   <label class="wide">Keterangan<textarea maxlength="4000" rows="2" bind:value={draft.notes}></textarea></label>
  </div>
  <div class="dialog-actions"><button type="button" onclick={() => dialog.close()}>Batal</button><button class="primary" disabled={busy || !draft.jobId}>{busy ? 'Menyimpan...' : 'Simpan'}</button></div>
 </form>
</dialog>

<style>
 .lessons-page{max-width:1200px}.toolbar{display:flex;align-items:end;gap:.75rem;flex-wrap:wrap;margin:1.5rem 0}.toolbar label{flex:1 1 180px}.toolbar button{min-height:43px}.lessons-page label,dialog label{display:grid;gap:.35rem;font-weight:600;font-size:.9rem}input,select,textarea{width:100%;border:1px solid var(--his-border);border-radius:8px;padding:.65rem;background:#fff;color:var(--his-text)}textarea{resize:vertical}.primary{background:var(--his-primary);color:#fff;border:1px solid var(--his-primary);border-radius:8px;padding:.65rem 1rem;font-weight:700}.lesson-list{display:grid;gap:1rem}.lesson{padding:1.25rem}.lesson h2{margin:.7rem 0;font-size:1.15rem}.lesson p{white-space:pre-wrap}.lesson-top,.lesson-footer,.reference-heading,.dialog-top,.dialog-actions{display:flex;align-items:center;justify-content:space-between;gap:.75rem;flex-wrap:wrap}.lesson-top{font-size:.82rem;color:var(--his-muted)}.tag{display:inline-block;background:var(--his-primary-soft);color:var(--his-primary);padding:.25rem .6rem;border-radius:999px;font-weight:700}.tag.failure{background:var(--his-danger-soft);color:var(--his-danger)}.details{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.25rem 1rem}.details strong{display:block;margin-bottom:.25rem}.lesson-footer{border-top:1px solid var(--his-border);padding-top:.75rem;margin-top:1rem}.lesson-footer div{display:flex;gap:.5rem}.lesson-footer button,.reference button,.dialog-actions button,.dialog-top button{border:1px solid var(--his-border);border-radius:8px;padding:.55rem .75rem;background:#fff}.lesson-footer .danger{color:var(--his-danger)}.reference{margin-top:1.5rem}.reference h2{margin:0}.reference p{margin:.35rem 0}.reference-list{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,300px),1fr));gap:.75rem;margin:1rem 0}.reference-list article{border:1px solid var(--his-border);padding:1rem;border-radius:8px}.reference-list h3{font-size:1rem}.empty,.message{padding:1rem;border-radius:8px;background:#fff;border:1px solid var(--his-border)}.error{color:var(--his-danger);border-color:var(--his-danger-border)}dialog{width:min(760px,calc(100vw - 24px));max-height:90dvh;overflow:auto;border:1px solid var(--his-border);border-radius:12px;padding:1.25rem}dialog::backdrop{background:#0008}.dialog-top h2{margin:.25rem 0}.dialog-top button{font-size:1.4rem}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem;margin:1.25rem 0}.wide{grid-column:1/-1}.dialog-actions{justify-content:flex-end}.dialog-actions .primary{background:var(--his-primary);color:#fff}@media(max-width:650px){.details,.form-grid{grid-template-columns:1fr}.wide{grid-column:auto}}
</style>
