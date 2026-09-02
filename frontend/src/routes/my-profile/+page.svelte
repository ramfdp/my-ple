<script lang="ts">
	import { onMount } from 'svelte';
	import { api, assetUrl, type User } from '$lib/api';
	let user = $state<User | null>(null), error = $state('');
	onMount(async () => { try { user = (await api<{ user: User }>('/me')).user; } catch (e) { error = e instanceof Error ? e.message : 'Gagal memuat profil.'; } });
	const fields = (user: User) => [['Name', user.name], ['Email', user.email], ['Username', user.username], ['Role', user.role], ['Job title', user.jobTitle], ['Residence', user.residence], ['Phone number', user.phoneNumber], ['Employment status', user.employmentStatus]];
</script>
<svelte:head><title>My Profile | my-ple</title></svelte:head>
<main class="profile">
	<h1>My Profile</h1><p>Your employee data. Changes are managed from Manage Employees.</p>
	{#if error}<p class="error">{error}</p>{:else if !user}<p>Loading profile…</p>{:else}
		<section><img src={assetUrl(user.avatarUrl) || '/Logo_PLE-removebg-preview-300x233.png'} alt={user.name} /><div><h2>{user.name}</h2><span>{user.jobTitle} · {user.role}</span></div></section>
		<dl>{#each fields(user) as field}<div><dt>{field[0]}</dt><dd>{field[1] || '—'}</dd></div>{/each}</dl>
	{/if}
</main>
<style>.profile{max-width:900px;padding:32px;margin:auto;color:#0f172a}.profile>p{color:#64748b}section{margin-top:24px;padding:24px;display:flex;align-items:center;gap:18px;border:1px solid #e2e8f0;border-radius:14px;background:#fff}img{width:76px;height:76px;border-radius:50%;object-fit:cover;background:#f1f5f9}h1,h2,p{margin:0}h2{margin-bottom:5px}span{color:#64748b}dl{display:grid;grid-template-columns:repeat(2,1fr);gap:1px;margin:20px 0;background:#e2e8f0;border:1px solid #e2e8f0;border-radius:14px;overflow:hidden}dl div{background:#fff;padding:16px}dt{font-size:13px;color:#64748b}dd{margin:5px 0 0;font-weight:600}.error{color:#b91c1c;margin-top:16px}@media(max-width:600px){.profile{padding:20px}dl{grid-template-columns:1fr}}</style>
