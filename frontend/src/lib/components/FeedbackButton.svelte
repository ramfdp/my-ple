<script lang="ts">
 import { api, currentUser } from '$lib/api';
 import { t as translate } from '$lib/i18n';
 import ToastNotification from './ToastNotification.svelte';
 let dialog: HTMLDialogElement;
 let message = $state('');
 let sending = $state(false);
 let error = $state('');
 let success = $state(false);
 function keepFocus(event: KeyboardEvent) {
  if (event.key !== 'Tab') return;
  const controls = [...dialog.querySelectorAll<HTMLElement>('textarea:not(:disabled), button:not(:disabled)')];
  const first = controls[0], last = controls.at(-1);
  if (!first) { event.preventDefault(); return; }
  if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
  else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus(); }
 }
 async function submit(event: SubmitEvent) {
  event.preventDefault();
  if (sending) return;
  if (!message.trim()) { error = 'Isi kritik atau saran antara 1 hingga 4.000 karakter.'; return; }
  sending = true; error = '';
  try {
   await api<void>('/feedback', { method: 'POST', body: JSON.stringify({ message }) });
   dialog.close(); message = ''; success = true;
  } catch (failure) { error = failure instanceof Error ? failure.message : 'Kritik dan saran gagal dikirim. Silakan coba lagi.'; }
  finally { sending = false; }
 }
</script>

<button class="feedback-button" type="button" aria-label={$translate('Kritik & Saran')} title={$translate('Kritik & Saran')} onclick={() => { success = false; dialog.showModal(); }}>
 <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 11.5a8.4 8.4 0 0 1-.9 3.8 8.5 8.5 0 0 1-7.6 4.7 8.4 8.4 0 0 1-3.8-.9L3 21l1.9-5.7a8.4 8.4 0 0 1-.9-3.8 8.5 8.5 0 0 1 4.7-7.6 8.4 8.4 0 0 1 3.8-.9h.5a8.5 8.5 0 0 1 8 8v.5Z"/><path d="M8 12h8M8 8h5"/></svg>
</button>
<dialog bind:this={dialog} aria-labelledby="feedback-title" onkeydown={keepFocus} oncancel={(event) => { if (sending) event.preventDefault(); }}>
 <form onsubmit={submit} aria-busy={sending}>
  <h2 id="feedback-title">{$translate('Kritik & Saran')}</h2>
  <p class="description">{$translate('Sampaikan masukan Anda untuk MyHIS.')}</p>
  <p class="identity">{$translate('Email akun')}: <strong>{$currentUser?.email}</strong></p>
  <label for="feedback-message">{$translate('Kritik atau saran Anda')}</label>
  <textarea id="feedback-message" bind:value={message} required maxlength="4000" rows="6" disabled={sending} aria-describedby="feedback-limit" placeholder={$translate('Tuliskan kritik atau saran di sini…')}></textarea>
  <small id="feedback-limit">{message.length}/4.000</small>
  {#if error}<p class="error" role="alert">{$translate(error)}</p>{/if}
  <footer><button type="button" disabled={sending} onclick={() => dialog.close()}>{$translate('Tutup')}</button><button class="send" type="submit" disabled={sending}>{$translate(sending ? 'Mengirim…' : 'Kirim')}</button></footer>
 </form>
</dialog>
<ToastNotification message="Kritik dan saran berhasil dikirim." visible={success} onclose={() => success = false}/>

<style>
 .feedback-button{position:fixed;right:max(24px,env(safe-area-inset-right));bottom:max(24px,env(safe-area-inset-bottom));z-index:5;width:56px;height:56px;display:grid;place-items:center;border:0;border-radius:50%;background:var(--his-primary);color:white;box-shadow:0 4px 16px #0003;cursor:pointer;transition:transform 180ms ease,box-shadow 180ms ease}
 .feedback-button:hover{transform:translateY(-2px);box-shadow:0 6px 20px #0004}.feedback-button svg{width:26px;height:26px}
 dialog{width:min(480px,calc(100vw - 32px));max-height:calc(100dvh - 32px);box-sizing:border-box;overflow:auto;border:1px solid var(--his-border);border-radius:16px;padding:24px;color:var(--his-text);background:white;opacity:0;transform:translateY(8px);transition:opacity 180ms ease,transform 180ms ease,display 180ms allow-discrete,overlay 180ms allow-discrete}
 dialog[open]{opacity:1;transform:translateY(0)}dialog::backdrop{background:#0006;opacity:0;transition:opacity 180ms ease,display 180ms allow-discrete,overlay 180ms allow-discrete}dialog[open]::backdrop{opacity:1}
 @starting-style{dialog[open]{opacity:0;transform:translateY(8px)}dialog[open]::backdrop{opacity:0}}
 h2{margin:0;font-size:1.4rem}.description{color:var(--his-muted)}.identity{overflow-wrap:anywhere;font-size:.9rem}label{display:block;font-weight:600;margin:20px 0 8px}
 textarea{display:block;width:100%;box-sizing:border-box;resize:vertical;min-height:120px;max-height:45dvh;border:1px solid var(--his-border);border-radius:8px;padding:12px;font:inherit;color:inherit;background:white}small{display:block;text-align:right;color:var(--his-muted);margin-top:4px}.error{color:#b42318;overflow-wrap:anywhere}
 footer{display:flex;justify-content:flex-end;gap:12px;margin-top:20px}footer button{min-height:44px;padding:10px 20px;border:1px solid var(--his-border);border-radius:8px;background:white;color:var(--his-text);font:inherit;cursor:pointer}footer .send{background:var(--his-primary);color:white;border-color:var(--his-primary)}button:disabled{opacity:.6;cursor:wait}button:focus-visible,textarea:focus-visible{outline:3px solid var(--his-primary);outline-offset:3px}
 @media(max-width:480px){dialog{padding:20px}.feedback-button{right:max(16px,env(safe-area-inset-right));bottom:max(16px,env(safe-area-inset-bottom))}}
 @media(prefers-reduced-motion:reduce){dialog,dialog::backdrop,.feedback-button{transition:none;transform:none}}
 @media print{.feedback-button,dialog{display:none}}
</style>
