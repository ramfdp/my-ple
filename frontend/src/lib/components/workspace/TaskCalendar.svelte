<script lang="ts">
 import {dateValue,taskSpan,taskStatus,labelDate,type Task} from '$lib/workspace';
 let {tasks,days,onopen}:{tasks:Task[];days:string[];onopen:(t:Task)=>void}=$props();
 let blanks=$derived((dateValue(days[0]).getUTCDay()+6)%7);
 let selectedDay=$state('');
 let dialog:HTMLDialogElement;
 function closeDay(){selectedDay='';dialog.close()}
 function openTask(task:Task){closeDay();onopen(task)}
</script>
<div class="ws-scroll"><div class="ws-calendar">
 {#each ['Sen','Sel','Rab','Kam','Jum','Sab','Min'] as day}<div class="ws-calendar-heading">{day}</div>{/each}
 {#each Array(blanks) as _}<div class="ws-calendar-day ws-calendar-blank"></div>{/each}
 {#each days as day}<section class="ws-calendar-day" aria-label={day}><button aria-label={'Lihat semua pekerjaan '+labelDate(day)} aria-pressed={selectedDay===day} onclick={()=>{selectedDay=day;dialog.showModal()}}>{Number(day.slice(-2))}</button>
 {#each tasks.filter(t=>t.status!=='DONE'&&taskSpan(t,day,day)) as task (task.id)}<button class={'ws-calendar-task '+task.status} onclick={()=>onopen(task)} title={task.title+' · '+taskStatus(task)}>{task.title} · {taskStatus(task)}</button>{/each}
 </section>{/each}
</div></div>
<dialog bind:this={dialog} class="ws-dialog" oncancel={()=>selectedDay=''} aria-labelledby="calendar-date-title"><h2 id="calendar-date-title">Semua pekerjaan · {labelDate(selectedDay)}</h2><button onclick={closeDay}>Tutup tanggal</button>{#if selectedDay&&days.includes(selectedDay)}{#each tasks.filter(t=>taskSpan(t,selectedDay,selectedDay)) as task (task.id)}<p><button class="ws-link" onclick={()=>openTask(task)}>{task.title}</button> · {taskStatus(task)}</p>{:else}<p>Tidak ada pekerjaan.</p>{/each}{/if}</dialog>
