<script lang="ts">
	import type { EmployeeProfile, UserRole } from '$lib/types/employee';
	import { formatRupiah } from '$lib/utils/formatters';
	import ProfileInformationItem from './ProfileInformationItem.svelte';
	import EmployeeStatusBadge from './EmployeeStatusBadge.svelte';

	let {
		profile,
		currentUserRole = 'Employee',
		onedit
	}: {
		profile: EmployeeProfile;
		currentUserRole?: UserRole;
		onedit?: () => void;
	} = $props();

	let isHR = $derived(currentUserRole === 'HR / Admin');
</script>

<section class="info-card" aria-labelledby="card-heading">
	<div class="card-header">
		<div class="header-left">
			<h2 id="card-heading" class="card-title">Employee Information</h2>
			<span class="card-subtitle">Data pribadi dan riwayat kepegawaian internal perusahaan</span>
		</div>

		<button
			type="button"
			class="edit-action-button"
			onclick={() => onedit?.()}
			aria-label="Edit employee information"
		>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
				<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
				<path d="m15 5 4 4" />
			</svg>
			<span>Edit</span>
		</button>
	</div>

	<!-- ponytail: clean 3-col/2-col/1-col CSS grid without complex masonry calculations -->
	<div class="info-grid">
		<!-- Row 1 -->
		<ProfileInformationItem
			icon="user"
			label="Name"
			value={profile.name}
		/>

		<ProfileInformationItem
			icon="building"
			label="Division"
			value={profile.division}
			isLocked={!isHR}
			lockedMessage="Hanya HR / Admin yang dapat mengubah divisi"
		/>

		<ProfileInformationItem
			icon="wallet"
			label="Current Salary"
			value="{formatRupiah(profile.currentSalary)} / month"
			isLocked={!isHR}
			lockedMessage="Hanya HR / Admin yang dapat mengubah gaji"
		/>

		<!-- Row 2 -->
		<ProfileInformationItem
			icon="map-pin"
			label="Residence"
			value={profile.residence}
		/>

		<ProfileInformationItem
			icon="phone"
			label="Phone Number"
			value={profile.phoneNumber}
		/>

		<ProfileInformationItem
			icon="mail"
			label="Email"
			value={profile.email}
		/>

		<!-- Row 3: Status -->
		<div class="status-item-wrapper">
			<ProfileInformationItem
				icon="briefcase"
				label="Employment Status"
				isLocked={!isHR}
				lockedMessage="Hanya HR / Admin yang dapat mengubah status kepegawaian"
			>
				<div class="status-badge-container">
					<EmployeeStatusBadge status={profile.employmentStatus} size="sm" />
				</div>
			</ProfileInformationItem>
		</div>
	</div>
</section>

<style>
	.info-card {
		background: #ffffff;
		border-radius: 16px;
		border: 1px solid #e2e8f0;
		padding: 24px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04), 0 1px 2px rgba(0, 0, 0, 0.02);
		margin-top: 24px;
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding-bottom: 20px;
		margin-bottom: 20px;
		border-bottom: 1px solid #f1f5f9;
		gap: 16px;
	}

	.header-left {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.card-title {
		font-size: 1.1875rem;
		font-weight: 700;
		color: #0f172a;
		letter-spacing: -0.02em;
		margin: 0;
	}

	.card-subtitle {
		font-size: 0.8125rem;
		color: #64748b;
	}

	.edit-action-button {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 8px 14px;
		border-radius: 10px;
		background: #ffffff;
		border: 1px solid #e2e8f0;
		color: #334155;
		font-size: 0.84375rem;
		font-weight: 600;
		transition: all 0.15s ease;
	}

	.edit-action-button:hover {
		background: #f8fafc;
		border-color: #cbd5e1;
		color: #0f172a;
		transform: translateY(-1px);
	}

	.edit-action-button svg {
		width: 14px;
		height: 14px;
		color: #64748b;
	}

	.info-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 16px;
	}

	.status-item-wrapper {
		grid-column: span 1;
	}

	.status-badge-container {
		margin-top: 2px;
	}

	@media (max-width: 1024px) {
		.info-grid {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 640px) {
		.info-card {
			padding: 16px;
			border-radius: 14px;
		}

		.card-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 12px;
		}

		.edit-action-button {
			width: 100%;
			justify-content: center;
		}

		.info-grid {
			grid-template-columns: 1fr;
			gap: 12px;
		}
	}
</style>
