import {notificationSound} from './notification-sound';
import { get, writable } from 'svelte/store';
import { api, apiUrl, session, type Notification } from '$lib/api';

export const notifications = writable<{ items: Notification[]; unread: number }>({ items: [], unread: 0 });

export function addNotification(item: Notification) {
	notifications.update((state) => state.items.some(({ id }) => id === item.id)
		? state : { items: [item, ...state.items], unread: state.unread + (item.readAt ? 0 : 1) });
}

export async function markRead(id: string) {
	const item = get(notifications).items.find((entry) => entry.id === id);
	if (!item || item.readAt) return;
	await api(`/notifications/${id}/read`, { method: 'PATCH' });
	notifications.update((state) => ({ items: state.items.map((entry) => entry.id === id ? { ...entry, readAt: new Date().toISOString() } : entry), unread: Math.max(0, state.unread - 1) }));
}

export async function markAllRead() {
	await api('/notifications/read-all', { method: 'PATCH' });
	const readAt = new Date().toISOString();
	notifications.update((state) => ({ items: state.items.map((item) => ({ ...item, readAt: item.readAt ?? readAt })), unread: 0 }));
}

export async function dismissNotification(id: string) {
	const item = get(notifications).items.find((entry) => entry.id === id);
	await api(`/notifications/${id}/dismiss`, { method: 'PATCH' });
	notifications.update((state) => ({ items: state.items.filter((entry) => entry.id !== id), unread: Math.max(0, state.unread - (item && !item.readAt ? 1 : 0)) }));
}

let seenUser = '';
const seen = new Set<string>();

export function startNotifications() {
	notifications.set({items:[],unread:0});
	const current = session();
	if (!current) return () => {};
	if (seenUser !== current.user.id) { seen.clear(); seenUser = current.user.id; }
 const sound = notificationSound();
 let stopped = false, hydrated = false;
	let socket: WebSocket | undefined;
	let retry = 1000;
	let timer: ReturnType<typeof setTimeout>;
	let refreshing = Promise.resolve();
 const refresh = (merge: boolean, live = false) => {
 const run = async () => {
		const [feed, count] = await Promise.all([
			api<{ notifications: Notification[] }>('/notifications'),
			api<{ count: number }>('/notifications/unread-count')
		]);
		if (stopped) return;
  const incoming = feed.notifications.some(item => !seen.has(item.id) && !item.readAt);
  feed.notifications.forEach(item => seen.add(item.id));
  notifications.update((state) => {
			const ids = new Set(feed.notifications.map(({ id }) => id));
			const items = merge ? [...feed.notifications, ...state.items.filter(({ id, type }) => type !== 'workspace_task' && type !== 'work_item_assigned' && !type.startsWith('leave_request_') && !ids.has(id))] : feed.notifications;
			return { items, unread: merge ? items.filter(({ readAt }) => !readAt).length : count.count };
		});
  if (live && hydrated && incoming) sound.play();
  hydrated = true;
 };
 refreshing = refreshing.catch(() => {}).then(run);
 return refreshing;
	};
	const connect = () => {
		if (stopped) return;
		const url = new URL(`${apiUrl}/ws/notifications`, window.location.origin);
		url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
		socket = new WebSocket(url);
		socket.onopen = () => { retry = 1000; window.dispatchEvent(new Event('realtime.resync')); void refresh(true).catch(() => {}); };
		socket.onmessage = ({ data }) => {
   if (stopped) return;
			try {
				const message = JSON.parse(data) as { event: string; notification: Notification; workspaceId?:number };
				if(message.event==='workspace.changed') { window.dispatchEvent(new CustomEvent('workspace.changed',{detail:message})); void refresh(false,true).catch(()=>{}); }
				if(message.event==='leave.changed') { window.dispatchEvent(new Event('leave.changed')); void refresh(false,true).catch(()=>{}); }
				if(['reports.changed','zoom.changed','jobs.changed','users.changed'].includes(message.event))window.dispatchEvent(new CustomEvent(message.event,{detail:message}));
    if (message.event === 'notification.created' && message.notification?.id && message.notification.actor?.id && message.notification.actor.id !== current.user.id) {
     const item = message.notification;
     if (!seen.has(item.id)) { seen.add(item.id); addNotification(item); if (!item.readAt) sound.play(); }
    }
			} catch { /* Ignore malformed server events. */ }
		};
		socket.onclose = () => { if (!stopped) { timer = setTimeout(connect, retry); retry = Math.min(retry * 2, 30_000); } };
	};
	void refresh(false).then(connect).catch(connect);
	// Bounded recovery for missed relay events and silently disconnected sockets.
	const resync=()=>{if(!stopped&&document.visibilityState==='visible')window.dispatchEvent(new Event('realtime.resync'))};
	const fallback=setInterval(resync,30_000);
	document.addEventListener('visibilitychange',resync);
	return () => { stopped = true; sound.stop(); clearTimeout(timer);clearInterval(fallback);document.removeEventListener('visibilitychange',resync); socket?.close(); notifications.set({items:[],unread:0}); };
}
