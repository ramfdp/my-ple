<script lang="ts">
 import { api, currentUser } from '$lib/api';
 import { language, t as translate } from '$lib/i18n';
 import { tick } from 'svelte';
 import { cubicInOut, cubicOut } from 'svelte/easing';
 import { fly } from 'svelte/transition';
 import ToastNotification from './ToastNotification.svelte';
 type Source = { id:string; kind:string; title:string; excerpt:string; url:string };
 type Entry = { role:'user'|'assistant'; content:string; sources?:Source[] };
 type Tab = 'chat'|'feedback';
 let open = $state(false), tab = $state<Tab>('chat'), busy = $state(false), question = $state(''), error = $state('');
 let entries = $state<Entry[]>([]);
 let message = $state(''), sending = $state(false), feedbackError = $state(''), success = $state(false);
 const english = $derived($language === 'en');
 const sourceKind = (kind:string) => kind==='lesson' ? (english?'Evaluation':'Evaluasi') : kind==='task' ? (english?'Task':'Pekerjaan') : (english?'Report':'Laporan');
 let launcher = $state<HTMLButtonElement>(), chatTab = $state<HTMLButtonElement>(), feedbackTab = $state<HTMLButtonElement>();
 let messageList = $state<HTMLDivElement>();
 function motionDuration(duration: number) { return typeof matchMedia!=='undefined'&&matchMedia('(prefers-reduced-motion: reduce)').matches?0:duration; }
 function morph(node: HTMLElement) {
  const {width,height}=node.getBoundingClientRect();
  return {
   duration:motionDuration(600), easing:cubicInOut,
   css:(t:number,u:number)=>`clip-path:inset(${Math.max(0,height-56)*u}px 0 0 ${Math.max(0,width-56)*u}px round ${28*u+20*t}px);transform:translateY(${68*u}px);background:color-mix(in srgb,var(--his-primary) ${100*u}%,var(--his-surface,#fff));--content-opacity:${Math.max(0,(t-.3)/.7)}`
  };
 }
 function focusTab() { if(open) (tab==='chat'?chatTab:feedbackTab)?.focus(); }
 async function scrollToEnd() { await tick(); if(messageList) messageList.scrollTop=messageList.scrollHeight; }
 function toggle() {
  if (sending) return;
  open = !open;
  if (open) success=false;
  else launcher?.focus();
 }
 function selectTab(next: Tab) {
  if (sending) return;
  tab = next;
  void tick().then(() => {
   (next==='chat'?chatTab:feedbackTab)?.focus();
   if (next==='chat') void scrollToEnd();
  });
 }
 function tabKeydown(event: KeyboardEvent) {
  if (sending) return;
  if (event.key==='ArrowRight' || event.key==='ArrowLeft') { event.preventDefault(); selectTab(tab==='chat'?'feedback':'chat'); }
  else if (event.key==='Home' || event.key==='End') { event.preventDefault(); selectTab(event.key==='Home'?'chat':'feedback'); }
 }
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
 async function submitFeedback(event: SubmitEvent) {
  event.preventDefault();
  if (sending) return;
  if (!message.trim()) { feedbackError='Isi kritik atau saran antara 1 hingga 4.000 karakter.'; return; }
  sending=true; feedbackError='';
  try {
   await api<void>('/feedback',{method:'POST',body:JSON.stringify({message})});
   message=''; open=false; success=true;
  } catch (failure) { feedbackError=failure instanceof Error?failure.message:'Kritik dan saran gagal dikirim. Silakan coba lagi.'; }
  finally { sending=false; }
  if(success) { await tick(); launcher?.focus(); }
 }
</script>

