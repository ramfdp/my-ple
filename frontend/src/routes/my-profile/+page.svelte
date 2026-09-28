<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
	import { onMount } from 'svelte';
	import { subscribeChanges } from '$lib/realtime';
	import { api, assetUrl, upload, type User } from '$lib/api';
	import EmployeeStatusBadge from '$lib/components/EmployeeStatusBadge.svelte';
	import ProfileAvatar from '$lib/components/ProfileAvatar.svelte';
	import ProfileInformationItem from '$lib/components/ProfileInformationItem.svelte';
	import { validateIndonesianPhone } from '$lib/utils/formatters';

	let user = $state<User | null>(null);
	let error = $state('');
	let editing = $state(false), saving = $state(false), editError = $state('');
	let phoneNumber = $state(''), residence = $state(''), avatar = $state<File | null>(null);
	let generation = 0;

	async function loadProfile() {
		if (saving) return;
		const request = ++generation;
		error = '';
		try { const result = (await api<{ user: User }>('/me')).user; if(request === generation) user = result; }
		catch (cause) { if(request === generation) error = cause instanceof Error ? cause.message : 'Gagal memuat profil.'; }
	}

	const display = (value?: string) => value?.trim() || '-';
	const joinedOn = (value: string) => {
		const date = new Date(value);
		return Number.isNaN(date.getTime()) ? '-' : new Intl.DateTimeFormat($locale, { dateStyle: 'long' }).format(date);
	};

	function openEdit() {
		if (!user) return;
		phoneNumber = user.phoneNumber;
		residence = user.residence;
		avatar = null;
		editError = '';
		editing = true;
	}

	function chooseAvatar(event: Event) {
		const file = (event.currentTarget as HTMLInputElement).files?.[0] ?? null;
		if (file && (!['image/jpeg', 'image/png', 'image/gif'].includes(file.type) || file.size > 1024 * 1024)) {
			avatar = null;
			editError = 'Photo must be a JPEG, PNG, or GIF up to 1 MB.';
			return;
		}
		avatar = file;
		editError = '';
	}

	async function saveProfile(event: SubmitEvent) {
		event.preventDefault();
		if (!validateIndonesianPhone(phoneNumber) || residence.trim().length > 255) {
			editError = 'Enter a valid Indonesian phone number and an address up to 255 characters.';
			return;
		}
		saving = true;
		generation++;
		editError = '';
		try {
			user = (await api<{ user: User }>('/profile', { method: 'PATCH', body: JSON.stringify({ phoneNumber, residence }) })).user;
			if (avatar) {
				const data = new FormData(); data.append('avatar', avatar);
				user = (await upload<{ user: User }>('/profile/avatar', data)).user;
			}
			editing = false;
		} catch (cause) { editError = cause instanceof Error ? cause.message : 'Failed to update profile.'; }
		finally { saving = false; void loadProfile(); }
	}

	function closeBackdrop(event: MouseEvent) {
		if (event.target === event.currentTarget) editing = false;
	}

	onMount(() => { void loadProfile(); const stop = subscribeChanges(['users.changed'], loadProfile, change => !change.entityId || change.entityId === user?.id); return () => { generation++; stop(); }; });
</script>

<svelte:head><title>{$translate("My Profile | MyHIS")}</title></svelte:head>

