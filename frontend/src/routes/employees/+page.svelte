<script lang="ts">
	import { onMount } from 'svelte';
	import { api, session, upload, type Role, type User } from '$lib/api';
	import { formatRupiah } from '$lib/utils/formatters';

	const blank = (): Partial<User> & { password?: string } => ({ name: '', email: '', username: '', password: '', roleId: 0, jobTitle: '', salary: 0, residence: '', phoneNumber: '', employmentStatus: 'Permanent Employee', avatarUrl: '', isActive: true });
	const employmentStatuses = ['Permanent Employee', 'Contract Employee', 'Intern'];
	let users = $state<User[]>([]), roles = $state<Role[]>([]), form = $state<Partial<User> & { password?: string }>(blank());
	let editing = $state(false), showForm = $state(false), loading = $state(true), error = $state(''), allowed = $state(false);
	const canManage = (role?: string) => ['admin', 'management', 'hr'].includes(role?.toLowerCase() ?? '');

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
		if (!confirm(`Hapus ${user.name}?`)) return;
		try { await api(`/users/${user.id}`, { method: 'DELETE' }); users = users.filter((item) => item.id !== user.id); } catch (e) { error = e instanceof Error ? e.message : 'Gagal menghapus pengguna.'; }
	}
</script>

<svelte:head><title>Manage Employees | my-ple</title></svelte:head>
<main class="page">
	{#if !loading && !allowed}
		<h1>Access denied</h1><p>Manage Employees is only available to Admin, Management, and HR.</p>
	{:else}
		<header><div><h1>Manage Employees</h1><p>All employee records from the users table.</p></div><button onclick={() => { editing = false; form = blank(); showForm = true; }}>Add employee</button></header>
		{#if error}<p class="error">{error}</p>{/if}
		{#if loading}<p>Loading employees…</p>{:else}
			{#if !showForm}
				<div class="table-wrap"><table><thead><tr><th>ID</th><th>Name</th><th>Email</th><th>Role</th><th>Role ID</th><th>Username</th><th>Job title</th><th>Salary</th><th>Residence</th><th>Phone</th><th>Status</th><th>Photo</th><th>Active</th><th>Created</th><th>Updated</th><th>Actions</th></tr></thead><tbody>{#each users as user}<tr><td>{user.id}</td><td>{user.name}</td><td>{user.email}</td><td>{user.role}</td><td>{user.roleId}</td><td>{user.username}</td><td>{user.jobTitle}</td><td>{formatRupiah(user.salary)}</td><td>{user.residence}</td><td>{user.phoneNumber}</td><td>{user.employmentStatus}</td><td>{user.avatarUrl ? 'Uploaded' : '—'}</td><td>{user.isActive ? 'Yes' : 'No'}</td><td>{new Date(user.createdAt).toLocaleString()}</td><td>{new Date(user.updatedAt).toLocaleString()}</td><td class="actions"><button onclick={() => edit(user)}>Edit</button><button class="danger" onclick={() => remove(user)}>Delete</button></td></tr>{/each}</tbody></table></div>
			{:else}
				<form onsubmit={(event) => { event.preventDefault(); save(); }}><h2>{editing ? 'Edit employee' : 'Add employee'}</h2><div class="grid">
					<label>Name<input bind:value={form.name} required /></label><label>Email<input type="email" bind:value={form.email} required /></label><label>Username<input bind:value={form.username} required /></label><label>Password<input type="password" bind:value={form.password} required={!editing} minlength="8" placeholder={editing ? 'Leave blank to keep' : ''} /></label>
					<label>Role<select bind:value={form.roleId} required><option value={0}>Select role</option>{#each roles as role}<option value={role.id}>{role.name}</option>{/each}</select></label><label>Job title<input bind:value={form.jobTitle} /></label><label>Salary<input type="number" min="0" bind:value={form.salary} /><small>{formatRupiah(Number(form.salary) || 0)}</small></label><label>Residence<input bind:value={form.residence} /></label><label>Phone<input bind:value={form.phoneNumber} /></label><label>Employment status<select bind:value={form.employmentStatus}>{#each employmentStatuses as status}<option value={status}>{status}</option>{/each}</select></label>{#if editing}<label>Profile photo<input type="file" accept="image/jpeg,image/png,image/gif" onchange={uploadAvatar} /><small>JPEG, PNG, or GIF; max. 1 MB.</small></label>{/if}<label class="check"><input type="checkbox" bind:checked={form.isActive} /> Active</label>
				</div><div class="form-actions"><button type="button" class="secondary" onclick={cancel}>Cancel</button><button type="submit">Save</button></div></form>
			{/if}
		{/if}
	{/if}
</main>

<style>
	.page { padding: 32px; color: #0f172a; } header { display:flex; justify-content:space-between; align-items:center; gap:16px; margin-bottom:24px } h1,h2,p{margin:0} header p{color:#64748b;margin-top:4px}.table-wrap{overflow:auto;border:1px solid #e2e8f0;border-radius:12px}table{border-collapse:collapse;width:100%;font-size:13px}th,td{padding:10px;white-space:nowrap;border-bottom:1px solid #e2e8f0;text-align:left}th{background:#f8fafc}.actions{display:flex;gap:6px}.grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin:18px 0}label{display:grid;gap:5px;font-weight:600;font-size:14px}input,select{padding:9px;border:1px solid #cbd5e1;border-radius:7px;font:inherit}.check{display:flex;align-items:center;gap:8px}button{border:0;border-radius:7px;background:#0f766e;color:white;padding:9px 13px;font-weight:600;cursor:pointer}.secondary{background:#e2e8f0;color:#334155}.danger{background:#dc2626}.form-actions{display:flex;gap:8px}.error{color:#b91c1c;margin:12px 0}@media(max-width:800px){.page{padding:20px}.grid{grid-template-columns:1fr}header{align-items:flex-start;flex-direction:column}}
</style>
