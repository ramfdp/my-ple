<script lang="ts">
 import { api, setAuthenticated, type User } from '$lib/api';
 import { language, setLanguage, t as translate } from '$lib/i18n';

 let busy = $state(false);
 let error = $state('');

 async function select(nextLang: 'id' | 'en') {
  if (busy || $language === nextLang) return;
  busy = true;
  error = '';
  const prevLang = $language;
  // Optimistic update for instant animation response
  setLanguage(nextLang);
  try {
   const { user } = await api<{ user: User }>('/profile/language', {
    method: 'PATCH',
    body: JSON.stringify({ language: nextLang })
   });
   setAuthenticated(user);
  } catch {
   error = 'Gagal menyimpan bahasa.';
   setLanguage(prevLang);
  } finally {
   busy = false;
  }
 }
</script>

<div class="lang-container">
 <!-- ponytail: sleek oval toggle with animated sliding pill thumb, zero extra libraries -->
 <div
  class="lang-pill"
  class:busy
  role="group"
  aria-label={$translate('Pilih bahasa')}
 >
  <span
   class="slider-thumb"
   class:is-en={$language === 'en'}
   aria-hidden="true"
  ></span>

  <button
   type="button"
   class="lang-btn"
   class:active={$language === 'id'}
   disabled={busy}
   aria-pressed={$language === 'id'}
   onclick={(e) => { e.stopPropagation(); void select('id'); }}
   title="Bahasa Indonesia"
  >
   <svg class="globe-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <circle cx="12" cy="12" r="10"/>
    <line x1="2" y1="12" x2="22" y2="12"/>
    <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
   </svg>
   <span>ID</span>
  </button>

  <button
   type="button"
   class="lang-btn"
   class:active={$language === 'en'}
   disabled={busy}
   aria-pressed={$language === 'en'}
   onclick={(e) => { e.stopPropagation(); void select('en'); }}
   title="English"
  >
   <span>EN</span>
  </button>
 </div>

 {#if error}
  <p class="lang-error" role="alert">{$translate(error)}</p>
 {/if}
</div>

<style>
 .lang-container {
  display: inline-flex;
  flex-direction: column;
  align-items: flex-end;
  position: relative;
 }

 .lang-pill {
  position: relative;
  display: inline-flex;
  align-items: center;
  width: 104px;
  height: 38px;
  padding: 3px;
  box-sizing: border-box;
  background: #ffffff;
  border: 1px solid var(--his-border, #dedede);
  border-radius: 9999px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), inset 0 1px 2px rgba(0, 0, 0, 0.03);
  cursor: pointer;
  user-select: none;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.16s ease;
 }

 .lang-pill:hover {
  border-color: #bbf7d0;
  box-shadow: 0 2px 8px rgba(0, 122, 65, 0.1), inset 0 1px 2px rgba(0, 0, 0, 0.02);
  transform: translateY(-1px);
 }

 .lang-pill:focus-visible {
  outline: 2px solid var(--his-primary, #007a41);
  outline-offset: 2px;
 }

 .slider-thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: calc(50% - 3px);
  height: calc(100% - 6px);
  border-radius: 9999px;
  background: linear-gradient(135deg, var(--his-primary, #007a41), #009b50);
  box-shadow: 0 2px 6px rgba(0, 122, 65, 0.3);
  transition: transform 0.28s cubic-bezier(0.34, 1.35, 0.64, 1);
  pointer-events: none;
  z-index: 1;
 }

 .slider-thumb.is-en {
  transform: translateX(100%);
 }

 .lang-btn {
  /* Override global button sizing from base.css */
  min-width: unset !important;
  min-height: unset !important;
  padding: 0 !important;
  flex: 1;
  height: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: 0;
  border-radius: 9999px;
  background: transparent;
  color: var(--his-muted, #595959);
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  z-index: 2;
  cursor: pointer;
  transition: color 0.2s ease, transform 0.16s ease;
 }

 .lang-btn.active {
  color: #ffffff;
  font-weight: 700;
 }

 .lang-btn:hover:not(.active):not(:disabled) {
  color: var(--his-text, #171717);
 }

 .globe-icon {
  width: 13px;
  height: 13px;
  stroke: currentColor;
  flex-shrink: 0;
  transition: stroke 0.2s ease;
 }

 .lang-btn.active .globe-icon {
  stroke: #ffffff;
 }

 .lang-pill.busy {
  opacity: 0.7;
  cursor: wait;
 }

 .lang-error {
  position: absolute;
  top: 100%;
  right: 0;
  margin: 4px 0 0;
  font-size: 0.75rem;
  color: var(--his-danger, #c52327);
  white-space: nowrap;
 }
</style>
