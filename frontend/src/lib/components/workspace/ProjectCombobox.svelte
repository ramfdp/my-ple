<script lang="ts">
 import { t as translate } from '$lib/i18n';
 import type {WorkspaceJob} from '$lib/workspace';
 let {jobs,value=$bindable<number|null>(null)}:{jobs:WorkspaceJob[];value?:number|null}=$props();
 const uid=$props.id();
 let root:HTMLDivElement;
 let open=$state(false),query=$state(''),active=$state(-1),noProjectChosen=$state(false);
 const options=$derived([...new Map(jobs.filter(j=>j.isActive||j.id===value).map(j=>[j.id,j])).values()]);
 const selected=$derived(options.find(j=>j.id===value));
 const matches=$derived(options.filter(j=>j.name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase())));
 const choices=$derived([{id:null,name:'Tanpa project',label:'Tanpa project'},...matches].filter(j=>j.id!==null||!query.trim()||j.name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase())));
 function close(){open=false;query='';active=-1}
 function choose(id:number|null){value=id;noProjectChosen=id===null;close()}
 function keydown(e:KeyboardEvent){
  if(e.key==='Escape'&&open){e.preventDefault();e.stopPropagation();close()}
  else if(e.key==='ArrowDown'||e.key==='ArrowUp'){
   e.preventDefault();open=true;
   active=choices.length?(active<0?(e.key==='ArrowDown'?0:choices.length-1):(active+(e.key==='ArrowDown'?1:-1)+choices.length)%choices.length):-1;
   requestAnimationFrame(()=>document.getElementById(uid+'-'+active)?.scrollIntoView({block:'nearest'}));
  }else if(e.key==='Enter'&&open){e.preventDefault();if(choices[active])choose(choices[active].id)}
  else if(e.key==='Tab')close();
 }
</script>

<svelte:window onpointerdown={(e)=>{if(root&&!root.contains(e.target as Node))close()}} />
<div bind:this={root} class="project-picker ws-full">
 <label for={uid}>{$translate("Project / Pekerjaan")}</label>
 <input id={uid} role="combobox" aria-autocomplete="list" aria-expanded={open} aria-controls={uid+'-list'} aria-activedescendant={open&&active>=0?uid+'-'+active:undefined}
  autocomplete="off" placeholder={$translate("Pilih atau cari project...")} value={open?query:selected?.label??selected?.name??(noProjectChosen?$translate('Tanpa project'):'')}
  onfocus={()=>{open=true;query='';active=-1}} onclick={()=>open=true} oninput={(e)=>{query=e.currentTarget.value;open=true;active=-1}} onkeydown={keydown}
  onblur={()=>close()} />
 {#if open}
  <ul id={uid+'-list'} role="listbox" aria-label={$translate("Project / Pekerjaan")}>
   {#each choices as job,i (job.id)}
    <li id={uid+'-'+i} role="option" aria-selected={value===job.id} class:active={active===i}>
     <button type="button" tabindex="-1" onpointerdown={(e)=>e.preventDefault()} onclick={()=>choose(job.id)}>{job.id===null?$translate('Tanpa project'):job.label??job.name}</button>
    </li>
   {:else}<li role="presentation">{$translate("Tidak ada project ditemukan")}</li>{/each}
  </ul>
 {/if}
</div>

<style>
 .project-picker{min-width:0;display:grid;gap:5px}
 input{width:100%;min-width:0}
 ul{list-style:none;margin:0;padding:4px;border:1px solid #cbd5e1;border-radius:6px;max-height:min(200px,30dvh);overflow-y:auto;overscroll-behavior:contain;background:white}
 li{margin:0;overflow-wrap: break-word}
 li[role=presentation]{padding:12px;color:#64748b}
 li button{width:100%;min-height:44px;border:0!important;text-align:left;white-space:normal;overflow-wrap: break-word}
 li.active button,li[aria-selected=true] button{background:#e1eff7}
</style>
