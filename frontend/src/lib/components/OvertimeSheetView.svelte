<script lang="ts">
 import {automaticDayType, calculateOvertime, calendarDates, hours, type OvertimeEntry, type OvertimeSheet} from '$lib/overtime';
 type Draft = Pick<OvertimeEntry, 'workDate'|'startTime'|'endTime'|'dayType'|'overrideReason'|'project'|'description'> & Partial<Pick<OvertimeEntry, 'actualMinutes'|'weightedMinutes'>>;
 let {sheet, period, rows = $bindable({}), editable = false, busy = false, onclear, onpersist}: {
  sheet: OvertimeSheet; period: string; rows?: Record<string,Draft>; editable?: boolean; busy?: boolean;
  onclear?: (date:string)=>void; onpersist?: ()=>void;
 } = $props();
 const dates = $derived(calendarDates(period));
 const savedRows = $derived(Object.fromEntries(sheet.entries.map(entry=>[entry.workDate,entry])));
 const month = $derived(new Intl.DateTimeFormat('id-ID',{month:'long',timeZone:'UTC'}).format(new Date(`${period}-01T00:00:00Z`)));
 const total = $derived(editable ? dates.reduce((sum,date)=>sum+(rows[date] ? calculateOvertime(rows[date].startTime,rows[date].endTime,rows[date].dayType)?.weightedMinutes??0 : 0),0) : sheet.weightedMinutes);
 const widths = [24.6,49.2,51,50.4,110.4,81.6,63,45.6,45.6,45.6,45.6,66.6,58.8,61.2,95.4,323.4,57.6];
</script>

