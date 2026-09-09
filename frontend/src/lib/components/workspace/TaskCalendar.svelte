<script lang="ts">
 import {dateValue,taskSpan,statuses,type Task} from '$lib/workspace';
 let {tasks,days,onopen}:{tasks:Task[];days:string[];onopen:(t:Task)=>void}=$props();
 let blanks=$derived((dateValue(days[0]).getUTCDay()+6)%7);
</script>
<div class="ws-scroll"><div class="ws-calendar">
 {#each ['Sen','Sel','Rab','Kam','Jum','Sab','Min'] as day}<div class="ws-calendar-heading">{day}</div>{/each}
 {#each Array(blanks) as _}<div class="ws-calendar-day ws-calendar-blank"></div>{/each}
 {#each days as day}<section class="ws-calendar-day" aria-label={day}><strong>{Number(day.slice(-2))}</strong>
 {#each tasks.filter(t=>taskSpan(t,day,day)) as task}<button class={'ws-calendar-task '+task.status} onclick={()=>onopen(task)} title={task.title+' · '+statuses[task.status]}>{task.title}</button>{/each}
 </section>{/each}
</div></div>
