<script lang="ts">
	import { usePrompt } from '../social/prompt';
	const ask = usePrompt();
	import BrowserPush from '../push/BrowserPush.svelte';
	import { disablePush, forgetPush } from '../push/client.ts';
	import { onMount, onDestroy, tick, untrack, type Snippet } from 'svelte';
	import { page } from '$app/state';
	import {
		CraftskyAPI,
		type OnboardingStatus,
		type EligibilityStatus,
		type SubscriptionAccess,
		type Profile
	} from '#lib/api/client.ts';
	import Icon from './Icon.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	import DiscoverySidebar from './DiscoverySidebar.svelte';
	import Onboarding from '../social/Onboarding.svelte';
	import { loadLanguagePreferences } from './languages';
	import { ownProfileRoute } from './navigation';
	import { profileStyle } from '../social/model';
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
	const version = __CRAFTSKY_VERSION__.trim();
	let accountProfiles = $state<Record<string, Profile>>({});
	$effect(() => {
		if (!accountMenu) return;
		let current = true;
		untrack(() => {
			for (const account of accounts) {
				if (account.id === accountId || accountProfiles[account.did]) continue;
				void api.profiles
					.get(account.did)
					.then((profile) => {
						if (current) accountProfiles[account.did] = profile;
					})
					.catch(() => {});
			}
		});
		return () => {
			current = false;
		};
	});
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
	let identity = $state<Profile>();
	let postTypeMenu = $state<HTMLDialogElement>(null!);
	let firstPost = $state(true);
	let comingSoonFeature = $state('');
	let comingSoonDialog: HTMLDialogElement;
	let profileScrolled = $state(false);
	let desktop = $state(false);
	let gated = $derived(
		!ready ||
			expired ||
			(eligibility?.state === 'eligible' && onboarding?.completed === false)
	);
	let discovery = $derived(
		desktop && !gated && eligibility?.state === 'eligible'
	);
	function choosePostType() {
		drawer = false;
		try {
			firstPost =
				localStorage.getItem('craftsky.post-type.seen.' + session.did) !==
				'true';
		} catch {
			firstPost = true;
		}
		postTypeMenu.showModal();
	}
	function chosePostType() {
		try {
			localStorage.setItem('craftsky.post-type.seen.' + session.did, 'true');
		} catch {}
		postTypeMenu.close();
	}
	const branchPaths = [
		'/app/home',
		'/app/search',
		'/app/messages',
		'/app/activity',
		'/app/saved',
		'/app/scheduled',
		'/app/drafts',
		'/app/settings'
	];
	let ownProfile = $derived(
		ownProfileRoute(
			page.url.pathname,
			session,
			identity?.handle || session.handle
		)
	);
	let savedChild = $derived(
		page.url.pathname === '/app/saved' &&
			(page.url.searchParams.has('folder') ||
				page.url.searchParams.get('unfiled') === 'true')
	);
	let shellRoot = $derived(
		(branchPaths.includes(page.url.pathname) && !savedChild) || ownProfile
	);
	let fullscreen = $derived(
		!ownProfile &&
			![
				'/app/home',
				'/app/search',
				'/app/messages',
				'/app/messages/requests',
				'/app/messages/new',
				'/app/activity',
				'/app/profile/' + encodeURIComponent(session.did)
			].includes(page.url.pathname)
	);
	let notice = $state('');
	let noticeTimer: ReturnType<typeof setTimeout> | undefined;
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
				const [status, tier, , profile] = await Promise.all([
					api.account.onboarding(),
					api.billing.access().catch(() => undefined),
					loadLanguagePreferences(
						api,
						navigator.languages || [navigator.language]
					),
					api.profiles.me().catch(() => undefined)
				]);
				if (dead) return;
				onboarding = status;
				access = tier;
				identity = profile;
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
		const showComingSoon = (event: Event) => {
			const feature = (event as CustomEvent<unknown>).detail;
			if (typeof feature !== 'string' || !feature || gated) return;
			comingSoonFeature = feature;
			comingSoonDialog.showModal();
		};
		window.addEventListener('craftsky-coming-soon', showComingSoon);
		const desktopQuery = window.matchMedia('(min-width: 1201px)');
		const updateDesktop = () => (desktop = desktopQuery.matches);
		updateDesktop();
		desktopQuery.addEventListener('change', updateDesktop);
		try {
			const theme = localStorage.getItem('craftsky.theme');
			if (theme === 'light' || theme === 'dark')
				document.documentElement.dataset.theme = theme;
		} catch {}
		void boot();
		timer = setInterval(counts, 30000);
		const draftSaved = (event: Event) => {
			const detail = (event as CustomEvent<unknown>).detail;
			if (dead || expired || !ready || !detail || typeof detail !== 'object')
				return;
			const value = detail as Record<string, unknown>;
			if (value.accountId !== session.id || value.did !== session.did) return;
			notice = 'Draft saved';
			clearTimeout(noticeTimer);
			noticeTimer = setTimeout(() => {
				notice = '';
			}, 4000);
		};
		let profileRevision = 0;
		const profileUpdated = async (event: Event) => {
			const detail = (event as CustomEvent<unknown>).detail;
			if (dead || expired || !ready || !detail || typeof detail !== 'object')
				return;
			const value = detail as Record<string, unknown>;
			if (
				value.accountId !== accountId ||
				value.accountId !== session.id ||
				value.did !== session.did
			)
				return;
			const revision = ++profileRevision,
				currentDid = session.did;
			try {
				const profile = await api.profiles.me();
				if (
					!dead &&
					!expired &&
					revision === profileRevision &&
					session.id === accountId &&
					session.did === currentDid &&
					profile.did === currentDid
				)
					identity = profile;
			} catch {
				// The save already succeeded; retain the last known identity until its next refresh.
			}
		};
		window.addEventListener('craftsky-draft-saved', draftSaved);
		window.addEventListener('craftsky-profile-updated', profileUpdated);
		return () => {
			window.removeEventListener('craftsky-coming-soon', showComingSoon);
			desktopQuery.removeEventListener('change', updateDesktop);
			profileRevision++;
			clearInterval(timer);
			clearTimeout(noticeTimer);
			window.removeEventListener('craftsky-draft-saved', draftSaved);
			window.removeEventListener('craftsky-profile-updated', profileUpdated);
		};
	});
	onDestroy(() => {
		dead = true;
	});
	async function changeAccount(id: string) {
		if (id === accountId) {
			accountMenu = false;
			return;
		}
		if (
			!(await ask({
				title: 'Switch accounts?',
				detail: 'Save any unfinished work first.',
				confirmLabel: 'Switch account'
			}))
		)
			return;
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
		if (
			!(await ask({
				title: 'Sign out?',
				detail: 'Sign out of this account?',
				confirmLabel: 'Sign out'
			}))
		)
			return;
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
		if (!drawer && !accountMenu) return;
		const previous = document.activeElement as HTMLElement | null;
		const overflow = document.documentElement.style.overflow;
		document.documentElement.style.overflow = 'hidden';
		void tick().then(() => {
			if (drawer || accountMenu)
				document
					.querySelector<HTMLElement>(
						accountMenu ? '.account-menu a[href]' : '.app-sidebar.open a[href]'
					)
					?.focus();
		});
		const trap = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				event.preventDefault();
				drawer = false;
				accountMenu = false;
				return;
			}
			if (event.key !== 'Tab') return;
			const controls = [
				...document.querySelectorAll<HTMLElement>(
					accountMenu
						? '.account-menu a[href],.account-menu button:not([disabled])'
						: '.app-sidebar.open a[href],.app-sidebar.open button:not([disabled])'
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
			window.scrollTo({
				top: 0,
				behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches
					? 'instant'
					: 'smooth'
			});
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
<svelte:window onscroll={() => (profileScrolled = window.scrollY > 72)} />
<header
	class="app-topbar"
	class:on-profile={ownProfile}
	class:profile-scrolled={profileScrolled}
	style={ownProfile ? profileStyle(identity?.customisation) : undefined}
	class:visible={shellRoot && !gated}
	inert={drawer || accountMenu}
>
	<button
		class="icon-button mobile-only"
		aria-label="Open navigation"
		aria-expanded={drawer}
		onclick={() => (drawer = !drawer)}><Icon name="menu" /></button
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
	inert={accountMenu}
	hidden={gated}
>
	<nav aria-label="App navigation">
		{#each nav as [key, label, href]}<div class="rail-row">
				<a
					{href}
					aria-current={active(key) ? 'page' : undefined}
					onclick={(event) => reselect(event, key)}
					><span class="rail-icon"
						><Icon name={key} filled={active(key)} /></span
					><span>{label === 'Activity' ? 'Notifications' : label}</span
					>{#if key === 'activity' && newActivity}<span class="count-badge"
							>{newActivity > 99 ? '99+' : newActivity}</span
						>{:else if key === 'messages' && newMessages}<span
							class="count-badge">{newMessages > 99 ? '99+' : newMessages}</span
						>{/if}</a
				>{#if key === 'profile'}<button
						class="icon-button rail-switch"
						aria-label={'Switch accounts, @' + session.handle}
						aria-expanded={accountMenu}
						onclick={() => {
							drawer = false;
							accountMenu = true;
						}}><Icon name="switch" size={20} /></button
					>{/if}
			</div>{/each}
	</nav>
	<button class="button" onclick={choosePostType} aria-label="Create post"
		><Icon name="plus" />New post</button
	>
	<div class="sidebar-about">
		<a href="https://craftsky.social/terms" target="_blank" rel="noreferrer"
			>Terms</a
		><a href="https://craftsky.social/privacy" target="_blank" rel="noreferrer"
			>Privacy</a
		><a
			href="https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/3mpr5izppvt2k?lang=en"
			target="_blank"
			rel="noreferrer"
			class="sidebar-feedback"><Icon name="messages" size={20} />Feedback</a
		>
		{#if version}<small>{version}</small>{/if}
	</div>
</aside>
{#if accountMenu}<button
		class="account-scrim"
		aria-label="Close account switcher"
		onclick={() => (accountMenu = false)}
	></button>
	<div
		class="account-menu panel"
		role="dialog"
		aria-modal="true"
		aria-label="Accounts"
		tabindex="-1"
	>
		{#each accounts as account}{@const profile =
				account.id === accountId ? identity : accountProfiles[account.did]}
			<button
				class="person-row"
				disabled={account.id === accountId}
				aria-current={account.id === accountId ? 'true' : undefined}
				onclick={() => changeAccount(account.id)}
			>
				<Avatar
					author={{
						did: account.did,
						handle: account.handle,
						displayName: profile?.displayName || undefined,
						avatar: profile?.avatar || undefined
					}}
				/>
				<div class="shell-identity">
					<CompactIdentity
						author={{
							did: account.did,
							handle: account.handle,
							displayName: profile?.displayName || undefined,
							crafts: profile?.crafts || []
						}}
					/>
				</div>
				{#if account.id === accountId}<Icon name="check" />{/if}
			</button>
		{/each}
		<a class="add-account" href="/sign-in?add=1"
			><Icon name="userPlus" />Add account</a
		>
	</div>{/if}
{#if notice}<div class="shell-notice" role="status">{notice}</div>{/if}
<main
	id="main"
	class="app-main"
	class:has-discovery={discovery}
	class:is-gated={gated}
	class:shell-root={shellRoot}
	class:fullscreen
	inert={drawer || accountMenu}
>
	{#if error && !expired}<div class="notice" role="alert">
			{#if !ready}<h1>Account unavailable</h1>{/if}
			{error}<button onclick={boot}>Retry</button>
			{#if !ready}<button onclick={() => (accountMenu = true)}
					>Switch account</button
				><button onclick={logout}>Sign out</button>{/if}
		</div>{/if}{#if expired}<section class="panel stack">
			<h1>Your session has expired</h1>
			<p>Sign in to continue using this account.</p>
			<a class="button" href="/sign-in">Sign in</a><button
				onclick={() => (accountMenu = true)}>Switch account</button
			><button onclick={logout}>Sign out</button>
		</section>{:else if !ready}{#if !error}<p class="empty" role="status">
				Loading your CraftSky…
			</p>{/if}{:else if eligibility?.state === 'restricted' && !allowedRestricted()}<section
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
{#if discovery}<aside
		class="app-discovery"
		aria-label="Discover CraftSky"
		inert={drawer || accountMenu}
	>
		<DiscoverySidebar
			{api}
			profile={identity}
			onProfile={(profile) => (identity = profile)}
		/>
	</aside>{/if}
{#if ready && !expired && !accountMenu && !drawer && eligibility?.state === 'eligible' && onboarding?.completed && ((page.url.pathname === '/app/home' && !page.url.searchParams.get('craft')) || ownProfile)}<button
		class="button compose-fab"
		onclick={choosePostType}
		aria-label="Create post"><Icon name="plus" />New post</button
	>{/if}
<nav
	class="bottom-nav"
	class:fullscreen={fullscreen || gated}
	aria-label="Main tabs"
	inert={drawer || accountMenu}
>
	{#each nav.slice(0, 5) as [key, label, href]}<a
			{href}
			aria-current={active(key) ? 'page' : undefined}
			aria-label={label}
			aria-describedby={key === 'activity' && newActivity
				? 'bottom-activity-count'
				: key === 'messages' && newMessages
					? 'bottom-message-count'
					: undefined}
			oncontextmenu={(event) => {
				if (key === 'profile') {
					event.preventDefault();
					accountMenu = true;
				}
			}}
			onclick={(event) => reselect(event, key)}
			>{#if key === 'profile'}<Avatar
					author={identity || session}
				/>{:else}<Icon name={key} filled={active(key)} />{/if}<span
				class="visually-hidden">{label}</span
			>{#if key === 'activity' && newActivity}<span
					class="bottom-count"
					id="bottom-activity-count"
					aria-label={`${newActivity} new activities`}
					>{newActivity > 99 ? '99+' : newActivity}</span
				>{:else if key === 'messages' && newMessages}<span
					class="bottom-count"
					id="bottom-message-count"
					aria-label={`${newMessages} unread conversations`}
					>{newMessages > 99 ? '99+' : newMessages}</span
				>{/if}</a
		>{/each}
</nav>
<dialog
	bind:this={postTypeMenu}
	class="post-type-chooser"
	aria-labelledby="post-type-title"
>
	<header>
		<h2 id="post-type-title">
			{firstPost ? 'What would you like to share?' : 'Create post'}
		</h2>
		<button
			class="icon-button"
			aria-label="Close post type chooser"
			onclick={() => postTypeMenu.close()}><Icon name="close" /></button
		>
	</header>
	<a href="/app/compose" onclick={chosePostType}
		><Icon name="document" /><span
			><strong>Regular post</strong><small
				>Share a quick update, thought or question.</small
			></span
		><Icon name="chevron" size={20} /></a
	>
	<a href="/app/compose/project" onclick={chosePostType}
		><Icon name="image" /><span
			><strong>Project post</strong><small
				>Add photos and structured project details.</small
			></span
		><Icon name="chevron" size={20} /></a
	>
</dialog>
<dialog
	bind:this={comingSoonDialog}
	class="coming-soon-dialog"
	aria-labelledby="coming-soon-title"
>
	<h2 id="coming-soon-title">{comingSoonFeature} is coming soon</h2>
	<p>
		{comingSoonFeature} isn't available yet. We'll let you know when it launches.
	</p>
	<div class="actions">
		<button onclick={() => comingSoonDialog.close()}>OK</button>
	</div>
</dialog>

<style>
	.coming-soon-dialog {
		width: min(360px, calc(100vw - 48px));
		border-radius: 14px;
		box-shadow: 10px 10px var(--shadow);
	}
	.coming-soon-dialog h2 {
		font-size: 20px;
	}
	.coming-soon-dialog .actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 16px;
	}
	.shell-notice {
		position: fixed;
		z-index: 100;
		bottom: calc(88px + env(safe-area-inset-bottom));
		left: 50%;
		transform: translateX(-50%);
		width: min(560px, calc(100vw - 32px));
		max-width: calc(100vw - 32px);
		padding: 14px 16px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		background: var(--surface);
		color: var(--ink);
		box-shadow: 0 3px 8px #0003;
		font-size: 14px;
	}

	.app-topbar.on-profile:not(.profile-scrolled) :global(.icon-button) {
		color: var(--profile-foreground);
	}
	.app-topbar.on-profile.profile-scrolled {
		background: var(--paper);
	}
	.post-type-chooser {
		width: min(440px, calc(100vw - 32px));
		border-radius: 22px;
		background: var(--paper);
		padding: 16px;
	}
	.post-type-chooser header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		margin-bottom: 12px;
	}
	.post-type-chooser h2 {
		font-size: 20px;
		line-height: 1.25;
		margin: 0;
	}
	.post-type-chooser > a {
		display: flex;
		gap: 16px;
		align-items: center;
		color: var(--ink);
		text-decoration: none;
		padding: 16px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		background: var(--surface);
		margin-top: 12px;
	}
	.post-type-chooser > a:hover {
		background: var(--primary-soft);
	}
	.post-type-chooser > a > span {
		flex: 1;
		min-width: 0;
	}
	.post-type-chooser strong {
		display: block;
		font-size: 16px;
		font-weight: 600;
	}
	.post-type-chooser small {
		display: block;
		font-size: 14px;
		line-height: 1.5;
		color: var(--ink-2);
		margin-top: 4px;
	}
</style>
