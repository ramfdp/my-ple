<script lang="ts">
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
  const q=new URLSearchParams({search,assignee,jobId,status,priority,start,end,sort,limit:'100',offset:String(offset)});
  if(view==='Overview'){for(const k of ['search','assignee','jobId','priority','start','end'])q.set(k,'');q.set('status','ACTIVE');q.set('sort','due');q.set('offset','0');q.set('limit','8')}
  if(view==='Timeline'||view==='Calendar'){q.set('start',start>currentPeriod.start?start:currentPeriod.start);q.set('end',end&&end<currentPeriod.end?end:currentPeriod.end)}
  const calendar=view==='Calendar';
  if(calendar){q.set('status','');q.set('offset','0')}
  try{const[t,w,o,a]=await Promise.all([api<{tasks:Task[];total:number}>(`/workspaces/${wid}/tasks?${q}`),api<{workload:Workload[]}>(`/workspaces/${wid}/workload`),api<{overview:Record<string,number>}>(`/workspaces/${wid}/overview`),api<{activity:Activity[]}>(`/workspaces/${wid}/activity`)]);while(calendar&&t.tasks.length<t.total){if(sequence!==generation)return;q.set('offset',String(t.tasks.length));const more=await api<{tasks:Task[];total:number}>(`/workspaces/${wid}/tasks?${q}`);if(!more.tasks.length)break;t.tasks.push(...more.tasks);t.total=more.total}if(sequence!==generation)return;tasks=t.tasks;total=t.total;workload=w.workload;overview=o.overview;activity=a.activity}
  catch(e){if(sequence===generation){error=(e as Error).message;tasks=[];workload=[];overview={};activity=[]}}
  finally{if(sequence===generation)loading=false}
 }
 $effect(()=>{const current=id;let active=true;generation++;opening++;workspace=null;tasks=[];selected=null;detail=false;loading=true;error='';
  void Promise.all([api<{workspace:Workspace;members:Member[]}>('/workspaces/'+current),api<{jobs:WorkspaceJob[]}>('/workspaces/'+current+'/jobs')]).then(([w,j])=>{if(!active)return;workspace=w.workspace;members=w.members;jobs=j.jobs;void refresh()}).catch(e=>{if(active){error=e.message;loading=false}});
  const stop=subscribeChanges(['workspace.changed','jobs.changed'],async change=>{if(change.workspaceId!==undefined&&change.workspaceId!==Number(current))return;if(change.event==='jobs.changed'){const result=await api<{jobs:WorkspaceJob[]}>('/workspaces/'+current+'/jobs');if(active)jobs=result.jobs}if(view!=='Notes')await refresh()});
  return()=>{active=false;generation++;opening++;stop()};
 });
 async function open(task:Pick<Task,'id'>){const sequence=++opening,wid=workspace?.id;try{const response=await api<{task:Task}>('/work-items/'+encodeURIComponent(task.id));if(sequence!==opening||workspace?.id!==wid)return;if(response.task.workspaceId!==wid)throw new Error('Pekerjaan tidak berada dalam Workspace ini.');selected=response.task;detail=true}catch(e){if(sequence===opening)error=(e as Error).message}}
 $effect(()=>{const taskId=page.url.searchParams.get('task');if(workspace&&taskId)untrack(()=>void open({id:taskId}))});
 function create(){opening++;selected=null;detail=true}
 function closeDetail(){opening++;detail=false}
 function changeView(v:string){view=v;offset=0;void refresh()}
 function navigate(n:number){if(scale==='week'&&view==='Timeline')anchor=addDays(anchor,n*7);else{const d=dateValue(anchor);anchor=dateString(new Date(Date.UTC(d.getUTCFullYear(),d.getUTCMonth()+n,1)))}offset=0;void refresh()}
