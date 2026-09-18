<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
 import {dateValue,taskSpan,taskStatus,labelDate,type Task} from '$lib/workspace';
 let {tasks,days,onopen}:{tasks:Task[];days:string[];onopen:(t:Task)=>void}=$props();
 let blanks=$derived((dateValue(days[0]).getUTCDay()+6)%7);
 let selectedDay=$state('');
 let dialog:HTMLDialogElement;
 function closeDay(){selectedDay='';dialog.close()}
 function openTask(task:Task){closeDay();onopen(task)}
</script>
<div class="ws-scroll"><div class="ws-calendar">
 {#each [0,1,2,3,4,5,6] as day}<div class="ws-calendar-heading">{new Date(Date.UTC(2026,0,5+day)).toLocaleDateString($locale,{weekday:'short',timeZone:'UTC'})}</div>{/each}
 {#each Array(blanks) as _}<div class="ws-calendar-day ws-calendar-blank"></div>{/each}
 {#each days as day}<section class="ws-calendar-day" aria-label={day}><button aria-label={$translate('Lihat semua pekerjaan ')+labelDate(day, $locale)} aria-pressed={selectedDay===day} onclick={()=>{selectedDay=day;dialog.showModal()}}>{Number(day.slice(-2))}</button>
 {#each tasks.filter(t=>t.status!=='DONE'&&taskSpan(t,day,day)) as task (task.id)}<button class={'ws-calendar-task '+task.status} onclick={()=>onopen(task)} title={task.title+' · '+$translate(taskStatus(task))}>{task.title} · {$translate(taskStatus(task))}</button>{/each}
 </section>{/each}
</div></div>
<dialog bind:this={dialog} class="ws-dialog" oncancel={()=>selectedDay=''} aria-labelledby="calendar-date-title"><h2 id="calendar-date-title">{$translate("Semua pekerjaan ·")} {labelDate(selectedDay, $locale)}</h2><button onclick={closeDay}>{$translate("Tutup tanggal")}</button>{#if selectedDay&&days.includes(selectedDay)}{#each tasks.filter(t=>taskSpan(t,selectedDay,selectedDay)) as task (task.id)}<p><button class="ws-link" onclick={()=>openTask(task)}>{task.title}</button> · {$translate(taskStatus(task))}</p>{:else}<p>{$translate("Tidak ada pekerjaan.")}</p>{/each}{/if}</dialog>
