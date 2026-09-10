<script lang="ts">
 import {formatActivity} from '$lib/activity';
 import {onMount} from 'svelte';
 import {api,APIError,currentUser} from '$lib/api';
 import {subscribeChanges} from '$lib/realtime';
 import {commentLinks} from '$lib/utils/comment-links';
 import ProjectCombobox from './ProjectCombobox.svelte';
 import {statuses,priorities,taskInput,type Task,type TaskInput,type Workspace,type WorkspaceJob,type Member,type Comment,type Activity} from '$lib/workspace';
 let {task,workspace,members,jobs,onclose,onsaved}:{task:Task|null;workspace:Workspace;members:Member[];jobs:WorkspaceJob[];onclose:()=>void;onsaved:(task?:Task)=>Promise<void>}=$props();
 let dialog:HTMLDialogElement;
 let input=$state<TaskInput>({requiresQa:false,qaStage:'',title:'',description:'',relatedUrl:'',jobId:null,startDate:'',dueDate:'',status:'TODO',priority:'MEDIUM',assigneeIds:[],version:0});
 let latest=$state<Task|null>(null),removed=$state(false);
 let active=true,syncGeneration=0,discussionGeneration=0,priorityChoice=$state('MEDIUM');
 async function sync(){if(!task)return;const sequence=++syncGeneration;try{const fresh=(await api<{task:Task}>('/work-items/'+task.id)).task;if(!active||sequence!==syncGeneration)return;if(fresh.version!==task.version){if(JSON.stringify(input)===JSON.stringify(taskInput(task))&&priorityChoice===task.priority){task=fresh;input=taskInput(fresh);priorityChoice=fresh.priority}else latest=fresh}await collaboration()}catch(e){if(active){error=(e as Error).message;if(e instanceof APIError&&e.status===404)removed=true}}}
 let saving=$state(false),error=$state(''),comments=$state<Comment[]>([]),activity=$state<Activity[]>([]),comment=$state(''),confirmArchive=$state(false),loading=$state(false);
 const editable=$derived(!task||task.canEdit);
 async function collaboration(){if(!task)return;const sequence=++discussionGeneration;loading=true;try{const[c,a]=await Promise.all([api<{comments:Comment[]}>('/work-items/'+task.id+'/comments'),api<{activity:Activity[]}>('/work-items/'+task.id+'/activity')]);if(active&&sequence===discussionGeneration){comments=c.comments;activity=a.activity}}catch(e){if(active)error=(e as Error).message}finally{if(active&&sequence===discussionGeneration)loading=false}}
 onMount(()=>{priorityChoice=task?.priority??'MEDIUM';if(!task)input.requestId=crypto.randomUUID();input=task?taskInput(task):{...input,assigneeIds:members.some(m=>m.id===$currentUser?.id)?[$currentUser!.id]:[]};dialog.showModal();void collaboration();const stop=subscribeChanges(['workspace.changed'],c=>{if(c.workspaceId===undefined||c.workspaceId===workspace.id)return sync()});return()=>{active=false;stop()}});
 async function save(e:SubmitEvent){e.preventDefault();if(saving||latest||removed)return;saving=true;error='';try{const base=task?taskInput(task):null;const patch=base?Object.fromEntries(Object.entries(input).filter(([key,value])=>key==='version'||JSON.stringify(value)!==JSON.stringify(base[key as keyof TaskInput]))):input;const response=await api<{task:Task}>(task?'/work-items/'+task.id:'/workspaces/'+workspace.id+'/tasks',{method:task?'PATCH':'POST',body:JSON.stringify(patch)});await onsaved(response.task);onclose()}catch(e){error=(e as Error).message;if(e instanceof APIError&&e.status===409)await sync()}finally{saving=false}}
 function reconcile(){if(!task||!latest)return;const base=taskInput(task),fresh=taskInput(latest);const changed=Object.fromEntries(Object.entries(input).filter(([key,value])=>key!=='version'&&JSON.stringify(value)!==JSON.stringify(base[key as keyof TaskInput])));input={...fresh,...changed};task=latest;latest=null;error=''}
 async function action(kind:'priority'|'qa',patch:Record<string,unknown>){if(!task||saving)return;saving=true;try{const result=await api<{task:Task}>(`/work-items/${task.id}/${kind}`,{method:'PATCH',body:JSON.stringify({...patch,version:task.version})});task=result.task;input=taskInput(task);priorityChoice=task.priority;await onsaved(task)}catch(e){error=(e as Error).message;if(e instanceof APIError&&e.status===409)await sync()}finally{saving=false}}
 async function archive(){if(!task||saving)return;saving=true;error='';try{await api('/work-items/'+task.id,{method:'DELETE',body:JSON.stringify({version:task.version})});await onsaved();onclose()}catch(e){error=(e as Error).message}finally{saving=false}}
 async function sendComment(e:SubmitEvent){e.preventDefault();if(!task||saving)return;saving=true;error='';try{await api('/work-items/'+task.id+'/comments',{method:'POST',body:JSON.stringify({content:comment})});comment='';await collaboration();await onsaved()}catch(e){error=(e as Error).message}finally{saving=false}}
 const time=(s:string)=>new Date(s).toLocaleString('id-ID');
