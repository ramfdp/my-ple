import { writable } from 'svelte/store';
import { api, session } from './api';

type DesktopState = 'loading' | 'off' | 'on' | 'blocked' | 'unsupported' | 'unconfigured' | 'error' | 'disableError';
export const desktopState = writable<DesktopState>('loading');
const ownerKey = 'my-ple.desktop-push.owner';
let publicKey = '';

function supported() {
 return window.isSecureContext && 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window;
}

async function registration() {
 await navigator.serviceWorker.register('/desktop-push.js');
 return navigator.serviceWorker.ready;
}

async function subscribe(user: string) {
 const reg = await registration();
 const key = Uint8Array.from(atob(publicKey.replace(/-/g, '+').replace(/_/g, '/')), c => c.charCodeAt(0));
 let sub = await reg.pushManager.getSubscription();
 if (sub && (localStorage.getItem(ownerKey) !== user ||
      !sub.options.applicationServerKey || String(new Uint8Array(sub.options.applicationServerKey)) !== String(key))) {
  await sub.unsubscribe();
  sub = null;
 }
 sub ??= await reg.pushManager.subscribe({userVisibleOnly: true, applicationServerKey: key});
 if (session()?.user.id !== user) return;
 await api('/desktop-push', {method: 'PUT', body: JSON.stringify({...sub.toJSON(), userId:user})});
 localStorage.setItem(ownerKey,user);
 desktopState.set('on');
}

export async function restoreDesktopPush(user: string) {
 if (!supported()) { desktopState.set('unsupported'); return; }
 try {
  desktopState.set('loading');
  const owner = localStorage.getItem(ownerKey);
  if (owner && owner !== user) await detachDesktopPush();
  if (Notification.permission === 'denied') { desktopState.set('blocked'); return; }
  const config = await api<{publicKey:string}>('/desktop-push');
  if (session()?.user.id !== user) return;
  publicKey = config.publicKey;
  if (!publicKey) desktopState.set('unconfigured');
  else if (owner === user && Notification.permission === 'granted') await subscribe(user);
  else desktopState.set('off');
 } catch { desktopState.set('error'); }
}

export async function enableDesktopPush() {
 const user = session()?.user.id;
 if (!user) return;
 try {
  desktopState.set('loading');
  const permission = await Notification.requestPermission();
  if (permission !== 'granted') { desktopState.set(permission === 'denied' ? 'blocked' : 'off'); return; }
  if (!publicKey) publicKey = (await api<{publicKey:string}>('/desktop-push')).publicKey;
  if (!publicKey) { desktopState.set('unconfigured'); return; }
  await subscribe(user);
 } catch { desktopState.set('error'); }
}

export async function detachDesktopPush(keepPreference = false) {
 if (!supported()) return;
 const reg = await navigator.serviceWorker.getRegistration('/');
 const sub = await reg?.pushManager.getSubscription();
 if (sub) await sub.unsubscribe();
 if (reg) for (const banner of await reg.getNotifications()) banner.close();
 if (!keepPreference) localStorage.removeItem(ownerKey);
 desktopState.set('off');
}

export async function disableDesktopPush() {
 desktopState.set('loading');
 try {
  const reg = await navigator.serviceWorker.getRegistration('/');
  const sub = await reg?.pushManager.getSubscription();
  if (sub) await api('/desktop-push', {method:'DELETE',body:JSON.stringify({endpoint:sub.endpoint})});
  await detachDesktopPush();
 } catch {desktopState.set('disableError');}
}

