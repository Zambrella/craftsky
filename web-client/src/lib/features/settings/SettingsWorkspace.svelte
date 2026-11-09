<script lang="ts">
	import { UserFacingError } from '#lib/user-facing-error.ts';
	import { usePrompt } from '../social/prompt';
	const ask = usePrompt();
	import type {
		CraftskyAPI,
		Profile,
		SubscriptionAccess
	} from '#lib/api/client.ts';
	import { getContext, onDestroy, untrack } from 'svelte';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import { colours } from './model.ts';
	import { disablePush, forgetPush } from '../push/client.ts';
	import { clearAccountChat } from './account-cleanup.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import type { Session } from './model.ts';
	import { visibleSettingsGroups } from './presentation.ts';
	import SettingsIcon from './SettingsIcon.svelte';
	import Preferences from './Preferences.svelte';
	import Connections from './Connections.svelte';
	import Account from './Account.svelte';
	import Billing from './Billing.svelte';
	import Instagram from './Instagram.svelte';
	import Business from './Business.svelte';
	import './settings.css';
	import { subscriptionsEnabled, comingSoon } from '../shell/release-config';
	let {
		api,
		path,
		session
	}: { api: CraftskyAPI; path: string[]; session: Session } = $props();
	let parts = $derived(
		path.includes('settings') ? path.slice(path.indexOf('settings') + 1) : path
	);
	let section = $derived(parts[0] || '');
	const menu = new Work('settings.SettingsWorkspace.action');
	let restricted = $state(false),
		profile = $state<Profile>(),
		access = $state<SubscriptionAccess>();
	let theme = $state('Use device setting');
	const openAccounts = getContext<() => void>('craftsky-open-accounts');
	let paid = $derived(!!access?.givesAccess && access.effectiveTier !== 'free');
	function loadMenu() {
		void menu.load(
			async () => {
				const policy = await api.account.eligibility();
				const [me, tier] =
					policy.state === 'eligible'
						? await Promise.all([
								api.profiles.me().catch(() => undefined),
								api.billing.access().catch(() => undefined)
							])
						: [undefined, undefined];
				return { policy, me, tier };
			},
			(value) => {
				restricted = value.policy.state === 'restricted';
				profile = value.me;
				access = value.tier;
				try {
					const mode = localStorage.getItem('craftsky.theme');
					theme =
						mode === 'dark'
							? 'Dark'
							: mode === 'light'
								? 'Light'
								: 'Use device setting';
				} catch {}
			},
			'settings.SettingsWorkspace.loadMenu'
		);
	}
	$effect(() => {
		if (!section) untrack(loadMenu);
	});
	onDestroy(() => menu.dispose());
	let shownGroups = $derived(
		visibleSettingsGroups(
			restricted,
			subscriptionsEnabled() &&
				!!access?.givesAccess &&
				access.effectiveTier === 'business'
		)
	);
	async function signOut() {
		if (
			!(await ask({
				title: 'Sign out?',
				detail: 'Sign out of this account?',
				confirmLabel: 'Sign out'
			}))
		)
			return;
		void menu.run(
			async () => {
				if (!session.id)
					throw new UserFacingError('Reopen Settings to sign out.');
				await disablePush(api, session.id).catch(() => {});
				const response = await fetch('/api/auth/logout', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ accountId: session.id })
				});
				if (!response.ok)
					throw new UserFacingError('Could not sign out. Try again.');
				await forgetPush(session.id).catch(() => {});
				try {
					clearAccountChat(localStorage, session.id, session.did);
				} catch {}
				location.assign('/feed');
			},
			() => {},
			'',
			'settings.SettingsWorkspace.signOut'
		);
	}
</script>

