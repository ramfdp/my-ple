<script lang="ts">
 import { onMount } from 'svelte';
 import { api, APIError, session } from '$lib/api';

 type Status = 'PRESENT' | 'LEAVE' | 'MISSING';
 interface Employee { userId: string; name: string; role: string; status: Status; leaveTypes: string; mode: string; clockIn: string | null; clockOut: string | null }
 interface Overview { date: string; todayDate: string; employees: Employee[] }
 const jakartaDate = () => new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Jakarta', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date());
 const formatTime = (value: string) => new Intl.DateTimeFormat('id-ID', { timeZone: 'Asia/Jakarta', dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value));
 let date = $state(''), today = $state(jakartaDate()), search = $state(''), filter = $state('');
 let data = $state<Overview | null>(null), allowed = $state(false), loading = $state(true), error = $state('');
 let missingLabel = $derived(data?.date === data?.todayDate ? 'Belum Clock In' : 'Tidak ada Clock In');
 let rows = $derived((data?.employees ?? []).filter(employee => (!filter || employee.status === filter) && employee.name.toLowerCase().includes(search.trim().toLowerCase())));
 let counts = $derived({
  PRESENT: data?.employees.filter(employee => employee.status === 'PRESENT').length ?? 0,
  LEAVE: data?.employees.filter(employee => employee.status === 'LEAVE').length ?? 0,
  MISSING: data?.employees.filter(employee => employee.status === 'MISSING').length ?? 0
 });

 async function load() {
  loading = true; error = ''; data = null;
  try {
   data = await api<Overview>(`/attendance/overview${date ? `?date=${encodeURIComponent(date)}` : ''}`);
   date = data.date; today = data.todayDate;
  } catch (failure) {
   if (failure instanceof APIError && failure.status === 403) allowed = false;
   else error = failure instanceof Error ? failure.message : 'Data absensi belum dapat dimuat. Coba lagi.';
  } finally { loading = false; }
 }
 function refresh() {
  if (!allowed || loading) return;
  if (date === today && jakartaDate() !== today) date = '';
  void load();
 }
 onMount(() => {
  allowed = ['admin', 'management', 'hr'].includes(session()?.user.role?.toLowerCase() ?? '');
  if (allowed) void load(); else loading = false;
  window.addEventListener('focus', refresh);
  return () => window.removeEventListener('focus', refresh);
 });
</script>

<svelte:head><title>Absensi Karyawan | MyHIS</title><meta name="description" content="Rekap kehadiran, izin, dan cuti karyawan per tanggal." /></svelte:head>

