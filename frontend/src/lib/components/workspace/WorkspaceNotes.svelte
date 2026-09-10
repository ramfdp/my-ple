<script lang="ts">
 import {onMount} from 'svelte';
 import {api,currentUser} from '$lib/api';
 import {subscribeChanges} from '$lib/realtime';
 import type {Workspace,Note,NoteInput,PageLine} from '$lib/workspace';
 let {workspace}:{workspace:Workspace}=$props();
 let pages=$state<Note[]>([]),selected=$state<Note|null>(null),editing=$state(false),busy=$state(false),loading=$state(true),error=$state(''),confirmDelete=$state(false);
 let input=$state<NoteInput>({title:'',parentId:null,visibility:'DIVISION',content:[],version:0});
 async function load(){loading=true;try{pages=(await api<{pages:Note[]}>('/workspaces/'+workspace.id+'/pages')).pages}catch(e){error=(e as Error).message}finally{loading=false}}
 onMount(()=>{void load();return subscribeChanges(['workspace.changed'],async c=>{if(c.workspaceId!==undefined&&c.workspaceId!==workspace.id)return;await load();if(selected){const latest=pages.find(p=>p.id===selected?.id);if(editing){if(!latest||latest.version!==selected.version)error='Catatan server berubah. Draf tetap tersedia; batalkan edit untuk memuat versi terbaru.'}else selected=latest??null}})});
 async function open(note:Note){if(busy)return;busy=true;selected=null;error='';try{selected=(await api<{page:Note}>('/workspace-pages/'+note.id)).page;editing=false;confirmDelete=false}catch(e){error=(e as Error).message}finally{busy=false}}
 function create(parent:Note|null=null){selected=null;input={title:'',parentId:parent?.id??null,visibility:parent?.visibility??'DIVISION',content:[{kind:'text',text:'',checked:false}],version:0};editing=true;error=''}
 function edit(){if(!selected)return;input={title:selected.title,parentId:selected.parentId,visibility:selected.visibility,content:selected.content.map(l=>({...l})),version:selected.version};editing=true}
 async function save(e:SubmitEvent){e.preventDefault();if(busy)return;busy=true;error='';try{selected=(await api<{page:Note}>(selected?'/workspace-pages/'+selected.id:'/workspaces/'+workspace.id+'/pages',{method:selected?'PATCH':'POST',body:JSON.stringify(input)})).page;editing=false;await load()}catch(e){error=(e as Error).message}finally{busy=false}}
 async function remove(){if(!selected||busy)return;busy=true;error='';try{await api('/workspace-pages/'+selected.id,{method:'DELETE',body:JSON.stringify({version:selected.version})});selected=null;confirmDelete=false;await load()}catch(e){error=(e as Error).message+' Hapus atau pindahkan halaman anak terlebih dahulu.'}finally{busy=false}}
 const kinds:{value:PageLine['kind'];label:string}[]=[{value:'text',label:'Paragraf'},{value:'heading',label:'Heading'},{value:'bold',label:'Bold'},{value:'italic',label:'Italic'},{value:'bullet',label:'List'},{value:'check',label:'Checklist'},{value:'link',label:'Link'}];
</script>
{#snippet tree(parentId:string|null)}
 <ul class="ws-note-tree">{#each pages.filter(p=>p.parentId===parentId) as p}<li><button disabled={editing||busy} class="ws-link" onclick={()=>open(p)}>{p.title} {p.visibility==='PERSONAL'?'(Personal)':''}</button>{@render tree(p.id)}</li>{/each}</ul>
{/snippet}
<div class="ws-notes">
 <aside class="ws-panel"><h2>Catatan Divisi</h2><button disabled={busy||editing} onclick={()=>create()}>+ Buat Catatan</button>{#if loading}<p role="status">Memuat catatan…</p>{:else}{@render tree(null)}{#if !pages.length}<p>Belum ada catatan.</p>{/if}{/if}</aside>
 <section class="ws-panel">
 {#if error}<p role="alert">{error}</p>{/if}
 {#if editing}<form onsubmit={save}><fieldset disabled={busy} class="ws-note-form">
 <label>Judul Halaman<input required maxlength="200" bind:value={input.title}/></label>
 <label>Visibility<select bind:value={input.visibility} onchange={()=>input.parentId=null}><option value="DIVISION">Division — anggota divisi</option><option value="PERSONAL">Personal — hanya pembuat</option></select></label>
 <label>Halaman Induk<select bind:value={input.parentId}><option value={null}>Root / tanpa induk</option>{#each pages.filter(p=>p.id!==selected?.id&&p.visibility===input.visibility&&(p.visibility==='DIVISION'||p.createdBy===$currentUser?.id)) as p}<option value={p.id}>{p.title}</option>{/each}</select></label>
 <p class="ws-muted">Tulis catatan dengan format sederhana. Maksimal 200 paragraf dan 10.000 karakter.</p>
 {#each input.content as line,i}<div class="ws-note-line"><label>Format<select bind:value={line.kind}>{#each kinds as kind}<option value={kind.value}>{kind.label}</option>{/each}</select></label>
 <label class="ws-note-text">Isi<textarea rows="2" bind:value={line.text} maxlength="10000"></textarea></label>
 {#if line.kind==='check'}<label>Selesai<input type="checkbox" bind:checked={line.checked}/></label>{/if}
 {#if line.kind==='link'}<label>URL<input required type="url" bind:value={line.url}/></label>{/if}
 <button type="button" aria-label={'Hapus paragraf '+(i+1)} onclick={()=>input.content=input.content.filter((_,n)=>n!==i)}>✕</button></div>{/each}
 <button type="button" disabled={input.content.length>=200} onclick={()=>input.content=[...input.content,{kind:'text',text:'',checked:false}]}>+ Tambah Paragraf</button>
 <div class="ws-row"><button class="ws-primary" type="submit">{busy?'Menyimpan…':'Simpan Catatan'}</button><button type="button" onclick={()=>editing=false}>Batal</button></div>
 </fieldset></form>
 {:else if selected}<header><div class="ws-row"><h2>{selected.title}</h2><span>{selected.visibility==='PERSONAL'?'Personal':'Division'}</span></div><div class="ws-row">{#if selected.canEdit}<button onclick={edit}>Edit Catatan</button><button onclick={()=>create(selected)}>+ Halaman Anak</button><button onclick={()=>confirmDelete=true}>Hapus Catatan</button>{/if}</div></header>
 {#if confirmDelete}<p>Hapus halaman ini? Halaman dengan anak harus dikosongkan terlebih dahulu.</p><button disabled={busy} onclick={remove}>Ya, hapus</button> <button onclick={()=>confirmDelete=false}>Batal</button>{/if}
 <article class="ws-document">{#each selected.content as line}
 {#if line.kind==='heading'}<h3>{line.text}</h3>{:else if line.kind==='bold'}<p><strong>{line.text}</strong></p>{:else if line.kind==='italic'}<p><em>{line.text}</em></p>{:else if line.kind==='bullet'}<ul><li>{line.text}</li></ul>{:else if line.kind==='check'}<p>{line.checked?'☑':'☐'} {line.text}</p>{:else if line.kind==='link'}<p><a href={line.url} target="_blank" rel="noopener noreferrer">{line.text||line.url} ↗</a></p>{:else}<p>{line.text}</p>{/if}
 {/each}</article>
 {:else}<div class="ws-empty">Pilih catatan atau buat halaman baru untuk tim Anda.</div>{/if}
 </section>
</div>
