<script lang="ts">
 import { t as translate } from '$lib/i18n';
 import { api, type JobPIC } from '$lib/api';
 let { jobId, pics, onchange }: { jobId:number; pics:JobPIC[]; onchange:(pics:JobPIC[])=>void }=$props();
 let open=$state(false), search=$state(''), users=$state<JobPIC[]>([]), busy=$state(false), loading=$state(false), error=$state(''), success=$state('');
 let options=$derived(users.filter(user=>!pics.some(pic=>pic.userId===user.userId)));
 $effect(()=>{
  const query=search;
  if(!open)return;
  let cancelled=false;
  loading=true;
  const timer=setTimeout(async()=>{
   try { const result=await api<{users:JobPIC[]}>(`/jobs/pic-users?q=${encodeURIComponent(query)}`); if(!cancelled){users=result.users;error='';} }
   catch(e){if(!cancelled)error=e instanceof Error?e.message:'Akun tidak dapat dimuat.';}
   finally{if(!cancelled)loading=false;}
  },200);
  return ()=>{cancelled=true;clearTimeout(timer);};
 });
 async function change(userId:string,remove=false){
  if(busy)return;busy=true;error='';success='';
  try {
   const result=await api<{pics:JobPIC[]}>(`/jobs/${jobId}/pics${remove?`/${userId}`:''}`,{method:remove?'DELETE':'POST',...(remove?{}:{body:JSON.stringify({userId})})});
   onchange(result.pics);success=remove?'PIC dihapus.':'PIC ditambahkan.';
  }catch(e){error=e instanceof Error?e.message:'PIC tidak dapat disimpan.';}
  finally{busy=false;}
 }
</script>

<div class="pics">
 <span>{$translate("PIC:")}</span>
 {#each pics as pic (pic.userId)}
  <span class="chip">{pic.name}{pic.isActive?'':$translate(' (Nonaktif)')} <button type="button" disabled={busy} aria-label={`Hapus PIC ${pic.name}`} onclick={()=>void change(pic.userId,true)}>×</button></span>
 {:else}<span>{$translate("Belum ada")}</span>{/each}
 <button type="button" aria-expanded={open} onclick={()=>open=!open}>{$translate("+ Tambah PIC")}</button>
</div>
{#if open}
 <div class="selector">
  <label>{$translate("Cari akun")}<input type="search" bind:value={search} placeholder={$translate("Nama, username, atau email")} /></label>
  {#if loading}<span role="status">{$translate("Memuat akun…")}</span>{:else}
   {#each options as user (user.userId)}<button type="button" disabled={busy} onclick={()=>void change(user.userId)}>{user.name} ({user.username}) +</button>
   {:else}<span>{$translate("Tidak ada akun lain yang cocok.")}</span>{/each}
   <small>{$translate("Ketik nama untuk mencari akun lainnya.")}</small>
  {/if}
 </div>
{/if}
{#if busy}<small role="status">{$translate("Menyimpan…")}</small>{/if}
{#if error}<small role="alert">{$translate(error)}</small>{/if}
{#if success}<small role="status">{$translate(success)}</small>{/if}

<style>
 .pics{display:flex;flex-wrap:wrap;align-items:center;gap:6px;margin-top:8px;font-size:.85rem;font-weight:400;overflow-wrap:anywhere}.chip{display:inline-flex;align-items:center;gap:4px;border-radius:12px;background:#e5f2eb;padding:2px 8px;max-width:100%}.pics button,.selector button{cursor:pointer;padding:4px 8px}.chip button{border:0;background:transparent}.selector{display:flex;flex-direction:column;gap:6px;margin-top:8px;max-height:240px;overflow:auto;font-weight:400}.selector label{display:flex;flex-direction:column;gap:4px}.selector input{min-width:0;width:100%;box-sizing:border-box}.selector button{text-align:left;white-space:normal;overflow-wrap:anywhere}small{display:block;font-weight:400}
</style>
