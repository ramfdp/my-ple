<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
 import { onMount } from 'svelte';
 import { api, session } from '$lib/api';
 import { subscribeChanges } from '$lib/realtime';

 type Mode = 'WFO' | 'WFA';
 interface Attendance { id: string; date: string; mode: Mode; clockIn: string; clockOut: string | null; latitude: number | null; longitude: number | null; notes: string }
 interface History { records: Attendance[]; today: Attendance | null; open: Attendance | null; todayDate: string; officeConfigured: boolean }
 const jakartaDate = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Jakarta', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());
 const formatTime = (value: string) => new Intl.DateTimeFormat($locale, { timeZone: 'Asia/Jakarta', dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
 let month = $state(jakartaDate().slice(0, 7));
 let data = $state<History | null>(null), mode = $state<Mode | null>(null);
 let busy = $state(false), loading = $state(true), ready = $state(false);
 let notes = $state('');
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
  if (!data?.open && [...notes.trim()].length > 4000) { error = "Catatan maksimal 4.000 karakter."; return; }
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
    await api('/attendance/clock-in', { method: 'POST', body: JSON.stringify({ mode, notes, latitude: coords.latitude, longitude: coords.longitude, accuracy: coords.accuracy }) });
   }
   notes = '';
   status = open ? 'Clock Out tersimpan.' : 'Clock In tersimpan.';
  } catch (failure) {
   status = '';
   error = failure instanceof Error ? failure.message : 'Absensi gagal diproses. Coba lagi.';
  } finally {
   if (submitted || pending) {
    pending = false;
    try { await load(); }
    catch { error = 'Status penyimpanan belum dapat diperiksa. Muat ulang data sebelum mencoba lagi.'; }
   }
   busy = false;
  }
 }
</script>

<svelte:head><title>{$translate("Absensi | MyHIS")}</title><meta name="description" content={$translate("Clock In, Clock Out, dan riwayat absensi pribadi.")} /></svelte:head>

