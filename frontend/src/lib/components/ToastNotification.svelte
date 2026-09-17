<script lang="ts">
	// ponytail: lightweight self-dismissing toast with zero animation libraries
	let {
		message = '',
		type = 'success',
		visible = false,
		onclose
	}: {
		message: string;
		type?: 'success' | 'error';
		visible: boolean;
		onclose?: () => void;
	} = $props();

	$effect(() => {
		if (visible) {
			const timer = setTimeout(() => {
				onclose?.();
			}, 3500);
			return () => clearTimeout(timer);
		}
	});
</script>

{#if visible}
	<div class="toast-wrapper {type}" role="status" aria-live="polite">
		<div class="toast-icon">
			{#if type === 'success'}
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="20 6 9 17 4 12" />
				</svg>
			{:else}
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<circle cx="12" cy="12" r="10" />
					<line x1="15" y1="9" x2="9" y2="15" />
					<line x1="9" y1="9" x2="15" y2="15" />
				</svg>
			{/if}
		</div>

		<p class="toast-message">{message}</p>

		<button type="button" class="toast-close" onclick={() => onclose?.()} aria-label="Tutup notifikasi">
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	</div>
{/if}

<style>
	.toast-wrapper {
		position: fixed;
		bottom: 24px;
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		background: #ffffff;
		border-radius: 12px;
		box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
		border: 1px solid var(--his-border);
		z-index: 60;
		animation: toastSlideUp 0.25s cubic-bezier(0.16, 1, 0.3, 1);
		width: max-content;
		max-width: min(480px, calc(100vw - 32px));
		right: 16px;
		max-height: calc(100dvh - 48px);
		overflow-y: auto;
	}

	@keyframes toastSlideUp {
		from {
			opacity: 0;
			transform: translateY(16px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.toast-wrapper.success {
		border-left: 4px solid var(--his-primary);
	}

	.toast-wrapper.error {
		border-left: 4px solid var(--his-danger);
	}

	.toast-icon {
		width: 24px;
		height: 24px;
		border-radius: 9999px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.toast-wrapper.success .toast-icon {
		background: var(--his-primary-soft);
		color: var(--his-primary);
	}

	.toast-wrapper.error .toast-icon {
		background: var(--his-danger-soft);
		color: var(--his-danger);
	}

	.toast-icon svg {
		width: 14px;
		height: 14px;
	}

	.toast-message {
		min-width: 0;
		overflow-wrap: anywhere;
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--his-text);
		margin: 0;
	}

	.toast-close {
		flex: 0 0 44px;
		min-height: 44px;
		border: none;
		background: transparent;
		color: #737373;
		padding: 2px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 4px;
		transition: color 0.15s ease;
	}

	.toast-close:hover {
		color: var(--his-muted);
	}

	.toast-close svg {
		width: 14px;
		height: 14px;
	}
</style>