</script>
<svelte:head><title>{workspace?.name??'Division'} Workspace | MyHIS</title></svelte:head>
<main class="workspace-page">
 <a href="/workspace">← Semua workspace</a>
 <a class="back-link" href="/dashboard">← Kembali ke Dashboard</a>
 {#if workspace}<header class="ws-row"><div><p class="dashboard-kicker">TEAM WORKSPACE</p><h1>{workspace.name} Workspace</h1></div><button class="ws-primary" onclick={create}>+ Tambah Pekerjaan</button></header>
 <nav class="ws-tabs" aria-label="Tampilan workspace">{#each tabs as tab}<button aria-current={view===tab?'page':undefined} onclick={()=>changeView(tab)}>{tab}</button>{/each}</nav>
 {#if view!=='Overview'&&view!=='Workload'&&view!=='Notes'}<form class="ws-filters" onsubmit={(e)=>{e.preventDefault();offset=0;void refresh()}}>
 <label>Cari<input type="search" bind:value={search} placeholder="Judul atau deskripsi"/></label><label>Assignee<select bind:value={assignee}><option value="">Semua anggota</option>{#each members as m}<option value={m.id}>{m.name}</option>{/each}</select></label>
 <label>Project<select bind:value={jobId}><option value="">Semua project</option>{#each jobs as j}<option value={j.id}>{j.label??j.name}</option>{/each}</select></label>
 <label>Status<select bind:value={status}><option value="">Semua status</option>{#each Object.entries(statuses) as [key,label]}<option value={key}>{label}</option>{/each}</select></label>
 <label>Priority<select bind:value={priority}><option value="">Semua priority</option>{#each Object.entries(priorities) as [key,label]}<option value={key}>{label}</option>{/each}</select></label>
 <label>Dari<input type="date" bind:value={start}/></label><label>Sampai<input type="date" min={start||undefined} bind:value={end}/></label><button type="submit" disabled={loading}>Terapkan</button><button type="button" onclick={()=>{search='';assignee='';jobId='';status='';priority='';start='';end='';offset=0;void refresh()}}>Reset</button>
 </form>{/if}{/if}
 {#if error}<div role="alert"><p>{error}</p>{#if workspace}<button onclick={refresh}>Muat ulang</button>{/if}</div>{/if}
 {#if loading}<p role="status">Memuat pekerjaan…</p>{/if}{#if workspace&&!error}
 {#if view==='Overview'}
 <div class="ws-metrics">{#each [['active','Active Tasks'],['inProgress','In Progress'],['needRevision','Need Revision'],['done','Completed'],['overdue','Overdue']] as [key,label]}<article class="ws-panel"><strong>{overview[key]??0}</strong><p>{label}</p></article>{/each}</div>
 <section class="ws-panel"><h2>Pekerjaan Mendekati Deadline</h2>{#each tasks.filter(t=>t.dueDate&&t.status!=='DONE').sort((a,b)=>a.dueDate.localeCompare(b.dueDate)).slice(0,8) as t}<p><button class="ws-link" onclick={()=>open(t)}>{t.title}</button> · {t.dueDate}</p>{:else}<p>Belum ada deadline terdekat.</p>{/each}</section>
 <section class="ws-panel"><h2>Team Workload Summary</h2><p>{workload.filter(w=>w.workloadLevel==='AVAILABLE').length} Available · {workload.filter(w=>w.workloadLevel==='NORMAL').length} Normal · {workload.filter(w=>w.workloadLevel==='BUSY').length} Busy · {workload.filter(w=>w.workloadLevel==='OVERLOADED').length} Overloaded</p></section>
 <section class="ws-panel"><h2>Aktivitas Terbaru</h2><ul class="ws-activity">{#each activity.slice(0,10) as a}{@const formatted=formatActivity(a)}<li>{formatted.title}<br/><small>{formatted.timestamp}</small></li>{:else}<li>Belum ada aktivitas terbaru.</li>{/each}</ul></section>
 {:else if view==='Table'}<TaskTable {tasks} onopen={open} {sort} onsort={(value)=>{sort=value;void refresh()}}/>
 {:else if view==='Board'}<TaskBoard {tasks} onopen={open} onchanged={refresh}/>
 {:else if view==='Timeline'||view==='Calendar'}
 <div class="ws-row ws-period"><div><button onclick={()=>navigate(-1)} aria-label="Periode sebelumnya">←</button> <button onclick={()=>{anchor=today();offset=0;void refresh()}}>Today</button> <button onclick={()=>navigate(1)} aria-label="Periode berikutnya">→</button></div><strong>{labelDate(currentPeriod.start)} – {labelDate(currentPeriod.end)}</strong>
 {#if view==='Timeline'}<label>Skala<select bind:value={scale} onchange={()=>{offset=0;void refresh()}}><option value="month">Month</option><option value="week">Week</option></select></label>{/if}</div>
 {#if view==='Timeline'}<TaskTimeline {tasks} days={currentPeriod.days} onopen={open}/>{:else}<TaskCalendar {tasks} days={currentPeriod.days} onopen={open}/>{/if}
 {:else if view==='Workload'}<WorkloadView {workload} onopen={open}/>
 {:else if view==='Notes'}<WorkspaceNotes {workspace}/>{/if}
 {#if view!=='Overview'&&view!=='Workload'&&view!=='Notes'}{#if !tasks.length}<div class="ws-empty">Belum ada pekerjaan. <button onclick={create}>Tambah Pekerjaan</button></div>{/if}{#if view!=='Calendar'}<div class="ws-row ws-pagination"><span>{total} pekerjaan · {total?offset+1:0}–{Math.min(offset+tasks.length,total)}</span><div><button disabled={offset===0} onclick={()=>{offset=Math.max(0,offset-100);void refresh()}}>Sebelumnya</button> <button disabled={offset+100>=total} onclick={()=>{offset+=100;void refresh()}}>Berikutnya</button></div></div>{/if}{/if}
 {/if}
 {#if detail&&workspace}{#key selected?.id??'new'}<TaskDetail task={selected} {workspace} {members} {jobs} onclose={closeDetail} onsaved={refresh}/>{/key}{/if}
</main>