<main class="overview-page">
 <header><p class="eyebrow">KEHADIRAN</p><h1>Absensi Karyawan</h1><p>Rekap Clock In dan Clock Out karyawan. Semua waktu dalam WIB, tanpa perhitungan keterlambatan.</p></header>
 {#if !allowed && !loading}
  <p class="notice" role="alert">Akses rekap absensi hanya untuk HR, Management, dan Admin.</p>
 {:else if allowed}
  <section class="overview-panel" aria-labelledby="overview-title" aria-busy={loading}>
   <div class="heading"><h2 id="overview-title">Kehadiran per tanggal</h2><a href="/attendance">Absensi saya</a></div>
   <div class="filters">
    <label>Tanggal<input type="date" min="0001-01-01" max={today} bind:value={date} disabled={loading} onchange={() => void load()} /></label>
    <label>Cari nama<input type="search" bind:value={search} placeholder="Nama karyawan" /></label>
    <label>Status<select aria-label="Status" bind:value={filter}><option value="">Semua status</option><option value="PRESENT">Hadir</option><option value="LEAVE">Izin/Cuti</option><option value="MISSING">{missingLabel}</option></select></label>
    <button disabled={loading} onclick={refresh}>Muat ulang</button>
   </div>
   {#if loading}<p role="status">Memuat absensi karyawan…</p>
   {:else if error}<p class="error" role="alert">{error}</p>
   {:else if data}
    <dl class="summary" aria-label="Ringkasan kehadiran">
     <div><dt>Karyawan aktif</dt><dd>{data.employees.length}</dd></div>
     <div><dt>Hadir</dt><dd>{counts.PRESENT}</dd></div>
     <div><dt>Izin/Cuti</dt><dd>{counts.LEAVE}</dd></div>
     <div><dt>{missingLabel}</dt><dd>{counts.MISSING}</dd></div>
    </dl>
    <p class="hint">Izin/Cuti hanya mencakup pengajuan yang disetujui. Karyawan yang sudah Clock In tetap ditampilkan sebagai Hadir.</p>
    <p class="hint" role="status">Menampilkan {rows.length} dari {data.employees.length} karyawan.</p>
    {#if !data.employees.length}<p>Belum ada karyawan aktif pada tanggal ini.</p>
    {:else if !rows.length}<p>Tidak ada karyawan yang sesuai dengan pencarian atau filter.</p>
    {:else}
     <div class="table-scroll"><table>
      <thead><tr><th scope="col">Karyawan</th><th scope="col">Role</th><th scope="col">Status</th><th scope="col">Mode</th><th scope="col">Clock In (WIB)</th><th scope="col">Clock Out (WIB)</th></tr></thead>
      <tbody>{#each rows as employee (employee.userId)}<tr>
       <td data-label="Karyawan">{employee.name}</td><td data-label="Role">{employee.role}</td>
       <td data-label="Status"><span class:present={employee.status === 'PRESENT'} class="badge">{employee.status === 'PRESENT' ? 'Hadir' : employee.status === 'LEAVE' ? employee.leaveTypes.replace('CUTI', 'Cuti').replace('IZIN', 'Izin') : missingLabel}</span></td>
       <td data-label="Mode">{employee.mode || '—'}</td>
       <td data-label="Clock In (WIB)">{employee.clockIn ? formatTime(employee.clockIn) : '—'}</td>
       <td data-label="Clock Out (WIB)">{employee.clockOut ? formatTime(employee.clockOut) : employee.clockIn ? 'Belum Clock Out' : '—'}</td>
      </tr>{/each}</tbody>
     </table></div>
    {/if}
   {/if}
  </section>
 {/if}
</main>

<style>
 .overview-page { padding: var(--his-page-padding); max-width: 1400px; margin: auto; }
 header { margin-bottom: 2rem; }
 h1, h2, p { margin-top: 0; }
 h1 { font-family: 'Plus Jakarta Sans', sans-serif; font-size: clamp(1.8rem, 4vw, 2.5rem); letter-spacing: -.04em; margin-bottom: .75rem; }
 h2 { font-size: 1.15rem; margin: 0; }
 header > p:last-child, .hint, dt { color: var(--his-muted); line-height: 1.6; }
 .eyebrow { color: var(--his-primary); font-size: .75rem; font-weight: 700; letter-spacing: .12em; }
 section { background: white; border: 1px solid var(--his-border); border-radius: 16px; padding: clamp(1rem, 3vw, 2rem); }
 .heading, .filters { display: flex; flex-wrap: wrap; align-items: center; gap: 1rem; margin-bottom: 1.5rem; }
 .heading { justify-content: space-between; }
 .filters { align-items: end; }
 label { display: grid; gap: .5rem; flex: 1 1 180px; min-width: 0; }
 button, input, select { border: 1px solid var(--his-border); border-radius: 8px; background: white; color: var(--his-text); padding: .75rem; min-height: 44px; }
 button { color: var(--his-primary); font-weight: 600; }
 button:disabled { opacity: .55; cursor: not-allowed; }
 .summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 1rem; margin: 0 0 1.5rem; }
 .summary div { padding: 1rem; background: var(--his-surface); border-radius: 8px; }
 dt { font-size: .85rem; } dd { margin: .5rem 0 0; font-size: 1.75rem; font-weight: 700; }
 .hint { font-size: .875rem; }
 .notice, .error { padding: 1rem; border-radius: 8px; line-height: 1.5; }
 .notice { background: white; border: 1px solid var(--his-border); }
 .error { background: var(--his-danger-soft); color: var(--his-danger); }
 .table-scroll { overflow-x: auto; }
 table { width: 100%; border-collapse: collapse; text-align: left; font-size: .875rem; }
 th, td { padding: 1rem .75rem; border-bottom: 1px solid var(--his-border); overflow-wrap: anywhere; }
 th { background: var(--his-surface); }
 .badge { display: inline-block; background: var(--his-surface); border-radius: 6px; padding: .3rem .6rem; font-weight: 600; }
 .present { background: var(--his-primary-soft); color: var(--his-primary); }
 a { color: var(--his-primary); text-underline-offset: 3px; }
 @media (max-width: 1100px) {
  table, tbody { display: block; }
  thead { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  tr { display: grid; grid-template-columns: 1fr 1fr; border-bottom: 1px solid var(--his-border); padding: .5rem 0; }
  td { border: 0; min-width: 0; padding: .75rem .25rem; }
  td::before { content: attr(data-label); display: block; color: var(--his-muted); font-size: .75rem; margin-bottom: .4rem; }
 }
 @media (max-width: 640px) { .summary { grid-template-columns: repeat(2, minmax(0, 1fr)); } .summary div { padding: .75rem; } }
</style>
