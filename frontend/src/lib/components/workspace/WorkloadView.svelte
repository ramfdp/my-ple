<script lang="ts">
 import type {Task,Workload} from '$lib/workspace';
 let {workload,onopen}:{workload:Workload[];onopen:(t:Task)=>void}=$props();
 const levels={AVAILABLE:'Available',NORMAL:'Normal',BUSY:'Busy',OVERLOADED:'Overloaded'};
</script>
<p class="ws-muted">Indikator beban kerja, bukan ukuran produktivitas. Skor mempertimbangkan priority, deadline, keterlambatan, durasi, dan pekerjaan yang berlangsung bersamaan.</p>
{#if !workload.length}<p class="ws-empty">Belum ada data pekerjaan aktif atau anggota divisi.</p>{/if}
<div class="ws-directory">{#each workload as person}<article class="ws-panel"><div class="ws-row"><h2>{person.member.name}</h2><span class={'ws-status '+person.workloadLevel}>{levels[person.workloadLevel]}</span></div><strong>{person.activeTaskCount} pekerjaan aktif · Skor {person.workloadScore}</strong>
 <p>{person.highPriorityCount} priority tinggi<br/>{person.dueSoonCount} deadline dalam 7 hari<br/>{person.overdueCount} terlambat<br/>{person.overlapCount} pekerjaan tumpang tindih tambahan</p>
 {#each person.tasks as task}<p><button class="ws-link" onclick={()=>onopen(task)}>{task.title}</button></p>{:else}<p>Belum ada pekerjaan aktif.</p>{/each}
</article>{/each}</div>
