<script lang="ts">
 import { api } from '$lib/api';
 import { language } from '$lib/i18n';
 import { tick } from 'svelte';
 type Source = { id:string; kind:string; title:string; excerpt:string; url:string };
 type Entry = { role:'user'|'assistant'; content:string; sources?:Source[] };
 let open = $state(false), busy = $state(false), question = $state(''), error = $state('');
 let entries = $state<Entry[]>([]);
 const english = $derived($language === 'en');
 const sourceKind = (kind:string) => kind==='lesson' ? (english?'Evaluation':'Evaluasi') : kind==='task' ? (english?'Task':'Pekerjaan') : (english?'Report':'Laporan');
 let input = $state<HTMLTextAreaElement>();
 let messageList = $state<HTMLDivElement>();
 async function scrollToEnd() { await tick(); if(messageList) messageList.scrollTop=messageList.scrollHeight; }
 async function send(event: Event) {
  event.preventDefault();
  const text=question.trim();
  if (!text || busy) return;
  const history=entries.slice(-6).map(({role,content})=>({role,content}));
  entries=[...entries,{role:'user',content:text}]; question=''; error=''; busy=true;
  void scrollToEnd();
  try {
   const result=await api<{answer:string;sources:Source[]}>('/hermes/chat',{method:'POST',body:JSON.stringify({question:text,history})});
   entries=[...entries,{role:'assistant',content:result.answer,sources:result.sources}];
  } catch (cause) { error=cause instanceof Error?cause.message:(english?'Hermes is unavailable.':'Hermes belum tersedia.'); entries=entries.slice(0,-1); question=text; }
  finally { busy=false; void scrollToEnd(); }
 }
 function toggle() { open=!open; if(open) setTimeout(()=>input?.focus(),0); }
</script>

<svelte:window onkeydown={(event)=>{if(open&&event.key==='Escape')toggle()}}/>
<div class="hermes-widget">
 {#if open}
  <section class="hermes-panel" aria-label={english?'Chat with Hermes':'Chat dengan Hermes'}>
   <header><div><strong>Hermes</strong><small>{english?'Answers from MyHIS sources':'Jawaban dari sumber MyHIS'}</small></div><button type="button" aria-label={english?'Close Hermes chat':'Tutup chat Hermes'} onclick={toggle}>×</button></header>
   <div class="hermes-messages" bind:this={messageList} role="log" aria-live="polite">
    {#if entries.length===0}<p class="hermes-intro">{english?'Ask about a work problem. Hermes searches evaluations, tasks, and reports you can access.':'Tanyakan masalah pekerjaan. Hermes mencari evaluasi, tugas, dan laporan yang dapat Anda akses.'}</p>{/if}
    {#each entries as entry}
     <article class:user={entry.role==='user'}><p>{entry.content}</p>
      {#if entry.sources?.length}<ul>{#each entry.sources as source}<li><a href={source.url}>{source.title}</a> <small>({sourceKind(source.kind)})</small></li>{/each}</ul>{/if}
     </article>
    {/each}
    {#if busy}<p role="status">{english?'Hermes is searching…':'Hermes mencari jawaban…'}</p>{/if}
   </div>
   {#if error}<p class="hermes-error" role="alert">{error}</p>{/if}
   <form onsubmit={send}><label for="hermes-question">{english?'Question':'Pertanyaan'}</label><textarea id="hermes-question" bind:this={input} bind:value={question} maxlength="1000" rows="2" placeholder={english?'Example: how can this issue be solved?':'Contoh: bagaimana mengatasi masalah ini?'} onkeydown={(event)=>{if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();void send(event)}}}></textarea><button disabled={busy||!question.trim()} type="submit">{english?'Send':'Kirim'}</button></form>
  </section>
 {/if}
 <button class="hermes-launcher" type="button" aria-label={open?(english?'Close Hermes chat':'Tutup chat Hermes'):(english?'Open Hermes chat':'Buka chat Hermes')} aria-expanded={open} onclick={toggle}>✦<span class="sr-only">Hermes</span></button>
</div>

<style>
 .hermes-widget{position:fixed;right:20px;bottom:20px;z-index:30;font-family:inherit}.hermes-launcher{width:54px;height:54px;border-radius:50%;border:0;background:var(--his-primary);color:white;box-shadow:0 6px 24px #0004;font-size:24px;cursor:pointer}.hermes-launcher:focus-visible,.hermes-panel button:focus-visible,.hermes-panel a:focus-visible,.hermes-panel textarea:focus-visible{outline:3px solid #fbbf24;outline-offset:2px}.hermes-panel{position:absolute;right:0;bottom:68px;width:min(390px,calc(100vw - 32px));height:min(540px,calc(100dvh - 110px));display:flex;flex-direction:column;background:var(--his-surface,#fff);color:var(--his-text);border:1px solid #ccd5df;border-radius:16px;box-shadow:0 12px 40px #0003;overflow:hidden}.hermes-panel header{display:flex;justify-content:space-between;align-items:center;padding:14px 16px;border-bottom:1px solid #d5dde5}.hermes-panel header div{display:flex;flex-direction:column}.hermes-panel header small{font-size:12px}.hermes-panel header button{border:0;background:transparent;font-size:26px;cursor:pointer;color:inherit}.hermes-messages{flex:1;overflow:auto;padding:14px;display:flex;flex-direction:column;gap:12px}.hermes-messages article{align-self:flex-start;max-width:95%;background:#eef4f9;border-radius:12px;padding:10px 12px;overflow-wrap:anywhere}.hermes-messages article.user{align-self:flex-end;background:#dbeafe}.hermes-messages p{margin:0;white-space:pre-wrap}.hermes-messages ul{padding-left:18px;margin:10px 0 0}.hermes-messages a{color:#165a9d}.hermes-intro{font-size:14px;line-height:1.5}.hermes-panel form{display:grid;gap:6px;padding:12px;border-top:1px solid #d5dde5}.hermes-panel form label{font-size:13px;font-weight:600}.hermes-panel textarea{resize:vertical;min-height:48px;border:1px solid #9aaaba;border-radius:8px;padding:8px;background:transparent;color:inherit;font:inherit}.hermes-panel form button{justify-self:end;border:0;border-radius:8px;background:var(--his-primary);color:white;padding:8px 16px;cursor:pointer}.hermes-panel form button:disabled{opacity:.55;cursor:default}.hermes-error{margin:0;padding:8px 14px;color:#9b1c1c;font-size:13px}.sr-only{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}@media(max-width:480px){.hermes-widget{right:12px;bottom:12px}.hermes-panel{height:min(560px,calc(100dvh - 90px))}}
</style>
