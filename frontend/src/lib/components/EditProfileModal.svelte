<script lang="ts">
	import type { EmployeeProfile, EmploymentStatus, ProfileFormErrors, UserRole } from '$lib/types/employee';
	import { validateEmail, validateIndonesianPhone } from '$lib/utils/formatters';
	import ProfileAvatar from './ProfileAvatar.svelte';

	let {
		isOpen = false,
		profile,
		currentUserRole = 'Employee',
		onclose,
		onsave
	}: {
		isOpen: boolean;
		profile: EmployeeProfile;
		currentUserRole: UserRole;
		onclose: () => void;
		onsave: (updated: EmployeeProfile) => Promise<void> | void;
	} = $props();

	const divisionOptions = [
		'Information Technology',
		'Software Engineering',
		'AI Dev',
		'Project Engineer',
		'Keuangan',
		'Business Development',
		'HR',
		'PMO'
	];

	const statusOptions: EmploymentStatus[] = [
		'Permanent Employee',
		'Contract Employee',
		'Probation',
		'Intern',
		'Freelancer',
		'Inactive'
	];

	// ponytail: local clone of profile for zero-mutation editing
	let formData = $state<EmployeeProfile>({
		id: '',
		name: '',
		jobTitle: '',
		division: '',
		currentSalary: 0,
		residence: '',
		phoneNumber: '',
		email: '',
		employmentStatus: 'Permanent Employee',
		avatarUrl: '',
		isActive: true
	});
	let errors = $state<ProfileFormErrors>({});
	let isSubmitting = $state(false);

	let isHR = $derived(currentUserRole === 'HR / Admin');

	$effect(() => {
		if (isOpen) {
			formData = { ...profile };
			errors = {};
		}
	});

	function validate(): boolean {
		const newErrors: ProfileFormErrors = {};

		if (!formData.name.trim()) {
			newErrors.name = 'Nama lengkap wajib diisi.';
		}

		if (!formData.division.trim()) {
			newErrors.division = 'Divisi wajib dipilih.';
		}

		if (formData.currentSalary <= 0 || isNaN(formData.currentSalary)) {
			newErrors.currentSalary = 'Gaji harus berupa angka positif lebih dari 0.';
		}

		if (!formData.residence.trim()) {
			newErrors.residence = 'Alamat domisili wajib diisi.';
		}

		if (!formData.phoneNumber.trim()) {
			newErrors.phoneNumber = 'Nomor telepon wajib diisi.';
		} else if (!validateIndonesianPhone(formData.phoneNumber)) {
			newErrors.phoneNumber = 'Format nomor Indonesia tidak valid (contoh: +62 812-3456-7890 atau 081234567890).';
		}

		if (!formData.email.trim()) {
			newErrors.email = 'Email wajib diisi.';
		} else if (!validateEmail(formData.email)) {
			newErrors.email = 'Format alamat email tidak valid.';
		}

		if (!formData.employmentStatus) {
			newErrors.employmentStatus = 'Status kepegawaian wajib dipilih.';
		}

		errors = newErrors;
		return Object.keys(newErrors).length === 0;
	}

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		if (!validate()) return;

		isSubmitting = true;
		try {
			// ponytail: simulate realistic async API response
			await new Promise((resolve) => setTimeout(resolve, 500));
			await onsave(formData);
		} finally {
			isSubmitting = false;
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && isOpen) {
			onclose();
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if isOpen}
	<div class="modal-backdrop" onclick={onclose} onkeydown={(e) => e.key === 'Escape' && onclose()} role="presentation">
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
		<div
			class="modal-dialog"
			onclick={(e) => e.stopPropagation()}
			role="dialog"
			aria-modal="true"
			aria-labelledby="modal-title"
			tabindex="-1"
		>
			<header class="modal-header">
				<div>
					<h2 id="modal-title" class="modal-title">Edit Employee Profile</h2>
					<p class="modal-subtitle">Perbarui informasi profil dan data kepegawaian karyawan.</p>
				</div>
				<button type="button" class="close-btn" onclick={onclose} aria-label="Tutup modal">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
						<line x1="18" y1="6" x2="6" y2="18" />
						<line x1="6" y1="6" x2="18" y2="18" />
					</svg>
				</button>
			</header>

			{#if !isHR}
				<div class="rbac-notice-banner" role="alert">
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
						<rect width="18" height="11" x="3" y="11" rx="2" ry="2" />
						<path d="M7 11V7a5 5 0 0 1 10 0v4" />
					</svg>
					<span>Anda sedang login sebagai <strong>Employee</strong>. Field Divisi, Gaji, dan Status dikunci (hanya dapat diubah oleh HR / Admin).</span>
				</div>
			{/if}

			<form onsubmit={handleSubmit} class="modal-form" novalidate>
				<!-- Avatar row -->
				<div class="avatar-edit-section">
					<ProfileAvatar
						src={formData.avatarUrl}
						alt={formData.name}
						size={72}
						borderWidth={3}
					/>
					<div class="avatar-inputs">
						<label for="avatarUrl" class="field-label">Foto Profil URL</label>
						<input
							id="avatarUrl"
							type="url"
							bind:value={formData.avatarUrl}
							placeholder="https://images.unsplash.com/..."
							class="text-input"
						/>
						<span class="field-hint">Gunakan URL foto berformat jpg, png, atau webp.</span>
					</div>
				</div>

				<div class="form-grid">
					<!-- Name -->
					<div class="form-field">
						<label for="employee-name-input" class="field-label">
							Nama Lengkap <span class="required-star">*</span>
						</label>
						<input
							id="employee-name-input"
							type="text"
							bind:value={formData.name}
							placeholder="Masukkan nama lengkap"
							class="text-input"
							class:has-error={!!errors.name}
							required
						/>
						{#if errors.name}
							<p class="error-text">{errors.name}</p>
						{/if}
					</div>

					<!-- Job Title -->
					<div class="form-field">
						<label for="job-title-input" class="field-label">
							Jabatan / Job Title
						</label>
						<input
							id="job-title-input"
							type="text"
							bind:value={formData.jobTitle}
							placeholder="e.g. Software Engineering"
							class="text-input"
						/>
					</div>

					<!-- Division (HR Only) -->
					<div class="form-field">
						<label for="division-select" class="field-label">
							Divisi <span class="required-star">*</span>
							{#if !isHR}<span class="badge-locked">HR Only</span>{/if}
						</label>
						<select
							id="division-select"
							bind:value={formData.division}
							disabled={!isHR}
							class="select-input"
							class:disabled-input={!isHR}
							class:has-error={!!errors.division}
						>
							{#each divisionOptions as division}
								<option value={division}>{division}</option>
							{/each}
						</select>
						{#if errors.division}
							<p class="error-text">{errors.division}</p>
						{/if}
					</div>

					<!-- Salary (HR Only) -->
					<div class="form-field">
						<label for="salary-input" class="field-label">
							Gaji Bulanan (IDR) <span class="required-star">*</span>
							{#if !isHR}<span class="badge-locked">HR Only</span>{/if}
						</label>
						<input
							id="salary-input"
							type="number"
							bind:value={formData.currentSalary}
							disabled={!isHR}
							min="1"
							step="100000"
							placeholder="12500000"
							class="text-input"
							class:disabled-input={!isHR}
							class:has-error={!!errors.currentSalary}
						/>
						{#if errors.currentSalary}
							<p class="error-text">{errors.currentSalary}</p>
						{/if}
					</div>

					<!-- Residence -->
					<div class="form-field">
						<label for="residence-input" class="field-label">
							Domisili / Tempat Tinggal <span class="required-star">*</span>
						</label>
						<input
							id="residence-input"
							type="text"
							bind:value={formData.residence}
							placeholder="e.g. South Jakarta"
							class="text-input"
							class:has-error={!!errors.residence}
							required
						/>
						{#if errors.residence}
							<p class="error-text">{errors.residence}</p>
						{/if}
					</div>

					<!-- Phone Number -->
					<div class="form-field">
						<label for="phone-input" class="field-label">
							Nomor Telepon <span class="required-star">*</span>
						</label>
						<input
							id="phone-input"
							type="tel"
							bind:value={formData.phoneNumber}
							placeholder="+62 812-3456-7890"
							class="text-input"
							class:has-error={!!errors.phoneNumber}
							required
						/>
						{#if errors.phoneNumber}
							<p class="error-text">{errors.phoneNumber}</p>
						{/if}
					</div>

					<!-- Email -->
					<div class="form-field">
						<label for="email-input" class="field-label">
							Alamat Email <span class="required-star">*</span>
						</label>
						<input
							id="email-input"
							type="email"
							bind:value={formData.email}
							placeholder="andi.pratama@company.com"
							class="text-input"
							class:has-error={!!errors.email}
							required
						/>
						{#if errors.email}
							<p class="error-text">{errors.email}</p>
						{/if}
					</div>

					<!-- Employment Status (HR Only) -->
					<div class="form-field">
						<label for="status-select" class="field-label">
							Status Kepegawaian <span class="required-star">*</span>
							{#if !isHR}<span class="badge-locked">HR Only</span>{/if}
						</label>
						<select
							id="status-select"
							bind:value={formData.employmentStatus}
							disabled={!isHR}
							class="select-input"
							class:disabled-input={!isHR}
							class:has-error={!!errors.employmentStatus}
						>
							{#each statusOptions as status}
								<option value={status}>{status}</option>
							{/each}
						</select>
						{#if errors.employmentStatus}
							<p class="error-text">{errors.employmentStatus}</p>
						{/if}
					</div>
				</div>

				<footer class="modal-footer">
					<button
						type="button"
						class="cancel-btn"
						onclick={onclose}
						disabled={isSubmitting}
					>
						Cancel
					</button>

					<button
						type="submit"
						class="save-btn"
						disabled={isSubmitting}
					>
						{#if isSubmitting}
							<span class="spinner" aria-hidden="true"></span>
							<span>Saving...</span>
						{:else}
							<span>Save Changes</span>
						{/if}
					</button>
				</footer>
			</form>
		</div>
	</div>
{/if}

<style>
	.modal-backdrop {
		position: fixed;
		inset: 0;
		background: rgba(23, 23, 23, 0.45);
		backdrop-filter: blur(4px);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 50;
		padding: 20px;
	}

	.modal-dialog {
		background: #ffffff;
		border-radius: 18px;
		width: 100%;
		max-width: 680px;
		max-height: 90vh;
		overflow-y: auto;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
		border: 1px solid var(--his-border);
		animation: modalFadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
	}

	@keyframes modalFadeIn {
		from {
			opacity: 0;
			transform: scale(0.97) translateY(8px);
		}
		to {
			opacity: 1;
			transform: scale(1) translateY(0);
		}
	}

	.modal-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		padding: 24px 28px 18px;
		border-bottom: 1px solid var(--his-surface);
		gap: 16px;
	}

	.modal-title {
		font-size: 1.25rem;
		font-weight: 700;
		color: var(--his-text);
		letter-spacing: -0.02em;
		margin: 0 0 4px;
	}

	.modal-subtitle {
		font-size: 0.84375rem;
		color: var(--his-muted);
		margin: 0;
	}

	.close-btn {
		width: 34px;
		height: 34px;
		border-radius: 8px;
		border: none;
		background: var(--his-surface);
		color: var(--his-muted);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.15s ease;
		flex-shrink: 0;
	}

	.close-btn:hover {
		background: var(--his-border);
		color: var(--his-text);
	}

	.close-btn svg {
		width: 18px;
		height: 18px;
	}

	.rbac-notice-banner {
		margin: 16px 28px 0;
		padding: 10px 14px;
		background: var(--his-surface);
		border: 1px solid var(--his-border);
		border-radius: 10px;
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 0.8125rem;
		color: var(--his-muted);
		line-height: 1.4;
	}

	.rbac-notice-banner svg {
		width: 16px;
		height: 16px;
		color: var(--his-primary);
		flex-shrink: 0;
	}

	.modal-form {
		padding: 24px 28px;
	}

	.avatar-edit-section {
		display: flex;
		align-items: center;
		gap: 20px;
		padding-bottom: 20px;
		border-bottom: 1px solid var(--his-surface);
		margin-bottom: 20px;
	}

	.avatar-inputs {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.field-hint {
		font-size: 0.75rem;
		color: #737373;
	}

	.form-grid {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: 18px 20px;
	}

	.form-field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.field-label {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 0.8125rem;
		font-weight: 600;
		color: var(--his-muted);
	}

	.required-star {
		color: var(--his-danger);
	}

	.badge-locked {
		font-size: 0.6875rem;
		font-weight: 600;
		background: var(--his-surface);
		color: var(--his-muted);
		padding: 2px 6px;
		border-radius: 4px;
		margin-left: auto;
	}

	.text-input,
	.select-input {
		width: 100%;
		padding: 10px 14px;
		border-radius: 10px;
		border: 1px solid var(--his-border);
		background: #ffffff;
		color: var(--his-text);
		font-size: 0.875rem;
		outline: none;
		transition: all 0.15s ease;
	}

	.text-input:focus,
	.select-input:focus {
		border-color: var(--his-primary);
		box-shadow: 0 0 0 3px rgba(0, 122, 65, 0.15);
	}

	.text-input.has-error,
	.select-input.has-error {
		border-color: var(--his-danger);
		box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.12);
	}

	.disabled-input {
		background: var(--his-surface);
		color: #737373;
		border-color: var(--his-border);
		cursor: not-allowed;
	}

	.error-text {
		font-size: 0.75rem;
		color: var(--his-danger);
		margin: 2px 0 0;
		font-weight: 500;
	}

	.modal-footer {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 12px;
		padding-top: 24px;
		margin-top: 24px;
		border-top: 1px solid var(--his-surface);
	}

	.cancel-btn {
		padding: 10px 20px;
		border-radius: 10px;
		border: 1px solid var(--his-border);
		background: #ffffff;
		color: var(--his-muted);
		font-size: 0.875rem;
		font-weight: 600;
		transition: all 0.15s ease;
	}

	.cancel-btn:hover:not(:disabled) {
		background: var(--his-surface);
		color: var(--his-text);
	}

	.save-btn {
		padding: 10px 24px;
		border-radius: 10px;
		border: none;
		background: var(--his-primary);
		color: #ffffff;
		font-size: 0.875rem;
		font-weight: 600;
		display: inline-flex;
		align-items: center;
		gap: 8px;
		transition: all 0.15s ease;
		box-shadow: 0 2px 6px rgba(0, 122, 65, 0.25);
	}

	.save-btn:hover:not(:disabled) {
		background: var(--his-primary);
		box-shadow: 0 4px 12px rgba(0, 122, 65, 0.35);
		transform: translateY(-1px);
	}

	.save-btn:disabled {
		opacity: 0.65;
		cursor: not-allowed;
	}

	.spinner {
		width: 14px;
		height: 14px;
		border: 2px solid rgba(255, 255, 255, 0.35);
		border-top-color: #ffffff;
		border-radius: 9999px;
		animation: spin 0.6s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	@media (max-width: 640px) {
		.form-grid {
			grid-template-columns: 1fr;
			gap: 14px;
		}

		.modal-dialog {
			max-height: 95vh;
			border-radius: 14px;
		}

		.modal-header,
		.modal-form {
			padding: 18px 20px;
		}

		.rbac-notice-banner {
			margin: 12px 20px 0;
		}

		.avatar-edit-section {
			flex-direction: column;
			align-items: flex-start;
		}
	}
</style>
