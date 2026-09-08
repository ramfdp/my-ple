<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, session, setAuthenticated } from '$lib/api';

	// ponytail: native Svelte reactivity for mobile drawer without external drawer lib
	let mobileOpen = $state(false);
	let canManageEmployees = $state(false);
	onMount(() => {
		const role = session()?.user.role?.toLowerCase() ?? '';
		canManageEmployees = ['admin', 'management', 'hr'].includes(role);
	});
	async function logout(event: MouseEvent) {
		event.preventDefault();
		try { await api('/auth/logout', { method: 'POST' }); } finally { setAuthenticated(null); await goto('/'); }
	}

	const navItems = [
		{ name: 'Dashboard', href: '/dashboard', icon: 'dashboard' },
		{ name: 'My Profile', href: '/my-profile', icon: 'user' },
		{ name: 'Attendance', href: '/attendance', icon: 'calendar' },
		{ name: 'Leave', href: '/leave', icon: 'clock' },
		{ name: 'Payroll', href: '/payroll', icon: 'wallet' },
		{ name: 'Documents', href: '/documents', icon: 'file' },
		{ name: 'Pengaturan Pekerjaan', href: '/settings/jobs', icon: 'settings' }
	];
	let visibleNavItems = $derived(canManageEmployees ? [{ name: 'Employees', href: '/employees', icon: 'users' }, ...navItems] : navItems);

	function isItemActive(href: string): boolean {
		if (href === '/my-profile') {
			return page.url.pathname === '/my-profile' || page.url.pathname === '/profile';
		}
		return page.url.pathname === href;
	}
</script>

