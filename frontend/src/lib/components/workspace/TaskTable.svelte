<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
 import {taskStatus,priorities,labelDate,type Task} from '$lib/workspace';
 let {tasks,onopen,onsort,sort}:{tasks:Task[];onopen:(t:Task)=>void;onsort:(s:string)=>void;sort:string}=$props();
</script>
<div class="ws-scroll"><table class="ws-table"><thead><tr>
 <th><button onclick={()=>onsort('title')} aria-pressed={sort==='title'}>{$translate("Pekerjaan ↕")}</button></th><th>{$translate("Assignee")}</th><th>{$translate("Project / Pekerjaan")}</th><th>{$translate("Start")}</th><th><button onclick={()=>onsort('due')} aria-pressed={sort==='due'}>{$translate("Due ↕")}</button></th><th>{$translate("Status")}</th><th><button onclick={()=>onsort('priority')} aria-pressed={sort==='priority'}>{$translate("Priority ↕")}</button></th><th>{$translate("URL")}</th><th>{$translate("Aksi")}</th>
 </tr></thead><tbody>{#each tasks as task (task.id)}<tr>
 <td><button class="ws-link" onclick={()=>onopen(task)}>{task.title}</button></td><td>{task.assignees.map(a=>a.name).join(', ')||$translate('Belum ditugaskan')}</td><td>{task.jobName||'—'}</td><td>{labelDate(task.startDate, $locale)}</td><td>{labelDate(task.dueDate, $locale)}</td><td><span class={'ws-status '+task.status}>{$translate(taskStatus(task))}{task.requiresQa?$translate(' · Butuh check QA'):''}</span></td><td>{$translate(priorities[task.priority])}</td><td>{#if task.relatedUrl}<a href={task.relatedUrl} target="_blank" rel="noopener noreferrer">{$translate("Buka ↗")}</a>{:else}—{/if}</td><td><button onclick={()=>onopen(task)}>{task.canEdit?$translate('Edit'):$translate('Detail')}</button></td>
 </tr>{/each}</tbody></table></div>
