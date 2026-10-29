<script lang="ts">
    import {onMount} from 'svelte';
    import {api} from '$lib/api';
    import {subscribeChanges} from '$lib/realtime';
    import NotificationBell from '$lib/components/NotificationBell.svelte';
    import {automaticDayType,calculateOvertime,calendarDates,defaultOvertimePeriod,hours,loadOvertimeDraft,overtimeStatuses,saveOvertimeDraft,type OvertimeDayType,type OvertimeEntry,type OvertimeSheet} from '$lib/overtime';
    import '$lib/styles/workspace.css';
    
    type Draft = Pick<OvertimeEntry,'workDate'|'startTime'|'endTime'|'dayType'|'overrideReason'|'project'|'description'> & Partial<Pick<OvertimeEntry,'actualMinutes'|'weightedMinutes'>>;
    
    let period = $state(defaultOvertimePeriod()),
        sheet = $state<OvertimeSheet|null>(null),
        rows = $state<Record<string,Draft>>({}),
        submissionOpen = $state(false);

    let loading = $state(true),
        busy = $state(false),
        error = $state(''),
        notice = $state('');

    const editable = $derived(sheet?.status === 'DRAFT'||sheet?.status === 'REJECTED');
    
    const dates = $derived(calendarDates(period));
    
    const hasEntries = $derived(dates.some(date=>!!(rows[date]?.startTime||rows[date]?.endTime||rows[date]?.project||rows[date]?.description)));

    const fmt = new Intl.DateTimeFormat('id-ID',{timeZone:'UTC',weekday:'short',day:'numeric',month:'short'});
    
    function blank(date:string):Draft{
        return{
            workDate:date,
            startTime:'',
            endTime:'',
            dayType:automaticDayType(date),
            overrideReason:'',
            project:'',
            description:''
        }
    }

    function draftKey(item=sheet){
        return item?`overtime:${item.userId}:${period}`:''
    }
    
    function hydrate(item:OvertimeSheet){
        sheet = item;
        const next:Record<string,Draft>={};
        
        for(const date of calendarDates(period))next[date] = blank(date);
        for(const entry of item.entries)next[entry.workDate] = {...entry};
        
        if(item.status==='DRAFT'||item.status==='REJECTED'){
            const saved=loadOvertimeDraft(sessionStorage,draftKey(item));

            if(saved)for(const date of calendarDates(period)){
                const row=saved[date];
                if(row&&typeof row==='object'&&!Array.isArray(row))next[date]={...next[date],...row}
            }
        }
        rows = next
    }

    function persistDraft(){
        if(sheet&&editable)saveOvertimeDraft(sessionStorage,draftKey(),rows)
    }

    async function load(){
        loading = true;
        error='';
        notice='';
        try{
            const result=await api<{sheet:OvertimeSheet; submissionOpen:boolean}>(`/overtime-sheets/${period}`);
            hydrate(result.sheet);
            submissionOpen=result.submissionOpen
        } catch(e){
            error=(e as Error).message
        } finally{
            loading=false
        }
    }

    onMount(()=>{
        void load();
        return subscribeChanges(['overtime.changed'],()=>load())
    });

    async function save(showNotice=true){
        if(!sheet||busy)
        return null;
        const entries=[];
        
        for(const date of dates){
            const row = rows[date];
            const touched = row.startTime||row.endTime||row.project.trim()||row.description.trim();
            
            if(!touched)continue;
            
            if(!row.startTime||!row.endTime||!row.project.trim()||!row.description.trim()||(row.dayType!==automaticDayType(date)&&!row.overrideReason.trim())){error=`Lengkapi data lembur ${date}.`;return null}
            
            entries.push({
                workDate:row.workDate,
                startTime:row.startTime,
                endTime:row.endTime,
                dayType:row.dayType,
                overrideReason:row.
                overrideReason.trim(),
                project:row.project.trim(),
                description:row.description.trim()
            })
        }
        
        busy=true;
        error='';
        notice='';
        try{
            const result=await api<{sheet:OvertimeSheet}>(`/overtime-sheets/${period}`,{method:'PUT',body:JSON.stringify({version:sheet.version,entries})});
            sessionStorage.removeItem(draftKey());
            hydrate(result.sheet);
            if(showNotice)notice='Draf lembur tersimpan.';
            return result.sheet
        } catch(e){
            error=(e as Error).message;
            return null
        } finally{
            busy=false
        }
    }

    async function submit(){
        const saved = await save(false);
        if(!saved)
        return;
        busy=true;
        error='';
        try{
            const result = await api<{sheet:OvertimeSheet}>(`/overtime-sheets/${period}/submit`,{method:'POST',body:JSON.stringify({version:saved.version})});
            hydrate(result.sheet);
            submissionOpen = false;
            notice = 'Lembar lembur berhasil dikirim ke Finance.'
        } catch(e){
            error=(e as Error).message
        } finally{
            busy=false
        }
    }
    function clearRow(date:string){
        rows[date] = blank(date);
        rows={...rows};
        persistDraft()
    }
    
    function total(field:'actualMinutes'|'weightedMinutes'){
        return dates.reduce((sum,date)=>sum+(rows[date]?(calculateOvertime(rows[date].startTime,rows[date].endTime,rows[date].dayType)?.[field]??0):0),0)
    }