<!-- Mobile Topbar with Hamburger Toggle -->
<div class="mobile-topbar">
	<button
		type="button"
		class="mobile-menu-btn"
		onclick={() => (mobileOpen = !mobileOpen)}
		aria-label="Toggle navigation menu"
		aria-expanded={mobileOpen}
	>
		{#if mobileOpen}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		{:else}
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<line x1="3" y1="12" x2="21" y2="12" />
				<line x1="3" y1="6" x2="21" y2="6" />
				<line x1="3" y1="18" x2="21" y2="18" />
			</svg>
		{/if}
	</button>
	<div class="mobile-brand">
		<img src="/harmoni%20insan%20phooto.webp" alt="my HIS" class="mobile-logo-img" />
		<span class="brand-text">my HIS</span>
	</div>
</div>

<!-- Mobile Drawer Backdrop -->
{#if mobileOpen}
	<div class="mobile-backdrop" onclick={() => (mobileOpen = false)} role="presentation"></div>
{/if}

<!-- Sidebar -->
<aside class="sidebar-container" class:open-mobile={mobileOpen} aria-label="Navigasi Utama">
	<!-- Logo Section -->
	<div class="sidebar-header">
		<a href="/dashboard" class="logo-link" onclick={() => (mobileOpen = false)}>
			<div class="logo-circle">
				<img src="/harmoni%20insan%20phooto.webp" alt="my HIS" class="sidebar-logo" />
			</div>
			<div class="brand-details">
				<span class="brand-title">my HIS</span>
			</div>
		</a>
	</div>

	<!-- Navigation Menu -->
	<nav class="sidebar-nav">
		{#each visibleNavItems as item}
			{@const active = isItemActive(item.href)}
			<a
				href={item.href}
				class="nav-item"
				class:active
				onclick={() => (mobileOpen = false)}
			>
				<span class="nav-icon" aria-hidden="true">
					{#if item.icon === 'dashboard'}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<rect width="7" height="9" x="3" y="3" rx="1" />
							<rect width="7" height="5" x="14" y="3" rx="1" />
							<rect width="7" height="9" x="14" y="12" rx="1" />
							<rect width="7" height="5" x="3" y="16" rx="1" />
						</svg>
					{:else if item.icon === 'users'}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
							<circle cx="9" cy="7" r="4" />
							<path d="M22 21v-2a4 4 0 0 0-3-3.87" />
							<path d="M16 3.13a4 4 0 0 1 0 7.75" />
						</svg>
					{:else if item.icon === 'user'}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
							<circle cx="12" cy="7" r="4" />
						</svg>
					{:else if item.icon === 'calendar'}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<rect width="18" height="18" x="3" y="4" rx="2" />
							<line x1="16" x2="16" y1="2" y2="6" />
							<line x1="8" x2="8" y1="2" y2="6" />
							<line x1="3" x2="21" y1="10" y2="10" />
						</svg>
					{:else if item.icon === 'clock'}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<circle cx="12" cy="12" r="10" />
							<polyline points="12 6 12 12 16 14" />
						</svg>
					{:else if item.icon === 'wallet'}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<rect width="20" height="14" x="2" y="5" rx="2" />
							<line x1="2" x2="22" y1="10" y2="10" />
							<circle cx="16" cy="14" r="1" />
						</svg>
					{:else if item.icon === 'file'}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z" />
							<path d="M14 2v4a2 2 0 0 0 2 2h4" />
						</svg>
					{:else if item.icon === 'settings'}
						<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<circle cx="12" cy="12" r="3" />
							<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
						</svg>
					{/if}
				</span>
				<span class="nav-text">{item.name}</span>
			</a>
		{/each}
	</nav>

	<!-- Footer / Logout -->
	<div class="sidebar-footer">
		<a href="/" class="logout-button" onclick={logout}>
			<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
				<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
				<polyline points="16 17 21 12 16 7" />
				<line x1="21" x2="9" y1="12" y2="12" />
			</svg>
			<span>Log out</span>
		</a>
	</div>
</aside>

<style>
	/* ponytail: clean sticky desktop sidebar + hardware-accelerated mobile drawer */
	.sidebar-container {
		position: sticky;
		top: 0;
		width: 250px;
		height: 100dvh;
		background: #ffffff;
		border-right: 1px solid var(--his-border);
		display: flex;
		flex-direction: column;
		flex-shrink: 0;
		z-index: 40;
		transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1);
	}

	.sidebar-header {
		padding: 24px 20px 20px;
		border-bottom: 1px solid var(--his-surface);
	}

	.logo-link {
		display: flex;
		align-items: center;
		gap: 12px;
		text-decoration: none;
		color: inherit;
	}

	.logo-circle {
		width: 42px;
		height: 42px;
		border-radius: 10px;
		background: var(--his-primary-soft);
		border: 1px solid var(--his-primary-soft);
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		flex-shrink: 0;
	}

	.sidebar-logo {
		width: 32px;
		height: 32px;
		object-fit: contain;
	}

	.brand-details {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.brand-title {
		font-family: 'Plus Jakarta Sans', sans-serif;
		font-size: 1.125rem;
		font-weight: 800;
		color: var(--his-text);
		letter-spacing: -0.03em;
		line-height: 1.2;
	}

	.brand-badge {
		font-size: 0.6875rem;
		font-weight: 600;
		color: var(--his-primary);
		letter-spacing: 0.02em;
		text-transform: uppercase;
	}

	.sidebar-nav {
		flex: 1;
		padding: 18px 12px;
		display: flex;
		flex-direction: column;
		gap: 4px;
		overflow-y: auto;
	}

	.nav-item {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 14px;
		border-radius: 10px;
		color: var(--his-muted);
		text-decoration: none;
		font-size: 0.875rem;
		font-weight: 600;
		transition: all 0.15s ease;
	}

	.nav-item:hover {
		background: var(--his-surface);
		color: var(--his-text);
	}

	.nav-item.active {
		background: var(--his-primary);
		color: #ffffff;
		box-shadow: 0 2px 8px rgba(0, 122, 65, 0.28);
	}

	.nav-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		flex-shrink: 0;
	}

	.nav-icon svg {
		width: 18px;
		height: 18px;
	}

	.nav-text {
		letter-spacing: -0.01em;
	}

	.sidebar-footer {
		padding: 16px 12px;
		border-top: 1px solid var(--his-surface);
	}

	.logout-button {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 10px 14px;
		border-radius: 10px;
		color: var(--his-danger);
		text-decoration: none;
		font-size: 0.875rem;
		font-weight: 600;
		transition: all 0.15s ease;
	}

	.logout-button:hover {
		background: var(--his-danger-soft);
	}

	.logout-button svg {
		width: 18px;
		height: 18px;
	}

	.mobile-topbar {
		display: none;
	}

	.mobile-backdrop {
		display: none;
	}

	@media (max-width: 868px) {
		.mobile-topbar {
			display: flex;
			align-items: center;
			justify-content: space-between;
			position: sticky;
			top: 0;
			left: 0;
			right: 0;
			height: 56px;
			padding: 0 16px;
			background: #ffffff;
		border-bottom: 1px solid var(--his-green);
			z-index: 35;
		}

		.mobile-menu-btn {
			width: 38px;
			height: 38px;
			border-radius: 8px;
			border: 1px solid var(--his-border);
			background: #ffffff;
			color: var(--his-muted);
			display: flex;
			align-items: center;
			justify-content: center;
		}

		.mobile-menu-btn svg {
			width: 20px;
			height: 20px;
		}

		.mobile-brand {
			display: flex;
			align-items: center;
			gap: 8px;
		}

		.mobile-logo-img {
			width: 26px;
			height: 26px;
			object-fit: contain;
		}

		.sidebar-container {
			position: fixed;
			left: 0;
			top: 0;
			bottom: 0;
			transform: translateX(-100%);
			box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
		}

		.sidebar-container.open-mobile {
			transform: translateX(0);
		}

		.mobile-backdrop {
			display: block;
			position: fixed;
			inset: 0;
			background: rgba(23, 23, 23, 0.4);
			backdrop-filter: blur(2px);
			z-index: 39;
		}
	}
</style>
