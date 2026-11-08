<script lang="ts">
	import { routeTitle } from './page-context';
	import RailNavigation from './RailNavigation.svelte';
	import NavigationFooter from './NavigationFooter.svelte';
	import { contentBounds } from './content-bounds';
	import { authenticatedRoute } from '#lib/routes.ts';
	import Snackbar from '../social/Snackbar.svelte';
	import { notify } from '../social/toasts.svelte';
	import { usePrompt, confirmExternalLink } from '../social/prompt';
	import {
		afterNavigate,
		beforeNavigate,
		goto,
		replaceState
	} from '$app/navigation';
	const ask = usePrompt();
	import BrowserPush from '../push/BrowserPush.svelte';
	import { disablePush, forgetPush } from '../push/client.ts';
	import { onMount, onDestroy, tick, untrack, type Snippet } from 'svelte';
	import { page, navigating } from '$app/state';
	import {
		CraftskyAPI,
		type OnboardingStatus,
		type EligibilityStatus,
		type SubscriptionAccess,
		type Profile
	} from '#lib/api/client.ts';
	import Icon from './Icon.svelte';
	import { contextMenu } from '../social/context-menu';
	import Avatar from '#lib/Avatar.svelte';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	import DiscoverySidebar from './DiscoverySidebar.svelte';
	import Onboarding from '../social/Onboarding.svelte';
	import { loadLanguagePreferences } from './languages';
	import { ownProfileRoute } from './navigation';
	import { profileStyle } from '../social/model';
	import { setContext } from 'svelte';
	import {
		NotificationBadge,
		notificationBadgeContext
	} from './notification-badge.svelte';
	import { PageHistory, pageHistoryContext } from './page-history';
	import {
		Publications,
		publicationsContext
	} from '../social/publications.svelte';
	import { reportFailure } from '../diagnostics/client.ts';
	let {
		session,
		accounts,
		children
	}: {
		session: { id: string; did: string; handle: string };
		accounts: { id: string; did: string; handle: string }[];
		children: Snippet<[CraftskyAPI]>;
	} = $props();
	const publications = new Publications(untrack(() => session.did));
	setContext(publicationsContext, publications);
	onDestroy(() => publications.dispose());
	setContext('craftsky-open-accounts', () => openAccounts());
	const pageHistory = new PageHistory();
	setContext(pageHistoryContext, pageHistory);
	beforeNavigate((navigation) => {
		if (
			navigation.to &&
			navigation.from &&
			navigation.to.url.href !== navigation.from.url.href
		)
			pageHistory.leave(
				navigationDepth,
				navigationDepth +
					(navigation.type === 'popstate' ? navigation.delta : 1),
				navigation.type === 'popstate',
				(/^\/feed(?:\/filters)?\/?$/.test(navigation.from.url.pathname) &&
					/^\/feed(?:\/filters)?\/?$/.test(navigation.to.url.pathname)) ||
					(navigation.from.url.pathname === '/search' &&
						navigation.to.url.pathname === '/search' &&
						navigation.from.url.searchParams.get('q') ===
							navigation.to.url.searchParams.get('q'))
			);
	});
	let navigationDepth = 0;
	let routeHistory: string[] = [];
	let returning: (() => void) | undefined;
	let replacing = false;
	afterNavigate((navigation) => {
		const current = page.url.href;
		if (navigation.type === 'enter') {
			navigationDepth = 0;
			routeHistory = [current];
		} else if (navigation.type === 'popstate')
			navigationDepth =
				page.state.navigationIndex ??
				Math.max(0, navigationDepth + navigation.delta);
		else if (navigation.from && !navigation.willUnload && !replacing) {
			navigationDepth++;
			routeHistory = routeHistory.slice(0, navigationDepth);
		}
		replaceState('', { ...page.state, navigationIndex: navigationDepth });
		routeHistory[navigationDepth] = current;
		pageHistory.arrive(navigationDepth, navigation.type === 'popstate');
		returning?.();
		returning = undefined;
	});
	setContext('craftsky-composer-origin', () => {
		const from = navigating.from?.url;
		const candidates = [
			from?.href,
			...routeHistory.slice(0, navigationDepth).reverse()
		];
		for (const value of candidates) {
			if (!value) continue;
			const url = new URL(value);
			if (
				url.origin === location.origin &&
				authenticatedRoute(url.pathname) &&
				!url.pathname.startsWith('/compose')
			)
				return url.pathname + url.search + url.hash;
		}
		return undefined;
	});
	setContext('craftsky-return', async (destination: string) => {
		// A route can render controls before its afterNavigate callback has run.
		// Settle that transition before choosing an entry to return to.
		if (navigating.to) await navigating.complete;
		const target = new URL(destination, location.href);
		const previous = routeHistory[navigationDepth - 1];
		if (
			previous &&
			new URL(previous).pathname === target.pathname &&
			(!target.search || new URL(previous).search === target.search)
		) {
			await new Promise<void>((resolve) => {
				returning = resolve;
				history.back();
			});
		} else {
			replacing = true;
			try {
				await goto(destination, { replaceState: true });
			} finally {
				replacing = false;
			}
		}
	});
	setContext('craftsky-back', async (event: MouseEvent) => {
		if (
			(navigationDepth > 0 || navigating.to) &&
			!event.metaKey &&
			!event.ctrlKey &&
			!event.shiftKey &&
			!event.altKey
		) {
			event.preventDefault();
			const fallback = (event.currentTarget as HTMLAnchorElement).href;
			if (navigating.to) await navigating.complete;
			if (navigationDepth > 0) history.back();
			else await goto(fallback);
		}
	});
	onMount(() => {
		const external = (event: MouseEvent) => {
			if (event.defaultPrevented || !(event.target instanceof Element)) return;
			const anchor = event.target.closest<HTMLAnchorElement>('a[href]');
			if (
				!anchor ||
				anchor.hasAttribute('download') ||
				anchor.hasAttribute('data-external-confirmed')
			)
				return;
			const url = new URL(anchor.href);
			if (
				['http:', 'https:', 'mailto:'].includes(url.protocol) &&
				url.origin !== location.origin
			)
				confirmExternalLink(event, anchor);
		};
		document.addEventListener('click', external);
		return () => document.removeEventListener('click', external);
	});
	let accountProfiles = $state<Record<string, Profile>>({});
	let accountAnchor = $state<{ left: number; top: number }>();
	function openAccounts(event?: MouseEvent) {
		const trigger = event?.currentTarget;
		if (trigger instanceof HTMLElement && window.innerWidth > 900) {
			const bounds =
				trigger.closest('.rail-row')?.getBoundingClientRect() ||
				trigger.getBoundingClientRect();
			accountAnchor = {
				left: Math.min(innerWidth - 288, bounds.right),
				top: bounds.top
			};
		} else accountAnchor = undefined;
		drawer = false;
		accountMenu = true;
	}
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
		{
			headers: { 'X-Craftsky-Account-Id': accountId },
			onRelationshipChanged: () => {
				if (!dead) pageHistory.invalidate('feed');
			}
		}
	);

	setContext('craftsky-api', api);
	let drawer = $state(false),
		accountMenu = $state(false),
		error = $state(''),
		newMessages = $state(0),
		onboarding = $state<OnboardingStatus>(),
		eligibility = $state<EligibilityStatus>(),
		ready = $state(false),
		access = $state<SubscriptionAccess>();
	const activityBadge = new NotificationBadge();
	setContext(notificationBadgeContext, () => {
		const finish = activityBadge.beginSeen();
		let settled = false;
		return (succeeded: boolean) => {
			if (settled) return;
			settled = true;
			finish(succeeded);
			void counts();
		};
	});
	let identity = $state<Profile>();
	setContext('craftsky-profile', () => identity);
	let postTypeMenu = $state<HTMLDialogElement>(null!);
	let firstPost = $state(true);
	let postTypeContext: HTMLDetailsElement;
	let postTypeTrigger: HTMLElement | undefined;
	let scrollY = $state(0);
	setContext('craftsky-create-post', choosePostType);
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
	function choosePostType(event: MouseEvent) {
		event.preventDefault();
		postTypeTrigger = event.currentTarget as HTMLElement;
		drawer = false;
		try {
			firstPost =
				localStorage.getItem('craftsky.post-type.seen.' + session.did) !==
				'true';
		} catch {
			firstPost = true;
		}
		if (firstPost) postTypeMenu.showModal();
		else postTypeContext.open = true;
	}
	function chosePostType() {
		try {
			localStorage.setItem('craftsky.post-type.seen.' + session.did, 'true');
		} catch {}
		postTypeMenu.close();
		postTypeContext.open = false;
	}
	const branchPaths = [
		'/feed',
		'/search',
		'/messages',
		'/notifications',
		'/saved',
		'/scheduled',
		'/drafts',
		'/settings'
	];
	let ownProfile = $derived(
		ownProfileRoute(
			page.url.pathname,
			session,
			identity?.handle || session.handle
		)
	);
	let savedChild = $derived(
		(page.url.pathname === '/saved' || page.url.pathname === '/saved/folder') &&
			(page.url.searchParams.has('folder') ||
				page.url.searchParams.get('unfiled') === 'true')
	);
	let shellRoot = $derived(
		(branchPaths.includes(page.url.pathname) && !savedChild) || ownProfile
	);
	let fullscreen = $derived(
		!ownProfile &&
			![
				'/feed',
				'/search',
				'/messages',
				'/messages/requests',
				'/messages/new',
				'/notifications',
				'/profile'
			].includes(page.url.pathname)
	);
	let notice = $state('');
	let dead = false;
	let timer: ReturnType<typeof setInterval>;
	setContext('craftsky-access', () => access);
	const nav = [
		['home', 'Home', '/feed'],
		['search', 'Search', '/search'],
		['messages', 'Messages', '/messages'],
		['activity', 'Activity', '/notifications'],
		['profile', 'Profile', '/profile'],
		['saved', 'Saved', '/saved'],
		['scheduled', 'Scheduled', '/scheduled'],
		['drafts', 'Drafts', '/drafts'],
		['settings', 'Settings', '/settings']
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
		} catch (cause) {
			void reportFailure(cause, 'startup');
			if (!dead)
				error = 'CraftSky could not load your account. Please try again.';
		}
	}
	async function counts() {
		if (dead || document.hidden || !ready) return;
		const activityRevision = activityBadge.capture();
		const results = await Promise.allSettled([
			api.notifications.count(),
			api.chat.unread()
		]);
		if (dead) return;
		if (results[0].status === 'fulfilled')
			activityBadge.accept(results[0].value.newCount, activityRevision);
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
			window.removeEventListener('craftsky-draft-saved', draftSaved);
			window.removeEventListener('craftsky-profile-updated', profileUpdated);
		};
	});
	onDestroy(() => {
		dead = true;
		pageHistory.dispose();
	});
	async function changeAccount(id: string) {
		if (id === accountId) {
			accountMenu = false;
			return;
		}
		accountMenu = false;
		try {
			const result = await fetch('/api/auth/switch', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ accountId: id })
			});
			if (!result.ok) throw new Error();
			location.assign('/feed');
		} catch {
			notify('The account could not be switched. Try again.', true);
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
			location.assign('/feed');
		} catch {
			notify('Could not sign out. Please try again.', true);
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
			? page.url.pathname.startsWith('/profile')
			: page.url.pathname.startsWith(
					'/' +
						(key === 'home'
							? 'feed'
							: key === 'activity'
								? 'notifications'
								: key)
				);
	}
	function reselect(event: MouseEvent, key: string) {
		drawer = false;
		if (
			active(key) &&
			key === 'home' &&
			page.url.pathname === '/feed' &&
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
		page.url.pathname === '/settings' ||
		page.url.pathname.startsWith('/settings/account') ||
		page.url.pathname.startsWith('/settings/moderation') ||
		page.url.pathname.startsWith('/settings/muted') ||
		page.url.pathname.startsWith('/settings/blocked') ||
		page.url.pathname.startsWith('/settings/about');
</script>

<svelte:head
	><link rel="manifest" href="/manifest.webmanifest" /><link
		rel="apple-touch-icon"
		href="/app-icon.png"
	/><meta name="theme-color" content="#f5efe4" /><title
		>{routeTitle(page.url.pathname)}</title
	><meta name="robots" content="noindex, nofollow" /></svelte:head
><svelte:document
	onvisibilitychange={() => {
		if (!document.hidden) void counts();
	}}
/>
<BrowserPush {api} {session} />
<svelte:window
	bind:scrollY
	onscroll={() => (profileScrolled = window.scrollY > 72)}
/>
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
	<RailNavigation {nav} {active} onSelect={reselect}>
		{#snippet icon(key)}
			{#if key === 'profile'}<Avatar author={identity || session} />{:else}<Icon
					name={key}
					filled={active(key)}
				/>{/if}
		{/snippet}
		{#snippet badge(key)}
			{#if key === 'activity' && activityBadge.count}<span class="count-badge"
					>{activityBadge.count > 99 ? '99+' : activityBadge.count}</span
				>
			{:else if key === 'messages' && newMessages}<span class="count-badge"
					>{newMessages > 99 ? '99+' : newMessages}</span
				>{/if}
		{/snippet}
		{#snippet trailing(key)}
			{#if key === 'profile'}<button
					class="icon-button rail-switch"
					aria-label={'Switch accounts, @' + session.handle}
					aria-expanded={accountMenu}
					onclick={openAccounts}><Icon name="switch" size={20} /></button
				>{/if}
		{/snippet}
	</RailNavigation>
	<button class="button" onclick={choosePostType} aria-label="Create post"
		><Icon name="plus" />New post</button
	>
	<NavigationFooter />
</aside>
{#if accountMenu}<button
		class="account-scrim"
		class:anchored={!!accountAnchor}
		aria-label="Close account switcher"
		onclick={() => (accountMenu = false)}
	></button>
	<div
		class="account-menu panel"
		class:anchored={!!accountAnchor}
		style:--account-left={accountAnchor ? `${accountAnchor.left}px` : undefined}
		style:--account-top={accountAnchor ? `${accountAnchor.top}px` : undefined}
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
			><Icon name="following" bold />Add account</a
		>
	</div>{/if}
{#if notice}<Snackbar message={notice} onDismiss={() => (notice = '')} />{/if}
<main
	id="main"
	use:contentBounds
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
			{#if !ready}<button onclick={() => openAccounts()}>Switch account</button
				><button onclick={logout}>Sign out</button>{/if}
		</div>{/if}{#if expired}<section class="panel stack">
			<h1>Your session has expired</h1>
			<p>Sign in to continue using this account.</p>
			<a class="button" href="/sign-in">Sign in</a><button
				onclick={() => openAccounts()}>Switch account</button
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
			<a href="/settings/moderation">Account standing</a><a
				href="/settings/account">Account settings</a
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
{#if ready && !expired && !accountMenu && !drawer && eligibility?.state === 'eligible' && onboarding?.completed && ((page.url.pathname === '/feed' && !page.url.searchParams.get('craft')) || ownProfile)}<button
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
			aria-describedby={key === 'activity' && activityBadge.count
				? 'bottom-activity-count'
				: key === 'messages' && newMessages
					? 'bottom-message-count'
					: undefined}
			oncontextmenu={(event) => {
				if (key === 'profile') {
					event.preventDefault();
					openAccounts();
				}
			}}
			onclick={(event) => reselect(event, key)}
			>{#if key === 'profile'}<Avatar
					author={identity || session}
				/>{:else}<Icon name={key} filled={active(key)} />{/if}<span
				class="visually-hidden">{label}</span
			>{#if key === 'activity' && activityBadge.count}<span
					class="bottom-count"
					id="bottom-activity-count"
					aria-label={`${activityBadge.count} new activities`}
					>{activityBadge.count > 99 ? '99+' : activityBadge.count}</span
				>{:else if key === 'messages' && newMessages}<span
					class="bottom-count"
					id="bottom-message-count"
					aria-label={`${newMessages} unread conversations`}
					>{newMessages > 99 ? '99+' : newMessages}</span
				>{/if}</a
		>{/each}
</nav>
{#if !gated && !drawer && !accountMenu && scrollY >= 200 && !page.url.pathname.includes('/compose') && !page.url.pathname.endsWith('/edit')}
	<button
		class="scroll-to-top"
		class:above-tabs={!fullscreen}
		aria-label="Scroll to top"
		onclick={() =>
			window.scrollTo({
				top: 0,
				behavior: matchMedia('(prefers-reduced-motion: reduce)').matches
					? 'instant'
					: 'smooth'
			})}><Icon name="moveUp" bold /></button
	>
{/if}
<details
	bind:this={postTypeContext}
	class="post-type-context"
	use:contextMenu={{ anchor: () => postTypeTrigger }}
>
	<summary class="visually-hidden" tabindex="-1">Create post</summary>
	<div class="menu-items">
		<a href="/compose" onclick={chosePostType}
			><Icon name="textPost" bold /><span
				><strong>Regular post</strong><small
					>Share a quick update, thought or question.</small
				></span
			></a
		>
		<a href="/compose/project" onclick={chosePostType}
			><Icon name="projectPost" bold /><span
				><strong>Project post</strong><small
					>Add photos and structured project details.</small
				></span
			></a
		>
	</div>
</details>
<dialog
	bind:this={postTypeMenu}
	class="post-type-chooser"
	aria-labelledby="post-type-title"
>
	<h2 id="post-type-title">What would you like to share?</h2>
	<a href="/compose" onclick={chosePostType}
		><Icon name="textPost" bold /><span
			><strong>Regular post</strong><small
				>Share a quick update, ask a question, or show what’s inspiring you.</small
			></span
		></a
	>
	<a href="/compose/project" onclick={chosePostType}
		><Icon name="projectPost" bold /><span
			><strong>Project post</strong><small
				>Document something you’re making, with details like the craft, pattern,
				materials, and progress.</small
			></span
		></a
	>
	<div class="post-type-cancel">
		<button onclick={() => postTypeMenu.close()}>Cancel</button>
	</div>
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
	.account-menu:not(.anchored) {
		top: auto;
		bottom: 0;
		left: 50%;
		transform: translateX(-50%);
		width: min(640px, 100vw);
		max-height: 80dvh;
		border: 0;
		border-radius: 22px 22px 0 0;
		padding: 56px 0 max(24px, env(safe-area-inset-bottom));
	}
	.account-menu:not(.anchored)::before {
		content: '';
		position: absolute;
		top: 20px;
		left: calc(50% - 16px);
		width: 32px;
		height: 4px;
		border-radius: 999px;
		background: var(--ink-2);
	}
	.account-menu.anchored {
		left: var(--account-left);
		top: var(--account-top);
		transform: none;
		width: 280px;
		padding: 8px 0;
		border-radius: 14px;
		background: var(--surface);
	}
	.account-scrim.anchored {
		background: transparent;
	}
	.account-menu.anchored .person-row {
		padding: 8px 16px;
		min-height: 56px;
	}
	.account-menu.anchored .add-account {
		min-height: 56px;
		padding: 8px 16px;
		font-size: 16px;
		gap: 16px;
	}
	.account-menu.anchored .person-row[aria-current] {
		color: var(--ink-4);
	}

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

	.app-topbar.on-profile:not(.profile-scrolled) :global(.icon-button) {
		color: var(--profile-foreground);
	}
	.app-topbar.on-profile.profile-scrolled {
		background: var(--paper);
	}
	.scroll-to-top {
		position: fixed;
		z-index: 20;
		left: 272px;
		bottom: 16px;
		width: 44px;
		height: 44px;
		min-width: 44px;
		padding: 8px;
		border: 1.5px solid var(--ink);
		border-radius: 50%;
		background: light-dark(#ededed, #24201d);
		color: var(--ink-2);
		box-shadow: 0 3px 5px #0003;
	}
	.scroll-to-top:hover {
		background: color-mix(
			in srgb,
			var(--primary) 8%,
			light-dark(#ededed, #24201d)
		);
		box-shadow: 0 4px 6px #0003;
	}
	.scroll-to-top:active {
		box-shadow: none;
	}
	@media (max-width: 900px) {
		.scroll-to-top {
			left: 16px;
			bottom: calc(16px + env(safe-area-inset-bottom));
		}
		.scroll-to-top.above-tabs {
			bottom: calc(80px + env(safe-area-inset-bottom));
		}
	}
	.post-type-context .menu-items a {
		min-height: 72px;
	}
	.post-type-context strong,
	.post-type-context small {
		display: block;
	}
	.post-type-context strong {
		font-size: 14px;
		line-height: 1.3;
	}
	.post-type-context small {
		font-size: 13px;
		line-height: 1.45;
		color: var(--ink-2);
		font-weight: 400;
		margin-top: 4px;
	}
	.post-type-chooser {
		width: min(360px, calc(100vw - 48px));
		padding: 24px;
		background: var(--surface);
	}
	.post-type-chooser h2 {
		font-size: 20px;
		line-height: 1.25;
		margin: 0 0 16px;
	}
	.post-type-chooser > a {
		display: flex;
		gap: 12px;
		align-items: center;
		color: var(--ink);
		text-decoration: none;
		padding: 16px;
		border: 1px solid var(--ink);
		border-radius: 12px;
		margin-top: 16px;
	}
	.post-type-chooser > a:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
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
	.post-type-cancel {
		display: flex;
		justify-content: flex-end;
		margin-top: 24px;
	}
	.post-type-cancel button {
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--primary);
	}
</style>
