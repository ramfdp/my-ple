<script lang="ts">
 import { language, locale, t as translate } from '$lib/i18n';
 import {formatActivity} from '$lib/activity';
 import {onMount} from 'svelte';
 import {api,APIError,currentUser} from '$lib/api';
 import {subscribeChanges} from '$lib/realtime';
 import {commentLinks} from '$lib/utils/comment-links';
 import ProjectCombobox from './ProjectCombobox.svelte';
 import {statuses,priorities,revisionLabel,taskInput,type Task,type TaskInput,type Workspace,type WorkspaceJob,type Member,type Comment,type Activity} from '$lib/workspace';
 let {task,workspace,members,jobs,onclose,onsaved}:{task:Task|null;workspace:Workspace;members:Member[];jobs:WorkspaceJob[];onclose:()=>void;onsaved:(task?:Task)=>Promise<void>}=$props();
 let dialog:HTMLDialogElement,qaDialog:HTMLDialogElement;
 let input=$state<TaskInput>({requiresQa:false,qaStage:'',title:'',description:'',relatedUrl:'',jobId:null,startDate:'',dueDate:'',status:'TODO',priority:'MEDIUM',assigneeIds:[],version:0});
 let latest=$state<Task|null>(null),removed=$state(false);
 let editingLink=$state(false),linkDraft=$state(''),linkFeedback=$state('');
 let active=true,syncGeneration=0,discussionGeneration=0,priorityChoice=$state('MEDIUM');
 async function sync(){if(!task)return;const sequence=++syncGeneration;try{const fresh=(await api<{task:Task}>('/work-items/'+task.id+archiveQuery)).task;if(!active||sequence!==syncGeneration)return;if(fresh.version!==task.version){if(!editingLink&&JSON.stringify(input)===JSON.stringify(taskInput(task))&&priorityChoice===task.priority){task=fresh;input=taskInput(fresh);priorityChoice=fresh.priority}else latest=fresh}await collaboration()}catch(e){if(active){error=(e as Error).message;if(e instanceof APIError&&e.status===404)removed=true}}}
 let saving=$state(false),error=$state(''),comments=$state<Comment[]>([]),activity=$state<Activity[]>([]),comment=$state(''),mentionIds=$state<string[]>([]),qaDecision=$state<'DONE'|'NEED_REVISION'>('DONE'),confirmRestore=$state(false),confirmArchive=$state(false),loading=$state(false);
	const mentionQuery=$derived(comment.match(/(?:^|\s)@([^\s@]*)$/)?.[1].toLowerCase()??null);
	const mentionChoices=$derived(mentionQuery===null?[]:members.filter(member=>member.id!==$currentUser?.id&&member.name.toLowerCase().includes(mentionQuery)).slice(0,8));
 const archived=$derived(!!task?.archivedAt);
 const archiveQuery=$derived(archived?'?archived=true':'');
 const editable=$derived(!archived&&(!task||task.canEdit));
 async function collaboration(){if(!task)return;const sequence=++discussionGeneration;loading=true;try{const[c,a]=await Promise.all([api<{comments:Comment[]}>('/work-items/'+task.id+'/comments'+archiveQuery),api<{activity:Activity[]}>('/work-items/'+task.id+'/activity'+archiveQuery)]);if(active&&sequence===discussionGeneration){comments=c.comments;activity=a.activity}}catch(e){if(active)error=(e as Error).message}finally{if(active&&sequence===discussionGeneration)loading=false}}
 onMount(()=>{priorityChoice=task?.priority??'MEDIUM';if(!task)input.requestId=crypto.randomUUID();input=task?taskInput(task):{...input,assigneeIds:members.some(m=>m.id===$currentUser?.id)?[$currentUser!.id]:[]};dialog.showModal();void collaboration();const stop=subscribeChanges(['workspace.changed'],c=>{if(!saving&&(c.workspaceId===undefined||c.workspaceId===workspace.id))return sync()});return()=>{active=false;stop()}});
 async function save(e:SubmitEvent){e.preventDefault();if(saving||latest||removed||editingLink)return;saving=true;error='';try{const base=task?taskInput(task):null;const patch=base?Object.fromEntries(Object.entries(input).filter(([key,value])=>key==='version'||JSON.stringify(value)!==JSON.stringify(base[key as keyof TaskInput]))):input;const response=await api<{task:Task}>(task?'/work-items/'+task.id:'/workspaces/'+workspace.id+'/tasks',{method:task?'PATCH':'POST',body:JSON.stringify(patch)});await onsaved(response.task);onclose()}catch(e){error=(e as Error).message;if(e instanceof APIError&&e.status===409)await sync()}finally{saving=false}}
 function reconcile(){if(!task||!latest)return;const base=taskInput(task),fresh=taskInput(latest);const changed=Object.fromEntries(Object.entries(input).filter(([key,value])=>key!=='version'&&JSON.stringify(value)!==JSON.stringify(base[key as keyof TaskInput])));input={...fresh,...changed};task=latest;latest=null;error=''}
 async function action(kind:'priority'|'qa'|'link',patch:Record<string,unknown>){if(!task||saving||latest||removed)return false;saving=true;error='';++syncGeneration;try{const result=await api<{task:Task}>(`/work-items/${task.id}${kind==='link'?'':'/'+kind}`,{method:'PATCH',body:JSON.stringify({...patch,version:task.version})});const base=taskInput(task),draft=Object.fromEntries(Object.entries(input).filter(([key,value])=>key!=='version'&&!(key in patch)&&JSON.stringify(value)!==JSON.stringify(base[key as keyof TaskInput])));if(kind==='priority'||priorityChoice===task.priority)priorityChoice=result.task.priority;task=result.task;input={...taskInput(task),...draft};if(kind==='link'){editingLink=false;linkFeedback='Link tersimpan.'}await onsaved(task);await collaboration();return true}catch(e){error=(e as Error).message;if(e instanceof APIError&&e.status===409)await sync()}finally{saving=false}}
 async function restore(){if(!task?.canRestore||saving)return;saving=true;error='';try{await api('/work-items/'+task.id+'/restore',{method:'POST',body:JSON.stringify({version:task.version})});await onsaved();onclose()}catch(e){error=(e as Error).message;if(e instanceof APIError&&e.status===409)await sync()}finally{saving=false}}
 async function archive(){if(!task||saving)return;saving=true;error='';try{await api('/work-items/'+task.id,{method:'DELETE',body:JSON.stringify({version:task.version})});await onsaved();onclose()}catch(e){error=(e as Error).message}finally{saving=false}}
 function mention(member:Member){comment=comment.replace(/@[^\s@]*$/,`@${member.name} `);if(!mentionIds.includes(member.id))mentionIds=[...mentionIds,member.id]}
 async function validate(){qaDialog.close();await action('qa',{decision:qaDecision})}
 async function sendComment(e:SubmitEvent){e.preventDefault();if(!task||saving)return;saving=true;error='';try{const selected=mentionIds.filter(id=>{const member=members.find(m=>m.id===id);return member&&comment.includes('@'+member.name)});await api('/work-items/'+task.id+'/comments',{method:'POST',body:JSON.stringify({content:comment,mentionIds:selected})});comment='';mentionIds=[];await collaboration();await onsaved()}catch(e){error=(e as Error).message}finally{saving=false}}
 const time=(s:string)=>new Date(s).toLocaleString($locale);
 async function copyLink(){
  linkFeedback='';
  try{await navigator.clipboard.writeText(task!.relatedUrl);linkFeedback='Link tersalin.'}
  catch{linkFeedback='Link tidak dapat disalin. Pilih teks link dan salin secara manual.'}
 }
 function editLink(){linkDraft=task?.relatedUrl??'';linkFeedback='';editingLink=true}
 async function saveLink(e:SubmitEvent){e.preventDefault();if(!editable)return;await action('link',{relatedUrl:linkDraft.trim()})}
