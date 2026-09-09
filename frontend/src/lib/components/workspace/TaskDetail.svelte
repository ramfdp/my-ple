<script lang="ts">
 import {formatActivity} from '$lib/activity';
 import {onMount} from 'svelte';
 import {api,currentUser} from '$lib/api';
 import ProjectCombobox from './ProjectCombobox.svelte';
 import {statuses,priorities,taskInput,type Task,type TaskInput,type Workspace,type WorkspaceJob,type Member,type Comment,type Activity} from '$lib/workspace';
 let {task,workspace,members,jobs,onclose,onsaved}:{task:Task|null;workspace:Workspace;members:Member[];jobs:WorkspaceJob[];onclose:()=>void;onsaved:(task?:Task)=>Promise<void>}=$props();
 let dialog:HTMLDialogElement;
 let input=$state<TaskInput>({title:'',description:'',relatedUrl:'',jobId:null,startDate:'',dueDate:'',status:'TODO',priority:'MEDIUM',assigneeIds:[],version:0});
 let saving=$state(false),error=$state(''),comments=$state<Comment[]>([]),activity=$state<Activity[]>([]),comment=$state(''),confirmArchive=$state(false),loading=$state(false);
 const editable=$derived(!task||task.canEdit);
 async function collaboration(){if(!task)return;loading=true;try{const[c,a]=await Promise.all([api<{comments:Comment[]}>('/work-items/'+task.id+'/comments'),api<{activity:Activity[]}>('/work-items/'+task.id+'/activity')]);comments=c.comments;activity=a.activity}catch(e){error=(e as Error).message}finally{loading=false}}
 onMount(()=>{if(!task)input.requestId=crypto.randomUUID();input=task?taskInput(task):{...input,assigneeIds:members.some(m=>m.id===$currentUser?.id)?[$currentUser!.id]:[]};dialog.showModal();void collaboration()});
 async function save(e:SubmitEvent){e.preventDefault();if(saving)return;saving=true;error='';try{const response=await api<{task:Task}>(task?'/work-items/'+task.id:'/workspaces/'+workspace.id+'/tasks',{method:task?'PATCH':'POST',body:JSON.stringify(input)});await onsaved(response.task);onclose()}catch(e){error=(e as Error).message}finally{saving=false}}
 async function archive(){if(!task||saving)return;saving=true;error='';try{await api('/work-items/'+task.id,{method:'DELETE',body:JSON.stringify({version:task.version})});await onsaved();onclose()}catch(e){error=(e as Error).message}finally{saving=false}}
 async function sendComment(e:SubmitEvent){e.preventDefault();if(!task||saving)return;saving=true;error='';try{await api('/work-items/'+task.id+'/comments',{method:'POST',body:JSON.stringify({content:comment})});comment='';await collaboration();await onsaved()}catch(e){error=(e as Error).message}finally{saving=false}}
 const time=(s:string)=>new Date(s).toLocaleString('id-ID');
</script>
<dialog bind:this={dialog} class="ws-dialog" oncancel={(e)=>{if(saving)e.preventDefault();else onclose()}} aria-labelledby="task-title">
 <header class="ws-row"><h2 id="task-title">{task?'Detail Pekerjaan':'Tambah Pekerjaan'}</h2><button disabled={saving} onclick={onclose} aria-label="Tutup detail">✕</button></header>
 {#if error}<p role="alert">{error}</p>{/if}
 <form onsubmit={save}>
 <fieldset disabled={!editable||saving} class="ws-form">
 <label class="ws-full">Judul<input required maxlength="200" bind:value={input.title}/></label>
 <ProjectCombobox {jobs} bind:value={input.jobId}/>
 <label>Status<select bind:value={input.status}>{#each Object.entries(statuses) as [key,label]}<option value={key}>{label}</option>{/each}</select></label>
 <label>Priority<select bind:value={input.priority}>{#each Object.entries(priorities) as [key,label]}<option value={key}>{label}</option>{/each}</select></label>
 <label>Start Date<input type="date" bind:value={input.startDate}/></label><label>Due Date<input type="date" min={input.startDate||undefined} bind:value={input.dueDate}/></label>
 <fieldset class="ws-full ws-assignees"><legend>Assignees</legend>{#each members as m}<label><input type="checkbox" value={m.id} bind:group={input.assigneeIds} disabled={!workspace.canManage&&m.id!==$currentUser?.id}/>{m.name}</label>{/each}{#if !members.length}<p>Belum ada anggota divisi.</p>{/if}</fieldset>
 <label class="ws-full">Deskripsi<textarea rows="5" maxlength="10000" bind:value={input.description}></textarea></label>
 <label class="ws-full">Related URL<input type="url" placeholder="https://…" maxlength="2000" bind:value={input.relatedUrl}/></label>
 </fieldset>
 {#if editable}<div class="ws-row"><button class="ws-primary" type="submit" disabled={saving}>{saving?'Menyimpan…':'Simpan Pekerjaan'}</button>{#if task?.canDelete}<button type="button" disabled={saving} onclick={()=>confirmArchive=true}>Arsipkan</button>{/if}</div>{/if}
 </form>
 {#if confirmArchive}<div class="ws-panel"><p>Arsipkan pekerjaan ini? Pekerjaan tidak lagi muncul pada tampilan aktif.</p><button disabled={saving} onclick={archive}>Ya, arsipkan</button> <button onclick={()=>confirmArchive=false}>Batal</button></div>{/if}
 {#if task}
 <section><h3>Komentar</h3>{#if loading}<p role="status">Memuat diskusi…</p>{/if}{#each comments as c}<article class="ws-comment"><strong>{c.name}</strong><small> · {time(c.createdAt)}</small><p>{c.content}</p></article>{:else}<p>Belum ada komentar.</p>{/each}
 <form class="ws-comment-form" onsubmit={sendComment}><label>Komentar baru<textarea required maxlength="4000" rows="2" bind:value={comment} disabled={saving}></textarea></label><button type="submit" disabled={saving||!comment.trim()}>Kirim Komentar</button></form></section>
 <section><h3>Activity History</h3><ul class="ws-activity">{#each activity as a}{@const formatted=formatActivity(a)}<li>{formatted.title}<br/><small>{formatted.timestamp}</small></li>{/each}</ul></section>
 {/if}
</dialog>
