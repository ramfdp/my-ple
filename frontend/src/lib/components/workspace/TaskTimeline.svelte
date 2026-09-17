<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
 import {taskStatus,priorities,labelDate,taskSpan,type Task} from '$lib/workspace';
 let {tasks,days,onopen}:{tasks:Task[];days:string[];onopen:(t:Task)=>void}=$props();
 const positioned=$derived(tasks.map(task=>({task,span:taskSpan(task,days[0],days.at(-1)!)})).filter(row=>row.span));
</script>
<p class="ws-muted">{$translate("Klik pekerjaan untuk detail dan mengubah tanggal. Tanggal mengikuti Asia/Jakarta.")}</p>
<div class="ws-scroll"><div class="ws-timeline" style:min-width={'calc(var(--ws-label-width) + '+(days.length*38)+'px)'}>
 <div class="ws-timeline-row"><div class="ws-timeline-label">{$translate("Pekerjaan")}</div><div class="ws-timeline-days" style:grid-template-columns={'repeat('+days.length+',1fr)'}>{#each days as day}<span>{day.slice(-2)}</span>{/each}</div></div>
 {#each positioned as {task,span}}<div class="ws-timeline-row"><div class="ws-timeline-label"><button class="ws-link" onclick={()=>onopen(task)}>{task.title}</button><small>{task.assignees.map(a=>a.name).join(', ')} · {$translate(taskStatus(task))}{task.requiresQa?$translate(' · Butuh check QA'):''} · {$translate(priorities[task.priority])}</small></div>
 <div class="ws-timeline-track" style:grid-template-columns={'repeat('+days.length+',1fr)'}><button class={'ws-bar '+task.status} style:grid-column={(days.indexOf(span!.start)+1)+' / '+(days.indexOf(span!.end)+2)} title={task.title+': '+labelDate(task.startDate, $locale)+' – '+labelDate(task.dueDate, $locale)} onclick={()=>onopen(task)}>{task.title}</button></div></div>{/each}
</div></div>
{#if !positioned.length}<p class="ws-empty">{$translate("Belum ada pekerjaan bertanggal pada periode ini.")}</p>{/if}
