<script lang="ts">
 import { language, locale, t as translate } from '$lib/i18n';
 import {formatActivity} from '$lib/activity';
 import {api} from '$lib/api';
 import {untrack} from 'svelte';
 import {page} from '$app/state';
 import {subscribeChanges} from '$lib/realtime';
 import {statuses,priorities,today,period,addDays,dateString,dateValue,labelDate,type Task,type Workspace,type WorkspaceJob,type Member,type Workload,type Activity} from '$lib/workspace';
 import TaskTable from './TaskTable.svelte';
 import TaskDetail from './TaskDetail.svelte';
 import TaskBoard from './TaskBoard.svelte';
 import TaskTimeline from './TaskTimeline.svelte';
 import TaskCalendar from './TaskCalendar.svelte';
 import WorkloadView from './WorkloadView.svelte';
 import WorkspaceNotes from './WorkspaceNotes.svelte';
 import '$lib/styles/workspace.css';
 let {id}:{id:string}=$props();
 let workspace=$state<Workspace|null>(null),members=$state<Member[]>([]),jobs=$state<WorkspaceJob[]>([]),tasks=$state<Task[]>([]),total=$state(0);
 let loading=$state(true),error=$state(''),view=$state('Overview'),selected=$state<Task|null>(null),detail=$state(false);
 let search=$state(''),assignee=$state(''),jobId=$state(''),status=$state(''),priority=$state(''),start=$state(''),end=$state(''),sort=$state('updated'),offset=$state(0);
 let anchor=$state(today()),scale=$state<'week'|'month'>('month');
 let workload=$state<Workload[]>([]),overview=$state<Record<string,number>>({}),activity=$state<Activity[]>([]);
 let generation=0,opening=0;
 const tabs=['Overview','Table','Board','Timeline','Calendar','Workload','Notes'];
 const currentPeriod=$derived(period(anchor,view==='Calendar'?'month':scale));
 async function refresh(){
  if(!workspace)return;const wid=workspace.id,sequence=++generation;loading=true;error='';
  const q=new URLSearchParams({archived:String(view==='Archive'),search,assignee,jobId,status,priority,start,end,sort,limit:'100',offset:String(offset)});
  if(view==='Overview'){for(const k of ['search','assignee','jobId','priority','start','end'])q.set(k,'');q.set('status','ACTIVE');q.set('sort','due');q.set('offset','0');q.set('limit','8')}
  if(view==='Timeline'||view==='Calendar'){q.set('start',start>currentPeriod.start?start:currentPeriod.start);q.set('end',end&&end<currentPeriod.end?end:currentPeriod.end)}
  const calendar=view==='Calendar';
  if(calendar){q.set('status','');q.set('offset','0')}
  try{if(view==='Archive'){const result=await api<{tasks:Task[];total:number}>(`/workspaces/${wid}/tasks?${q}`);if(sequence===generation){if(offset>0&&offset>=result.total){offset=Math.max(0,Math.floor((result.total-1)/100)*100);await refresh();return}tasks=result.tasks;total=result.total}return}const[t,w,o,a]=await Promise.all([api<{tasks:Task[];total:number}>(`/workspaces/${wid}/tasks?${q}`),api<{workload:Workload[]}>(`/workspaces/${wid}/workload`),api<{overview:Record<string,number>}>(`/workspaces/${wid}/overview`),api<{activity:Activity[]}>(`/workspaces/${wid}/activity`)]);while(calendar&&t.tasks.length<t.total){if(sequence!==generation)return;q.set('offset',String(t.tasks.length));const more=await api<{tasks:Task[];total:number}>(`/workspaces/${wid}/tasks?${q}`);if(!more.tasks.length)break;t.tasks.push(...more.tasks);t.total=more.total}if(sequence!==generation)return;tasks=t.tasks;total=t.total;workload=w.workload;overview=o.overview;activity=a.activity}
  catch(e){if(sequence===generation){error=(e as Error).message;tasks=[];workload=[];overview={};activity=[]}}
  finally{if(sequence===generation)loading=false}
 }
 $effect(()=>{const current=id;let active=true;generation++;opening++;workspace=null;tasks=[];selected=null;detail=false;loading=true;error='';
  void Promise.all([api<{workspace:Workspace;members:Member[]}>('/workspaces/'+current),api<{jobs:WorkspaceJob[]}>('/workspaces/'+current+'/jobs')]).then(([w,j])=>{if(!active)return;workspace=w.workspace;members=w.members;jobs=j.jobs;void refresh()}).catch(e=>{if(active){error=e.message;loading=false}});
  const stop=subscribeChanges(['workspace.changed','jobs.changed'],async change=>{if(change.workspaceId!==undefined&&change.workspaceId!==Number(current))return;if(change.event==='jobs.changed'){const result=await api<{jobs:WorkspaceJob[]}>('/workspaces/'+current+'/jobs');if(active)jobs=result.jobs}if(view!=='Notes')await refresh()});
  return()=>{active=false;generation++;opening++;stop()};
 });
 async function open(task:Pick<Task,'id'>){const sequence=++opening,wid=workspace?.id;try{const response=await api<{task:Task}>('/work-items/'+encodeURIComponent(task.id)+(view==='Archive'?'?archived=true':''));if(sequence!==opening||workspace?.id!==wid)return;if(response.task.workspaceId!==wid)throw new Error('Pekerjaan tidak berada dalam Workspace ini.');selected=response.task;detail=true}catch(e){if(sequence===opening)error=(e as Error).message}}
 $effect(()=>{const taskId=page.url.searchParams.get('task');if(workspace&&taskId)untrack(()=>void open({id:taskId}))});
 function create(){opening++;selected=null;detail=true}
 function closeDetail(){opening++;detail=false}
 function changeView(v:string){if(v==='Archive'||view==='Archive'){search='';assignee='';jobId='';status='';priority='';start='';end='';sort=v==='Archive'?'archived':'updated'}view=v;offset=0;void refresh()}
 function navigate(n:number){if(scale==='week'&&view==='Timeline')anchor=addDays(anchor,n*7);else{const d=dateValue(anchor);anchor=dateString(new Date(Date.UTC(d.getUTCFullYear(),d.getUTCMonth()+n,1)))}offset=0;void refresh()}