</script>
<dialog bind:this={dialog} class="ws-dialog" oncancel={(e)=>{if(saving)e.preventDefault();else onclose()}} aria-labelledby="task-title">
 <header class="ws-row"><h2 id="task-title">{task?$translate('Detail Pekerjaan'):$translate('Tambah Pekerjaan')}</h2><button disabled={saving} onclick={onclose} aria-label={$translate("Tutup detail")}>✕</button></header>
 {#if error}<p role="alert">{$translate(error)}</p>{#if archived}<button disabled={saving} onclick={async()=>{await onsaved();onclose()}}>{$translate("Muat ulang")}</button>{/if}{/if}
 {#if archived}<p>{$translate("Tanggal diarsipkan")}: {time(task!.archivedAt!)}</p>{#if task?.canRestore}<button disabled={saving||removed||!!latest} onclick={()=>confirmRestore=true}>{$translate("Pulihkan")}</button>{/if}{/if}
 {#if confirmRestore}<section class="ws-panel"><p>{$translate("Pulihkan pekerjaan ini ke workspace aktif?")}</p><button disabled={saving||removed||!!latest} onclick={restore}>{$translate("Ya, pulihkan")}</button> <button disabled={saving} onclick={()=>confirmRestore=false}>{$translate("Batal")}</button></section>{/if}
 {#if latest}<section role="status"><p>{$translate("Versi server")} {latest.version} {$translate("sudah berubah. Draf tetap tersedia.")}</p><dl><dt>{$translate("Judul terbaru")}</dt><dd>{latest.title}</dd><dt>{$translate("Deskripsi terbaru")}</dt><dd style="white-space:pre-wrap">{latest.description}</dd><dt>{$translate("Status / Priority")}</dt><dd>{$translate(statuses[latest.status])} / {$translate(priorities[latest.priority])}</dd><dt>{$translate("Jadwal")}</dt><dd>{latest.startDate||'—'} – {latest.dueDate||'—'}</dd><dt>{$translate("Assignees")}</dt><dd>{latest.assignees.map(a=>a.name).join(', ')||'—'}</dd><dt>{$translate("Project / URL")}</dt><dd>{latest.jobName||'—'} / {latest.relatedUrl||'—'}</dd></dl><button onclick={reconcile}>{$translate("Sudah dibandingkan — gabungkan draf dengan versi terbaru")}</button></section>{/if}
 {#if removed&&!archived}<p role="alert">{$translate("Pekerjaan telah diarsipkan. Draf tetap tersedia untuk disalin.")}</p>{/if}
 {#if archived&&task}
 <dl class="ws-form">
 <div><dt>{$translate('Judul')}</dt><dd>{task.title}</dd></div>
 <div><dt>{$translate('Project')}</dt><dd>{task.jobName||$translate('Tanpa project')}</dd></div>
 <div><dt>{$translate('Status')}</dt><dd>{$translate(statuses[task.status])}</dd></div>
 <div><dt>{$translate('Priority')}</dt><dd>{$translate(priorities[task.priority])}</dd></div>
 <div><dt>{$translate('Start Date')}</dt><dd>{task.startDate||$translate('Tanpa tanggal')}</dd></div>
 <div><dt>{$translate('Due Date')}</dt><dd>{task.dueDate||$translate('Tanpa deadline')}</dd></div>
 <div class="ws-full"><dt>{$translate('Assignees')}</dt><dd>{task.assignees.map(a=>a.name).join(', ')||$translate('Belum ditugaskan')}</dd></div>
 <div class="ws-full"><dt>{$translate('Deskripsi')}</dt><dd style="white-space:pre-wrap;overflow-wrap:anywhere">{task.description}</dd></div>
 </dl>
 {:else}<form onsubmit={save}>
 <fieldset disabled={!editable||saving} class="ws-form">
 <label class="ws-full">{$translate("Judul")}<input required maxlength="200" bind:value={input.title}/></label>
 <ProjectCombobox jobs={jobs.filter(j=>j.role!=='QA'||j.id===task?.jobId)} bind:value={input.jobId}/>
 <label class="ws-full ws-check"><input type="checkbox" bind:checked={input.requiresQa}/> {$translate("Butuh validasi QA sebelum pekerjaan dapat diselesaikan")}</label>
 <label>{$translate("Status")}<select bind:value={input.status}>{#each Object.entries(statuses).filter(([key])=>!task?key==='TODO':key!=='DONE'||!input.requiresQa||task.status==='DONE') as [key,label]}<option value={key}>{$translate(label)}</option>{/each}</select></label>
 <label>{$translate("Priority")}<select bind:value={input.priority}>{#each Object.entries(priorities) as [key,label]}<option value={key}>{$translate(label)}</option>{/each}</select></label>
 <label>{$translate("Start Date")}<input type="date" bind:value={input.startDate}/></label><label>{$translate("Due Date")}<input type="date" min={input.startDate||undefined} bind:value={input.dueDate}/></label>
 <fieldset class="ws-full ws-assignees" disabled={!!task&&task.createdBy!==$currentUser?.id&&!task.assignees.some(a=>a.id===$currentUser?.id)}><legend>{$translate("Assignees")}</legend>{#each members as m}<label><input type="checkbox" value={m.id} bind:group={input.assigneeIds}/>{m.name}</label>{/each}{#if !members.length}<p>{$translate("Belum ada anggota divisi.")}</p>{/if}</fieldset>
 <label class="ws-full">{$translate("Deskripsi")}<textarea rows="5" maxlength="10000" bind:value={input.description}></textarea></label>
 {#if !task}<label class="ws-full">{$translate("Related URL")}<input type="url" placeholder="https://…" maxlength="2000" bind:value={input.relatedUrl}/></label>{/if}
 </fieldset>
 {#if editable}<div class="ws-row"><button class="ws-primary" type="submit" disabled={saving||!!latest||removed||editingLink}>{saving?$translate('Menyimpan…'):$translate('Simpan Pekerjaan')}</button>{#if task?.canDelete}<button type="button" disabled={saving} onclick={()=>confirmArchive=true}>{$translate("Arsipkan")}</button>{/if}</div>{/if}
 </form>{/if}
 {#if task}
 <section class="related-link" aria-labelledby="related-link-title">
  <h3 id="related-link-title">{$translate("Related URL")}</h3>
  {#if task.relatedUrl}<a href={task.relatedUrl} target="_blank" rel="noopener noreferrer">{task.relatedUrl}</a>
   <button type="button" onclick={copyLink}>{$translate("Salin link")}</button>
  {:else}<p>{$translate("Belum ada link.")}</p>{/if}
  {#if editingLink&&editable}
   <form onsubmit={saveLink}>
    <label>{$translate("URL baru")}<input type="url" pattern="https?://.*" maxlength="2000" placeholder="https://…" bind:value={linkDraft} disabled={saving}/></label>
    <p>{$translate("Kosongkan untuk menghapus link. Simpan atau batalkan edit link sebelum menyimpan pekerjaan.")}</p>
    <div class="ws-row"><button class="ws-primary" disabled={saving||!!latest||removed}>{$translate("Simpan link")}</button><button type="button" disabled={saving} onclick={()=>{editingLink=false;linkDraft='';linkFeedback=''}}>{$translate("Batal")}</button></div>
   </form>
  {:else if editable}<button type="button" disabled={saving||!!latest||removed} onclick={editLink}>{task.relatedUrl?$translate('Edit link'):$translate('Tambah link')}</button>{/if}
  <p role="status">{$translate(linkFeedback)}</p>
 </section>
 {/if}
 {#if task}<p><strong>{$translate("Revisi")}</strong>: {$translate(revisionLabel(task.revisionCount))}</p>{/if}
 {#if task?.qaValidatedAt}<p role="status"><strong>{$translate("QA Validated")}</strong> · {task.qaValidatedByName} · {time(task.qaValidatedAt)}</p>{:else if task?.requiresQa&&task.status!=='DONE'}<p>{$translate("Belum divalidasi QA.")}</p>{/if}
 {#if !archived&&task?.requiresQa&&(task.status==='IN_PROGRESS'||task.status==='NEED_REVISION')}<button class="ws-primary" type="button" disabled={saving||!!latest||removed} title={$translate("Validasi hasil pekerjaan")} onclick={()=>qaDialog.showModal()}>{$translate("Validasi QA")}</button>{/if}
 {#if task&&!editable&&!archived}<label>{$translate("Priority")}<select bind:value={priorityChoice}>{#each Object.entries(priorities) as [value,label]}<option {value}>{$translate(label)}</option>{/each}</select></label><button disabled={saving||removed||!!latest} onclick={()=>action('priority',{priority:priorityChoice})}>{$translate("Simpan Priority")}</button>{/if}
 {#if confirmArchive}<div class="ws-panel"><p>{$translate("Arsipkan pekerjaan ini? Pekerjaan tidak lagi muncul pada tampilan aktif.")}</p><button disabled={saving} onclick={archive}>{$translate("Ya, arsipkan")}</button> <button onclick={()=>confirmArchive=false}>{$translate("Batal")}</button></div>{/if}
 {#if task}
 <section><h3>{$translate("Komentar")}</h3>{#if loading}<p role="status">{$translate("Memuat diskusi…")}</p>{/if}{#each comments as c (c.id)}<article class="ws-comment"><strong>{c.name}</strong><small> · {time(c.createdAt)}</small><p style="white-space:pre-wrap">{#each commentLinks(c.content) as part}{#if part.href}<a href={part.href} target="_blank" rel="noopener noreferrer">{part.text}</a>{:else}{part.text}{/if}{/each}</p></article>{:else}<p>{$translate("Belum ada komentar.")}</p>{/each}
 {#if !archived}<form class="ws-comment-form" onsubmit={sendComment}><label>{$translate("Komentar baru")}<textarea required maxlength="4000" rows="2" bind:value={comment} disabled={saving} aria-autocomplete="list" aria-controls="mention-list"></textarea></label>{#if mentionChoices.length}<ul id="mention-list" class="ws-mentions" aria-label={$translate("Pilih user yang disebut")}>{#each mentionChoices as member}<li><button type="button" onclick={()=>mention(member)}>@{member.name}</button></li>{/each}</ul>{/if}<button type="submit" disabled={saving||!comment.trim()}>{$translate("Kirim Komentar")}</button></form>{/if}</section>
 <section><h3>{$translate("Activity History")}</h3><ul class="ws-activity">{#each activity as a}{@const formatted=formatActivity(a,$language)}<li>{formatted.title}<br/><small>{formatted.timestamp}</small></li>{/each}</ul></section>
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
  <h2 id="qa-title">{$translate("Validasi QA")}</h2>
  <label>{$translate("Hasil validasi")}<select bind:value={qaDecision}><option value="DONE">{$translate("Done")}</option><option value="NEED_REVISION">{$translate("Need Revision")}</option></select></label>
  <div class="ws-row"><button type="button" onclick={()=>qaDialog.close()}>{$translate("Batal")}</button><button class="ws-primary" disabled={saving}>{$translate("Simpan Validasi")}</button></div>
 </form>
</dialog>
