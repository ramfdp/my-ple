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

export function startNotifications() {
	const current = session();
	if (!current) return () => {};
	let stopped = false;
	let socket: WebSocket | undefined;
	let retry = 1000;
	let timer: ReturnType<typeof setTimeout>;
	const refresh = async (merge: boolean) => {
		const [feed, count] = await Promise.all([
			api<{ notifications: Notification[] }>('/notifications'),
			api<{ count: number }>('/notifications/unread-count')
		]);
		if (stopped) return;
		notifications.update((state) => {
			const ids = new Set(feed.notifications.map(({ id }) => id));
			const items = merge ? [...feed.notifications, ...state.items.filter(({ id }) => !ids.has(id))] : feed.notifications;
			return { items, unread: merge ? items.filter(({ readAt }) => !readAt).length : count.count };
		});
	};
	const connect = () => {
		if (stopped) return;
		const url = new URL('/ws/notifications', apiUrl);
		url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
		socket = new WebSocket(url);
		socket.onopen = () => { retry = 1000; void refresh(true).catch(() => {}); };
		socket.onmessage = ({ data }) => {
			try {
				const message = JSON.parse(data) as { event: string; notification: Notification };
				if (message.event === 'notification.created' && message.notification.actor.id !== current.user.id) addNotification(message.notification);
			} catch { /* Ignore malformed server events. */ }
		};
		socket.onclose = () => { if (!stopped) { timer = setTimeout(connect, retry); retry = Math.min(retry * 2, 30_000); } };
	};
	void refresh(false).then(connect).catch(connect);
	return () => { stopped = true; clearTimeout(timer); socket?.close(); };
}
