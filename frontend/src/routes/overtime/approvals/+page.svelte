<script lang="ts">
    import {onMount, tick} from 'svelte';
    import {api,currentUser} from '$lib/api';
    import {subscribeChanges} from '$lib/realtime';
    import OvertimeSheetView from '$lib/components/OvertimeSheetView.svelte';
    import NotificationBell from '$lib/components/NotificationBell.svelte';
    import {defaultOvertimePeriod,hours,overtimeStatuses,type OvertimeRosterItem,type OvertimeSheet} from '$lib/overtime';
    import '$lib/styles/workspace.css';
    import {locale,t as translate} from '$lib/i18n';

    const authorized=$derived(['Finance','Admin'].includes($currentUser?.role??''));
    let period = $state(defaultOvertimePeriod()),status=$state(''),employees=$state<OvertimeRosterItem[]>([]),selected=$state<OvertimeSheet|null>(null);

    let loading=$state(true),busy=$state(false),error=$state(''),notice=$state(''),reviewerNote=$state(''),offset=$state(0);
    let detailElement = $state<HTMLElement|null> (null);

    const reviewerNoteDrafts = new Map<string,string>();
    function updateReviewerNote(value:string){
        reviewerNote = value;
        if(selected) reviewerNoteDrafts.set(selected.id,value);
    }

    async function load(background=false){
        if(!authorized){
            loading=false;
            return
        }

        if(!background){
            loading = true;
            selected = null;
            error='';
        }
        try{
            const query = new URLSearchParams({
                period,status,offset:String(offset)
            });
            employees = (await api<{employees:OvertimeRosterItem[]}>
            (`/overtime-approvals?${query}`)).employees
        } catch(e){
            error = (e as Error).message
        } finally{
            loading=false
        }
    }

        onMount(()=>{
            void load();
            return subscribeChanges(['overtime.changed'],()=>load())
        });

        function scrollToDetail(){
            if (!detailElement) return;
            const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
            detailElement.scrollIntoView({behavior:reduced ? 'auto' : 'smooth', block:'start'})
        }

        async function open(item:OvertimeRosterItem){
            if(!item.sheetId||(item.status==='DRAFT'&&$currentUser?.role==='Finance')) return;
            error='';
            try{
                selected=(await api<{sheet:OvertimeSheet}>
                (`/overtime-approvals/${item.sheetId}`)).sheet;
                reviewerNote=reviewerNoteDrafts.get(selected.id)??'';
                await tick();
                scrollToDetail();
            } catch(e){
                error=(e as Error).message
            }
        }

        async function decide(decision:'APPROVED'|'REJECTED'){
            if(!selected||busy||(decision==='REJECTED'&&!reviewerNote.trim())) return;
            busy=true;
            error='';
            notice='';
            try{
                selected=(await api<{sheet:OvertimeSheet}>(`/overtime-approvals/${selected.id}/decision`,{
                    method:'POST',body:JSON.stringify({status:decision,reviewerNote:reviewerNote.trim()})
                })).sheet;
                reviewerNoteDrafts.delete(selected.id);
                notice = decision === 'APPROVED'?'Lembur disetujui.':'Lembur dikembalikan untuk diperbaiki.';
                await load()
            } catch(e){
                error=(e as Error).message
            } finally{
                busy=false
            }
        }
    const time=(value:string|null)=>value?new Intl.DateTimeFormat($locale,{timeZone:'Asia/Jakarta',dateStyle:'medium',timeStyle:'short'}).format(new Date(value)):'—';
