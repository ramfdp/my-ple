<script lang="ts">
	import type { EmployeeProfile, UserRole } from '$lib/types/employee';
	import ProfileHeader from '$lib/components/ProfileHeader.svelte';
	import ProfileInformationCard from '$lib/components/ProfileInformationCard.svelte';
	import EditProfileModal from '$lib/components/EditProfileModal.svelte';
	import ToastNotification from '$lib/components/ToastNotification.svelte';

	// Initial realistic employee data
	let employee = $state<EmployeeProfile>({
		id: 'EMP-2024-0089',
		name: 'Andi Pratama',
		jobTitle: 'Software Engineering',
		division: 'Information Technology',
		currentSalary: 12500000,
		residence: 'South Jakarta',
		phoneNumber: '+62 812-3456-7890',
		email: 'andi.pratama@company.com',
		employmentStatus: 'Permanent Employee',
		avatarUrl: 'https://images.unsplash.com/photo-1534528741775-53994a69daeb?auto=format&fit=crop&q=80&w=400',
		isActive: true
	});

	let currentUserRole = $state<UserRole>('Employee');
	let isEditModalOpen = $state(false);
	let isLoading = $state(false);

	let toast = $state<{ visible: boolean; message: string; type: 'success' | 'error' | 'info' }>({
		visible: false,
		message: '',
		type: 'success'
	});

	function showToast(message: string, type: 'success' | 'error' | 'info' = 'success') {
		toast = { visible: true, message, type };
	}

	async function handleSaveProfile(updated: EmployeeProfile) {
		employee = { ...updated };
		isEditModalOpen = false;
		showToast('Employee profile updated successfully.', 'success');
	}

	function simulateLoading() {
		isLoading = true;
		setTimeout(() => {
			isLoading = false;
		}, 800);
	}
</script>

<svelte:head>
	<title>{employee.name} | Employee Profile - my-ple</title>
	<meta name="description" content="View and manage employee profile and employment details." />
</svelte:head>

