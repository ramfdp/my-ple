<script lang="ts">
	// ponytail: lightweight inline SVG icon registry to avoid 50kb+ lucide package load
	let {
		icon = 'user',
		label,
		value,
		children
	}: {
		icon?: 'user' | 'wallet' | 'map-pin' | 'phone' | 'mail' | 'briefcase' | 'shield';
		label: string;
		value?: string | number;
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
		border: 1px solid var(--his-surface);
		transition: all 0.18s ease;
	}

	.info-item:hover {
		border-color: var(--his-border);
		background: var(--his-surface);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.02);
	}

	.info-icon-wrapper {
		width: 38px;
		height: 38px;
		border-radius: 10px;
		background: var(--his-primary-soft);
		border: 1px solid var(--his-primary-soft);
		color: var(--his-primary);
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
		color: var(--his-muted);
		letter-spacing: -0.01em;
	}



	.info-value {
		font-size: 0.9375rem;
		font-weight: 600;
		color: var(--his-text);
		letter-spacing: -0.015em;
		word-break: break-word;
	}
</style>