</script>
    <svelte:head>
        <title>{$translate('Persetujuan Lembur | MyHIS')}</title>
    </svelte:head>

    <main class="workspace-page approval-page">
    <header class="ws-row">
        <div>
            <p class="dashboard-kicker">FINANCE</p>
            <h1>{$translate('Persetujuan Lembur')}</h1>
            <p>{$translate('Pantau pengisian dan proses lembar lembur bulanan.')}</p>
        </div>
        <NotificationBell/>
    </header>

    {#if !authorized}
        <p role="alert">{$translate('Halaman ini hanya tersedia untuk Finance dan Admin.')}</p>
    {:else}
    <form class="ws-panel approval-filters" onsubmit={(e)=>{
        e.preventDefault();offset=0;void load()
        }
    }>
        <label>{$translate('Periode')}
            <input type="month" bind:value={period} />
        </label>
        <label>{$translate('Status')}
            <select bind:value={status}>
                <option value="">{$translate('Semua')}</option>
                {#each Object.entries(overtimeStatuses) as [value,label]}
                <option {value}>{$translate(label)}</option>
                {/each}
            </select>
        </label>

        <button disabled={loading}>{$translate('Terapkan')}</button></form>

        {#if error}
            <p role="alert" class="approval-error">{$translate(error)}</p>
        {/if}
        {#if notice}
            <p role="status" class="approval-notice">{$translate(notice)}</p>
        {/if}
        {#if loading}
            <p role="status">{$translate('Memuat daftar karyawan…')}</p>
        {:else}
            <div class="approval-grid">
                <section class="ws-panel roster">
                    <h2>{$translate('Status Karyawan')}</h2>
                    <div class="roster-wrap">
                        <table>
                            <thead>
                                <tr>
                                    <th>{$translate('Nama')}</th>
                                    <th>{$translate('Role')}</th>
                                    <th>{$translate('Status')}</th>
                                    <th>{$translate('Dikirim')}</th>
                                    <th></th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each employees as employee (employee.userId)}
                                <tr>
                                    <td>{employee.employeeName}</td>
                                    <td>{employee.employeeRole}</td>
                                    <td>
                                        <span class="status status-{employee.status.toLowerCase()}">{$translate(overtimeStatuses[employee.status])}</span>
                                    </td>
                                    <td>{time(employee.submittedAt)}</td>
                                    <td>{#if employee.sheetId&&!(employee.status==='DRAFT'&&$currentUser?.role==='Finance')}
                                        <button type="button" onclick={()=>open(employee)}>{$translate('Detail')}</button>
                                        {:else}
                                            —
                                        {/if}
                                    </td>
                                </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>

                    <div class="pager">
                        <button disabled={offset===0} onclick={()=>{offset-=50;void load()}}>{$translate('Sebelumnya')}</button>
                        <button disabled={employees.length<50} onclick={()=>{offset+=50;void load()}}>{$translate('Berikutnya')}</button>
                    </div>
                </section>

                {#if selected}
                <section class="ws-panel detail" bind:this={detailElement}>
                    <div class="ws-row">
                        <div>
                            <h2>{selected.employeeName}</h2>
                            <p>{selected.employeeRole} · {selected.period}</p>
                        </div>
                        <button aria-label={$translate('Tutup detail')} onclick={()=>selected=null}>×</button>
                    </div>
                    <dl>
                        <dt>{$translate('Status')}</dt>
                        <dd>{$translate(overtimeStatuses[selected.status])}</dd>
                        <dt>{$translate('Total aktual')}</dt>
                        <dd>{hours(selected.actualMinutes,$locale)} {$translate('jam')}</dd>
                        <dt>{$translate('Total bobot')}</dt>
                        <dd>{hours(selected.weightedMinutes,$locale)} {$translate('jam')}</dd>
                        <dt>{$translate('Dikirim')}</dt>
                        <dd>{time(selected.submittedAt)}</dd>
                        {#if selected.reviewedAt}
                            <dt>{$translate('Reviewer')}</dt>
                            <dd>{selected.reviewerName}</dd>
                            <dt>{$translate('Keputusan')}</dt>
                            <dd>{time(selected.reviewedAt)}</dd>
                            <dt>{$translate('Catatan')}</dt>
                            <dd>{selected.reviewerNote||'—'}</dd>
                        {/if}
                    </dl>

                    <OvertimeSheetView sheet={selected} period={selected.period}/>

                    {#if selected.status==='SUBMITTED'}
                        {#if selected.userId===$currentUser?.id}
                            <p>{$translate('Pengajuan sendiri harus diperiksa Finance/Admin lain.')}</p>
                        {:else}
                            <label class="review-note">{$translate('Catatan Reviewer')}
                                <textarea rows="3" maxlength="4000" bind:value={()=>reviewerNote,updateReviewerNote}></textarea>
                            </label>
                            <div class="decision-actions">
                                <button disabled={busy||!reviewerNote.trim()} onclick={()=>decide('REJECTED')}>{$translate('Tolak')}</button>
                                <button class="ws-primary" disabled={busy} onclick={()=>decide('APPROVED')}>{$translate('Setujui')}</button>
                            </div>
                        {/if}
                    {/if}
                </section>
                {/if}
            </div>
        {/if}
    {/if}
</main>
<style>
.approval-page{
    max-width:1800px
}
.approval-filters{
    display:flex;
    gap:16px;
    align-items:end;
    flex-wrap:wrap;
    margin-bottom:16px
}
.approval-filters label{
    display:grid;
    gap:5px
}.approval-filters button{
    min-width: 120px;
}.approval-grid{
    display:grid;
    grid-template-columns:minmax(0,1fr);
    gap:16px
}
.roster h2, .detail h2{
    margin-top:0
}
.roster-wrap{
    overflow:auto
}.roster table{
    width:100%;
    border-collapse:collapse;
    min-width:650px
}
.roster th, .roster td{
    padding:9px;
    border-bottom:1px solid #e5e7eb;
    text-align:left;
    vertical-align:top
}.roster td button{
    min-width: 80px;
}.detail{
    scroll-margin-top: 12px;
}.status{
    display:inline-block;
    padding:5px 8px;
    border-radius:6px;
    background:#e2e8f0;
    white-space:nowrap
}.status-submitted{
    background:#fef3c7
}.status-approved{
    background:#dcfce7
}.status-rejected{
    background:#fee2e2
}.detail dl{
    display:grid;
    grid-template-columns:max-content minmax(0,1fr);
    gap:6px 12px
}.detail dt{
    font-weight:600
}.detail dd{
    margin:0;
    overflow-wrap:anywhere
}.review-note{
    display:grid;
    gap:6px;
    margin-top:16px
}.decision-actions, .pager{
    display:flex;
    flex-wrap:wrap;
    justify-content:flex-end;
    gap:10px;
    margin-top:14px
}.decision-actions, .pager{
    min-width: 140px;
    min-height:  40px;
    padding-inline: 24px;
}.approval-error{
    color:#991b1b
}.approval-notice{
    color:#166534
}
@media(max-width:1050px){
    .approval-grid{
        grid-template-columns:minmax(0,1fr)
        }
}
@media(max-width: 700px){
    .decision-actions, .pager{
        display: grid;
        grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
        padding-inline: 0;
    }
    .decision-actions button, .pager button{
        width: 100%;
        min-width: 0;
    }
}
 @media print{.approval-page{max-width:none;padding:0}.approval-page>header,.approval-filters,.approval-error,.approval-notice,.roster,.detail>.ws-row,.detail>dl,.review-note,.decision-actions,.detail>p{display:none}.approval-grid{display:block}.approval-page section.detail{padding:0;margin:0;border:0;box-shadow:none;background:white;animation:none;max-width:none;min-width:0;width:100%}}
</style>
