<script lang="ts">
 import { locale, t as translate } from '$lib/i18n';
	import { onMount } from 'svelte';
	import {subscribeChanges} from '$lib/realtime';
	import { api, session, upload, type Role, type User } from '$lib/api';

	const blank = (): Partial<User> & { password?: string } => ({ name: '', email: '', username: '', password: '', roleId: 0, jobTitle: '', salary: 0, residence: '', phoneNumber: '', employmentStatus: 'Permanent Employee', avatarUrl: '', isActive: true });
	const employmentStatuses = ['Permanent Employee', 'Contract Employee', 'Intern'];
	let users = $state<User[]>([]), roles = $state<Role[]>([]), form = $state<Partial<User> & { password?: string }>(blank());
	let editing = $state(false), showForm = $state(false), loading = $state(true), error = $state(''), allowed = $state(false);
	const canManage = (role?: string) => ['admin', 'management', 'hr'].includes(role?.toLowerCase() ?? '');
	onMount(()=>subscribeChanges(['users.changed'],async()=>{if(!allowed)return;try{users=(await api<{users:User[]}>('/users')).users}catch(e){error=(e as Error).message}}));

	onMount(async () => {
		allowed = canManage(session()?.user.role);
		if (!allowed) { loading = false; return; }
		try { const [userData, roleData] = await Promise.all([api<{ users: User[] }>('/users'), api<{ roles: Role[] }>('/roles')]); users = userData.users; roles = roleData.roles; } catch (e) { error = e instanceof Error ? e.message : 'Gagal memuat data.'; } finally { loading = false; }
	});

	function edit(user: User) { form = { ...user, password: '' }; editing = true; showForm = true; }
	function cancel() { form = blank(); editing = false; showForm = false; }
	async function uploadAvatar(event: Event) {
		const file = (event.currentTarget as HTMLInputElement).files?.[0];
		if (!file || !form.id) return;
		if (file.size > 1024 * 1024) { error = 'Ukuran foto maksimal 1 MB.'; return; }
		error = '';
		try { const data = new FormData(); data.append('avatar', file); const result = await upload<{ user: User }>(`/users/${form.id}/avatar`, data); form = { ...form, avatarUrl: result.user.avatarUrl }; users = users.map((user) => user.id === result.user.id ? result.user : user); } catch (e) { error = e instanceof Error ? e.message : 'Gagal mengunggah foto.'; }
	}
	async function save() {
		error = '';
		try {
			const method = editing ? 'PATCH' : 'POST', path = editing ? `/users/${form.id}` : '/users';
			const result = await api<{ user: User }>(path, { method, body: JSON.stringify(form) });
			users = editing ? users.map((user) => user.id === result.user.id ? result.user : user) : [...users, result.user]; cancel();
		} catch (e) { error = e instanceof Error ? e.message : 'Gagal menyimpan pengguna.'; }
	}
	async function remove(user: User) {
		if (!confirm($translate('Hapus {name}?',{name:user.name}))) return;
		try { await api(`/users/${user.id}`, { method: 'DELETE' }); users = users.filter((item) => item.id !== user.id); } catch (e) { error = e instanceof Error ? e.message : 'Gagal menghapus pengguna.'; }
	}
</script>

