<script lang="ts">
 import { t as translate } from '$lib/i18n';
 import {onMount} from 'svelte';
 import {api,APIError,currentUser} from '$lib/api';
 import {subscribeChanges} from '$lib/realtime';
 import type {Workspace,Note,NoteInput,PageLine} from '$lib/workspace';
 let {workspace}:{workspace:Workspace}=$props();
 let pages=$state<Note[]>([]),selected=$state<Note|null>(null),editing=$state(false),busy=$state(false),loading=$state(true),error=$state(''),confirmDelete=$state(false);
 let input=$state<NoteInput>({title:'',parentId:null,visibility:'DIVISION',content:[],version:0});
 let request=0;
 async function load(background=false){const sequence=++request;if(!background)loading=true;try{const result=await api<{pages:Note[]}>('/workspaces/'+workspace.id+'/pages');if(sequence!==request)return false;pages=result.pages;return true}catch(e){if(sequence===request)error=(e as Error).message;return false}finally{if(sequence===request)loading=false}}
 onMount(()=>{void load();const stop=subscribeChanges(['workspace.changed'],async c=>{
  if(c.workspaceId!==undefined&&c.workspaceId!==workspace.id)return;
  const id=selected?.id,version=selected?.version;
  if(!await load(true)||!id||selected?.id!==id)return;
  const latest=pages.find(p=>p.id===id);
  if(editing){if(!latest||latest.version!==version)error='Catatan server berubah. Draf tetap tersedia; batalkan edit untuk memuat versi terbaru.';return}
  if(!latest){selected=null;confirmDelete=false;return}
  if(latest.version===version)return;
  try{const detail=(await api<{page:Note}>('/workspace-pages/'+id)).page;if(selected?.id===id&&selected.version===version&&!editing)selected=detail}
  catch(e){if(selected?.id!==id||selected.version!==version||editing)return;if(e instanceof APIError&&(e.status===403||e.status===404)){selected=null;confirmDelete=false}else error=(e as Error).message}
 });return()=>{request++;stop()}});
 async function open(note:Note){if(busy)return;busy=true;selected=null;error='';try{selected=(await api<{page:Note}>('/workspace-pages/'+note.id)).page;editing=false;confirmDelete=false}catch(e){error=(e as Error).message}finally{busy=false}}
 function create(parent:Note|null=null){selected=null;input={title:'',parentId:parent?.id??null,visibility:parent?.visibility??'DIVISION',content:[{kind:'text',text:'',checked:false}],version:0};editing=true;error=''}
 function edit(){if(!selected)return;input={title:selected.title,parentId:selected.parentId,visibility:selected.visibility,content:selected.content.map(l=>({...l})),version:selected.version};editing=true}
 async function save(e:SubmitEvent){e.preventDefault();if(busy)return;busy=true;error='';try{selected=(await api<{page:Note}>(selected?'/workspace-pages/'+selected.id:'/workspaces/'+workspace.id+'/pages',{method:selected?'PATCH':'POST',body:JSON.stringify(input)})).page;editing=false;await load()}catch(e){error=(e as Error).message}finally{busy=false}}
 async function remove(){if(!selected||busy)return;busy=true;error='';try{await api('/workspace-pages/'+selected.id,{method:'DELETE',body:JSON.stringify({version:selected.version})});selected=null;confirmDelete=false;await load()}catch(e){error=(e as Error).message+' Hapus atau pindahkan halaman anak terlebih dahulu.'}finally{busy=false}}
 const kinds:{value:PageLine['kind'];label:string}[]=[{value:'text',label:'Paragraf'},{value:'heading',label:'Heading'},{value:'bold',label:'Bold'},{value:'italic',label:'Italic'},{value:'bullet',label:'List'},{value:'check',label:'Checklist'},{value:'link',label:'Link'}];