<main class="attendance-page">
 <header><p class="eyebrow">{$translate("KEHADIRAN")}</p><h1>{$translate("Absensi saya")}</h1><p>{$translate("Catat kehadiran dan lihat riwayat kerja Anda. Semua waktu dalam WIB.")}</p></header>
 <section class="clock-panel" aria-labelledby="clock-title" aria-busy={busy || loading}>
  <div class="section-heading"><h2 id="clock-title">{data?.open && data.open.date !== data.todayDate ? $translate('Absensi belum selesai') : $translate('Kehadiran hari ini')}</h2>{#if active}<span class="badge">{active.mode}</span>{/if}</div>
  {#if loading && !data}<p role="status">{$translate("Memuat absensi…")}</p>
  {:else if active}
   {#if data?.open && data.open.date !== data.todayDate}<p class="notice">{$translate("Absensi tanggal")} {data.open.date} {$translate("masih terbuka. Selesaikan Clock Out sebelum memulai hari ini.")}</p>{/if}
   <dl><div><dt>{$translate("Clock In")}</dt><dd>{formatTime(active.clockIn)}</dd></div><div><dt>{$translate("Clock Out")}</dt><dd>{active.clockOut ? formatTime(active.clockOut) : $translate('Belum Clock Out')}</dd></div></dl>
   <p class="attendance-notes"><strong>{$translate("Catatan")}:</strong> <span>{active.notes || "—"}</span></p>
   {#if data?.open}<button class="primary" disabled={busy || loading || !ready} onclick={clock}>{$translate("Clock Out")}</button><p class="hint">{$translate("Clock Out mencatat waktu saja dan tidak memerlukan lokasi.")}</p>
   {:else}<p class="complete">{$translate("Absensi hari ini selesai.")}</p>{/if}
  {:else if data}
   <p>{$translate("Pilih lokasi kerja Anda hari ini.")}</p>
   <div class="modes" role="group" aria-label={$translate("Lokasi kerja")}>
    <button class:selected={mode === 'WFO'} aria-pressed={mode === 'WFO'} disabled={busy || loading || !ready} onclick={() => { mode = 'WFO'; error = ''; status = ''; }}><strong>{$translate("WFO")}</strong><span>{$translate("Work From Office")}</span></button>
    <button class:selected={mode === 'WFA'} aria-pressed={mode === 'WFA'} disabled={busy || loading || !ready} onclick={() => { mode = 'WFA'; error = ''; status = ''; }}><strong>{$translate("WFA")}</strong><span>{$translate("Work From Anywhere")}</span></button>
   </div>
   {#if mode}
    <p class="hint">{mode === 'WFO' ? $translate('Aktifkan GPS/lokasi presisi. Clock In tersedia dalam radius 100 meter dari kantor. Lokasi hanya diperiksa, tidak disimpan.') : $translate('Aktifkan GPS/lokasi perangkat. Waktu dan lokasi Anda disimpan saat Clock In. Tidak ada pelacakan latar belakang.')}</p>
    {#if mode === 'WFO' && !data.officeConfigured}<p class="notice">{$translate("Lokasi kantor belum dikonfigurasi. Hubungi administrator untuk absensi WFO.")}</p>{/if}
    <label class="notes-field">{$translate("Catatan (opsional)")}<textarea rows="3" maxlength="4000" bind:value={notes} disabled={busy || loading || !ready}></textarea></label>
    <button class="primary" disabled={busy || loading || !ready || (mode === 'WFO' && !data.officeConfigured)} onclick={clock}>{$translate("Clock In")}</button>
   {/if}
  {/if}
  <p class="status" role="status">{$translate(status)}</p>
  {#if error}<p class="error" role="alert">{$translate(error)}</p>{/if}
  {#if !ready && !loading}<button disabled={busy} onclick={refresh}>{$translate("Muat ulang data")}</button>{/if}
  <p class="hint policy">{$translate("Absensi hanya mencatat waktu Clock In dan Clock Out, tanpa perhitungan keterlambatan atau lembur.")}</p>
 </section>

 <section class="history-panel" aria-labelledby="history-title" aria-busy={loading}>
  <div class="section-heading"><h2 id="history-title">{$translate("Riwayat absensi")}</h2><label>{$translate("Bulan")} <input type="month" min="0001-01" max="9999-12" bind:value={month} disabled={busy || loading} onchange={() => { if (!month) month = jakartaDate().slice(0, 7); void refresh(); }} /></label></div>
  {#if loading}<p role="status">{$translate("Memuat riwayat…")}</p>
  {:else if !ready}<p>{$translate("Riwayat belum tersedia. Muat ulang data untuk mencoba lagi.")}</p>
  {:else if !data?.records.length}<p>{$translate("Belum ada absensi pada bulan ini.")}</p>
  {:else}
   <div class="table-scroll" role="region" aria-label={$translate("Riwayat absensi bulanan")}><table>
    <thead><tr><th scope="col">{$translate("Tanggal")}</th><th scope="col">{$translate("Mode")}</th><th scope="col">{$translate("Clock In (WIB)")}</th><th scope="col">{$translate("Clock Out (WIB)")}</th><th scope="col">{$translate("Lokasi Clock In")}</th><th scope="col">{$translate("Catatan")}</th></tr></thead>
    <tbody>{#each data.records as record (record.id)}<tr>
     <td data-label={$translate("Tanggal")}>{record.date}</td><td data-label={$translate("Mode")}><span class="badge">{record.mode}</span></td><td data-label={$translate("Clock In (WIB)")}>{formatTime(record.clockIn)}</td><td data-label={$translate("Clock Out (WIB)")}>{record.clockOut ? formatTime(record.clockOut) : $translate('Belum Clock Out')}</td>
     <td data-label={$translate("Lokasi Clock In")}>{#if record.mode === 'WFA' && record.latitude !== null && record.longitude !== null}<a href={`https://www.google.com/maps?q=${record.latitude},${record.longitude}`} target="_blank" rel="noopener noreferrer" aria-label={`Lihat lokasi Clock In ${record.date} di Google Maps (tab baru)`}>{$translate("Lihat lokasi ↗")}</a>{:else}{$translate("Tidak disimpan")}{/if}</td>
     <td class="attendance-notes" data-label={$translate("Catatan")}><span>{record.notes || "—"}</span></td></tr>{/each}</tbody>
    </table></div>
  {/if}
 </section>
</main>

<style>
 @keyframes slideIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }
 @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

 .attendance-notes { white-space: pre-wrap; overflow-wrap: break-word; word-break: break-word; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
 .attendance-notes span { display: block; }
 .notes-field { display: grid; margin-bottom: 1rem; gap: .5rem; }
 textarea { width: 100%; min-width: 0; box-sizing: border-box; padding: 1rem; border: 2px solid var(--his-border); border-radius: 12px; font: inherit; resize: vertical; transition: all 200ms ease; }
 textarea:focus { border-color: var(--his-primary); box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1); outline: none; }
 .attendance-page { padding: var(--his-page-padding); max-width: 1200px; margin: auto; }
 header { margin-bottom: 2rem; animation: slideIn 400ms ease-out; }
 h1, h2, p { margin-top: 0; margin-bottom: 0; }
 h1 { font-family: 'Plus Jakarta Sans', sans-serif; font-size: clamp(1.8rem, 4vw, 2.5rem); letter-spacing: -.04em; margin-bottom: .75rem; }
 h2 { font-size: 1.15rem; }
 header > p:last-child, .hint, dt { color: var(--his-muted); line-height: 1.6; font-size: .95rem; }
 .eyebrow { color: var(--his-primary); font-size: .75rem; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; }
 section { background: linear-gradient(135deg, white 0%, rgba(79, 70, 229, 0.02) 100%); border: 1px solid var(--his-border); border-radius: 16px; padding: clamp(1.5rem, 4vw, 2.5rem); margin-bottom: 1.5rem; box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04); animation: fadeIn 300ms ease-out; }
 .section-heading { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 1rem; margin-bottom: 1.5rem; }
 button, input { border: 2px solid var(--his-border); border-radius: 12px; background: white; color: var(--his-text); padding: .875rem 1.25rem; min-height: 44px; font-size: 1rem; transition: all 200ms ease; font-family: inherit; }
 button, input[type="month"] { cursor: pointer; }
 button:hover:not(:disabled) { border-color: var(--his-primary); background: rgba(79, 70, 229, 0.05); }
 button:active:not(:disabled) { transform: scale(0.98); }
 button:disabled { cursor: not-allowed; opacity: .5; }
 .modes { display: flex; gap: 1rem; margin: 1rem 0; flex-wrap: wrap; }
 .modes button { flex: 1; min-width: 140px; max-width: 240px; text-align: left; padding: 1.25rem; border: 2px solid var(--his-border); transition: all 250ms ease; }
 .modes strong, .modes span { display: block; }
 .modes strong { font-size: 1.2rem; margin-bottom: .5rem; font-weight: 700; }
 .modes span { font-size: .85rem; color: var(--his-muted); }
 .modes .selected { border-color: var(--his-primary); background: linear-gradient(135deg, var(--his-primary-soft) 0%, rgba(79, 70, 229, 0.08) 100%); box-shadow: 0 4px 12px rgba(79, 70, 229, 0.15); transform: translateY(-2px); }
 .modes .selected strong { color: var(--his-primary); }
 .primary { background: linear-gradient(135deg, var(--his-primary) 0%, rgba(79, 70, 229, 0.85) 100%); color: white; border-color: var(--his-primary); min-width: 180px; font-weight: 700; margin-bottom: .75rem; box-shadow: 0 4px 12px rgba(79, 70, 229, 0.25); transition: all 250ms ease; }
 .primary:hover:not(:disabled) { transform: translateY(-2px); box-shadow: 0 6px 16px rgba(79, 70, 229, 0.35); }
 .primary:active:not(:disabled) { transform: translateY(0); }
 .badge { display: inline-block; background: linear-gradient(135deg, var(--his-primary-soft) 0%, rgba(79, 70, 229, 0.1) 100%); color: var(--his-primary); border-radius: 8px; padding: .4rem .8rem; font-size: .8rem; font-weight: 700; border: 1px solid rgba(79, 70, 229, 0.2); }
 dl { display: flex; flex-wrap: wrap; gap: 2rem 3rem; margin: 0 0 1.5rem; padding: 1.5rem; background: rgba(79, 70, 229, 0.04); border-radius: 12px; border-left: 4px solid var(--his-primary); }
 dd { margin: .5rem 0 0; font-weight: 700; font-size: 1.1rem; }
 .notice { padding: 1rem 1.25rem; border-radius: 12px; line-height: 1.6; background: linear-gradient(135deg, var(--his-surface) 0%, rgba(79, 70, 229, 0.04) 100%); border: 2px solid var(--his-border); border-left: 4px solid var(--his-primary); }
 .error { padding: 1rem 1.25rem; border-radius: 12px; line-height: 1.6; background: linear-gradient(135deg, var(--his-danger-soft) 0%, rgba(220, 38, 38, 0.08) 100%); color: var(--his-danger); border: 2px solid rgba(220, 38, 38, 0.2); border-left: 4px solid var(--his-danger); animation: slideIn 250ms ease-out; }
 .status, .complete { color: var(--his-primary); font-weight: 600; animation: fadeIn 200ms ease-out; }
 .status:empty { display: none; }
 .hint { font-size: .875rem; }
 .policy { border-top: 2px solid var(--his-border); padding-top: 1rem; margin: 1.5rem 0 0; font-size: .85rem; }
 label { display: flex; align-items: center; flex-wrap: wrap; gap: .75rem; font-weight: 500; }
 .table-scroll { overflow-x: auto; border-radius: 12px; border: 1px solid var(--his-border); }
 table { width: 100%; border-collapse: collapse; text-align: left; font-size: .875rem; }
 th, td { padding: 1rem; border-bottom: 1px solid var(--his-border); overflow-wrap: break-word; word-break: break-word; }
 th { background: linear-gradient(135deg, var(--his-surface) 0%, rgba(79, 70, 229, 0.05) 100%); font-weight: 700; }
 a { color: var(--his-primary); text-underline-offset: 3px; font-weight: 500; transition: all 150ms ease; }
 a:hover { opacity: .8; }
 @media (max-width: 1100px) {
  table, tbody { display: block; }
  thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  tr { display: grid; grid-template-columns: 1fr 1fr; gap: 0; padding: 1rem 0; border-bottom: 2px solid var(--his-border); margin-bottom: .5rem; background: linear-gradient(135deg, white 0%, rgba(79, 70, 229, 0.02) 100%); border-radius: 12px; }
  td { border: 0; min-width: 0; padding: .75rem 1rem; overflow-wrap: break-word; word-break: break-word; }
  td:last-child { grid-column: 1 / -1; }
  td::before { content: attr(data-label); display: block; color: var(--his-primary); font-size: .75rem; font-weight: 700; margin-bottom: .5rem; text-transform: uppercase; letter-spacing: .05em; }
 }
 @media (max-width: 640px) { .modes { gap: .75rem; } .modes button { padding: 1rem; font-size: .95rem; min-width: 0; } .primary { width: 100%; } dl { gap: 1.5rem; padding: 1rem; } dd { font-size: 1rem; } section { padding: 1.25rem; } }
</style>