<svelte:head><title>{$translate("Manage Employees | MyHIS")}</title></svelte:head>
<main class="page">
	{#if !loading && !allowed}
		<h1>{$translate("Access denied")}</h1><p>{$translate("Manage Employees is only available to Admin, Management, and HR.")}</p>
	{:else}
		<header><div><h1>{$translate("Manage Employees")}</h1><p>{$translate("All employee records from the users table.")}</p></div><button onclick={() => { editing = false; form = blank(); showForm = true; }}>{$translate("Add employee")}</button></header>
		{#if error}<p class="error">{$translate(error)}</p>{/if}
		{#if loading}<p>{$translate("Loading employees…")}</p>{:else}
			{#if !showForm}
				<div class="table-wrap"><table><thead><tr><th>{$translate("No.")}</th><th>{$translate("Name")}</th><th>{$translate("Email")}</th><th>{$translate("Role")}</th><th>{$translate("Username")}</th><th>{$translate("Job title")}</th><th>{$translate("Residence")}</th><th>{$translate("Phone")}</th><th>{$translate("Status")}</th><th>{$translate("Photo")}</th><th>{$translate("Active")}</th><th>{$translate("Created")}</th><th>{$translate("Updated")}</th><th>{$translate("Actions")}</th></tr></thead><tbody>{#each users as user, index (user.id)}<tr><td>{index + 1}</td><td>{user.name}</td><td>{user.email}</td><td>{$translate(user.role)}</td><td>{user.username}</td><td>{user.jobTitle}</td><td>{user.residence}</td><td>{user.phoneNumber}</td><td>{$translate(user.employmentStatus)}</td><td>{user.avatarUrl ? $translate('Uploaded') : '—'}</td><td>{user.isActive ? $translate('Yes') : $translate('No')}</td><td>{new Date(user.createdAt).toLocaleString($locale)}</td><td>{new Date(user.updatedAt).toLocaleString($locale)}</td><td class="actions"><button onclick={() => edit(user)}>{$translate("Edit")}</button><button class="danger" onclick={() => remove(user)}>{$translate("Delete")}</button></td></tr>{/each}</tbody></table></div>
			{:else}
				<form onsubmit={(event) => { event.preventDefault(); save(); }}><h2>{editing ? $translate('Edit employee') : $translate('Add employee')}</h2><div class="grid">
					<label>{$translate("Name")}<input bind:value={form.name} required /></label><label>{$translate("Email")}<input type="email" bind:value={form.email} required /></label><label>{$translate("Username")}<input bind:value={form.username} required /></label><label>{$translate("Password")}<input type="password" bind:value={form.password} required={!editing} minlength="8" placeholder={editing ? $translate('Leave blank to keep') : ''} /></label>
					<label>{$translate("Role")}<select bind:value={form.roleId} required><option value={0}>{$translate("Select role")}</option>{#each roles as role}<option value={role.id}>{$translate(role.name)}</option>{/each}</select></label><label>{$translate("Job title")}<input bind:value={form.jobTitle} /></label><label>{$translate("Residence")}<input bind:value={form.residence} /></label><label>{$translate("Phone")}<input bind:value={form.phoneNumber} /></label><label>{$translate("Employment status")}<select bind:value={form.employmentStatus}>{#each employmentStatuses as status}<option value={status}>{$translate(status)}</option>{/each}</select></label>{#if editing}<label>{$translate("Profile photo")}<input type="file" accept="image/jpeg,image/png,image/gif" onchange={uploadAvatar} /><small>{$translate("JPEG, PNG, or GIF; max. 1 MB.")}</small></label>{/if}<label class="check"><input type="checkbox" bind:checked={form.isActive} /> {$translate("Active")}</label>
				</div><div class="form-actions"><button type="button" class="secondary" onclick={cancel}>{$translate("Cancel")}</button><button type="submit">{$translate("Save")}</button></div></form>
			{/if}
		{/if}
	{/if}
</main>

<style>
	header, .form-actions { flex-wrap: wrap; }
	label, input, select { min-width: 0; }
	input:not([type="checkbox"]), select { width: 100%; }
	.page { max-width: 1800px; margin: auto; overflow-wrap: break-word; padding: var(--his-page-padding); color: var(--his-text); } header { display:flex; justify-content:space-between; align-items:center; gap:16px; margin-bottom:24px } h1,h2,p{margin:0} header p{color:var(--his-muted);margin-top:4px}.table-wrap{min-width:0;max-width:100%;overflow:auto;border:1px solid var(--his-border);border-radius:12px}table{border-collapse:collapse;width:100%;font-size:13px}th,td{padding:10px;white-space:nowrap;border-bottom:1px solid var(--his-border);text-align:left}th{background:var(--his-surface)}.actions{display:flex;gap:6px}.grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin:18px 0}label{display:grid;gap:5px;font-weight:600;font-size:14px}input,select{padding:9px;border:1px solid var(--his-border);border-radius:7px;font:inherit}.check{display:flex;align-items:center;gap:8px}button{border:0;border-radius:7px;background:var(--his-primary);color:white;padding:9px 13px;font-weight:600;cursor:pointer}.secondary{background:var(--his-border);color:var(--his-muted)}.danger{background:var(--his-danger)}.form-actions{display:flex;gap:8px}form{max-width:1000px}.error{color:var(--his-danger);margin:12px 0}@media(max-width:800px){.page{padding:var(--his-page-padding)}.grid{grid-template-columns:1fr}header{align-items:flex-start;flex-direction:column}}
</style>
