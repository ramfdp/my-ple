<script lang="ts">
 import { onMount } from 'svelte';
 import { api, session } from '$lib/api';
 import { subscribeChanges } from '$lib/realtime';

 type Mode = 'WFO' | 'WFA';
 interface Attendance { id: string; date: string; mode: Mode; clockIn: string; clockOut: string | null; latitude: number | null; longitude: number | null }
 interface History { records: Attendance[]; today: Attendance | null; open: Attendance | null; todayDate: string; officeConfigured: boolean }
 const jakartaDate = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Jakarta', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());
 const formatTime = (value: string) => new Intl.DateTimeFormat('id-ID', { timeZone: 'Asia/Jakarta', dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
 let month = $state(jakartaDate().slice(0, 7));
 let data = $state<History | null>(null), mode = $state<Mode | null>(null);
 let busy = $state(false), loading = $state(true), ready = $state(false);
 let error = $state(''), status = $state('');
 let active = $derived(data?.open ?? data?.today);

 let generation = 0, pending = false;
 async function load(background = false) {
  const request = ++generation;
  if (!background) { ready = false; loading = true; }
  try { const result = await api<History>(`/attendance?month=${encodeURIComponent(month)}`); if(request === generation) { data = result; ready = true; } }
  finally { if(request === generation) loading = false; }
 }
 async function refresh() {
  if (busy || loading) return;
  error = '';
  try { await load(); } catch { error = 'Data absensi belum dapat dimuat. Coba muat ulang sebelum melanjutkan.'; }
 }
 onMount(() => {
  void load().catch(() => { error = 'Data absensi belum dapat dimuat. Coba muat ulang.'; });
  const focus = () => { void refresh(); };
  window.addEventListener('focus', focus);
  const timer = window.setInterval(() => { if (data && data.todayDate !== jakartaDate()) void refresh(); }, 60000);
  const stop = subscribeChanges(['attendance.changed'], async () => {
   if(busy) { pending = true; return; }
   try { await load(true); } catch { error = 'Data absensi belum dapat dimuat. Coba muat ulang.'; }
  }, change => !change.entityId || change.entityId === session()?.user.id);
  return () => { generation++; stop(); window.removeEventListener('focus', focus); window.clearInterval(timer); };
 });

 function locate(): Promise<GeolocationPosition> {
  return new Promise((resolve, reject) => {
   if (!window.isSecureContext || !navigator.geolocation) {
    reject(new Error('Lokasi tidak tersedia. Gunakan browser yang mendukung lokasi melalui HTTPS.')); return;
   }
   navigator.geolocation.getCurrentPosition(resolve, failure => {
    reject(new Error(failure.code === 1 ? 'Izin lokasi ditolak. Aktifkan izin lokasi untuk situs ini, lalu coba lagi.'
     : failure.code === 3 ? 'Pencarian lokasi habis waktu. Aktifkan GPS/lokasi presisi dan coba lagi.'
     : 'Lokasi tidak dapat ditemukan. Aktifkan GPS/lokasi perangkat dan coba lagi.'));
   }, { enableHighAccuracy: true, maximumAge: 0, timeout: 20000 });
  });
 }
 async function clock() {
  if (busy || !ready || loading) return;
  const open = data?.open;
  if (!open && (!mode || data?.today)) return;
  busy = true; error = ''; status = '';
  let submitted = false;
  try {
   if (open) {
    status = 'Menyimpan Clock Out…'; submitted = true;
    await api(`/attendance/${open.id}/clock-out`, { method: 'POST' });
   } else {
    status = 'Mencari lokasi perangkat…';
    const { coords } = await locate();
    status = 'Menyimpan Clock In…'; submitted = true;
    await api('/attendance/clock-in', { method: 'POST', body: JSON.stringify({ mode, latitude: coords.latitude, longitude: coords.longitude, accuracy: coords.accuracy }) });
   }
   status = open ? 'Clock Out tersimpan.' : 'Clock In tersimpan.';
  } catch (failure) {
   status = '';
   error = failure instanceof Error ? failure.message : 'Absensi gagal diproses. Coba lagi.';
  } finally {
   // Reconcile even when the response was lost: the server may have saved the action.
   if (submitted || pending) {
    pending = false;
    try { await load(); }
    catch { error = 'Status penyimpanan belum dapat diperiksa. Muat ulang data sebelum mencoba lagi.'; }
   }
   busy = false;
  }
 }
</script>

<svelte:head><title>Absensi | MyHIS</title><meta name="description" content="Clock In, Clock Out, dan riwayat absensi pribadi." /></svelte:head>

<main class="attendance-page">
 <header><p class="eyebrow">KEHADIRAN</p><h1>Absensi saya</h1><p>Catat kehadiran dan lihat riwayat kerja Anda. Semua waktu dalam WIB.</p></header>
 <section class="clock-panel" aria-labelledby="clock-title" aria-busy={busy || loading}>
  <div class="section-heading"><h2 id="clock-title">{data?.open && data.open.date !== data.todayDate ? 'Absensi belum selesai' : 'Kehadiran hari ini'}</h2>{#if active}<span class="badge">{active.mode}</span>{/if}</div>
  {#if loading && !data}<p role="status">Memuat absensi…</p>
  {:else if active}
   {#if data?.open && data.open.date !== data.todayDate}<p class="notice">Absensi tanggal {data.open.date} masih terbuka. Selesaikan Clock Out sebelum memulai hari ini.</p>{/if}
   <dl><div><dt>Clock In</dt><dd>{formatTime(active.clockIn)}</dd></div><div><dt>Clock Out</dt><dd>{active.clockOut ? formatTime(active.clockOut) : 'Belum Clock Out'}</dd></div></dl>
   {#if data?.open}<button class="primary" disabled={busy || loading || !ready} onclick={clock}>Clock Out</button><p class="hint">Clock Out mencatat waktu saja dan tidak memerlukan lokasi.</p>
   {:else}<p class="complete">Absensi hari ini selesai.</p>{/if}
  {:else if data}
   <p>Pilih lokasi kerja Anda hari ini.</p>
   <div class="modes" role="group" aria-label="Lokasi kerja">
    <button class:selected={mode === 'WFO'} aria-pressed={mode === 'WFO'} disabled={busy || loading || !ready} onclick={() => { mode = 'WFO'; error = ''; status = ''; }}><strong>WFO</strong><span>Work From Office</span></button>
    <button class:selected={mode === 'WFA'} aria-pressed={mode === 'WFA'} disabled={busy || loading || !ready} onclick={() => { mode = 'WFA'; error = ''; status = ''; }}><strong>WFA</strong><span>Work From Anywhere</span></button>
   </div>
   {#if mode}
    <p class="hint">{mode === 'WFO' ? 'Aktifkan GPS/lokasi presisi. Clock In tersedia dalam radius 100 meter dari kantor. Lokasi hanya diperiksa, tidak disimpan.' : 'Aktifkan GPS/lokasi perangkat. Waktu dan lokasi Anda disimpan saat Clock In. Tidak ada pelacakan latar belakang.'}</p>
    {#if mode === 'WFO' && !data.officeConfigured}<p class="notice">Lokasi kantor belum dikonfigurasi. Hubungi administrator untuk absensi WFO.</p>{/if}
    <button class="primary" disabled={busy || loading || !ready || (mode === 'WFO' && !data.officeConfigured)} onclick={clock}>Clock In</button>
   {/if}
  {/if}
  <p class="status" role="status">{status}</p>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  {#if !ready && !loading}<button disabled={busy} onclick={refresh}>Muat ulang data</button>{/if}
  <p class="hint policy">Absensi hanya mencatat waktu Clock In dan Clock Out, tanpa perhitungan keterlambatan atau lembur.</p>
 </section>

 <section class="history-panel" aria-labelledby="history-title" aria-busy={loading}>
  <div class="section-heading"><h2 id="history-title">Riwayat absensi</h2><label>Bulan <input type="month" min="0001-01" max="9999-12" bind:value={month} disabled={busy || loading} onchange={() => { if (!month) month = jakartaDate().slice(0, 7); void refresh(); }} /></label></div>
  {#if loading}<p role="status">Memuat riwayat…</p>
  {:else if !ready}<p>Riwayat belum tersedia. Muat ulang data untuk mencoba lagi.</p>
  {:else if !data?.records.length}<p>Belum ada absensi pada bulan ini.</p>
  {:else}
   <div class="table-scroll" role="region" aria-label="Riwayat absensi bulanan"><table>
    <thead><tr><th scope="col">Tanggal</th><th scope="col">Mode</th><th scope="col">Clock In (WIB)</th><th scope="col">Clock Out (WIB)</th><th scope="col">Lokasi Clock In</th></tr></thead>
    <tbody>{#each data.records as record (record.id)}<tr>
     <td data-label="Tanggal">{record.date}</td><td data-label="Mode"><span class="badge">{record.mode}</span></td><td data-label="Clock In (WIB)">{formatTime(record.clockIn)}</td><td data-label="Clock Out (WIB)">{record.clockOut ? formatTime(record.clockOut) : 'Belum Clock Out'}</td>
     <td data-label="Lokasi Clock In">{#if record.mode === 'WFA' && record.latitude !== null && record.longitude !== null}<a href={`https://www.google.com/maps?q=${record.latitude},${record.longitude}`} target="_blank" rel="noopener noreferrer" aria-label={`Lihat lokasi Clock In ${record.date} di Google Maps (tab baru)`}>Lihat lokasi ↗</a>{:else}Tidak disimpan{/if}</td>
    </tr>{/each}</tbody>
   </table></div>
  {/if}
 </section>
</main>

<style>
 .attendance-page { padding: var(--his-page-padding); max-width: 1200px; margin: auto; }
 header { margin-bottom: 2rem; }
 h1, h2, p { margin-top: 0; }
 h1 { font-family: 'Plus Jakarta Sans', sans-serif; font-size: clamp(1.8rem, 4vw, 2.5rem); letter-spacing: -.04em; margin-bottom: .75rem; }
 h2 { font-size: 1.15rem; margin-bottom: 0; }
 header > p:last-child, .hint, dt { color: var(--his-muted); line-height: 1.6; }
 .eyebrow { color: var(--his-primary); font-size: .75rem; font-weight: 700; letter-spacing: .12em; }
 section { background: white; border: 1px solid var(--his-border); border-radius: 16px; padding: clamp(1rem, 3vw, 2rem); margin-bottom: 1.5rem; }
 .section-heading { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 1rem; margin-bottom: 1.5rem; }
 button, input { border: 1px solid var(--his-border); border-radius: 8px; background: white; color: var(--his-text); padding: .75rem 1rem; min-height: 44px; }
 button:disabled { cursor: not-allowed; opacity: .55; }
 .modes { display: flex; gap: 1rem; margin: 1rem 0; }
 .modes button { flex: 1; max-width: 240px; text-align: left; padding: 1rem; }
 .modes strong, .modes span { display: block; }
 .modes strong { font-size: 1.2rem; margin-bottom: .4rem; }
 .modes span { font-size: .85rem; color: var(--his-muted); }
 .modes .selected { border-color: var(--his-primary); background: var(--his-primary-soft); box-shadow: inset 0 0 0 1px var(--his-primary); }
 .primary { background: var(--his-primary); color: white; border-color: var(--his-primary); min-width: 180px; font-weight: 700; margin-bottom: .75rem; }
 .badge { display: inline-block; background: var(--his-primary-soft); color: var(--his-primary); border-radius: 6px; padding: .3rem .6rem; font-size: .8rem; font-weight: 700; }
 dl { display: flex; flex-wrap: wrap; gap: 1.5rem 3rem; margin: 0 0 1.5rem; }
 dd { margin: .5rem 0 0; font-weight: 600; }
 .notice, .error { padding: 1rem; border-radius: 8px; line-height: 1.5; }
 .notice { background: var(--his-surface); border: 1px solid var(--his-border); }
 .error { background: var(--his-danger-soft); color: var(--his-danger); }
 .status, .complete { color: var(--his-primary); font-weight: 600; }
 .status:empty { display: none; }
 .hint { font-size: .875rem; }
 .policy { border-top: 1px solid var(--his-border); padding-top: 1rem; margin: 1rem 0 0; }
 label { display: flex; align-items: center; flex-wrap: wrap; gap: .75rem; }
 .table-scroll { overflow-x: auto; }
 table { width: 100%; border-collapse: collapse; text-align: left; font-size: .875rem; }
 th, td { padding: 1rem .75rem; border-bottom: 1px solid var(--his-border); white-space: nowrap; }
 th { background: var(--his-surface); }
 a { color: var(--his-primary); text-underline-offset: 3px; }
 @media (max-width: 1100px) {
  table, tbody { display: block; }
  thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  tr { display: grid; grid-template-columns: 1fr 1fr; padding: .5rem 0; border-bottom: 1px solid var(--his-border); }
  td { border: 0; padding: .75rem .25rem; white-space: normal; min-width: 0; }
  td::before { content: attr(data-label); display: block; color: var(--his-muted); font-size: .75rem; margin-bottom: .4rem; }
  td:last-child { grid-column: 1 / -1; }
 }
 @media (max-width: 480px) { .modes { gap: .75rem; } .modes button { padding: .75rem; } .primary { width: 100%; } }
</style>
