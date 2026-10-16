<script lang="ts">
 import { onMount } from 'svelte';
 import { api, currentUser, type Job } from '$lib/api';
 import { subscribeChanges } from '$lib/realtime';
 import type { LessonLearned, RekapSectionLayout } from '$lib/lessons-learned';
 import '$lib/styles/workspace.css';

 const authorized = $derived(['project engineer','admin','management'].includes(($currentUser?.role ?? '').toLowerCase()));
 let jobs = $state<Job[]>([]), lessons = $state<LessonLearned[]>([]), sections = $state<RekapSectionLayout[]>([]);
 let loading = $state(true), busy = $state(false), error = $state(''), notice = $state('');
 let groupFilter = $state(''), projectFilter = $state(''), search = $state(''), jobFilter = $state(0);
 let dialog: HTMLDialogElement;
 let draft = $state<LessonLearned>(blank());
 let generation = 0;
 const shownSections = $derived(sections.filter(s => (!groupFilter || s.group === groupFilter) && (!projectFilter || s.project === projectFilter)));
 const appLessons = $derived(lessons.filter(l => !l.sourceRow && (!jobFilter || l.jobId === jobFilter)));
 const jobName = (id:number) => jobs.find(j => j.id === id)?.name ?? '';
 const finding = (row:number,kind:'failure'|'success') => lessons.find(l => l.sourceRow === row && l.kind === kind);
 function rowMatches(row:number) {
  const term = search.trim().toLocaleLowerCase();
  return !term || (['failure','success'] as const).some(kind => {
   const l = finding(row,kind);
   return l && [l.sourceNumber,l.title,l.internalFactor,l.externalFactor,l.impact,l.action,l.notes].join(' ').toLocaleLowerCase().includes(term);
  });
 }
 function blank():LessonLearned { return {id:0,jobId:jobFilter || 0,sourceGroup:'',sourceProject:'',sourceRow:0,sourceNumber:'',kind:'failure',foundOn:'',title:'',location:'',internalFactor:'',externalFactor:'',impact:'',action:'',status:'open',notes:'',resolvedOn:'',version:0}; }
 async function load(background = false) {
  const request = ++generation;
  if (!authorized) { loading = false; return; }
  if (!background) loading = true;
  error = '';
  try {
   const result = await api<{jobs:Job[];lessons:LessonLearned[];sections:RekapSectionLayout[]}>('/lessons-learned');
   if (request === generation) { jobs = result.jobs; lessons = result.lessons; sections = result.sections; }
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
   lessons = [...lessons.filter(item => item.id !== lesson.id),lesson];
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

{#snippet findingCells(item:LessonLearned|undefined,kind:'failure'|'success',row:number)}
 {#if item}
  <td class="number">{item.sourceNumber}</td>
  <td>{item.title}</td><td>{item.internalFactor}</td><td>{item.externalFactor}</td>
  <td>{item.impact}</td><td>{item.action}</td><td>{item.notes}</td>
  <td class="edit-cell"><button aria-label={'Edit '+(kind === 'failure' ? 'kegagalan' : 'keberhasilan')+' baris '+row} onclick={() => open(item)}>Edit</button></td>
 {:else}
  {#each Array(8) as _}<td class="blank"></td>{/each}
 {/if}
{/snippet}

<svelte:head><title>Evaluasi & Lesson Learned | MyHIS</title></svelte:head>
<main class="workspace-page lessons-page">
 <header>
  <p class="dashboard-kicker">REKAP</p>
  <h1>LESSON LEARNED - PROJECT ENGINEER</h1>
  <p>Rekap temuan kegagalan dan keberhasilan per proyek.</p>
 </header>
 {#if !authorized}<p role="alert">Akses ditolak. Fitur ini tersedia untuk Project Engineer, Admin, dan Management.</p>
 {:else if loading}<p role="status">Memuat evaluasi...</p>
 {:else}
  {#if error}<p role="alert" class="message error">{error}</p>{/if}
  {#if notice}<p role="status" class="message">{notice}</p>{/if}
  <div class="toolbar">
   <label>Kelompok<select bind:value={groupFilter} onchange={() => projectFilter = ''}><option value="">Semua kelompok</option><option value="PENGAWASAN">PENGAWASAN</option><option value="NONPENGAWASAN">NONPENGAWASAN</option></select></label>
   <label>Proyek<select bind:value={projectFilter}><option value="">Semua proyek</option>{#each sections.filter(s => !groupFilter || s.group === groupFilter) as section}<option value={section.project}>{section.project}</option>{/each}</select></label>
   <label>Cari<input type="search" bind:value={search} placeholder="Cari isi REKAP..." /></label>
  </div>
  {#if !shownSections.length}<p class="message">Tidak ada proyek untuk pilihan ini.</p>{/if}
  {#each shownSections as section,i (section.headingRow)}
   {#if i === 0 || shownSections[i-1].group !== section.group}<h2 class="group-title">{section.group}</h2>{/if}
   <section class="project-section" aria-labelledby={'project-'+section.headingRow}>
    <div class="section-heading"><h3 id={'project-'+section.headingRow}>{section.project}</h3><span>{section.rows.length} baris</span></div>
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div class="table-scroll" tabindex="0" role="region" aria-label={'Rekap '+section.project}>
     <table>
      <caption>Evaluasi {section.project}</caption>
      <thead>
       <tr><th colspan="8" scope="colgroup">Identifikasi Kegagalan</th><th colspan="8" scope="colgroup">Identifikasi Keberhasilan</th></tr>
       <tr>
        <th scope="col">No</th><th scope="col">Uraian Masalah</th><th scope="col">Penyebab Internal</th><th scope="col">Penyebab Eksternal</th><th scope="col">Dampak</th><th scope="col">Tindak Lanjut</th><th scope="col">Keterangan</th><th scope="col">Aksi</th>
        <th scope="col">No</th><th scope="col">Uraian Keberhasilan</th><th scope="col">Faktor Internal</th><th scope="col">Faktor Eksternal</th><th scope="col">Dampak</th><th scope="col">Strategi</th><th scope="col">Keterangan</th><th scope="col">Aksi</th>
       </tr>
      </thead>
      <tbody>
       {#each section.rows.filter(rowMatches) as row (row)}
        <tr>{@render findingCells(finding(row,'failure'),'failure',row)}{@render findingCells(finding(row,'success'),'success',row)}</tr>
       {:else}<tr><td colspan="16" class="empty-row">{section.rows.length ? 'Tidak ada temuan yang cocok.' : 'Belum ada temuan pada proyek ini.'}</td></tr>{/each}
      </tbody>
     </table>
    </div>
   </section>
  {/each}
  <p class="source-link"><a href="https://docs.google.com/spreadsheets/d/16jkdK2DqlPGyVWOhfH71mPWU-7RznQiG1CzF1kmmUlI/edit?gid=608317333#gid=608317333" target="_blank" rel="noopener noreferrer">Buka sumber REKAP</a></p>
  <section class="app-section" aria-labelledby="app-title">
   <div class="section-heading"><div><h2 id="app-title">Temuan MyHIS</h2><p>Catatan yang dibuat langsung di aplikasi. Data lama tetap tersimpan.</p></div><button class="primary" onclick={() => open()} disabled={!jobs.length}>+ Tambah temuan</button></div>
   <label class="job-filter">Pekerjaan<select bind:value={jobFilter}><option value={0}>Semua pekerjaan</option>{#each jobs as job}<option value={job.id}>{job.name}</option>{/each}</select></label>
   {#if !appLessons.length}<p class="message">Belum ada temuan MyHIS untuk pilihan ini.</p>
   {:else}<div class="app-list">{#each appLessons as item (item.id)}<article class="ws-panel"><div class="app-head"><strong>{item.kind === 'failure' ? 'Kegagalan / isu' : 'Keberhasilan'}</strong><span>{jobName(item.jobId)}</span></div><h3>{item.title}</h3>{#if item.foundOn}<p><b>Tanggal:</b> {item.foundOn}</p>{/if}{#if item.location}<p><b>Lokasi:</b> {item.location}</p>{/if}{#if item.internalFactor}<p><b>Internal:</b> {item.internalFactor}</p>{/if}{#if item.externalFactor}<p><b>Eksternal:</b> {item.externalFactor}</p>{/if}{#if item.impact}<p><b>Dampak:</b> {item.impact}</p>{/if}{#if item.action}<p><b>Tindak lanjut / strategi:</b> {item.action}</p>{/if}{#if item.notes}<p><b>Keterangan:</b> {item.notes}</p>{/if}{#if item.kind === 'failure'}<p><b>Status:</b> {item.status === 'open' ? 'Terbuka' : item.status === 'in_progress' ? 'Dalam proses' : `Selesai ${item.resolvedOn}`}</p>{/if}<div class="app-actions"><button onclick={() => open(item)}>Edit</button><button disabled={busy} onclick={() => remove(item)}>Hapus</button></div></article>{/each}</div>{/if}
  </section>
 {/if}
</main>

<dialog bind:this={dialog} aria-labelledby="lesson-dialog-title">
 <form onsubmit={save}>
  <div class="dialog-top"><h2 id="lesson-dialog-title">{draft.id ? 'Edit temuan' : 'Tambah temuan'}</h2><button type="button" onclick={() => dialog.close()} aria-label="Tutup">×</button></div>
  {#if draft.sourceRow}<p class="source-note">{draft.sourceGroup} / {draft.sourceProject} · REKAP baris {draft.sourceRow} · No {draft.sourceNumber || '(kosong)'}</p>{/if}
  {#if error}<p role="alert" class="message error">{error}</p>{/if}
  <div class="form-grid">
   {#if !draft.sourceRow}
    <label>Pekerjaan<select required bind:value={draft.jobId}><option value={0} disabled>Pilih pekerjaan</option>{#each jobs as job}<option value={job.id}>{job.name}</option>{/each}</select></label>
    <label>Jenis<select bind:value={draft.kind} onchange={() => { draft.status='open'; draft.resolvedOn=''; }}><option value="failure">Kegagalan / isu</option><option value="success">Keberhasilan</option></select></label>
    <label>Tanggal temuan<input type="date" bind:value={draft.foundOn} /></label>
    <label>Lokasi (opsional)<input maxlength="4000" bind:value={draft.location} /></label>
   {/if}
   <label class="wide">{draft.kind === 'failure' ? 'Uraian masalah' : 'Uraian keberhasilan'}<textarea required maxlength="2000" rows="3" bind:value={draft.title}></textarea></label>
   <label>Faktor internal<textarea maxlength="4000" rows="4" bind:value={draft.internalFactor}></textarea></label>
   <label>Faktor eksternal<textarea maxlength="4000" rows="4" bind:value={draft.externalFactor}></textarea></label>
   <label class="wide">Dampak / analisis<textarea maxlength="4000" rows="4" bind:value={draft.impact}></textarea></label>
   <label class="wide">{draft.kind === 'failure' ? 'Tindak lanjut' : 'Strategi'}<textarea maxlength="4000" rows="4" bind:value={draft.action}></textarea></label>
   {#if !draft.sourceRow && draft.kind === 'failure'}<label>Status<select bind:value={draft.status} onchange={() => { if (draft.status !== 'resolved') draft.resolvedOn=''; }}><option value="open">Terbuka</option><option value="in_progress">Dalam proses</option><option value="resolved">Selesai</option></select></label>{#if draft.status === 'resolved'}<label>Tanggal penyelesaian<input required type="date" bind:value={draft.resolvedOn} /></label>{/if}{/if}
   <label class="wide">Keterangan<textarea maxlength="4000" rows="3" bind:value={draft.notes}></textarea></label>
  </div>
  <div class="dialog-actions"><button type="button" onclick={() => dialog.close()}>Batal</button><button class="primary" disabled={busy || (!draft.sourceRow && !draft.jobId)}>{busy ? 'Menyimpan...' : 'Simpan'}</button></div>
 </form>
</dialog>

<style>
 .lessons-page{max-width:none}.toolbar{display:flex;gap:.75rem;flex-wrap:wrap;margin:1.5rem 0}.toolbar label{flex:1 1 220px}.lessons-page label,dialog label{display:grid;gap:.35rem;font-weight:600;font-size:.9rem}input,select,textarea{width:100%;border:1px solid var(--his-border);border-radius:8px;padding:.65rem;background:#fff;color:var(--his-text)}textarea{resize:vertical}.primary{background:var(--his-primary);color:#fff;border:1px solid var(--his-primary);border-radius:8px;padding:.65rem 1rem;font-weight:700}.group-title{margin:2rem 0 .75rem;padding-bottom:.5rem;border-bottom:2px solid var(--his-primary)}.project-section{margin-bottom:1.5rem;background:#fff;border:1px solid var(--his-border);border-radius:10px;overflow:hidden}.section-heading,.dialog-top,.dialog-actions,.app-head,.app-actions{display:flex;align-items:center;justify-content:space-between;gap:.75rem;flex-wrap:wrap}.section-heading{padding:1rem}.section-heading h3,.section-heading h2{margin:0}.section-heading span,.section-heading p{color:var(--his-muted)}.table-scroll{overflow:auto;max-height:70vh;border-top:1px solid var(--his-border)}table{border-collapse:separate;border-spacing:0;min-width:2600px;width:100%;font-size:.84rem}caption{position:absolute;clip-path:inset(50%);width:1px;height:1px;overflow:hidden}th,td{border-right:1px solid var(--his-border);border-bottom:1px solid var(--his-border);padding:.65rem;vertical-align:top;text-align:left;min-width:145px;max-width:270px;white-space:pre-wrap;overflow-wrap:anywhere}th{background:#edf5ef;position:sticky;top:0;z-index:1}thead tr:first-child th{top:0;background:var(--his-primary);color:#fff;text-align:center}thead tr:nth-child(2) th{top:35px}th:nth-child(8),td:nth-child(8){border-right:3px solid var(--his-primary)}.number{min-width:50px;width:50px}.blank{background:#fafafa}.edit-cell{min-width:70px}.edit-cell button,.app-actions button,.dialog-top button,.dialog-actions button{border:1px solid var(--his-border);border-radius:8px;padding:.5rem .7rem;background:#fff}.empty-row{text-align:center;color:var(--his-muted)}.source-link{margin:1rem 0 2rem}.app-section{border-top:1px solid var(--his-border);padding-top:1rem}.app-section .section-heading{padding:0 0 1rem}.job-filter{max-width:360px;margin-bottom:1rem}.app-list{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,330px),1fr));gap:1rem}.app-list article{white-space:pre-wrap}.app-list h3{margin:.75rem 0}.app-actions{justify-content:flex-end}.message{padding:1rem;background:#fff;border:1px solid var(--his-border);border-radius:8px}.error{color:var(--his-danger);border-color:var(--his-danger-border)}dialog{width:min(760px,calc(100vw - 24px));max-height:90dvh;overflow:auto;border:1px solid var(--his-border);border-radius:12px;padding:1.25rem}dialog::backdrop{background:#0008}.dialog-top button{font-size:1.4rem}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem;margin:1.25rem 0}.wide{grid-column:1/-1}.dialog-actions{justify-content:flex-end}.dialog-actions .primary{background:var(--his-primary);color:#fff}.source-note{color:var(--his-muted)}@media(max-width:650px){.form-grid{grid-template-columns:1fr}.wide{grid-column:auto}.table-scroll{max-height:65vh}}
</style>