</script>
<dialog bind:this={dialog} class="ws-dialog" oncancel={(e)=>{if(saving)e.preventDefault();else onclose()}} aria-labelledby="task-title">
 <header class="ws-row"><h2 id="task-title">{task?'Detail Pekerjaan':'Tambah Pekerjaan'}</h2><button disabled={saving} onclick={onclose} aria-label="Tutup detail">✕</button></header>
 {#if error}<p role="alert">{error}</p>{/if}
 {#if latest}<section role="status"><p>Versi server {latest.version} sudah berubah. Draf tetap tersedia.</p><dl><dt>Judul terbaru</dt><dd>{latest.title}</dd><dt>Deskripsi terbaru</dt><dd style="white-space:pre-wrap">{latest.description}</dd><dt>Status / Priority</dt><dd>{statuses[latest.status]} / {priorities[latest.priority]}</dd><dt>Jadwal</dt><dd>{latest.startDate||'—'} – {latest.dueDate||'—'}</dd><dt>Assignees</dt><dd>{latest.assignees.map(a=>a.name).join(', ')||'—'}</dd><dt>Project / URL</dt><dd>{latest.jobName||'—'} / {latest.relatedUrl||'—'}</dd></dl><button onclick={reconcile}>Sudah dibandingkan — gabungkan draf dengan versi terbaru</button></section>{/if}
 {#if removed}<p role="alert">Pekerjaan telah diarsipkan. Draf tetap tersedia untuk disalin.</p>{/if}
 <form onsubmit={save}>
 <fieldset disabled={!editable||saving} class="ws-form">
 <label class="ws-full">Judul<input required maxlength="200" bind:value={input.title}/></label>
 <ProjectCombobox jobs={jobs.filter(j=>j.role!=='QA'||j.id===task?.jobId)} bind:value={input.jobId}/>
 {#if !task}<label class="ws-full"><input type="checkbox" bind:checked={input.requiresQa}/>Perlu check QA</label>{:else if task.requiresQa}<p>Butuh check QA · {task.qaStage==='WAITING'?'Menunggu QA':task.qaStage==='CHECKING'?'Progress Checking':task.qaStage==='COMPLETE'?'Selesai QA':'Belum diajukan'}</p>{/if}
 <label>Status<select bind:value={input.status}>{#each Object.entries(statuses).filter(([key])=>!task?key==='TODO':key!=='DONE'||!input.requiresQa||input.qaStage==='COMPLETE') as [key,label]}<option value={key}>{label}</option>{/each}</select></label>
 <label>Priority<select bind:value={input.priority}>{#each Object.entries(priorities) as [key,label]}<option value={key}>{label}</option>{/each}</select></label>
 <label>Start Date<input type="date" bind:value={input.startDate}/></label><label>Due Date<input type="date" min={input.startDate||undefined} bind:value={input.dueDate}/></label>
 <fieldset class="ws-full ws-assignees"><legend>Assignees</legend>{#each members as m}<label><input type="checkbox" value={m.id} bind:group={input.assigneeIds} disabled={!workspace.canManage&&m.id!==$currentUser?.id}/>{m.name}</label>{/each}{#if !members.length}<p>Belum ada anggota divisi.</p>{/if}</fieldset>
 <label class="ws-full">Deskripsi<textarea rows="5" maxlength="10000" bind:value={input.description}></textarea></label>
 <label class="ws-full">Related URL<input type="url" placeholder="https://…" maxlength="2000" bind:value={input.relatedUrl}/></label>
 </fieldset>
 {#if editable}<div class="ws-row"><button class="ws-primary" type="submit" disabled={saving||!!latest||removed}>{saving?'Menyimpan…':'Simpan Pekerjaan'}</button>{#if task?.canDelete}<button type="button" disabled={saving} onclick={()=>confirmArchive=true}>Arsipkan</button>{/if}</div>{/if}
 </form>
 {#if task&&!editable}<label>Priority<select bind:value={priorityChoice}>{#each Object.entries(priorities) as [value,label]}<option {value}>{label}</option>{/each}</select></label><button disabled={saving||removed||!!latest} onclick={()=>action('priority',{priority:priorityChoice})}>Simpan Priority</button>{/if}
 {#if task?.requiresQa}<div class="ws-row">{#if task.status==='IN_PROGRESS'&&!task.qaStage&&editable}<button disabled={saving||JSON.stringify(input)!==JSON.stringify(taskInput(task))} onclick={()=>action('qa',{qaStage:'WAITING'})}>Ajukan check QA</button>{:else if task.canCheckQa&&task.qaStage==='WAITING'}<button disabled={saving||JSON.stringify(input)!==JSON.stringify(taskInput(task))} onclick={()=>action('qa',{qaStage:'CHECKING'})}>Mulai Checking</button>{:else if task.canCheckQa&&task.qaStage==='CHECKING'}<button disabled={saving||JSON.stringify(input)!==JSON.stringify(taskInput(task))} onclick={()=>action('qa',{qaStage:'COMPLETE'})}>Selesaikan QA</button>{/if}</div>{/if}
 {#if confirmArchive}<div class="ws-panel"><p>Arsipkan pekerjaan ini? Pekerjaan tidak lagi muncul pada tampilan aktif.</p><button disabled={saving} onclick={archive}>Ya, arsipkan</button> <button onclick={()=>confirmArchive=false}>Batal</button></div>{/if}
 {#if task}
 <section><h3>Komentar</h3>{#if loading}<p role="status">Memuat diskusi…</p>{/if}{#each comments as c (c.id)}<article class="ws-comment"><strong>{c.name}</strong><small> · {time(c.createdAt)}</small><p style="white-space:pre-wrap">{#each commentLinks(c.content) as part}{#if part.href}<a href={part.href} target="_blank" rel="noopener noreferrer">{part.text}</a>{:else}{part.text}{/if}{/each}</p></article>{:else}<p>Belum ada komentar.</p>{/each}
 <form class="ws-comment-form" onsubmit={sendComment}><label>Komentar baru<textarea required maxlength="4000" rows="2" bind:value={comment} disabled={saving}></textarea></label><button type="submit" disabled={saving||!comment.trim()}>Kirim Komentar</button></form></section>
 <section><h3>Activity History</h3><ul class="ws-activity">{#each activity as a}{@const formatted=formatActivity(a)}<li>{formatted.title}<br/><small>{formatted.timestamp}</small></li>{/each}</ul></section>
 {/if}
</dialog>
