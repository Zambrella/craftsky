<script lang="ts">
	import BrowserPush from '../push/BrowserPush.svelte';
	import { disablePush, forgetPush } from '../push/client.ts';
	import { onMount, onDestroy, tick, untrack, type Snippet } from 'svelte';
	import { page } from '$app/state';
	import {
		CraftskyAPI,
		type OnboardingStatus,
		type EligibilityStatus,
		type SubscriptionAccess
	} from '#lib/api/client.ts';
	import Icon from './Icon.svelte';
	import Onboarding from '../social/Onboarding.svelte';
	import { setContext } from 'svelte';
	let {
		session,
		accounts,
		children
	}: {
		session: { id: string; did: string; handle: string };
		accounts: { id: string; did: string; handle: string }[];
		children: Snippet<[CraftskyAPI]>;
	} = $props();
	const accountId = untrack(() => session.id);
	let expired = $state(false);
	const api = new CraftskyAPI(
		async (input, init) => {
			const response = await fetch(input, init);
			if (response.status === 401) {
				expired = true;
				return response;
			}
			return response;
		},
		'/api/appview',
		{ headers: { 'X-Craftsky-Account-Id': accountId } }
	);

	let drawer = $state(false),
		accountMenu = $state(false),
		error = $state(''),
		newActivity = $state(0),
		newMessages = $state(0),
		onboarding = $state<OnboardingStatus>(),
		eligibility = $state<EligibilityStatus>(),
		ready = $state(false),
		access = $state<SubscriptionAccess>();
	let dead = false;
	let timer: ReturnType<typeof setInterval>;
	setContext('craftsky-access', () => access);
	const nav = [
		['home', 'Home', '/app/home'],
		['search', 'Search', '/app/search'],
		['messages', 'Messages', '/app/messages'],
		['activity', 'Activity', '/app/activity'],
		[
			'profile',
			'Profile',
			'/app/profile/' + encodeURIComponent(untrack(() => session.did))
		],
		['saved', 'Saved', '/app/saved'],
		['scheduled', 'Scheduled', '/app/scheduled'],
		['drafts', 'Drafts', '/app/drafts'],
		['settings', 'Settings', '/app/settings']
	];
	async function boot() {
		error = '';
		try {
			const policy = await api.account.eligibility();
			if (dead) return;
			eligibility = policy;
			if (policy.state === 'eligible') {
				const [status, tier] = await Promise.all([
					api.account.onboarding(),
					api.billing.access().catch(() => undefined)
				]);
				if (dead) return;
				onboarding = status;
				access = tier;
			}
			ready = true;
			void counts();
		} catch {
			if (!dead)
				error = 'CraftSky could not load your account. Please try again.';
		}
	}
	async function counts() {
		if (dead || document.hidden || !ready) return;
		const results = await Promise.allSettled([
			api.notifications.count(),
			api.chat.unread()
		]);
		if (dead) return;
		if (results[0].status === 'fulfilled')
			newActivity = results[0].value.newCount;
		if (results[1].status === 'fulfilled')
			newMessages = results[1].value.accepted;
	}
	onMount(() => {
		try {
			const theme = localStorage.getItem('craftsky.theme');
			if (theme === 'light' || theme === 'dark')
				document.documentElement.dataset.theme = theme;
		} catch {}
		void boot();
		timer = setInterval(counts, 30000);
		return () => clearInterval(timer);
	});
	onDestroy(() => {
		dead = true;
	});
	async function changeAccount(id: string) {
		if (id === accountId) return;
		if (!confirm('Switch accounts? Save any unfinished work first.')) return;
		try {
			const result = await fetch('/api/auth/switch', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ accountId: id })
			});
			if (!result.ok) throw new Error();
			location.assign('/app/home');
		} catch {
			error = 'The account could not be switched. Try again.';
		}
	}
	async function logout() {
		if (!confirm('Sign out of this account?')) return;
		try {
			await disablePush(api, accountId).catch(() => {});
			const response = await fetch('/api/auth/logout', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ accountId })
			});
			if (!response.ok) throw new Error();
			await forgetPush(accountId).catch(() => {});
			try {
				const privatePrefix = `craftsky.chat.v1:${encodeURIComponent(accountId + ':' + session.did)}:`;
				for (const key of Object.keys(localStorage))
					if (key.startsWith(privatePrefix)) localStorage.removeItem(key);
			} catch {}
			location.assign('/app/home');
		} catch {
			error = 'Could not sign out. Please try again.';
		}
	}
	$effect(() => {
		if (!drawer) return;
		const previous = document.activeElement as HTMLElement | null;
		const overflow = document.documentElement.style.overflow;
		document.documentElement.style.overflow = 'hidden';
		void tick().then(() => {
			if (drawer)
				document
					.querySelector<HTMLElement>('.app-sidebar.open button')
					?.focus();
		});
		const trap = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				event.preventDefault();
				drawer = false;
				return;
			}
			if (event.key !== 'Tab') return;
			const controls = [
				...document.querySelectorAll<HTMLElement>(
					'.app-sidebar.open a[href],.app-sidebar.open button:not([disabled])'
				)
			].filter((item) => item.getClientRects().length);
			const first = controls[0],
				last = controls.at(-1);
			if (event.shiftKey && document.activeElement === first) {
				event.preventDefault();
				last?.focus();
			} else if (!event.shiftKey && document.activeElement === last) {
				event.preventDefault();
				first?.focus();
			}
		};
		window.addEventListener('keydown', trap);
		return () => {
			window.removeEventListener('keydown', trap);
			document.documentElement.style.overflow = overflow;
			previous?.focus();
		};
	});

	function active(key: string) {
		return key === 'profile'
			? page.url.pathname.startsWith('/app/profile')
			: page.url.pathname.startsWith('/app/' + key);
	}
	function reselect(event: MouseEvent, key: string) {
		drawer = false;
		if (
			active(key) &&
			key === 'home' &&
			page.url.pathname === '/app/home' &&
			!page.url.search
		) {
			event.preventDefault();
			window.scrollTo({ top: 0, behavior: 'smooth' });
			window.dispatchEvent(new Event('craftsky-refresh'));
		}
	}
	const allowedRestricted = () =>
		page.url.pathname === '/app/settings' ||
		page.url.pathname.startsWith('/app/settings/account') ||
		page.url.pathname.startsWith('/app/settings/moderation') ||
		page.url.pathname.startsWith('/app/settings/muted') ||
		page.url.pathname.startsWith('/app/settings/blocked') ||
		page.url.pathname.startsWith('/app/settings/about');