</script>
{#snippet tree(parentId:string|null)}
 <ul class="ws-note-tree">{#each pages.filter(p=>p.parentId===parentId) as p}<li><button disabled={editing||busy} class="ws-link" onclick={()=>open(p)}>{p.title} {p.visibility==='PERSONAL'?$translate('(Personal)'):''}</button>{@render tree(p.id)}</li>{/each}</ul>
{/snippet}
<div class="ws-notes">
 <aside class="ws-panel"><h2>{$translate("Catatan Divisi")}</h2><button disabled={busy||editing} onclick={()=>create()}>{$translate("+ Buat Catatan")}</button>{#if loading}<p role="status">{$translate("Memuat catatan…")}</p>{:else}{@render tree(null)}{#if !pages.length}<p>{$translate("Belum ada catatan.")}</p>{/if}{/if}</aside>
 <section class="ws-panel">
 {#if error}<p role="alert">{$translate(error)}</p>{/if}
 {#if editing}<form onsubmit={save}><fieldset disabled={busy} class="ws-note-form">
 <label>{$translate("Judul Halaman")}<input required maxlength="200" bind:value={input.title}/></label>
 <label>{$translate("Visibility")}<select bind:value={input.visibility} onchange={()=>input.parentId=null}><option value="DIVISION">{$translate("Division — anggota divisi")}</option><option value="PERSONAL">{$translate("Personal — hanya pembuat")}</option></select></label>
 <label>{$translate("Halaman Induk")}<select bind:value={input.parentId}><option value={null}>{$translate("Root / tanpa induk")}</option>{#each pages.filter(p=>p.id!==selected?.id&&p.visibility===input.visibility&&(p.visibility==='DIVISION'||p.createdBy===$currentUser?.id)) as p}<option value={p.id}>{p.title}</option>{/each}</select></label>
 <p class="ws-muted">{$translate("Tulis catatan dengan format sederhana. Maksimal 200 paragraf dan 10.000 karakter.")}</p>
 {#each input.content as line,i}<div class="ws-note-line"><label>{$translate("Format")}<select bind:value={line.kind}>{#each kinds as kind}<option value={kind.value}>{$translate(kind.label)}</option>{/each}</select></label>
 <label class="ws-note-text">{$translate("Isi")}<textarea rows="6" bind:value={line.text} maxlength="10000"></textarea></label>
 {#if line.kind==='check'}<label>{$translate("Selesai")}<input type="checkbox" bind:checked={line.checked}/></label>{/if}
 {#if line.kind==='link'}<label>{$translate("URL")}<input required type="url" bind:value={line.url}/></label>{/if}
 <button type="button" aria-label={$translate('Hapus paragraf ')+(i+1)} onclick={()=>input.content=input.content.filter((_,n)=>n!==i)}>✕</button></div>{/each}
 <button type="button" disabled={input.content.length>=200} onclick={()=>input.content=[...input.content,{kind:'text',text:'',checked:false}]}>{$translate("+ Tambah Paragraf")}</button>
 <div class="ws-row"><button class="ws-primary" type="submit">{busy?$translate('Menyimpan…'):$translate('Simpan Catatan')}</button><button type="button" onclick={()=>editing=false}>{$translate("Batal")}</button></div>
 </fieldset></form>
 {:else if selected}<header><div class="ws-row"><h2>{selected.title}</h2><span>{selected.visibility==='PERSONAL'?$translate('Personal'):$translate('Division')}</span></div><div class="ws-row">{#if selected.canEdit}<button onclick={edit}>{$translate("Edit Catatan")}</button><button onclick={()=>create(selected)}>{$translate("+ Halaman Anak")}</button><button onclick={()=>confirmDelete=true}>{$translate("Hapus Catatan")}</button>{/if}</div></header>
 {#if confirmDelete}<p>{$translate("Hapus halaman ini? Halaman dengan anak harus dikosongkan terlebih dahulu.")}</p><button disabled={busy} onclick={remove}>{$translate("Ya, hapus")}</button> <button onclick={()=>confirmDelete=false}>{$translate("Batal")}</button>{/if}
 <article class="ws-document">{#each selected.content as line}
 {#if line.kind==='heading'}<h3>{line.text}</h3>{:else if line.kind==='bold'}<p><strong>{line.text}</strong></p>{:else if line.kind==='italic'}<p><em>{line.text}</em></p>{:else if line.kind==='bullet'}<ul><li>{line.text}</li></ul>{:else if line.kind==='check'}<p>{line.checked?'☑':'☐'} {line.text}</p>{:else if line.kind==='link'}<p><a href={line.url} target="_blank" rel="noopener noreferrer">{line.text||line.url} ↗</a></p>{:else}<p>{line.text}</p>{/if}
 {/each}</article>
 {:else}<div class="ws-empty">{$translate("Pilih catatan atau buat halaman baru untuk tim Anda.")}</div>{/if}
 </section>
</div>
