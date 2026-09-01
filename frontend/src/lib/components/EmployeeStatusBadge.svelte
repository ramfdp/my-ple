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
			bg: '#ecfdf5',
			text: '#047857',
			border: '#a7f3d0',
			dot: '#10b981'
		},
		'Contract Employee': {
			bg: '#eff6ff',
			text: '#1d4ed8',
			border: '#bfdbfe',
			dot: '#3b82f6'
		},
		'Probation': {
			bg: '#fffbeb',
			text: '#b45309',
			border: '#fde68a',
			dot: '#f59e0b'
		},
		'Intern': {
			bg: '#faf5ff',
			text: '#6b21a8',
			border: '#e9d5ff',
			dot: '#a855f7'
		},
		'Freelancer': {
			bg: '#ecfeff',
			text: '#0e7490',
			border: '#a5f3fc',
			dot: '#06b6d4'
		},
		'Inactive': {
			bg: '#fef2f2',
			text: '#b91c1c',
			border: '#fecaca',
			dot: '#ef4444'
		},
		'Active Employee': {
			bg: '#f0fdf4',
			text: '#15803d',
			border: '#bbf7d0',
			dot: '#22c55e'
		}
	};

	let currentStyle = $derived(
		statusStyles[status] ?? {
			bg: '#f8fafc',
			text: '#475569',
			border: '#e2e8f0',
			dot: '#94a3b8'
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
