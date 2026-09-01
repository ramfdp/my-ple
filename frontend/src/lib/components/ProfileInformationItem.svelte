<script lang="ts">
	// ponytail: lightweight inline SVG icon registry to avoid 50kb+ lucide package load
	let {
		icon = 'user',
		label,
		value,
		isLocked = false,
		lockedMessage = 'HR/Admin only',
		children
	}: {
		icon?: 'user' | 'building' | 'wallet' | 'map-pin' | 'phone' | 'mail' | 'briefcase' | 'shield';
		label: string;
		value?: string | number;
		isLocked?: boolean;
		lockedMessage?: string;
		children?: import('svelte').Snippet;
	} = $props();
</script>

<div class="info-item">
	<div class="info-icon-wrapper" aria-hidden="true">
		{#if icon === 'user'}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
				<circle cx="12" cy="7" r="4" />
			</svg>
		{:else if icon === 'building'}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z" />
				<path d="M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2" />
				<path d="M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2" />
				<path d="M10 6h4" />
				<path d="M10 10h4" />
				<path d="M10 14h4" />
				<path d="M10 18h4" />
			</svg>
		{:else if icon === 'wallet'}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<rect width="20" height="14" x="2" y="5" rx="2" />
				<line x1="2" x2="22" y1="10" y2="10" />
				<circle cx="16" cy="14" r="1" />
			</svg>
		{:else if icon === 'map-pin'}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M20 10c0 4.993-5.539 10.193-7.399 11.799a1 1 0 0 1-1.202 0C9.539 20.193 4 10.193 4 10a8 8 0 0 1 16 0" />
				<circle cx="12" cy="10" r="3" />
			</svg>
		{:else if icon === 'phone'}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
			</svg>
		{:else if icon === 'mail'}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<rect width="20" height="16" x="2" y="4" rx="2" />
				<path d="m22 7-8.97 5.7a1.94 1.94 0 0 1-2.06 0L2 7" />
			</svg>
		{:else if icon === 'briefcase'}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M16 20V4a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16" />
				<rect width="20" height="14" x="2" y="6" rx="2" />
			</svg>
		{:else}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10" />
			</svg>
		{/if}
	</div>

	<div class="info-content">
		<div class="info-label-row">
			<span class="info-label">{label}</span>
			{#if isLocked}
				<span class="locked-indicator" title={lockedMessage}>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
						<path d="M7 11V7a5 5 0 0 1 10 0v4" />
					</svg>
					HR Only
				</span>
			{/if}
		</div>

		<div class="info-value">
			{#if value !== undefined && value !== null && value !== ''}
				{value}
			{:else}
				{@render children?.()}
			{/if}
		</div>
	</div>
</div>

<style>
	.info-item {
		display: flex;
		align-items: flex-start;
		gap: 14px;
		padding: 12px 14px;
		border-radius: 12px;
		background: #ffffff;
		border: 1px solid #f1f5f9;
		transition: all 0.18s ease;
	}

	.info-item:hover {
		border-color: #e2e8f0;
		background: #fbfdff;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.02);
	}

	.info-icon-wrapper {
		width: 38px;
		height: 38px;
		border-radius: 10px;
		background: #f0fdfa;
		border: 1px solid #ccfbf1;
		color: #0f766e;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.info-icon-wrapper svg {
		width: 18px;
		height: 18px;
	}

	.info-content {
		display: flex;
		flex-direction: column;
		gap: 4px;
		min-width: 0;
		flex: 1;
	}

	.info-label-row {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.info-label {
		font-size: 0.8125rem;
		font-weight: 500;
		color: #64748b;
		letter-spacing: -0.01em;
	}

	.locked-indicator {
		display: inline-flex;
		align-items: center;
		gap: 3px;
		font-size: 0.6875rem;
		font-weight: 600;
		color: #94a3b8;
		background: #f1f5f9;
		padding: 2px 6px;
		border-radius: 4px;
		line-height: 1;
	}

	.locked-indicator svg {
		width: 10px;
		height: 10px;
	}

	.info-value {
		font-size: 0.9375rem;
		font-weight: 600;
		color: #0f172a;
		letter-spacing: -0.015em;
		word-break: break-word;
	}
</style>