</script>
<svelte:head><title>{workspace?.name??$translate('Division')} {$translate("Workspace | MyHIS")}</title></svelte:head>
<main class="workspace-page">
 <a href="/workspace">{$translate("← Semua workspace")}</a>
 <a class="back-link" href="/dashboard">{$translate("← Kembali ke Dashboard")}</a>
 {#if workspace}<header class="ws-row"><div><p class="dashboard-kicker">{$translate("TEAM WORKSPACE")}</p><h1>{workspace.name} {$translate("Workspace")}</h1></div><div class="ws-row"><button aria-pressed={view==='Archive'} onclick={()=>changeView(view==='Archive'?'Table':'Archive')}>{view==='Archive'?$translate('Kembali ke pekerjaan aktif'):$translate('Arsip Pekerjaan')}</button><button class="ws-primary" onclick={create}>{$translate("+ Tambah Pekerjaan")}</button></div></header>
 <nav class="ws-tabs" aria-label={$translate("Tampilan workspace")}>{#each tabs as tab}<button aria-current={view===tab?'page':undefined} onclick={()=>changeView(tab)}>{$translate(tab)}</button>{/each}</nav>
 {#if view!=='Overview'&&view!=='Workload'&&view!=='Notes'}<form class="ws-filters" onsubmit={(e)=>{e.preventDefault();offset=0;void refresh()}}>
 <label>{$translate("Cari")}<input type="search" bind:value={search} placeholder={$translate("Judul atau deskripsi")}/></label><label>{$translate("Assignee")}<select bind:value={assignee}><option value="">{$translate("Semua anggota")}</option>{#each members as m}<option value={m.id}>{m.name}</option>{/each}</select></label>
 <label>{$translate("Project")}<select bind:value={jobId}><option value="">{$translate("Semua project")}</option>{#each jobs as j}<option value={j.id}>{j.label??j.name}</option>{/each}</select></label>
 <label>{$translate("Status")}<select bind:value={status}><option value="">{$translate("Semua status")}</option>{#each Object.entries(statuses) as [key,label]}<option value={key}>{$translate(label)}</option>{/each}</select></label>
 <label>{$translate("Priority")}<select bind:value={priority}><option value="">{$translate("Semua priority")}</option>{#each Object.entries(priorities) as [key,label]}<option value={key}>{$translate(label)}</option>{/each}</select></label>
 <label>{$translate("Dari")}<input type="date" bind:value={start}/></label><label>{$translate("Sampai")}<input type="date" min={start||undefined} bind:value={end}/></label><button type="submit" disabled={loading}>{$translate("Terapkan")}</button><button type="button" onclick={()=>{search='';assignee='';jobId='';status='';priority='';start='';end='';offset=0;void refresh()}}>{$translate("Reset")}</button>
 </form>{/if}{/if}
 {#if error}<div role="alert"><p>{$translate(error)}</p>{#if workspace}<button onclick={refresh}>{$translate("Muat ulang")}</button>{/if}</div>{/if}
 {#if loading}<p role="status">{$translate("Memuat pekerjaan…")}</p>{/if}{#if workspace&&!error}
 {#if view==='Overview'}
 <div class="ws-metrics">{#each [['active',$translate('Active Tasks')],['inProgress',$translate('In Progress')],['needRevision',$translate('Need Revision')],['done',$translate('Completed')],['overdue',$translate('Overdue')]] as [key,label]}<article class="ws-panel"><strong>{overview[key]??0}</strong><p>{$translate(label)}</p></article>{/each}</div>
 <section class="ws-panel"><h2>{$translate("Pekerjaan Mendekati Deadline")}</h2>{#each tasks.filter(t=>t.dueDate&&t.status!=='DONE').sort((a,b)=>a.dueDate.localeCompare(b.dueDate)).slice(0,8) as t}<p><button class="ws-link" onclick={()=>open(t)}>{t.title}</button> · {t.dueDate}</p>{:else}<p>{$translate("Belum ada deadline terdekat.")}</p>{/each}</section>
 <section class="ws-panel"><h2>{$translate("Team Workload Summary")}</h2><p>{workload.filter(w=>w.workloadLevel==='AVAILABLE').length} {$translate("Available ·")} {workload.filter(w=>w.workloadLevel==='NORMAL').length} {$translate("Normal ·")} {workload.filter(w=>w.workloadLevel==='BUSY').length} {$translate("Busy ·")} {workload.filter(w=>w.workloadLevel==='OVERLOADED').length} {$translate("Overloaded")}</p></section>
 <section class="ws-panel"><h2>{$translate("Aktivitas Terbaru")}</h2><ul class="ws-activity">{#each activity.slice(0,10) as a}{@const formatted=formatActivity(a,$language)}<li>{formatted.title}<br/><small>{formatted.timestamp}</small></li>{:else}<li>{$translate("Belum ada aktivitas terbaru.")}</li>{/each}</ul></section>
 {:else if view==='Table'||view==='Archive'}{#if view==='Archive'}<h2>{$translate('Arsip Pekerjaan')}</h2>{/if}<TaskTable archived={view==='Archive'} {tasks} onopen={open} {sort} onsort={(value)=>{sort=value;void refresh()}}/>
 {:else if view==='Board'}<TaskBoard {tasks} onopen={open} onchanged={refresh}/>
 {:else if view==='Timeline'||view==='Calendar'}
 <div class="ws-row ws-period"><div><button onclick={()=>navigate(-1)} aria-label={$translate("Periode sebelumnya")}>←</button> <button onclick={()=>{anchor=today();offset=0;void refresh()}}>{$translate("Today")}</button> <button onclick={()=>navigate(1)} aria-label={$translate("Periode berikutnya")}>→</button></div><strong>{labelDate(currentPeriod.start, $locale)} – {labelDate(currentPeriod.end, $locale)}</strong>
 {#if view==='Timeline'}<label>{$translate("Skala")}<select bind:value={scale} onchange={()=>{offset=0;void refresh()}}><option value="month">{$translate("Month")}</option><option value="week">{$translate("Week")}</option></select></label>{/if}</div>
 {#if view==='Timeline'}<TaskTimeline {tasks} days={currentPeriod.days} onopen={open}/>{:else}<TaskCalendar {tasks} days={currentPeriod.days} onopen={open}/>{/if}
 {:else if view==='Workload'}<WorkloadView {workload} onopen={open}/>
 {:else if view==='Notes'}<WorkspaceNotes {workspace}/>{/if}
 {#if view!=='Overview'&&view!=='Workload'&&view!=='Notes'}{#if !tasks.length}<div class="ws-empty">{view==='Archive'?$translate('Belum ada pekerjaan yang diarsipkan.'):$translate('Belum ada pekerjaan.')} {#if view!=='Archive'}<button onclick={create}>{$translate('Tambah Pekerjaan')}</button>{/if}</div>{/if}{#if view!=='Calendar'}<div class="ws-row ws-pagination"><span>{total} {$translate("pekerjaan ·")} {total?offset+1:0}–{Math.min(offset+tasks.length,total)}</span><div><button disabled={offset===0} onclick={()=>{offset=Math.max(0,offset-100);void refresh()}}>{$translate("Sebelumnya")}</button> <button disabled={offset+100>=total} onclick={()=>{offset+=100;void refresh()}}>{$translate("Berikutnya")}</button></div></div>{/if}{/if}
 {/if}
 {#if detail&&workspace}{#key selected?.id??'new'}<TaskDetail task={selected} {workspace} {members} {jobs} onclose={closeDetail} onsaved={refresh}/>{/key}{/if}
</main>
