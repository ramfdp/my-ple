<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
	import {labelDate} from '$lib/workspace';
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
	const date = (value: string) => new Date(value).toLocaleDateString($locale, { day: 'numeric', month: 'short', year: 'numeric' });
	const time = (value: string) => new Date(value).toLocaleTimeString($locale, { hour: '2-digit', minute: '2-digit' });
	const relative = (value: string) => {
		const minutes = Math.max(0, Math.floor((Date.now() - new Date(value).getTime()) / 60000));
		return minutes < 1 ? $translate('Baru saja') : new Intl.RelativeTimeFormat($locale, {numeric:'auto'}).format(minutes < 60 ? -minutes : -Math.floor(minutes/60), minutes < 60 ? 'minute' : 'hour');
	};
</script>

<svelte:window onresize={() => panel?.hidePopover()} onscroll={() => panel?.hidePopover()} />

<div class="notification-wrap">
	<button class="notification-button" type="button" aria-label={$translate("Notifikasi")} aria-expanded={open} onclick={toggle}>
		<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 9a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" /></svg>
		{#if $notifications.unread}<span class="notification-badge">{$notifications.unread > 99 ? '99+' : $notifications.unread}</span>{/if}
	</button>
		<section bind:this={panel} popover="auto" ontoggle={(event) => open = event.newState === 'open'} class="notification-panel" style:left={left+'px'} style:top={top+'px'} style:max-height={'min(480px, calc(100dvh - '+(top+12)+'px))'} aria-label={$translate("Daftar notifikasi")}>
			<header><strong>{$translate("Notifikasi")}</strong>{#if $notifications.unread}<button type="button" onclick={() => void markAllRead().catch(() => {})}>{$translate("Tandai semua dibaca")}</button>{/if}</header>
			{#if !$notifications.items.length}<p class="notification-empty">{$translate("Belum ada notifikasi.")}</p>{/if}
			<ul>
				{#each $notifications.items as item (item.id)}
					<li class:unread={!item.readAt}>
						<div class="notification-content">
							<strong>{#if item.leave}{$translate(item.type==='leave_request_approved'?'{actor} menyetujui pengajuan':item.type==='leave_request_rejected'?'{actor} menolak pengajuan':'{actor} mengajukan izin/cuti',{actor:item.actor.name})}{:else if item.overtime}{item.overtime.title}{:else if item.workspace}{$translate(item.type==='work_item_assigned'?'{actor} menetapkan pekerjaan':item.workspace.message==='Komentar baru'?'{actor} menambahkan komentar':'{actor} memperbarui pekerjaan',{actor:item.actor.name})} — {item.workspace.title}{:else}{item.actor.name} {$translate("membooking")} {item.booking?.zoomName ?? $translate('Zoom')}{/if}</strong>
							{#if item.leave}<span>{labelDate(item.leave.startDate, $locale)} &ndash; {labelDate(item.leave.endDate, $locale)}</span><span class="booking-description">{item.leave.reason}</span>{#if item.leave.reviewerNote}<span class="booking-description">{$translate("Catatan:")} {item.leave.reviewerNote}</span>{/if}{/if}
							{#if item.overtime}<span>{item.overtime.employeeName} · {item.overtime.period}</span><span class="booking-description">{item.overtime.message}</span>{/if}
							{#if item.booking}<span>{date(item.booking.startAt)} · {time(item.booking.startAt)}–{time(item.booking.endAt)}</span>{/if}
							{#if item.type==='work_item_assigned'&&item.workspace}<span>{$translate("Project:")} {item.workspace.projectName??$translate('Tanpa project')}</span><span>{item.workspace.startDate||$translate('Tanpa tanggal')} – {item.workspace.dueDate||$translate('Tanpa deadline')} · {$translate(item.workspace.priority ?? '')}</span>{/if}
							{#if item.booking?.description}<span class="booking-description">{$translate("Deskripsi:")} {item.booking.description}</span>{/if}
							<small>{relative(item.createdAt)}</small>
						</div>
						{#if item.leave}<a href={(item.type==='leave_request_created'?'/leave/approvals':'/leave')+'?request='+encodeURIComponent(item.entityId)} onclick={() => void markRead(item.id).catch(() => {})}>{$translate("Lihat Pengajuan →")}</a>{/if}
						{#if item.overtime}<a href={item.overtime.path} onclick={() => void markRead(item.id).catch(() => {})}>Lihat Lembur →</a>{/if}
						{#if item.workspace}<a href={'/workspace/'+item.workspace.workspaceId} onclick={() => void markRead(item.id).catch(() => {})}>{$translate("Buka Workspace →")}</a>{/if}
						{#if item.booking}<a href="/booking-zoom" onclick={() => void markRead(item.id).catch(() => {})}>{$translate("Buka Booking Zoom →")}</a>{/if}
						<button class="notification-dismiss" type="button" onclick={() => void dismissNotification(item.id).catch(() => {})}>{$translate("Hapus")}</button>
					</li>
				{/each}
			</ul>
		</section>
</div>

<style>
	.booking-description { display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 3; line-clamp: 3; overflow: hidden; overflow-wrap: break-word; white-space: pre-wrap; }
</style>