<svelte:window onkeydown={(event)=>{if(open&&event.key==='Escape'){event.preventDefault();toggle()}}}/>
<div class="hermes-widget">
 {#if open}
  <div class="hermes-panel" role="dialog" aria-modal="false" aria-label={english?'Chat and feedback':'Chat dan masukan'} transition:morph onintrostart={event=>{event.currentTarget.inert=false}} onintroend={focusTab} onoutrostart={event=>{event.currentTarget.inert=true}}>
   <header><div class="header-brand"><span class="hermes-mark" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path d="m12 3 2.7 6.3L21 12l-6.3 2.7L12 21l-2.7-6.3L3 12l6.3-2.7Z"/></svg></span><div><strong>MyHIS</strong><small>{english?'Your work companion':'Teman kerja Anda'}</small></div></div><button class="panel-close" type="button" aria-label={english?'Close panel':'Tutup panel'} disabled={sending} onclick={toggle}>×</button></header>
   <div class="tabs" class:feedback={tab==='feedback'} role="tablist" aria-label={english?'Choose service':'Pilih layanan'} tabindex="-1" onkeydown={tabKeydown}>
    <button bind:this={chatTab} id="hermes-tab" type="button" role="tab" aria-selected={tab==='chat'} aria-controls="hermes-content" tabindex={tab==='chat'?0:-1} disabled={sending} onclick={()=>selectTab('chat')}>{english?'Hermes chat':'Chat Hermes'}</button>
    <button bind:this={feedbackTab} id="feedback-tab" type="button" role="tab" aria-selected={tab==='feedback'} aria-controls="feedback-content" tabindex={tab==='feedback'?0:-1} disabled={sending} onclick={()=>selectTab('feedback')}>{english?'Send to Gmail':'Kirim ke Gmail'}</button>
   </div>
    <div id="hermes-content" class="chat-content" role="tabpanel" aria-labelledby="hermes-tab" hidden={tab!=='chat'}>
     <div class="hermes-messages" bind:this={messageList} role="log" aria-live="polite">
      {#if entries.length===0}<p class="hermes-intro">{english?'Ask about a work problem. Hermes searches evaluations, tasks, and reports you can access.':'Tanyakan masalah pekerjaan. Hermes mencari evaluasi, tugas, dan laporan yang dapat Anda akses.'}</p>{/if}
      {#each entries as entry}
       <article class:user={entry.role==='user'} in:fly={{y:8,duration:motionDuration(300),easing:cubicOut}}><small class="message-author">{entry.role==='user'?(english?'You':'Anda'):'Hermes'}</small><p>{entry.content}</p>
        {#if entry.sources?.length}<ul>{#each entry.sources as source}<li><a href={source.url}>{source.title}</a> <small>({sourceKind(source.kind)})</small></li>{/each}</ul>{/if}
       </article>
      {/each}
      {#if busy}<p class="search-status" role="status"><span aria-hidden="true">✦</span>{english?'Hermes is searching…':'Hermes mencari jawaban…'}</p>{/if}
     </div>
     {#if error}<p class="panel-error" role="alert">{error}</p>{/if}
     <form class="chat-form" onsubmit={send}><label for="hermes-question">{english?'Question':'Pertanyaan'}</label><textarea id="hermes-question" bind:value={question} maxlength="1000" rows="2" placeholder={english?'Example: how can this issue be solved?':'Contoh: bagaimana mengatasi masalah ini?'} onkeydown={(event)=>{if(event.key==='Enter'&&!event.shiftKey){event.preventDefault();void send(event)}}}></textarea><button disabled={busy||!question.trim()} type="submit">{english?'Send':'Kirim'}</button></form>
    </div>
    <div id="feedback-content" class="feedback-content" role="tabpanel" aria-labelledby="feedback-tab" hidden={tab!=='feedback'}>
     <form onsubmit={submitFeedback} aria-busy={sending}>
      <h2>{$translate('Kritik & Saran')}</h2>
      <p>{$translate('Sampaikan masukan Anda untuk MyHIS.')}</p>
      <p class="identity">{$translate('Email akun')}: <strong>{$currentUser?.email}</strong></p>
      <label for="feedback-message">{$translate('Kritik atau saran Anda')}</label>
      <textarea id="feedback-message" bind:value={message} required maxlength="4000" rows="6" disabled={sending} aria-describedby="feedback-limit" placeholder={$translate('Tuliskan kritik atau saran di sini…')}></textarea>
      <small id="feedback-limit">{message.length}/4.000</small>
      {#if feedbackError}<p class="panel-error" role="alert">{$translate(feedbackError)}</p>{/if}
      <footer><button type="button" disabled={sending} onclick={toggle}>{$translate('Tutup')}</button><button class="send" type="submit" disabled={sending}>{$translate(sending?'Mengirim…':'Kirim')}</button></footer>
     </form>
    </div>
  </div>
 {/if}
 <button bind:this={launcher} class="hermes-launcher" class:expanded={open} type="button" aria-label={open?(english?'Close chat and feedback':'Tutup chat dan masukan'):(english?'Open chat and feedback':'Buka chat dan masukan')} aria-expanded={open} disabled={sending} onclick={toggle}><svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7"><path class="launcher-spark" d="m12 3 2.7 6.3L21 12l-6.3 2.7L12 21l-2.7-6.3L3 12l6.3-2.7Z"/><path class="launcher-close" d="m6 6 12 12M18 6 6 18"/></svg></button>
</div>
<ToastNotification message="Kritik dan saran berhasil dikirim." visible={success} onclose={() => success=false}/>

<style>
 .hermes-widget{--his-primary:#165a9d;position:fixed;right:max(16px,env(safe-area-inset-right));bottom:max(16px,env(safe-area-inset-bottom));z-index:30;font-family:inherit}
 .hermes-launcher{display:grid;place-items:center;width:56px;height:56px;border-radius:50%;border:1px solid #ffffff40;background:var(--his-primary);color:white;box-shadow:0 8px 24px #17376530;cursor:pointer;transition:box-shadow 240ms ease,background 240ms ease}
 .hermes-launcher:hover{box-shadow:0 10px 28px #17376545}.hermes-launcher svg{width:28px;height:28px}.hermes-launcher path{transform-origin:center;transition:opacity 240ms ease,transform 240ms ease}.launcher-close{opacity:0;transform:rotate(-45deg)}.expanded .launcher-close{opacity:1;transform:rotate(0)}.expanded .launcher-spark{opacity:0;transform:rotate(45deg)}
 .hermes-panel,.hermes-panel *{box-sizing:border-box}
 .hermes-panel{position:absolute;right:0;bottom:68px;width:min(420px,calc(100vw - 32px));height:min(560px,calc(100dvh - 100px - env(safe-area-inset-bottom)));display:flex;flex-direction:column;background:var(--his-surface,#fff);color:var(--his-text);border:1px solid var(--his-border,#dce4ee);border-radius:20px;box-shadow:0 16px 48px #17376526,0 2px 8px #17376512;overflow:hidden}
 .hermes-panel header,.tabs,.chat-content,.feedback-content{opacity:var(--content-opacity,1)}
 .hermes-panel header{display:flex;align-items:center;justify-content:space-between;flex-shrink:0;padding:16px 16px 12px;gap:12px}
 .header-brand{display:flex;align-items:center;gap:12px;min-width:0}.header-brand>div{display:grid;gap:3px}.header-brand strong{font-size:16px;letter-spacing:.02em}.header-brand small{color:var(--his-muted);font-size:12px}
 .hermes-mark{display:grid;place-items:center;width:40px;height:40px;flex-shrink:0;background:var(--his-primary);color:white;border-radius:12px}.hermes-mark svg{width:26px;height:26px}
 .panel-close{border:0;border-radius:12px;background:transparent;font-size:26px;color:var(--his-muted);cursor:pointer;min-width:44px;min-height:44px;transition:background 240ms ease,color 240ms ease}.panel-close:hover{background:#edf2f8;color:var(--his-text)}
 .tabs{position:relative;display:flex;flex-shrink:0;margin:0 16px 12px;padding:4px;border:1px solid var(--his-border,#dce4ee);border-radius:14px;background:#f2f5fa}
 .tabs::before{content:'';position:absolute;top:4px;bottom:4px;left:4px;width:calc((100% - 8px)/2);border-radius:10px;background:var(--his-surface,#fff);box-shadow:0 2px 6px #17376514;transition:transform 240ms cubic-bezier(.22,.61,.36,1)}.tabs.feedback::before{transform:translateX(100%)}
 .tabs button{position:relative;flex:1;min-width:0;min-height:44px;padding:8px 4px;border:0;border-radius:10px;background:transparent;color:var(--his-muted);font:inherit;font-size:13px;font-weight:600;cursor:pointer;transition:color 240ms ease}.tabs button[aria-selected="true"]{color:var(--his-primary)}
 .chat-content{display:flex;flex:1;flex-direction:column;min-height:0}.chat-content[hidden],.feedback-content[hidden]{display:none}
 .hermes-messages{flex:1;min-height:0;overflow:auto;padding:16px;display:flex;flex-direction:column;gap:14px;background:#f6f8fc;border-top:1px solid #edf1f7;scrollbar-width:thin}
 .hermes-messages article{align-self:flex-start;flex-shrink:0;max-width:94%;background:var(--his-surface,#fff);border:1px solid #e1e8f2;border-radius:16px 16px 16px 4px;padding:12px 14px;overflow-wrap:anywhere;font-size:14px;line-height:1.6;box-shadow:0 2px 4px #17376505}
 .hermes-messages article.user{align-self:flex-end;background:color-mix(in srgb,var(--his-primary) 9%,white);border-color:color-mix(in srgb,var(--his-primary) 15%,white);border-radius:16px 16px 4px 16px}
 .message-author{display:block;margin-bottom:4px;color:var(--his-primary);font-size:11px;font-weight:700;letter-spacing:.03em}.hermes-messages p{margin:0;white-space:pre-wrap}
 .hermes-messages ul{display:flex;flex-wrap:wrap;gap:6px;list-style:none;padding:10px 0 0;margin:10px 0 0;border-top:1px solid #e6ecf4}.hermes-messages li{max-width:100%;padding:4px 8px;border-radius:8px;background:#f1f5fb;font-size:12px}.hermes-messages a{color:var(--his-primary);text-underline-offset:3px}.hermes-messages li small{color:var(--his-muted)}
 .hermes-intro{padding:16px;border:1px dashed #c8d5e7;border-radius:14px;background:#ffffffb3;color:var(--his-muted);font-size:14px;line-height:1.7}.search-status{display:flex;align-items:center;gap:8px;color:var(--his-muted);font-size:13px}.search-status span{color:var(--his-primary)}
 .chat-form{display:grid;flex-shrink:0;gap:8px;padding:12px 16px 16px;border-top:1px solid var(--his-border,#dce4ee)}
 .chat-form label,.feedback-content label{font-size:12px;font-weight:600;color:var(--his-muted)}
 textarea{display:block;width:100%;resize:vertical;border:1px solid #c5d2e3;border-radius:12px;padding:12px;background:#fbfcfe;color:inherit;font:inherit;font-size:14px;line-height:1.5;transition:border-color 240ms ease,box-shadow 240ms ease,background 240ms ease}textarea:focus{background:var(--his-surface,#fff);border-color:var(--his-primary);box-shadow:0 0 0 3px color-mix(in srgb,var(--his-primary) 12%,transparent)}
 .chat-form textarea{min-height:64px;max-height:25dvh}.chat-form button{justify-self:end;border:0;border-radius:10px;background:var(--his-primary);color:white;padding:10px 20px;min-height:44px;font:inherit;font-size:14px;font-weight:600;cursor:pointer}
 .feedback-content{flex:1;min-height:0;overflow:auto;padding:16px;scrollbar-width:thin}.feedback-content h2{margin:0;font-size:18px}.feedback-content p{color:var(--his-muted);font-size:14px;line-height:1.6}.feedback-content .identity{padding:10px 12px;background:#f2f5fa;border:1px solid #e3eaf4;border-radius:10px;overflow-wrap:anywhere;font-size:12px}.feedback-content label{display:block;margin:18px 0 8px}
 .feedback-content textarea{min-height:120px;max-height:35dvh}.feedback-content small{display:block;text-align:right;color:var(--his-muted);font-size:11px;margin-top:6px}
 .feedback-content footer{position:sticky;bottom:-16px;display:flex;justify-content:flex-end;gap:10px;margin:20px -16px -16px;padding:12px 16px;background:var(--his-surface,#fff);border-top:1px solid var(--his-border,#dce4ee)}.feedback-content footer button{min-height:44px;padding:8px 18px;border:1px solid var(--his-border);border-radius:10px;background:transparent;color:inherit;font:inherit;font-size:14px;cursor:pointer}.feedback-content footer .send{background:var(--his-primary);color:white;border-color:var(--his-primary);font-weight:600}
 .panel-error{margin:0;padding:10px 16px;color:#9b1c1c;overflow-wrap:anywhere;font-size:13px;background:#fff4f3}.feedback-content .panel-error{margin-top:12px;padding:10px;border-radius:10px;color:#9b1c1c}
 button:disabled{opacity:.55;cursor:default}.hermes-widget button:focus-visible,.hermes-widget a:focus-visible,.hermes-widget textarea:focus-visible{outline:3px solid #e6ad32;outline-offset:2px}
 @media(prefers-reduced-motion:reduce){.hermes-widget *,.tabs::before{transition:none!important}}
 @media print{.hermes-widget{display:none}}
</style>