{#snippet unavailable()}
 <span aria-label="Tidak tersedia" title="Tidak tersedia">—</span>
{/snippet}

<!-- svelte-ignore a11y_no_noninteractive_tabindex (Keyboard users need to focus the horizontal scroll region.) -->
<div class="overtime-sheet-scroll" role="region" aria-label="Form rekapitulasi lembur" tabindex="0" oninput={()=>onpersist?.()} onchange={()=>onpersist?.()}>
 <article class="overtime-sheet">
  <header class="sheet-heading">
   <div><h2>FORM REKAPITULASI LEMBUR</h2><strong>Parama Laga Energi</strong></div>
   <img src="/Logo_PLE-removebg-preview-300x233.png" alt="Parama Laga Energi" width="70" height="54" />
  </header>
  <div class="identity"><div><strong>Nama</strong><span>{sheet.employeeName}</span></div><div><strong>Bulan</strong><span>{month}</span><strong>{period.slice(0,4)}</strong></div></div>
  <table>
   <caption class="sr-only">Rekapitulasi lembur {sheet.employeeName}, {month} {period.slice(0,4)}</caption>
   <colgroup>{#each widths as width}<col style={`width:${width/1275.6*100}%`} />{/each}</colgroup>
   <thead>
    <tr>
     <th rowspan="3" scope="col">No</th><th rowspan="3" scope="col">Tanggal</th>
     <th rowspan="3" scope="col">Jam Datang <small>(harus diisi)</small></th><th rowspan="3" scope="col">Jam Pulang <small>(harus diisi)</small></th>
     <th rowspan="3" scope="col">Jam Mulai (Decimal)</th><th rowspan="3" scope="col">Jam Selesai (Decimal)</th><th rowspan="3" scope="col">Aktual Jam</th>
     <th colspan="6" scope="colgroup">Perhitungan untuk Pembayaran Lembur</th>
     <th rowspan="3" scope="col">Nama &amp; Paraf Penanggung jawab</th>
     <th colspan="2" rowspan="2" scope="colgroup" class="project-heading">Project <small>(*harus diisi)</small></th>
     <th rowspan="3" scope="col">Hari Kerja? <small>(ya atau tidak)</small></th>
    </tr>
    <tr><th colspan="2" scope="colgroup">Hari Kerja</th><th colspan="4" scope="colgroup">Hari Libur (Sabtu, Minggu dan Tanggal Merah)</th></tr>
    <tr><th scope="col">Jam 1 (x1)</th><th scope="col">Jam 2-dst (x1)</th><th scope="col">Jam 1–8 (x2)</th><th scope="col">Jam ke-9 (x2)</th><th scope="col">Jam ke-10–dst (x2)</th><th scope="col">Total Jam</th><th scope="col">Kode <small>*harus diisi</small></th><th scope="col">Deskripsi <small>*harus diisi</small></th></tr>
   </thead>
   <tbody>
    {#each dates as date, index (date)}
     {@const row = editable ? rows[date] : savedRows[date]}
     {@const calculation = editable && row ? calculateOvertime(row.startTime,row.endTime,row.dayType) : row}
     <tr>
      <td class="number">{index+1}{#if editable && row}<button class="clear" type="button" aria-label={`Kosongkan ${date}`} title="Kosongkan baris" disabled={busy} onclick={()=>onclear?.(date)}>×</button>{/if}</td>
      <th scope="row" title={date}>{index+1}</th>
      <td>{#if editable && row}<input aria-label={`Jam mulai ${date}`} type="time" bind:value={rows[date].startTime} disabled={busy}/><span class="print-value">{row.startTime}</span>{:else}{row?.startTime??''}{/if}</td>
      <td>{#if editable && row}<input aria-label={`Jam selesai ${date}`} type="time" bind:value={rows[date].endTime} disabled={busy}/><span class="print-value">{row.endTime}</span>{:else}{row?.endTime??''}{/if}</td>
      <td class="calculated">{@render unavailable()}</td><td class="calculated">{@render unavailable()}</td>
      <td class="result">{calculation?.actualMinutes==null?'—':hours(calculation.actualMinutes)}</td>
      {#each [1,2,3,4,5] as bucket}<td class="calculated">{@render unavailable()}</td>{/each}
      <td class="result">{calculation?.weightedMinutes==null?'—':hours(calculation.weightedMinutes)}</td>
      <td>{@render unavailable()}</td>
      <td>{#if editable && row}<input aria-label={`Project ${date}`} maxlength="255" bind:value={rows[date].project} disabled={busy}/><span class="print-value">{row.project}</span>{:else}{row?.project??''}{/if}</td>
      <td class="description">{#if editable && row}<textarea aria-label={`Deskripsi ${date}`} rows="2" maxlength="4000" bind:value={rows[date].description} disabled={busy}></textarea><span class="print-value">{row.description}</span>{:else}{row?.description??''}{/if}
       {#if row?.overrideReason}<small class="override-text">Alasan perubahan hari: {row.overrideReason}</small>{/if}
      </td>
      <td class:holiday={row?.dayType==='HOLIDAY'}>
       {#if editable && row}<select aria-label={`Jenis hari ${date}`} bind:value={rows[date].dayType} disabled={busy}><option value="WORKDAY">ya</option><option value="HOLIDAY">tidak</option></select><span class="print-value">{row.dayType==='WORKDAY'?'ya':'tidak'}</span>
        {#if row.dayType!==automaticDayType(date)}<textarea aria-label={`Alasan perubahan jenis hari ${date}`} placeholder="Alasan perubahan" maxlength="500" rows="2" bind:value={rows[date].overrideReason} disabled={busy}></textarea>{/if}
       {:else if row}{row.dayType==='WORKDAY'?'ya':'tidak'}{/if}
      </td>
     </tr>
    {/each}
   </tbody>
   <tfoot><tr><th colspan="12" scope="row">T O T A L</th><td class="result">{hours(total)}</td><td colspan="4"></td></tr></tfoot>
  </table>
  <p class="formula">Perhitungan Lembur : Jumlah Jam Lembur x (Gaji Pokok+Tunjangan Tetap)/173</p>
  <footer>
   <div class="terms"><h3>Ketentuan</h3><ol><li>Isi form lembur setiap hari agar pekerjaan tidak terlewat.</li><li>Pengajuan tersedia tanggal 26 sampai 5 bulan berikutnya. Draf tetap dapat disimpan.</li><li>Lengkapi jam datang, jam pulang, kode project, deskripsi, dan jenis hari. Sertakan alasan bila jenis hari diubah.</li><li>Informasikan hasil pekerjaan lembur kepada atasan.</li></ol></div>
   <div class="signature"><span>Mengetahui,</span><span>{sheet.reviewedAt?sheet.reviewerName:'(                          )'}</span></div>
   <div class="signature"><span>Yang Membuat,</span><span>{sheet.employeeName}</span></div>
  </footer>
 </article>
</div>

<style>
 .overtime-sheet-scroll{overflow:auto;background:#fff;color:#000;border:1px solid #ccc}
 .overtime-sheet-scroll:focus-visible{outline:3px solid #2563eb;outline-offset:3px}
 .overtime-sheet{width:1700.8px;padding:24px;box-sizing:content-box;font:14.67px Calibri,Arial,sans-serif}
 .sheet-heading{display:flex;justify-content:space-between;align-items:start;margin-bottom:16px}.sheet-heading>div{flex:1;margin-right:80px}.overtime-sheet.overtime-sheet .sheet-heading h2{font:24px Broadway,Georgia,serif;border-bottom:1px solid #000;margin:0 0 2px}.sheet-heading strong{font: bold 14.67px Cambria,Georgia,serif}.sheet-heading img{object-fit:contain}
 .identity{display:flex;justify-content:space-between;margin-bottom:14px;gap:24px}.identity>div{display:flex;border:1px solid #000;min-width:34%;text-align:center}.identity strong{background:#f2f2f2;padding:3px 8px;border-right:1px solid #000}.identity span{padding:3px 12px;flex:1}
 table{border-collapse:collapse;table-layout:fixed;width:100%;border:2px solid #000}th,td{border:1px solid #000;padding:3px;text-align:center;vertical-align:middle;overflow-wrap:anywhere}thead{background:#f2f2f2}thead th{font-size:13px;font-weight:600}thead small{display:block;font-size:10px}thead .project-heading{font-size:32px}.project-heading small{display:inline}.calculated,tbody th,.number{background:#f2f2f2}.result,.holiday{background:#d9d9d9}.result{font-variant-numeric:tabular-nums}tbody th{font-weight:400}tbody tr{height:30px}.description{font-size:12px;white-space:pre-wrap}.number{position:relative}.clear{display:block;padding:0;margin:auto;border:0;background:transparent;color:#555;cursor:pointer;min-height:22px;min-width:22px;font-size:16px}
 .overtime-sheet input,.overtime-sheet select,.overtime-sheet textarea{font:inherit;color:inherit;background:transparent;border:1px solid transparent;border-radius:0;width:100%;min-width:0;box-sizing:border-box;padding:3px 0}.overtime-sheet input[type=time]{font-size:11px;min-height:28px;padding:2px 0}.overtime-sheet input[type=time]::-webkit-calendar-picker-indicator{display:none}.overtime-sheet textarea{field-sizing:content;resize:vertical;min-height:36px;white-space:pre-wrap}.overtime-sheet input:focus-visible,.overtime-sheet select:focus-visible,.overtime-sheet textarea:focus-visible,.overtime-sheet button:focus-visible{outline:2px solid #2563eb;outline-offset:-2px;background:#fff}input:disabled,select:disabled,textarea:disabled{opacity:1}tfoot{background:#f2f2f2;font-weight:bold;border:2px solid #000}.formula{font:21.33px Cambria,Georgia,serif;text-align:center;border:1px solid #000;padding:5px;margin:16px 0 8px}footer{display:grid;grid-template-columns:2fr 1fr 1fr;gap:32px;font-size:12px}.terms h3{font:16px Cambria,Georgia,serif;text-decoration:underline;margin:0}.terms ol{padding-left:18px;margin:4px 0}.signature{display:flex;flex-direction:column;justify-content:space-between;text-align:center;min-height:95px;padding:24px 0 8px}.override-text{display:block;text-align:left;font-size:11px}.print-value{display:none}.sr-only{position:absolute;width:1px;height:1px;padding:0;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
 @page{size:A4 landscape;margin:8mm}
 @media print{:global(.sidebar-container),:global(.mobile-topbar),:global(.mobile-backdrop){display:none!important}:global(.feature-shell){display:block!important}:global(.feature-content){width:100%!important;max-width:none!important;padding:0!important}.overtime-sheet-scroll{overflow:visible;border:0}.overtime-sheet{width:100%;padding:0;font-size:7pt}.sheet-heading{margin-bottom:6px}.overtime-sheet.overtime-sheet .sheet-heading h2{font-size:14pt}.sheet-heading strong{font-size:8pt}.sheet-heading img{width:35px;height:28px}.identity{margin-bottom:6px}.identity strong,.identity span{padding:1px 4px}thead th{font-size:6pt}thead small{font-size:5pt}thead .project-heading{font-size:14pt}th,td{padding:1px}tbody tr{height:12px}tr{break-inside:avoid}.overtime-sheet input,.overtime-sheet select,.overtime-sheet textarea,.clear{display:none}.print-value{display:inline;white-space:pre-wrap}.description{font-size:6pt}.override-text{font-size:5pt}.formula{font-size:10pt;margin:7px 0 4px;padding:2px}footer{font-size:6pt;gap:16px}.terms h3{font-size:8pt}.signature{min-height:50px;padding:10px 0 3px}*{print-color-adjust:exact;-webkit-print-color-adjust:exact}}
</style>
