<script lang="ts">
	import { notifications, dismissNotification, markAllRead, markRead } from '$lib/notifications';
	let open = $state(false);
	let panel: HTMLElement;
	let left = $state(12), top = $state(68);
	function toggle(event: MouseEvent) {
		const anchor = (event.currentTarget as HTMLElement).getBoundingClientRect();
		left = Math.max(12, Math.min(anchor.right - 380, window.innerWidth - 392));
		top = Math.max(12, Math.min(anchor.bottom + 8, window.innerHeight - 160));
		panel.togglePopover();
	}
	const date = (value: string) => new Date(value).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' });
	const time = (value: string) => new Date(value).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
	const relative = (value: string) => {
		const minutes = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 60000));
		return minutes < 1 ? 'Baru saja' : minutes < 60 ? `${minutes} menit lalu` : `${Math.floor(minutes / 60)} jam lalu`;
	};
</script>

<svelte:window onresize={() => panel?.hidePopover()} />

<div class="notification-wrap">
	<button class="notification-button" type="button" aria-label="Notifikasi" aria-expanded={open} onclick={toggle}>
		<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" /></svg>
		{#if $notifications.unread}<span class="notification-badge">{$notifications.unread > 99 ? '99+' : $notifications.unread}</span>{/if}
	</button>
		<section bind:this={panel} popover="auto" ontoggle={(event) => open = event.newState === 'open'} class="notification-panel" style:left={left+'px'} style:top={top+'px'} style:max-height={'min(480px, calc(100dvh - '+(top+12)+'px))'} aria-label="Daftar notifikasi">
			<header><strong>Notifikasi</strong>{#if $notifications.unread}<button type="button" onclick={() => void markAllRead().catch(() => {})}>Tandai semua dibaca</button>{/if}</header>
			{#if !$notifications.items.length}<p class="notification-empty">Belum ada notifikasi.</p>{/if}
			<ul>
				{#each $notifications.items as item (item.id)}
					<li class:unread={!item.readAt}>
						<button class="notification-content" type="button" onclick={() => void markRead(item.id).catch(() => {})}>
							<strong>{#if item.workspace}{item.actor.name}: {item.workspace.message} — {item.workspace.title}{:else}{item.actor.name} membooking {item.booking?.zoomName ?? 'Zoom'}{/if}</strong>
							{#if item.booking}<span>{date(item.booking.startAt)} · {time(item.booking.startAt)}–{time(item.booking.endAt)}</span>{/if}
							<small>{relative(item.createdAt)}</small>
						</button>
						{#if item.workspace}<a href={'/workspace/'+item.workspace.workspaceId}>Buka Workspace →</a>{/if}
						<button class="notification-dismiss" type="button" onclick={() => void dismissNotification(item.id).catch(() => {})}>Hapus</button>
					</li>
				{/each}
			</ul>
		</section>
</div>