<div class="settings-workspace" data-settings-page={section || 'index'}>
	{#if section}<a
			class="back settings-back"
			href="/settings"
			aria-label="Back to Settings"
			><SettingsIcon name="back" /><span>Settings</span></a
		>{/if}
	{#key `${session.did}:${section}`}
		{#if !section}
			<h1 class="workspace-heading">Settings</h1>
			<Feedback work={menu} reload={loadMenu} />
			{#if !menu.loading && !menu.error}
				<div
					class="settings-identity identity"
					style={`--profile-accent:${colours[profile?.customisation?.colour || 'cobalt'] || colours.cobalt}`}
				>
					<Avatar
						author={{
							did: session.did,
							handle: session.handle,
							displayName: profile?.displayName || session.handle,
							avatar: profile?.avatar
						}}
					/>
					<CompactIdentity
						author={{
							did: session.did,
							handle: session.handle,
							displayName: profile?.displayName,
							crafts: profile?.crafts || []
						}}
					/>
				</div>
				<button class="settings-tile" onclick={openAccounts}
					><SettingsIcon name="switch" /><span class="tile-copy"
						>Switch account</span
					><SettingsIcon name="chevron" size={24} /></button
				>
				{#if !restricted && subscriptionsEnabled()}<a
						class="panel subscription-callout"
						href="/settings/subscriptions"
						><h2>
							<SettingsIcon name="subscription" size={28} />{paid
								? 'Your CraftSky subscription'
								: 'Make more with CraftSky'}
						</h2>
						<p>
							{paid
								? 'Review your access and subscription details.'
								: 'Unlock scheduling, saved folders and more with Plus or Business.'}
						</p>
						<span class="callout-action"
							>{paid ? 'View subscription' : 'Explore subscriptions'}</span
						></a
					>{/if}
				{#each shownGroups as group}<section class="settings-group">
						<h2 class="section-label">{group.title}</h2>
						<nav class="nav-list" aria-label={group.title}>
							{#each group.items as [slug, label]}{@const external =
									slug === 'feedback'}<a
									class="settings-tile"
									class:coming-soon={!subscriptionsEnabled() &&
										['growth', 'customisation'].includes(slug)}
									aria-label={!subscriptionsEnabled() &&
									['growth', 'customisation'].includes(slug)
										? label + ' is coming soon'
										: undefined}
									onclick={(event) => {
										if (
											!subscriptionsEnabled() &&
											['growth', 'customisation'].includes(slug)
										) {
											event.preventDefault();
											comingSoon(label);
										}
									}}
									href={external
										? 'https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/3mpr5izppvt2k?lang=en'
										: slug === 'safety'
											? '/safety'
											: slug === 'notifications'
												? '/notifications/settings'
												: `/settings/${slug}`}
									target={external ? '_blank' : undefined}
									rel={external ? 'noopener noreferrer' : undefined}
									><span class="tile-leading"
										><SettingsIcon
											name={slug}
										/>{#if subscriptionsEnabled() && !paid && ['growth', 'customisation'].includes(slug)}<span
												class="plus-mark">+</span
											>{/if}</span
									><span class="tile-copy"
										><span>{label}</span>{#if slug === 'appearance'}<small
												>{theme}</small
											>{:else if slug === 'instagram'}<small
												>Verify your account or privately import handles.</small
											>{/if}</span
									><SettingsIcon
										name={external ? 'external' : 'chevron'}
										size={24}
									/></a
								>{/each}
						</nav>
					</section>{/each}
				<div class="settings-divider"></div>
				<button
					class="settings-tile danger"
					disabled={menu.busy}
					onclick={signOut}
					><SettingsIcon name="logout" /><span class="tile-copy">Sign out</span
					></button
				>
			{/if}
		{:else if !subscriptionsEnabled() && ['customisation', 'growth', 'subscriptions', 'billing', 'products', 'events', 'business'].includes(section)}<h1
				class="workspace-heading"
			>
				{section === 'growth'
					? 'Growth'
					: section === 'customisation'
						? 'Customisation'
						: 'Subscriptions'} is coming soon
			</h1>
			<p class="empty">
				This feature isn't available yet. We'll let you know when it launches.
			</p>
		{:else if ['appearance', 'customisation', 'languages', 'notifications', 'privacy', 'messaging'].includes(section)}<Preferences
				{api}
				{section}
				{session}
			/>
		{:else if ['followers', 'following', 'muted', 'blocked', 'growth'].includes(section)}<Connections
				{api}
				{section}
			/>
		{:else if ['account', 'moderation', 'about'].includes(section)}<Account
				{api}
				{session}
				{section}
				reference={parts[1]}
			/>
		{:else if ['subscriptions', 'billing'].includes(section)}<Billing
				{api}
				{session}
			/>
		{:else if section === 'instagram'}<Instagram {api} />
		{:else if ['business', 'products', 'events'].includes(section)}<Business
				{api}
				{session}
				{section}
			/>
		{:else}<h1 class="workspace-heading">Page not found</h1>
			<p>Choose a page in <a href="/settings">Settings</a>.</p>{/if}
	{/key}
</div>
