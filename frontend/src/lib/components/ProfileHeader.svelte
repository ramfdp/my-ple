<script lang="ts">
	import type { EmployeeProfile, UserRole } from '$lib/types/employee';
	import ProfileAvatar from './ProfileAvatar.svelte';
	import EmployeeStatusBadge from './EmployeeStatusBadge.svelte';

	let {
		profile,
		currentUserRole = 'Employee',
		onedit,
		onrolechange
	}: {
		profile: EmployeeProfile;
		currentUserRole?: UserRole;
		onedit?: () => void;
		onrolechange?: (role: UserRole) => void;
	} = $props();
</script>

<header class="profile-header-card">
	<!-- Gradient Banner -->
	<div class="banner-gradient" aria-hidden="true">
		<div class="banner-pattern"></div>
		<div class="role-selector-pill">
			<span class="role-label">Current Viewing Role:</span>
			<button
				type="button"
				class:active={currentUserRole === 'Employee'}
				onclick={() => onrolechange?.('Employee')}
				aria-label="Switch to Employee role view"
			>
				Employee
			</button>
			<button
				type="button"
				class:active={currentUserRole === 'HR / Admin'}
				onclick={() => onrolechange?.('HR / Admin')}
				aria-label="Switch to HR/Admin role view"
			>
				HR / Admin
			</button>
		</div>
	</div>

	<!-- Profile Body with Overlapping Avatar -->
	<div class="profile-header-content">
		<div class="profile-primary-row">
			<div class="avatar-overlap-wrapper">
				<ProfileAvatar
					src={profile.avatarUrl}
					alt={profile.name}
					size={104}
					borderWidth={4}
					editable={true}
					{onedit}
				/>
			</div>

			<div class="profile-identity">
				<div class="name-status-row">
					<h1 class="employee-name">{profile.name}</h1>
					<EmployeeStatusBadge
						status={profile.isActive ? 'Active Employee' : 'Inactive'}
						size="sm"
					/>
				</div>

				<p class="employee-meta">
					<span class="job-title">{profile.jobTitle}</span>
					<span class="meta-separator" aria-hidden="true">•</span>
					<span class="division-title">{profile.division}</span>
				</p>
			</div>
		</div>

		<div class="profile-header-actions">
			<button
				type="button"
				class="edit-profile-btn"
				onclick={() => onedit?.()}
			>
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
					<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
					<path d="m15 5 4 4" />
				</svg>
				<span>Edit Profile</span>
			</button>
		</div>
	</div>
</header>

<style>
	.profile-header-card {
		background: #ffffff;
		border-radius: 16px;
		border: 1px solid #e2e8f0;
		overflow: hidden;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.02);
	}

	/* ponytail: standard CSS linear-gradient matching reference #99F6E4 -> #D1FAE5 -> #ECFCCB */
	.banner-gradient {
		height: 145px;
		background: linear-gradient(90deg, #99f6e4 0%, #d1fae5 50%, #ecfccb 100%);
		position: relative;
		display: flex;
		justify-content: flex-end;
		align-items: flex-start;
		padding: 16px 20px;
	}

	.banner-pattern {
		position: absolute;
		inset: 0;
		background-image: radial-gradient(rgba(15, 118, 110, 0.08) 1px, transparent 1px);
		background-size: 16px 16px;
		pointer-events: none;
	}

	.role-selector-pill {
		position: relative;
		z-index: 2;
		display: inline-flex;
		align-items: center;
		background: rgba(255, 255, 255, 0.88);
		backdrop-filter: blur(8px);
		border: 1px solid rgba(255, 255, 255, 0.6);
		padding: 3px 4px 3px 10px;
		border-radius: 9999px;
		gap: 6px;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
	}

	.role-label {
		font-size: 0.6875rem;
		font-weight: 700;
		color: #0f766e;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	.role-selector-pill button {
		border: none;
		background: transparent;
		font-size: 0.75rem;
		font-weight: 600;
		color: #475569;
		padding: 4px 10px;
		border-radius: 9999px;
		transition: all 0.15s ease;
	}

	.role-selector-pill button.active {
		background: #0f766e;
		color: #ffffff;
		box-shadow: 0 1px 4px rgba(15, 118, 110, 0.25);
	}

	.profile-header-content {
		padding: 0 28px 24px;
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: 20px;
		position: relative;
	}

	.profile-primary-row {
		display: flex;
		align-items: flex-end;
		gap: 24px;
		min-width: 0;
	}

	.avatar-overlap-wrapper {
		margin-top: -52px;
		position: relative;
		z-index: 3;
	}

	.profile-identity {
		padding-bottom: 4px;
		display: flex;
		flex-direction: column;
		gap: 6px;
		min-width: 0;
	}

	.name-status-row {
		display: flex;
		align-items: center;
		gap: 12px;
		flex-wrap: wrap;
	}

	.employee-name {
		font-size: 1.625rem;
		font-weight: 800;
		color: #0f172a;
		letter-spacing: -0.03em;
		margin: 0;
		line-height: 1.2;
	}

	.employee-meta {
		margin: 0;
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 0.9375rem;
		color: #64748b;
		font-weight: 500;
		flex-wrap: wrap;
	}

	.job-title {
		color: #334155;
		font-weight: 600;
	}

	.meta-separator {
		color: #cbd5e1;
	}

	.division-title {
		color: #0f766e;
		background: #f0fdfa;
		padding: 2px 8px;
		border-radius: 6px;
		font-size: 0.8125rem;
		font-weight: 600;
	}

	.profile-header-actions {
		padding-bottom: 4px;
		flex-shrink: 0;
	}

	.edit-profile-btn {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		background: #ffffff;
		color: #1e293b;
		border: 1px solid #cbd5e1;
		padding: 10px 18px;
		border-radius: 10px;
		font-size: 0.875rem;
		font-weight: 600;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03);
		transition: all 0.15s ease;
	}

	.edit-profile-btn:hover {
		background: #f8fafc;
		border-color: #94a3b8;
		color: #0f172a;
		transform: translateY(-1px);
		box-shadow: 0 3px 8px rgba(0, 0, 0, 0.05);
	}

	.edit-profile-btn svg {
		width: 16px;
		height: 16px;
		color: #0f766e;
	}

	@media (max-width: 768px) {
		.profile-header-content {
			flex-direction: column;
			align-items: flex-start;
			padding: 0 18px 20px;
			gap: 16px;
		}

		.profile-primary-row {
			flex-direction: column;
			align-items: flex-start;
			gap: 12px;
			width: 100%;
		}

		.avatar-overlap-wrapper {
			margin-top: -46px;
		}

		.employee-name {
			font-size: 1.375rem;
		}

		.profile-header-actions {
			width: 100%;
		}

		.edit-profile-btn {
			width: 100%;
			justify-content: center;
		}

		.role-selector-pill {
			padding: 2px 4px 2px 8px;
		}

		.role-label {
			display: none;
		}
	}
</style>
