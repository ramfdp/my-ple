<script lang="ts">
	import { onMount } from 'svelte';
	import { api, assetUrl, type User } from '$lib/api';
	import EmployeeStatusBadge from '$lib/components/EmployeeStatusBadge.svelte';
	import ProfileAvatar from '$lib/components/ProfileAvatar.svelte';
	import ProfileInformationItem from '$lib/components/ProfileInformationItem.svelte';

	let user = $state<User | null>(null);
	let error = $state('');

	async function loadProfile() {
		error = '';
		try { user = (await api<{ user: User }>('/me')).user; }
		catch (cause) { error = cause instanceof Error ? cause.message : 'Gagal memuat profil.'; }
	}

	const display = (value?: string) => value?.trim() || '-';
	const joinedOn = (value: string) => {
		const date = new Date(value);
		return Number.isNaN(date.getTime()) ? '-' : new Intl.DateTimeFormat('en-GB', { dateStyle: 'long' }).format(date);
	};

	onMount(loadProfile);
</script>

<svelte:head><title>My Profile | my-ple</title></svelte:head>

<main class="profile-page">
	<div class="page-intro">
		<p class="eyebrow">ACCOUNT</p><h1>My Profile</h1>
		<p>Your employee information from the company directory.</p>
	</div>

	{#if error}
		<section class="state-card error-state" role="alert">
			<div><h2>Unable to load your profile</h2><p>{error}</p></div>
			<button type="button" onclick={loadProfile}>Try again</button>
		</section>
	{:else if !user}
		<section class="profile-card loading-card" aria-busy="true" aria-label="Loading profile">
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
					<div class="name-row"><h2 id="profile-name">{display(user.name)}</h2><EmployeeStatusBadge status={user.isActive ? 'Active Employee' : 'Inactive'} size="sm" /></div>
					<p>{display(user.jobTitle)}{user.role ? ` · ${user.role}` : ''}</p>
				</div>
				<div class="join-date"><span>Account created</span><strong>{joinedOn(user.createdAt)}</strong></div>
			</div>
		</section>

		<section class="details-card" aria-labelledby="details-heading">
			<header>
				<div><h2 id="details-heading">Profile Details</h2><p>Personal and employment information</p></div>
				<span class="read-only">Managed by HR</span>
			</header>
			<div class="details-grid">
				<ProfileInformationItem icon="user" label="Full Name" value={display(user.name)} />
				<ProfileInformationItem icon="mail" label="Email" value={display(user.email)} />
				<ProfileInformationItem icon="shield" label="Username / Employee ID" value={display(user.username)} />
				<ProfileInformationItem icon="phone" label="Phone Number" value={display(user.phoneNumber)} />
				<ProfileInformationItem icon="briefcase" label="Role" value={display(user.role)} />
				<ProfileInformationItem icon="briefcase" label="Job Title" value={display(user.jobTitle)} />
				<ProfileInformationItem icon="map-pin" label="Address / Residence" value={display(user.residence)} />
				<ProfileInformationItem icon="shield" label="Account Created" value={joinedOn(user.createdAt)} />
				<ProfileInformationItem icon="briefcase" label="Employment Status"><EmployeeStatusBadge status={display(user.employmentStatus)} size="sm" /></ProfileInformationItem>
			</div>
		</section>
	{/if}
</main>

<style>
	.profile-page { width: min(100%, 1120px); margin: 0 auto; padding: clamp(24px, 5vw, 56px); color: #102a43; }
	.page-intro { margin-bottom: 24px; }
	.page-intro h1, .page-intro p, h2 { margin: 0; }
	.page-intro h1 { font-family: 'Plus Jakarta Sans', sans-serif; font-size: clamp(1.8rem, 4vw, 2.5rem); letter-spacing: -0.04em; }
	.page-intro > p:last-child { margin-top: 6px; color: #708090; }
	.eyebrow { margin-bottom: 8px !important; color: #0c6656; font-size: .7rem; font-weight: 700; letter-spacing: .14em; }
	.profile-card, .details-card, .state-card { overflow: hidden; border: 1px solid #dfe8e6; border-radius: 20px; background: #fff; box-shadow: 0 10px 28px rgb(16 42 67 / 6%); }
	.cover { height: clamp(132px, 18vw, 184px); background: linear-gradient(105deg, #d9f5f1, #dff5df 58%, #edf6cb); }
	.identity { display: flex; align-items: flex-end; gap: 22px; min-width: 0; padding: 0 28px 26px; }
	.avatar-wrap { z-index: 1; flex: 0 0 auto; margin-top: -54px; }
	.identity-copy { min-width: 0; flex: 1; padding-top: 20px; }
	.name-row { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
	.name-row h2 { min-width: 0; overflow-wrap: anywhere; font-family: 'Plus Jakarta Sans', sans-serif; font-size: clamp(1.35rem, 3vw, 1.75rem); letter-spacing: -.035em; }
	.identity-copy p { margin: 6px 0 0; color: #64748b; overflow-wrap: anywhere; }
	.join-date { display: grid; gap: 3px; flex: 0 0 auto; text-align: right; }
	.join-date span { color: #7b8a99; font-size: .75rem; font-weight: 600; text-transform: uppercase; letter-spacing: .08em; }
	.join-date strong { font-size: .9rem; }
	.details-card { margin-top: 20px; padding: 24px; }
	.details-card > header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 20px; padding-bottom: 18px; border-bottom: 1px solid #edf1f0; }
	.details-card h2, .state-card h2 { font-family: 'Plus Jakarta Sans', sans-serif; font-size: 1.15rem; letter-spacing: -.025em; }
	.details-card header p, .state-card p { margin: 4px 0 0; color: #708090; font-size: .84rem; }
	.read-only { flex: 0 0 auto; padding: 6px 10px; border: 1px solid #dce7e5; border-radius: 999px; color: #526775; background: #f8fbfa; font-size: .75rem; font-weight: 600; }
	.details-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 14px; }
	.state-card { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 24px; }
	.error-state { border-color: #fecaca; }
	.error-state h2 { color: #991b1b; }
	.state-card button { flex: 0 0 auto; padding: 9px 14px; border: 0; border-radius: 9px; color: #fff; background: #0c6656; font-weight: 700; }
	.loading-card, .loading-details { animation: pulse 1.5s ease-in-out infinite; }
	.loading-cover { height: 160px; background: #e8efed; }
	.loading-body { display: flex; align-items: center; gap: 20px; padding: 0 28px 24px; }
	.loading-body > span { width: 104px; height: 104px; margin-top: -48px; border: 5px solid #fff; border-radius: 50%; background: #dce6e4; }
	.loading-body div { display: grid; gap: 10px; flex: 1; }
	.loading-body i, .loading-details i, .loading-details span { display: block; height: 14px; border-radius: 7px; background: #e8efed; }
	.loading-body i:first-child { width: 38%; height: 22px; }
	.loading-body i:last-child { width: 24%; }
	.loading-details > i { width: 20%; height: 20px; margin-bottom: 24px; }
	.loading-details div { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
	.loading-details span { height: 76px; }
	@keyframes pulse { 50% { opacity: .6; } }
	@media (max-width: 900px) { .details-grid, .loading-details div { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
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
	@media (prefers-reduced-motion: reduce) { .loading-card, .loading-details { animation: none; } }
</style>