</script>

<svelte:head
	><link rel="manifest" href="/manifest.webmanifest" /><link
		rel="apple-touch-icon"
		href="/app-icon.png"
	/><meta name="theme-color" content="#f5efe4" /><title>CraftSky</title><meta
		name="robots"
		content="noindex, nofollow"
	/></svelte:head
><svelte:document
	onvisibilitychange={() => {
		if (!document.hidden) void counts();
	}}
/>
<BrowserPush {api} {session} />
<header class="app-topbar" inert={drawer}>
	<button
		class="icon-button mobile-only"
		aria-label="Open navigation"
		aria-expanded={drawer}
		onclick={() => (drawer = !drawer)}><Icon name="menu" /></button
	><a class="brand" href="/app/home"
		><img src="/favicon.svg" width="32" height="32" alt="" />CraftSky</a
	><button
		class="account-trigger"
		aria-expanded={accountMenu}
		onclick={() => (accountMenu = !accountMenu)}
		>@{session.handle}<span aria-hidden="true">⌄</span></button
	>
</header>
{#if drawer}<button
		class="drawer-scrim"
		aria-label="Close navigation"
		onclick={() => (drawer = false)}
	></button>{/if}
<aside
	class="app-sidebar"
	class:open={drawer}
	role={drawer ? 'dialog' : undefined}
	aria-modal={drawer ? true : undefined}
	aria-label="Navigation"
>
	<button
		class="icon-button mobile-only"
		aria-label="Close navigation"
		onclick={() => (drawer = false)}>Close</button
	>
	<nav aria-label="App navigation">
		{#each nav as [key, label, href]}<a
				{href}
				aria-current={active(key) ? 'page' : undefined}
				onclick={(event) => reselect(event, key)}
				><Icon name={key} /><span>{label}</span
				>{#if key === 'activity' && newActivity}<span class="count-badge"
						>{newActivity > 99 ? '99+' : newActivity}</span
					>{:else if key === 'messages' && newMessages}<span class="count-badge"
						>{newMessages > 99 ? '99+' : newMessages}</span
					>{/if}</a
			>{/each}
	</nav>
	<a class="button" href="/app/compose"><Icon name="plus" />Create post</a><a
		class="sidebar-about"
		href="https://craftsky.social"
		target="_blank"
		rel="noreferrer">Your craft. Your community.</a
	>
</aside>
{#if accountMenu}<section class="account-menu panel" aria-label="Accounts">
		<h2>Your accounts</h2>
		{#each accounts as account}<button
				class="person-row"
				aria-current={account.id === accountId ? 'true' : undefined}
				onclick={() => changeAccount(account.id)}
				>@{account.handle}{#if account.id === accountId}<span
						>Current account</span
					>{/if}</button
			>{/each}<a href="/sign-in?add=1">Add another account</a><button
			onclick={logout}>Sign out @{session.handle}</button
		><button onclick={() => (accountMenu = false)}>Close</button>
	</section>{/if}
<main id="main" class="app-main" inert={drawer}>
	{#if error}<div class="notice" role="alert">
			{error}<button onclick={boot}>Retry</button>
		</div>{/if}{#if expired}<section class="panel stack">
			<h1>Your session has expired</h1>
			<p>Sign in to continue using this account.</p>
			<a class="button" href="/sign-in">Sign in</a><a href="/app/home"
				>Use another account</a
			>
		</section>{:else if !ready}<p class="empty" role="status">
			Loading your CraftSky…
		</p>{:else if eligibility?.state === 'restricted' && !allowedRestricted()}<section
			class="panel stack"
		>
			<h1>Your account is restricted</h1>
			<p>
				{eligibility.appealGuidance ||
					'Review your account standing to learn more.'}
			</p>
			<a href="/app/settings/moderation">Account standing</a><a
				href="/app/settings/account">Account settings</a
			>
		</section>{:else if onboarding && !onboarding.completed && eligibility?.state === 'eligible'}<Onboarding
			{api}
			{session}
			status={onboarding}
			onComplete={() => {
				onboarding = { ...onboarding!, completed: true };
			}}
		/>{:else}{@render children(api)}{/if}
</main>
<nav class="bottom-nav" aria-label="Main tabs" inert={drawer}>
	{#each nav.slice(0, 5) as [key, label, href]}<a
			{href}
			aria-current={active(key) ? 'page' : undefined}
			onclick={(event) => reselect(event, key)}
			><Icon name={key} /><span>{label}</span
			>{#if key === 'activity' && newActivity}<span class="dot"
				></span>{:else if key === 'messages' && newMessages}<span class="dot"
				></span>{/if}</a
		>{/each}
</nav>
