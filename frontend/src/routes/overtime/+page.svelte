<script lang="ts">
    import {onMount} from 'svelte';
    import {api} from '$lib/api';
    import {subscribeChanges} from '$lib/realtime';
    import OvertimeSheetView from '$lib/components/OvertimeSheetView.svelte';
    import NotificationBell from '$lib/components/NotificationBell.svelte';
    import {automaticDayType,calendarDates,defaultOvertimePeriod, loadOvertimeDraft,overtimeStatuses,saveOvertimeDraft,type OvertimeEntry,type OvertimeSheet} from '$lib/overtime';
    import '$lib/styles/workspace.css';
    import {t as translate} from '$lib/i18n';

    type Draft = Pick<OvertimeEntry,'workDate'|'startTime'|'endTime'|'dayType'|'overrideReason'|'project'|'description'> & Partial<Pick<OvertimeEntry,'actualMinutes'|'weightedMinutes'>>;

    let period = $state(defaultOvertimePeriod()),
        sheet = $state<OvertimeSheet|null>(null),
        rows = $state<Record<string,Draft>>({}),
        submissionOpen = $state(false);

    let errorDate = $state('');
    let loading = $state(true),
        busy = $state(false),
        error = $state(''),
        notice = $state('');

    const editable = $derived(sheet?.status === 'DRAFT'||sheet?.status === 'REJECTED');

    const dates = $derived(calendarDates(period));

    const hasEntries = $derived(dates.some(date=>!!(rows[date]?.startTime||rows[date]?.endTime||rows[date]?.project||rows[date]?.description)));



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

            if(!row.startTime||!row.endTime||!row.project.trim()||!row.description.trim()||(row.dayType!==automaticDayType(date)&&!row.overrideReason.trim())){error='Lengkapi data lembur {date}.';errorDate=date;return null}

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


</script>
<svelte:head>
    <title>{$translate('Lembur | MyHIS')}</title>
</svelte:head>

<main class="workspace-page overtime-page">
    <header class="ws-row">
        <div>
            <p class="dashboard-kicker">{$translate('LEMBUR KARYAWAN')}</p>
            <h1>{$translate('Form Rekapitulasi Lembur')}</h1>
            <p>{$translate('Isi jam, project, dan uraian pekerjaan. Perhitungan dilakukan oleh server.')}</p>
        </div>
        <NotificationBell/>
    </header>

    <section class="ws-panel overtime-toolbar">
        <label>{$translate('Periode')}<input type="month" value={period} onchange={(e)=>{
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
            <span class="overtime-status status-{sheet.status.toLowerCase()}">{$translate(overtimeStatuses[sheet.status])}</span>
        {/if}
    </section>

    {#if error}
        <p role="alert" class="overtime-error">{$translate(error,{date:errorDate})}</p>
    {/if}
    {#if notice}
        <p role="status" class="overtime-notice">{$translate(notice)}</p>
    {/if}

    {#if loading}
        <p role="status">{$translate('Memuat lembar lembur…')}</p>
    {:else if sheet}
        <OvertimeSheetView {sheet} {period} bind:rows {editable} {busy} onclear={clearRow} onpersist={persistDraft}/>

        {#if sheet.reviewerNote}
        <p class="ws-panel">
            <strong>{$translate('Catatan Reviewer:')}</strong>
            {sheet.reviewerNote}
        </p>
        {/if}

        <div class="overtime-actions">
            {#if editable}
                <button type="button" onclick={()=>void save()} disabled={busy}>{busy?$translate('Menyimpan…'):$translate('Simpan Draf')}</button>
                <button class="ws-primary" type="button" onclick={submit} disabled={busy||!submissionOpen||!hasEntries}>{$translate('Ajukan lembur')}</button>
            {/if}
        </div>

        {#if editable&&!submissionOpen}
            <p class="window-note">{$translate('Submit tersedia tanggal 26 sampai 5 bulan berikutnya. Draf tetap dapat disimpan.')}</p>
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
    display:grid;
    gap:5px;
    min-width:0;
    max-width:100%;
    overflow-wrap:anywhere
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
}.overtime-actions{
    display:flex;
    flex-wrap:wrap;
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
    .overtime-actions button{flex:1 1 140px}
    .overtime-status{margin-left:0}.overtime-actions{position:sticky;
    bottom:0;
    background:white;
    padding:10px;z-index:2
    }
}
 @media print{.overtime-page{max-width:none;padding:0}.overtime-page>header,.overtime-toolbar,.overtime-error,.overtime-notice,.overtime-actions,.window-note,.overtime-page>.ws-panel{display:none}}
</style>
