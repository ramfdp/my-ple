<script lang="ts">
	// ponytail: pure native SVG fallback with zero external avatar library dependencies
	let {
		src = '',
		alt = 'Employee Avatar',
		size = 100,
		borderWidth = 4,
		editable = false,
		onedit
	}: {
		src?: string;
		alt?: string;
		size?: number;
		borderWidth?: number;
		editable?: boolean;
		onedit?: () => void;
	} = $props();

	let imageFailed = $state(false);
	$effect(() => { src; imageFailed = false; });
</script>

<div
	class="avatar-container"
	style:width="{size}px"
	style:height="{size}px"
	style:border-width="{borderWidth}px"
>
	{#if src && !imageFailed}
		<img
			{src}
			{alt}
			class="avatar-image"
			onerror={() => (imageFailed = true)}
		/>
	{:else}
		<div class="avatar-fallback" aria-label={alt}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
				<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
				<circle cx="12" cy="7" r="4" />
			</svg>
		</div>
	{/if}

	{#if editable}
		<button
			type="button"
			class="avatar-edit-badge"
			aria-label="Ubah foto profil"
			onclick={() => onedit?.()}
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<path d="M23 19a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h4l2-3h6l2 3h4a2 2 0 0 1 2 2z" />
				<circle cx="12" cy="13" r="4" />
			</svg>
		</button>
	{/if}
</div>

<style>
	.avatar-container {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border-radius: 9999px;
		border-style: solid;
		border-color: #ffffff;
		background: var(--his-surface);
		box-shadow: 0 4px 14px -2px rgba(0, 0, 0, 0.08), 0 2px 6px -1px rgba(0, 0, 0, 0.04);
		flex-shrink: 0;
	}

	.avatar-image {
		width: 100%;
		height: 100%;
		border-radius: 9999px;
		object-fit: cover;
		display: block;
	}

	.avatar-fallback {
		width: 100%;
		height: 100%;
		border-radius: 9999px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: linear-gradient(135deg, var(--his-primary-soft), var(--his-primary-soft));
		color: var(--his-primary);
	}

	.avatar-fallback svg {
		width: 50%;
		height: 50%;
	}

	.avatar-edit-badge {
		position: absolute;
		bottom: 0;
		right: 0;
		width: 32px;
		height: 32px;
		border-radius: 9999px;
		background: var(--his-primary);
		color: #ffffff;
		border: 2px solid #ffffff;
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 2px 5px rgba(0, 0, 0, 0.15);
		transition: transform 0.15s ease, background-color 0.15s ease;
	}

	.avatar-edit-badge:hover {
		background: var(--his-primary);
		transform: scale(1.08);
	}

	.avatar-edit-badge svg {
		width: 15px;
		height: 15px;
	}
</style>