<main class="profile-page-root">
	<div class="profile-container">
		<!-- Top Breadcrumb / Page Header -->
		<div class="page-top-bar">
			<div class="breadcrumb-trail">
				<a href="/dashboard" class="crumb-link">Dashboard</a>
				<span class="crumb-separator" aria-hidden="true">/</span>
				<span class="crumb-current">My Profile</span>
			</div>

			<div class="page-quick-actions">
				<button
					type="button"
					class="reload-skeleton-btn"
					onclick={simulateLoading}
					title="Simulasikan refresh / skeleton loading state"
					aria-label="Refresh profile"
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
						<path d="M21 12a9 9 0 0 0-9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
						<path d="M3 3v5h5" />
						<path d="M3 12a9 9 0 0 0 9 9 9.75 9.75 0 0 0 6.74-2.74L21 16" />
						<path d="M16 21h5v-5" />
					</svg>
					<span>Refresh</span>
				</button>
			</div>
		</div>

		{#if isLoading}
			<!-- Skeleton Loading State -->
			<div class="skeleton-wrapper" aria-busy="true" aria-label="Memuat data profil...">
				<div class="skeleton-header">
					<div class="skeleton-banner shimmer"></div>
					<div class="skeleton-header-body">
						<div class="skeleton-avatar shimmer"></div>
						<div class="skeleton-lines">
							<div class="skeleton-line lg shimmer"></div>
							<div class="skeleton-line md shimmer"></div>
						</div>
					</div>
				</div>

				<div class="skeleton-card shimmer">
					<div class="skeleton-line sm shimmer"></div>
					<div class="skeleton-grid">
						<div class="skeleton-box shimmer"></div>
						<div class="skeleton-box shimmer"></div>
						<div class="skeleton-box shimmer"></div>
						<div class="skeleton-box shimmer"></div>
						<div class="skeleton-box shimmer"></div>
						<div class="skeleton-box shimmer"></div>
					</div>
				</div>
			</div>
		{:else}
			<!-- Live Profile Header -->
			<ProfileHeader
				profile={employee}
				{currentUserRole}
				onedit={() => (isEditModalOpen = true)}
				onrolechange={(newRole) => {
					currentUserRole = newRole;
					showToast(`Role switched to ${newRole}`, 'info');
				}}
			/>

			<!-- Live Employee Information Card -->
			<ProfileInformationCard
				profile={employee}
				{currentUserRole}
				onedit={() => (isEditModalOpen = true)}
			/>
		{/if}
	</div>

	<!-- Edit Modal -->
	<EditProfileModal
		isOpen={isEditModalOpen}
		profile={employee}
		{currentUserRole}
		onclose={() => (isEditModalOpen = false)}
		onsave={handleSaveProfile}
	/>

	<!-- Toast Notification -->
	<ToastNotification
		message={toast.message}
		type={toast.type}
		visible={toast.visible}
		onclose={() => (toast.visible = false)}
	/>
</main>

<style>
	/* ponytail: clear layout tokens matching specifications */
	.profile-page-root {
		background: #f8fafc;
		min-height: 100vh;
		padding: 28px 36px 48px;
	}

	.profile-container {
		max-width: 1120px;
		margin: 0 auto;
	}

	.page-top-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: 20px;
		gap: 16px;
	}

	.breadcrumb-trail {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 0.84375rem;
		font-weight: 500;
	}

	.crumb-link {
		color: #64748b;
		text-decoration: none;
		transition: color 0.15s ease;
	}

	.crumb-link:hover {
		color: #0f766e;
	}

	.crumb-separator {
		color: #cbd5e1;
	}

	.crumb-current {
		color: #0f172a;
		font-weight: 600;
	}

	.reload-skeleton-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		font-size: 0.8125rem;
		font-weight: 600;
		color: #64748b;
		background: #ffffff;
		border: 1px solid #e2e8f0;
		padding: 6px 12px;
		border-radius: 8px;
		transition: all 0.15s ease;
	}

	.reload-skeleton-btn:hover {
		color: #0f172a;
		border-color: #cbd5e1;
		background: #f8fafc;
	}

	.reload-skeleton-btn svg {
		width: 14px;
		height: 14px;
	}

	/* Skeleton styles */
	.skeleton-wrapper {
		display: flex;
		flex-direction: column;
		gap: 24px;
	}

	.skeleton-header {
		background: #ffffff;
		border-radius: 16px;
		border: 1px solid #e2e8f0;
		overflow: hidden;
	}

	.skeleton-banner {
		height: 145px;
		background: #e2e8f0;
	}

	.skeleton-header-body {
		padding: 0 28px 24px;
		display: flex;
		align-items: flex-end;
		gap: 24px;
	}

	.skeleton-avatar {
		width: 104px;
		height: 104px;
		border-radius: 9999px;
		background: #e2e8f0;
		border: 4px solid #ffffff;
		margin-top: -52px;
		flex-shrink: 0;
	}

	.skeleton-lines {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 8px;
		padding-bottom: 8px;
	}

	.skeleton-line {
		height: 14px;
		border-radius: 6px;
		background: #e2e8f0;
	}

	.skeleton-line.lg {
		width: 220px;
		height: 24px;
	}

	.skeleton-line.md {
		width: 160px;
	}

	.skeleton-line.sm {
		width: 120px;
	}

	.skeleton-card {
		background: #ffffff;
		border-radius: 16px;
		border: 1px solid #e2e8f0;
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 20px;
	}

	.skeleton-grid {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 16px;
	}

	.skeleton-box {
		height: 64px;
		border-radius: 12px;
		background: #f1f5f9;
	}

	/* Shimmer Animation */
	.shimmer {
		background: linear-gradient(90deg, #f1f5f9 25%, #e2e8f0 50%, #f1f5f9 75%);
		background-size: 200% 100%;
		animation: shimmer 1.5s infinite;
	}

	@keyframes shimmer {
		0% {
			background-position: -200% 0;
		}
		100% {
			background-position: 200% 0;
		}
	}

	@media (max-width: 768px) {
		.profile-page-root {
			padding: 20px 16px 36px;
		}

		.skeleton-grid {
			grid-template-columns: 1fr;
		}
	}
</style>
