<script lang="ts">
 import { api, setAuthenticated, type User } from '$lib/api';
 import { language, t as translate } from '$lib/i18n';
 let busy=$state(false), error=$state('');
 async function change(event: Event) {
  const select=event.currentTarget as HTMLSelectElement;
  busy=true;error='';
  try { const {user}=await api<{user:User}>('/profile/language',{method:'PATCH',body:JSON.stringify({language:select.value})});setAuthenticated(user); }
  catch {error='Gagal menyimpan bahasa.';select.value=$language;}
  finally {busy=false;}
 }
</script>
<div><select aria-label={$translate('Bahasa')} value={$language} disabled={busy} onchange={change}><option value="id">Bahasa Indonesia</option><option value="en">English</option></select>{#if error}<p role="alert">{$translate(error)}</p>{/if}</div>
<style>select{font:inherit;min-height:44px;border:1px solid var(--his-border);border-radius:10px;background:white;color:var(--his-text);padding:8px}p{font-size:.8rem;max-width:220px;color:var(--his-danger)}</style>