</script>
<svelte:head>
    <title>Lembur | MyHIS</title>
</svelte:head>

<main class="workspace-page overtime-page">
    <header class="ws-row">
        <div>
            <p class="dashboard-kicker">LEMBUR KARYAWAN</p>
            <h1>Form Rekapitulasi Lembur</h1>
            <p>Isi jam, project, dan uraian pekerjaan. Perhitungan dilakukan oleh server.</p>
        </div>
        <NotificationBell/>
    </header>

    <section class="ws-panel overtime-toolbar">
        <label>Periode <input type="month" value={period} onchange={(e)=>{
            const v=e.currentTarget.value;
            if(!v)return;
            period=v;
            void load()
            }}/>
        </label>
        {#if sheet}
            <div>
                <strong>{sheet.employeeName}</strong>
                <small>{sheet.employeeRole}</small>
            </div>
            <span class="overtime-status status-{sheet.status.toLowerCase()}">{overtimeStatuses[sheet.status]}</span>
        {/if}
    </section>

    {#if error}
        <p role="alert" class="overtime-error">{error}</p>
    {/if}
    {#if notice}
        <p role="status" class="overtime-notice">{notice}</p>
    {/if}

    {#if loading}
        <p role="status">Memuat lembar lembur…</p>
    {:else if sheet}
        <div class="overtime-table-wrap" oninput={persistDraft} onchange={persistDraft}>
            <table class="overtime-table">
                <thead>
                    <tr>
                        <th>Tanggal</th>
                        <th>Mulai</th>
                        <th>Selesai</th>
                        <th>Jenis hari</th>
                        <th>Project</th>
                        <th>Deskripsi</th>
                        <th>Aktual</th>
                        <th>Bobot</th>
                        <th></th>
                    </tr>
                </thead>
                
                <tbody>
                {#each dates as date}
                    {#if rows[date]}
                    {@const calculation=calculateOvertime(rows[date].startTime,rows[date].endTime,rows[date].dayType)}
                    <tr class:weekend={automaticDayType(date)==='HOLIDAY'}>
                        <th scope="row">{fmt.format(new Date(`${date}T00:00:00Z`))}</th>
                            <td>
                                <input aria-label={`Jam mulai ${date}`} type="time" bind:value={rows[date].startTime} disabled={!editable||busy}/>
                            </td>
                            <td>
                                <input aria-label={`Jam selesai ${date}`} type="time" bind:value={rows[date].endTime} disabled={!editable||busy}/>
                            </td>
                            <td>
                                <select aria-label={`Jenis hari ${date}`} bind:value={rows[date].dayType} disabled={!editable||busy}>
                                    <option value="WORKDAY">Hari kerja</option>
                                    <option value="HOLIDAY">Libur</option>
                                    {#if rows[date].dayType!==automaticDayType(date)}
                                    <input class="override" aria-label={`Alasan perubahan jenis hari ${date}`} placeholder="Alasan override" maxlength="500" bind:value={rows[date].overrideReason} disabled={!editable||busy}/>{/if}
                                </select>
                            </td>
                            <td>
                                <input aria-label={`Project ${date}`} maxlength="255" bind:value={rows[date].project} disabled={!editable||busy}/>
                            </td>
                            <td>
                                <textarea aria-label={`Deskripsi ${date}`} rows="2" maxlength="4000" bind:value={rows[date].description} disabled={!editable||busy}></textarea>
                            </td>
                            <td>
                                {calculation===null?'—':hours(calculation.actualMinutes)}</td><td>{calculation===null?'—':hours(calculation.weightedMinutes)}
                            </td>
                            <td>
                                {#if editable}
                                    <button class="row-clear" type="button" aria-label={`Kosongkan ${date}`} onclick={()=>clearRow(date)} disabled={busy}>×</button>
                                {/if}
                            </td>
                        </tr>
                        {/if}
                    {/each}
                </tbody>
                
                <tfoot>
                    <tr>
                        <th colspan="6">TOTAL</th>
                        <td>{hours(total('actualMinutes'))}</td>
                        <td>{hours(total('weightedMinutes'))}</td>
                        <td></td>
                    </tr>
                </tfoot>
            </table>
        </div>
    
        {#if sheet.reviewerNote}
        <p class="ws-panel">
            <strong>Catatan reviewer:</strong> 
            {sheet.reviewerNote}
        </p>
        {/if}
    
        <div class="overtime-actions">
            {#if editable}
                <button type="button" onclick={()=>void save()} disabled={busy}>{busy?'Menyimpan…':'Simpan Draf'}</button>
                <button class="ws-primary" type="button" onclick={submit} disabled={busy||!submissionOpen||!hasEntries}>Submit Overtime</button>
            {/if}
        </div>
    
        {#if editable&&!submissionOpen}
            <p class="window-note">Submit tersedia tanggal 26 sampai 5 bulan berikutnya. Draf tetap dapat disimpan.</p>
        {/if}
    {/if}
</main>

<style>
.overtime-page{
    max-width:1800px
}.overtime-toolbar{
    display:flex;
    align-items:end;
    gap:24px;
    flex-wrap:wrap;
    margin-bottom:16px
}.overtime-toolbar label,.overtime-toolbar div{
    display:grid;gap:5px
}.overtime-toolbar small{
    color:#64748b
}.overtime-status{
    margin-left:auto;
    padding:7px 10px;
    border-radius:6px;
    background:#e2e8f0
}.status-submitted{
    background:#fef3c7
}.status-approved{
    background:#dcfce7
}.status-rejected{
    background:#fee2e2
}.overtime-table-wrap{
    overflow:auto;
    border:1px solid #dbe3ed;
    border-radius:10px;
    background:white
}.overtime-table{
    border-collapse:collapse;
    width:100%;
    min-width:1250px
}.overtime-table th,.overtime-table td{
    padding:8px;
    border-bottom:1px solid #e5e7eb;
    vertical-align:middle
}.overtime-table thead th{
    position:sticky;
    top:0;z-index:1;
    background:#1e3a5f;
    color:white;
    text-align:center
}.overtime-table tbody th{
    white-space:nowrap;
    text-align:left
}.overtime-table input,.overtime-table select,.overtime-table textarea{
    width:100%;
    min-height:40px
}.overtime-table textarea{
    min-width:280px;
    resize:vertical
}.overtime-table .override{
    margin-top:6px;
    min-width:150px
}.overtime-table .weekend{
    background:#fff8ed
}.overtime-table tfoot{
    font-weight:700;background:#edf3f8
}.row-clear{
    min-width:40px;min-height:40px
}.overtime-actions{
    display:flex;
    justify-content:flex-end;
    gap:12px;margin-top:16px
}.overtime-actions button{
    min-height:44px
}.overtime-error{
    color:#991b1b
}.overtime-notice{
    color:#166534
}.window-note{
    text-align:right;
    color:#64748b
}

@media(max-width:700px){
    .overtime-status{margin-left:0}.overtime-actions{position:sticky;
    bottom:0;
    background:white;
    padding:10px;z-index:2
    }
}
</style>
