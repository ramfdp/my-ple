<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
 import {statuses,priorities,labelDate,updateTask,type Task,type Status} from '$lib/workspace';
 import {taskStatus} from '$lib/workspace';
 let {tasks,onopen,onchanged}:{tasks:Task[];onopen:(t:Task)=>void;onchanged:()=>Promise<void>}=$props();
 let busy=$state(false),error=$state('');
 async function move(task:Task,status:Status){if(busy||!task.canEdit||task.status===status)return;if(status==='DONE'&&task.requiresQa&&!task.qaValidatedAt){error='Pekerjaan ini harus divalidasi QA.';return}busy=true;error='';try{await updateTask(task.id,{status,version:task.version});await onchanged()}catch(e){error=(e as Error).message}finally{busy=false}}
</script>
{#if error}<p role="alert">{$translate(error)}</p>{/if}
<div class="ws-board">{#each Object.entries(statuses) as [key,label]}
 <section class="ws-column" aria-label={$translate(label)}><h2>{$translate(label)} <small>{tasks.filter(t=>t.status===key).length}</small></h2>
 {#each tasks.filter(t=>t.status===key) as task (task.id)}<article class="ws-task-card">
 <button class="ws-link" onclick={()=>onopen(task)}>{task.title}</button>
 <p>{task.assignees.map(a=>a.name).join(', ')||$translate('Belum ditugaskan')}</p><p>{task.jobName}</p><small>{labelDate(task.dueDate, $locale)} · {$translate(priorities[task.priority])}</small>{#if task.requiresQa||task.qaValidatedAt}<p>{task.qaValidatedAt?'QA':$translate('Butuh check QA')} · {$translate(taskStatus(task))}</p>{/if}
 {#if task.canEdit}<label>{$translate("Status")}<select value={task.status} disabled={busy} onchange={(e)=>{const next=e.currentTarget.value as Status;e.currentTarget.value=task.status;void move(task,next)}}>{#each Object.entries(statuses).filter(([value])=>value!=='DONE'||!task.requiresQa||task.status==='DONE') as [value,text]}<option {value}>{$translate(text)}</option>{/each}</select></label>{/if}
 </article>{:else}<p class="ws-muted">{$translate("Belum ada pekerjaan.")}</p>{/each}</section>
{/each}</div>