<main class="profile-page">
	<div class="page-intro">
		<p class="eyebrow">{$translate("ACCOUNT")}</p><h1>{$translate("My Profile")}</h1>
		<p>{$translate("Your employee information from the company directory.")}</p>
	</div>

	{#if error && user}<p role="alert">{$translate(error)}</p>{/if}
	{#if error && !user}
		<section class="state-card error-state" role="alert">
			<div><h2>{$translate("Unable to load your profile")}</h2><p>{$translate(error)}</p></div>
			<button type="button" onclick={loadProfile}>{$translate("Try again")}</button>
		</section>
	{:else if !user}
		<section class="profile-card loading-card" aria-busy="true" aria-label={$translate("Loading profile")}>
			<div class="loading-cover"></div><div class="loading-body"><span></span><div><i></i><i></i></div></div>
		</section>
		<section class="details-card loading-details" aria-hidden="true">
			<i></i><div>{#each Array(6) as _}<span></span>{/each}</div>
		</section>
	{:else}
		<section class="profile-card" aria-labelledby="profile-name">
			<div class="cover" aria-hidden="true"></div>
			<div class="identity">
				<div class="avatar-wrap"><ProfileAvatar src={assetUrl(user.avatarUrl)} alt={`${user.name}'s profile photo`} size={112} borderWidth={5} /></div>
				<div class="identity-copy">
					<div class="name-row"><h2 id="profile-name">{display(user.name)}</h2><EmployeeStatusBadge status={user.isActive ? 'Active Employee' : 'Inactive'} /></div>
					<p>{display(user.jobTitle)}{user.role ? ` · ${$translate(user.role)}` : ''}</p>
				</div>
				<div class="join-date"><span>{$translate("Account created")}</span><strong>{joinedOn(user.createdAt)}</strong></div>
			</div>
		</section>

		<section class="details-card" aria-labelledby="details-heading">
			<header>
				<div><h2 id="details-heading">{$translate("Profile Details")}</h2><p>{$translate("Personal and employment information")}</p></div>
				<button class="edit-button" type="button" onclick={openEdit}>{$translate("Edit Profile")}</button>
			</header>
			<div class="details-grid">
				<ProfileInformationItem icon="user" label={$translate("Full Name")} value={display(user.name)} />
				<ProfileInformationItem icon="mail" label={$translate("Email")} value={display(user.email)} />
				<ProfileInformationItem icon="shield" label={$translate("Username / Employee ID")} value={display(user.username)} />
				<ProfileInformationItem icon="phone" label={$translate("Phone Number")} value={display(user.phoneNumber)} />
				<ProfileInformationItem icon="briefcase" label={$translate("Role")} value={$translate(display(user.role))} />
				<ProfileInformationItem icon="briefcase" label={$translate("Job Title")} value={display(user.jobTitle)} />
				<ProfileInformationItem icon="map-pin" label={$translate("Address / Residence")} value={display(user.residence)} />
				<ProfileInformationItem icon="shield" label={$translate("Account Created")} value={joinedOn(user.createdAt)} />
				<ProfileInformationItem icon="briefcase" label={$translate("Employment Status")}><EmployeeStatusBadge status={display(user.employmentStatus)} /></ProfileInformationItem>
			</div>
		</section>
	{/if}

	{#if editing && user}
		<div class="modal-backdrop" role="presentation" onclick={closeBackdrop}>
			<div class="edit-modal" role="dialog" aria-modal="true" aria-labelledby="edit-title">
				<form onsubmit={saveProfile}>
					<header><div><h2 id="edit-title">{$translate("Edit Profile")}</h2><p>{$translate("Update your contact information and profile photo.")}</p></div><button type="button" class="close" aria-label={$translate("Close")} onclick={() => (editing = false)}>×</button></header>
					{#if editError}<p class="edit-error" role="alert">{$translate(editError)}</p>{/if}
					<label>{$translate("Profile Photo")}<input type="file" accept="image/jpeg,image/png,image/gif" onchange={chooseAvatar} /><small>{$translate("JPEG, PNG, or GIF; max. 1 MB.")}</small></label>
					<label>{$translate("Phone Number")}<input type="tel" bind:value={phoneNumber} maxlength="50" required placeholder="+62 812-3456-7890" /></label>
					<label>{$translate("Address / Residence")}<textarea bind:value={residence} maxlength="255" rows="4"></textarea></label>
					<div class="modal-actions"><button type="button" class="cancel" onclick={() => (editing = false)}>{$translate("Cancel")}</button><button type="submit" disabled={saving}>{saving ? $translate('Saving…') : $translate('Save Changes')}</button></div>
				</form>
			</div>
		</div>
	{/if}
</main>

<style>
	.profile-page { width: min(100%, 1120px); margin: 0 auto; padding: var(--his-page-padding); color: var(--his-text); }
	.page-intro { margin-bottom: 24px; }
	.page-intro h1, .page-intro p, h2 { margin: 0; }
	.page-intro h1 { font-family: 'Plus Jakarta Sans', sans-serif; font-size: clamp(1.8rem, 4vw, 2.5rem); letter-spacing: -0.04em; }
	.page-intro > p:last-child { margin-top: 6px; color: var(--his-muted); }
	.eyebrow { margin-bottom: 8px !important; color: var(--his-primary); font-size: .7rem; font-weight: 700; letter-spacing: .14em; }
	.profile-card, .details-card, .state-card { overflow: hidden; border: 1px solid var(--his-border); border-radius: 20px; background: #fff; box-shadow: 0 10px 28px rgb(23 23 23 / 0.08); }
	.cover { height: clamp(132px, 18vw, 184px); background: linear-gradient(105deg, var(--his-primary-soft), var(--his-primary-soft) 48%, var(--his-primary-soft)); }
	.identity { display: flex; align-items: flex-end; gap: 22px; min-width: 0; padding: 0 28px 26px; }
	.avatar-wrap { z-index: 1; flex: 0 0 auto; margin-top: -54px; }
	.identity-copy { min-width: 0; flex: 1; padding-top: 20px; }
	.name-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
	.name-row h2 { min-width: 0; overflow-wrap: anywhere; font-family: 'Plus Jakarta Sans', sans-serif; font-size: clamp(1.35rem, 3vw, 1.75rem); letter-spacing: -.035em; }
	.identity-copy p { margin: 6px 0 0; color: var(--his-muted); overflow-wrap: anywhere; }
	.join-date { display: grid; gap: 3px; flex: 0 0 auto; text-align: right; }
	.join-date span { color: var(--his-muted); font-size: .75rem; font-weight: 600; text-transform: uppercase; letter-spacing: .08em; }
	.join-date strong { font-size: .9rem; }
	.details-card { margin-top: 20px; padding: 24px; animation-delay: 160ms; }
	.details-card > header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 20px; padding-bottom: 18px; border-bottom: 1px solid var(--his-border); }
	.details-card h2, .state-card h2 { font-family: 'Plus Jakarta Sans', sans-serif; font-size: 1.15rem; letter-spacing: -.025em; }
	.details-card header p, .state-card p { margin: 4px 0 0; color: var(--his-muted); font-size: .84rem; }
	.edit-button, .modal-actions button { flex: 0 0 auto; padding: 9px 14px; border: 0; border-radius: 9px; color: #fff; background: var(--his-primary); font-weight: 700; cursor: pointer; }
	.details-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 240px), 1fr)); gap: 14px; }
	.state-card { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 24px; }
	.error-state { border-color: var(--his-danger-border); }
	.error-state h2 { color: var(--his-danger); }
	.state-card button { flex: 0 0 auto; padding: 9px 14px; border: 0; border-radius: 9px; color: #fff; background: var(--his-primary); font-weight: 700; }
	.modal-backdrop { position: fixed; inset: 0; z-index: 50; display: grid; place-items: center; padding: 18px; background: rgb(23 23 23 / .5); }
	.edit-modal { min-width: 0; width: min(100%, 520px); max-height: calc(100dvh - 36px); overflow: auto; padding: 24px; border-radius: 18px; background: #fff; box-shadow: 0 20px 50px rgb(23 23 23 / .2); }
	.edit-modal header { display: flex; justify-content: space-between; gap: 16px; margin-bottom: 20px; }
	.edit-modal header p { margin: 4px 0 0; color: var(--his-muted); font-size: .84rem; }
	.edit-modal .close { flex: 0 0 44px; min-height: 44px; border: 0; background: transparent; color: var(--his-muted); font-size: 1.6rem; cursor: pointer; }
	.edit-modal label { display: grid; gap: 7px; margin-top: 16px; font-size: .84rem; font-weight: 700; }
	.edit-modal input, .edit-modal textarea { box-sizing: border-box; width: 100%; padding: 10px 12px; border: 1px solid var(--his-border); border-radius: 9px; font: inherit; }
	.edit-modal small { color: var(--his-muted); font-weight: 400; }
	.edit-error { padding: 10px; border-radius: 8px; color: var(--his-danger); background: var(--his-danger-soft); }
	.modal-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 10px; margin-top: 22px; }
	.edit-modal label, .edit-modal input, .edit-modal textarea { min-width: 0; }
	.identity, .details-card > header, .state-card { flex-wrap: wrap; }
	@media (min-width: 641px) { .identity-copy { flex-basis: 180px; } }
	.modal-actions .cancel { color: var(--his-muted); background: var(--his-border); }
	.loading-card, .loading-details { animation: pulse 1.5s ease-in-out infinite; }
	.loading-cover { height: 160px; background: var(--his-surface); }
	.loading-body { display: flex; align-items: center; gap: 20px; padding: 0 28px 24px; }
	.loading-body > span { width: 104px; height: 104px; margin-top: -48px; border: 5px solid #fff; border-radius: 50%; background: var(--his-border); }
	.loading-body div { display: grid; gap: 10px; flex: 1; }
	.loading-body i, .loading-details i, .loading-details span { display: block; height: 14px; border-radius: 7px; background: var(--his-surface); }
	.loading-body i:first-child { width: 38%; height: 22px; }
	.loading-body i:last-child { width: 24%; }
	.loading-details > i { width: 20%; height: 20px; margin-bottom: 24px; }
	.loading-details div { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
	.loading-details span { height: 76px; }
	@keyframes pulse { 50% { opacity: .6; } }
	@media (max-width: 900px) { .loading-details div { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
	@media (max-width: 640px) {
		.profile-page { padding: 24px 18px; }
		.profile-card, .details-card, .state-card { border-radius: 16px; }
		.identity { align-items: flex-start; flex-direction: column; gap: 12px; padding: 0 18px 20px; }
		.identity-copy { padding-top: 0; }
		.join-date { text-align: left; }
		.details-card { padding: 18px; }
		.details-card > header, .state-card { align-items: flex-start; flex-direction: column; }
		.details-grid, .loading-details div { grid-template-columns: 1fr; }
		.state-card button { width: 100%; }
	}
	@media (max-width: 480px) { .profile-page { padding: 20px 14px; } .page-intro { margin-bottom: 18px; } .page-intro h1 { font-size: 1.75rem; } .identity { padding-inline: 14px; } .details-card { padding: 14px; } .details-card > header { gap: 10px; } }
	@media (prefers-reduced-motion: reduce) { .loading-card, .loading-details { animation: none; } }
</style>
