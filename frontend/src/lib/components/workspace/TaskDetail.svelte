<script lang="ts">
 import {formatActivity} from '$lib/activity';
 import {onMount} from 'svelte';
 import {api,APIError,currentUser} from '$lib/api';
 import {subscribeChanges} from '$lib/realtime';
 import {commentLinks} from '$lib/utils/comment-links';
 import ProjectCombobox from './ProjectCombobox.svelte';
 import {statuses,priorities,taskInput,type Task,type TaskInput,type Workspace,type WorkspaceJob,type Member,type Comment,type Activity} from '$lib/workspace';
 let {task,workspace,members,jobs,onclose,onsaved}:{task:Task|null;workspace:Workspace;members:Member[];jobs:WorkspaceJob[];onclose:()=>void;onsaved:(task?:Task)=>Promise<void>}=$props();
 let dialog:HTMLDialogElement,qaDialog:HTMLDialogElement;
 let input=$state<TaskInput>({requiresQa:false,qaStage:'',title:'',description:'',relatedUrl:'',jobId:null,startDate:'',dueDate:'',status:'TODO',priority:'MEDIUM',assigneeIds:[],version:0});
 let latest=$state<Task|null>(null),removed=$state(false);
 let editingLink=$state(false),linkDraft=$state(''),linkFeedback=$state('');
 let active=true,syncGeneration=0,discussionGeneration=0,priorityChoice=$state('MEDIUM');
 async function sync(){if(!task)return;const sequence=++syncGeneration;try{const fresh=(await api<{task:Task}>('/work-items/'+task.id)).task;if(!active||sequence!==syncGeneration)return;if(fresh.version!==task.version){if(!editingLink&&JSON.stringify(input)===JSON.stringify(taskInput(task))&&priorityChoice===task.priority){task=fresh;input=taskInput(fresh);priorityChoice=fresh.priority}else latest=fresh}await collaboration()}catch(e){if(active){error=(e as Error).message;if(e instanceof APIError&&e.status===404)removed=true}}}
 let saving=$state(false),error=$state(''),comments=$state<Comment[]>([]),activity=$state<Activity[]>([]),comment=$state(''),mentionIds=$state<string[]>([]),qaDecision=$state<'DONE'|'NEED_REVISION'>('DONE'),confirmArchive=$state(false),loading=$state(false);
	const mentionQuery=$derived(comment.match(/(?:^|\s)@([^\s@]*)$/)?.[1].toLowerCase()??null);
	const mentionChoices=$derived(mentionQuery===null?[]:members.filter(member=>member.id!==$currentUser?.id&&member.name.toLowerCase().includes(mentionQuery)).slice(0,8));
 const editable=$derived(!task||task.canEdit);
 async function collaboration(){if(!task)return;const sequence=++discussionGeneration;loading=true;try{const[c,a]=await Promise.all([api<{comments:Comment[]}>('/work-items/'+task.id+'/comments'),api<{activity:Activity[]}>('/work-items/'+task.id+'/activity')]);if(active&&sequence===discussionGeneration){comments=c.comments;activity=a.activity}}catch(e){if(active)error=(e as Error).message}finally{if(active&&sequence===discussionGeneration)loading=false}}
 onMount(()=>{priorityChoice=task?.priority??'MEDIUM';if(!task)input.requestId=crypto.randomUUID();input=task?taskInput(task):{...input,assigneeIds:members.some(m=>m.id===$currentUser?.id)?[$currentUser!.id]:[]};dialog.showModal();void collaboration();const stop=subscribeChanges(['workspace.changed'],c=>{if(!saving&&(c.workspaceId===undefined||c.workspaceId===workspace.id))return sync()});return()=>{active=false;stop()}});
 async function save(e:SubmitEvent){e.preventDefault();if(saving||latest||removed||editingLink)return;saving=true;error='';try{const base=task?taskInput(task):null;const patch=base?Object.fromEntries(Object.entries(input).filter(([key,value])=>key==='version'||JSON.stringify(value)!==JSON.stringify(base[key as keyof TaskInput]))):input;const response=await api<{task:Task}>(task?'/work-items/'+task.id:'/workspaces/'+workspace.id+'/tasks',{method:task?'PATCH':'POST',body:JSON.stringify(patch)});await onsaved(response.task);onclose()}catch(e){error=(e as Error).message;if(e instanceof APIError&&e.status===409)await sync()}finally{saving=false}}
 function reconcile(){if(!task||!latest)return;const base=taskInput(task),fresh=taskInput(latest);const changed=Object.fromEntries(Object.entries(input).filter(([key,value])=>key!=='version'&&JSON.stringify(value)!==JSON.stringify(base[key as keyof TaskInput])));input={...fresh,...changed};task=latest;latest=null;error=''}
 async function action(kind:'priority'|'qa'|'link',patch:Record<string,unknown>){if(!task||saving||latest||removed)return false;saving=true;error='';++syncGeneration;try{const result=await api<{task:Task}>(`/work-items/${task.id}${kind==='link'?'':'/'+kind}`,{method:'PATCH',body:JSON.stringify({...patch,version:task.version})});const base=taskInput(task),draft=Object.fromEntries(Object.entries(input).filter(([key,value])=>key!=='version'&&!(key in patch)&&JSON.stringify(value)!==JSON.stringify(base[key as keyof TaskInput])));if(kind==='priority'||priorityChoice===task.priority)priorityChoice=result.task.priority;task=result.task;input={...taskInput(task),...draft};if(kind==='link'){editingLink=false;linkFeedback='Link tersimpan.'}await onsaved(task);await collaboration();return true}catch(e){error=(e as Error).message;if(e instanceof APIError&&e.status===409)await sync()}finally{saving=false}}
 async function archive(){if(!task||saving)return;saving=true;error='';try{await api('/work-items/'+task.id,{method:'DELETE',body:JSON.stringify({version:task.version})});await onsaved();onclose()}catch(e){error=(e as Error).message}finally{saving=false}}
 function mention(member:Member){comment=comment.replace(/@[^\s@]*$/,`@${member.name} `);if(!mentionIds.includes(member.id))mentionIds=[...mentionIds,member.id]}
 async function validate(){qaDialog.close();await action('qa',{decision:qaDecision})}
 async function sendComment(e:SubmitEvent){e.preventDefault();if(!task||saving)return;saving=true;error='';try{const selected=mentionIds.filter(id=>{const member=members.find(m=>m.id===id);return member&&comment.includes('@'+member.name)});await api('/work-items/'+task.id+'/comments',{method:'POST',body:JSON.stringify({content:comment,mentionIds:selected})});comment='';mentionIds=[];await collaboration();await onsaved()}catch(e){error=(e as Error).message}finally{saving=false}}
 const time=(s:string)=>new Date(s).toLocaleString('id-ID');
 async function copyLink(){
  linkFeedback='';
  try{await navigator.clipboard.writeText(task!.relatedUrl);linkFeedback='Link tersalin.'}
  catch{linkFeedback='Link tidak dapat disalin. Pilih teks link dan salin secara manual.'}
 }
 function editLink(){linkDraft=task?.relatedUrl??'';linkFeedback='';editingLink=true}
 async function saveLink(e:SubmitEvent){e.preventDefault();if(!editable)return;await action('link',{relatedUrl:linkDraft.trim()})}
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
 <label class="ws-full ws-check"><input type="checkbox" bind:checked={input.requiresQa}/> Butuh validasi QA sebelum pekerjaan dapat diselesaikan</label>
 <label>Status<select bind:value={input.status}>{#each Object.entries(statuses).filter(([key])=>!task?key==='TODO':key!=='DONE'||!input.requiresQa||task.status==='DONE') as [key,label]}<option value={key}>{label}</option>{/each}</select></label>
 <label>Priority<select bind:value={input.priority}>{#each Object.entries(priorities) as [key,label]}<option value={key}>{label}</option>{/each}</select></label>
 <label>Start Date<input type="date" bind:value={input.startDate}/></label><label>Due Date<input type="date" min={input.startDate||undefined} bind:value={input.dueDate}/></label>
 <fieldset class="ws-full ws-assignees" disabled={!!task&&task.createdBy!==$currentUser?.id}><legend>Assignees</legend>{#each members as m}<label><input type="checkbox" value={m.id} bind:group={input.assigneeIds}/>{m.name}</label>{/each}{#if !members.length}<p>Belum ada anggota divisi.</p>{/if}</fieldset>
 <label class="ws-full">Deskripsi<textarea rows="5" maxlength="10000" bind:value={input.description}></textarea></label>
 {#if !task}<label class="ws-full">Related URL<input type="url" placeholder="https://…" maxlength="2000" bind:value={input.relatedUrl}/></label>{/if}
 </fieldset>
 {#if editable}<div class="ws-row"><button class="ws-primary" type="submit" disabled={saving||!!latest||removed||editingLink}>{saving?'Menyimpan…':'Simpan Pekerjaan'}</button>{#if task?.canDelete}<button type="button" disabled={saving} onclick={()=>confirmArchive=true}>Arsipkan</button>{/if}</div>{/if}
 </form>
 {#if task}
 <section class="related-link" aria-labelledby="related-link-title">
  <h3 id="related-link-title">Related URL</h3>
  {#if task.relatedUrl}<a href={task.relatedUrl} target="_blank" rel="noopener noreferrer">{task.relatedUrl}</a>
   <button type="button" onclick={copyLink}>Salin link</button>
  {:else}<p>Belum ada link.</p>{/if}
  {#if editingLink&&editable}
   <form onsubmit={saveLink}>
    <label>URL baru<input type="url" pattern="https?://.*" maxlength="2000" placeholder="https://…" bind:value={linkDraft} disabled={saving}/></label>
    <p>Kosongkan untuk menghapus link. Simpan atau batalkan edit link sebelum menyimpan pekerjaan.</p>
    <div class="ws-row"><button class="ws-primary" disabled={saving||!!latest||removed}>Simpan link</button><button type="button" disabled={saving} onclick={()=>{editingLink=false;linkDraft='';linkFeedback=''}}>Batal</button></div>
   </form>
  {:else if editable}<button type="button" disabled={saving||!!latest||removed} onclick={editLink}>{task.relatedUrl?'Edit link':'Tambah link'}</button>{/if}
  <p role="status">{linkFeedback}</p>
 </section>
 {/if}
 {#if task?.qaValidatedAt}<p role="status"><strong>QA Validated</strong> · {task.qaValidatedByName} · {time(task.qaValidatedAt)}</p>{:else if task?.requiresQa&&task.status!=='DONE'}<p>Belum divalidasi QA.</p>{/if}
 {#if task?.requiresQa&&task.status==='IN_PROGRESS'}<button class="ws-primary" type="button" disabled={saving||!!latest||removed} title="Validasi hasil pekerjaan" onclick={()=>qaDialog.showModal()}>Validasi QA</button>{/if}
 {#if task&&!editable}<label>Priority<select bind:value={priorityChoice}>{#each Object.entries(priorities) as [value,label]}<option {value}>{label}</option>{/each}</select></label><button disabled={saving||removed||!!latest} onclick={()=>action('priority',{priority:priorityChoice})}>Simpan Priority</button>{/if}
 {#if confirmArchive}<div class="ws-panel"><p>Arsipkan pekerjaan ini? Pekerjaan tidak lagi muncul pada tampilan aktif.</p><button disabled={saving} onclick={archive}>Ya, arsipkan</button> <button onclick={()=>confirmArchive=false}>Batal</button></div>{/if}
 {#if task}
 <section><h3>Komentar</h3>{#if loading}<p role="status">Memuat diskusi…</p>{/if}{#each comments as c (c.id)}<article class="ws-comment"><strong>{c.name}</strong><small> · {time(c.createdAt)}</small><p style="white-space:pre-wrap">{#each commentLinks(c.content) as part}{#if part.href}<a href={part.href} target="_blank" rel="noopener noreferrer">{part.text}</a>{:else}{part.text}{/if}{/each}</p></article>{:else}<p>Belum ada komentar.</p>{/each}
 <form class="ws-comment-form" onsubmit={sendComment}><label>Komentar baru<textarea required maxlength="4000" rows="2" bind:value={comment} disabled={saving} aria-autocomplete="list" aria-controls="mention-list"></textarea></label>{#if mentionChoices.length}<ul id="mention-list" class="ws-mentions" aria-label="Pilih user yang disebut">{#each mentionChoices as member}<li><button type="button" onclick={()=>mention(member)}>@{member.name}</button></li>{/each}</ul>{/if}<button type="submit" disabled={saving||!comment.trim()}>Kirim Komentar</button></form></section>
 <section><h3>Activity History</h3><ul class="ws-activity">{#each activity as a}{@const formatted=formatActivity(a)}<li>{formatted.title}<br/><small>{formatted.timestamp}</small></li>{/each}</ul></section>
 {/if}
</dialog>
<style>
 .related-link { margin: 1.5rem 0; }
 .related-link > a { display: block; overflow-wrap: anywhere; margin-bottom: .75rem; }
 .related-link > button { margin: 0 .5rem .5rem 0; }
 .related-link form { display: grid; gap: .75rem; margin-top: .75rem; }
 .related-link form label { display: grid; gap: .5rem; }
 .related-link p { margin: .5rem 0; }
 .related-link [role='status']:empty { display: none; }
</style>
<dialog bind:this={qaDialog} class="ws-dialog ws-qa-dialog" aria-labelledby="qa-title">
 <form onsubmit={(e)=>{e.preventDefault();void validate()}}>
  <h2 id="qa-title">Validasi QA</h2>
  <label>Hasil validasi<select bind:value={qaDecision}><option value="DONE">Done</option><option value="NEED_REVISION">Need Revision</option></select></label>
  <div class="ws-row"><button type="button" onclick={()=>qaDialog.close()}>Batal</button><button class="ws-primary" disabled={saving}>Simpan Validasi</button></div>
 </form>
</dialog>
