<script lang="ts">
	import type { EmploymentStatus } from '$lib/types/employee';

	// ponytail: lookup dictionary replaces multi-level conditional branches
	let {
		status = 'Permanent Employee',
		size = 'md'
	}: {
		status: EmploymentStatus | 'Active Employee' | string;
		size?: 'sm' | 'md' | 'lg';
	} = $props();

	const statusStyles: Record<string, { bg: string; text: string; border: string; dot: string }> = {
		'Permanent Employee': {
			bg: 'var(--his-primary-soft)',
			text: 'var(--his-primary)',
			border: 'var(--his-primary-soft)',
			dot: 'var(--his-primary)'
		},
		'Contract Employee': {
			bg: 'var(--his-primary-soft)',
			text: 'var(--his-primary)',
			border: 'var(--his-primary-soft)',
			dot: 'var(--his-primary)'
		},
		'Probation': {
			bg: '#fffbeb',
			text: '#b45309',
			border: '#fde68a',
			dot: '#f59e0b'
		},
		'Intern': {
			bg: 'var(--his-danger-soft)',
			text: 'var(--his-danger)',
			border: 'var(--his-danger-border)',
			dot: 'var(--his-danger)'
		},
		'Freelancer': {
			bg: 'var(--his-primary-soft)',
			text: 'var(--his-primary)',
			border: 'var(--his-primary-soft)',
			dot: 'var(--his-primary)'
		},
		'Inactive': {
			bg: 'var(--his-danger-soft)',
			text: 'var(--his-danger)',
			border: 'var(--his-danger-border)',
			dot: 'var(--his-danger)'
		},
		'Active Employee': {
			bg: 'var(--his-primary-soft)',
			text: 'var(--his-primary)',
			border: 'var(--his-primary-soft)',
			dot: 'var(--his-primary)'
		}
	};

	let currentStyle = $derived(
		statusStyles[status] ?? {
			bg: 'var(--his-surface)',
			text: 'var(--his-muted)',
			border: 'var(--his-border)',
			dot: '#737373'
		}
	);
</script>

<span
	class="status-badge {size}"
	style:background-color={currentStyle.bg}
	style:color={currentStyle.text}
	style:border-color={currentStyle.border}
>
	<span class="status-dot" style:background-color={currentStyle.dot} aria-hidden="true"></span>
	{status}
</span>

<style>
	.status-badge {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-weight: 600;
		border-radius: 9999px;
		border: 1px solid;
		letter-spacing: -0.01em;
		width: fit-content;
		white-space: nowrap;
		line-height: 1;
	}

	.status-badge.sm {
		padding: 4px 10px;
		font-size: 0.75rem;
	}

	.status-badge.md {
		padding: 6px 12px;
		font-size: 0.8125rem;
	}

	.status-badge.lg {
		padding: 8px 16px;
		font-size: 0.875rem;
	}

	.status-dot {
		width: 6px;
		height: 6px;
		border-radius: 9999px;
		flex-shrink: 0;
	}
</style>
