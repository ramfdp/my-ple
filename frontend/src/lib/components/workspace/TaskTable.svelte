<script lang="ts">
 import {taskStatus,priorities,labelDate,type Task} from '$lib/workspace';
 let {tasks,onopen,onsort,sort}:{tasks:Task[];onopen:(t:Task)=>void;onsort:(s:string)=>void;sort:string}=$props();
</script>
<div class="ws-scroll"><table class="ws-table"><thead><tr>
 <th><button onclick={()=>onsort('title')} aria-pressed={sort==='title'}>Pekerjaan ↕</button></th><th>Assignee</th><th>Project / Pekerjaan</th><th>Start</th><th><button onclick={()=>onsort('due')} aria-pressed={sort==='due'}>Due ↕</button></th><th>Status</th><th><button onclick={()=>onsort('priority')} aria-pressed={sort==='priority'}>Priority ↕</button></th><th>URL</th><th>Aksi</th>
 </tr></thead><tbody>{#each tasks as task (task.id)}<tr>
 <td><button class="ws-link" onclick={()=>onopen(task)}>{task.title}</button></td><td>{task.assignees.map(a=>a.name).join(', ')||'Belum ditugaskan'}</td><td>{task.jobName||'—'}</td><td>{labelDate(task.startDate)}</td><td>{labelDate(task.dueDate)}</td><td><span class={'ws-status '+task.status}>{taskStatus(task)}{task.requiresQa?' · Butuh check QA':''}</span></td><td>{priorities[task.priority]}</td><td>{#if task.relatedUrl}<a href={task.relatedUrl} target="_blank" rel="noopener noreferrer">Buka ↗</a>{:else}—{/if}</td><td><button onclick={()=>onopen(task)}>{task.canEdit?'Edit':'Detail'}</button></td>
 </tr>{/each}</tbody></table></div>
