<script lang="ts">
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import { onDestroy } from 'svelte';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import type { Session } from './model.ts';
	import Preferences from './Preferences.svelte';
	import Connections from './Connections.svelte';
	import Account from './Account.svelte';
	import Billing from './Billing.svelte';
	import Instagram from './Instagram.svelte';
	import Business from './Business.svelte';
	import './settings.css';
	let {
		api,
		path,
		session
	}: { api: CraftskyAPI; path: string[]; session: Session } = $props();
	let parts = $derived(
		path.includes('settings') ? path.slice(path.indexOf('settings') + 1) : path
	);
	let section = $derived(parts[0] || '');
	const menu = new Work();
	let restricted = $state(false);
	function loadMenu() {
		void menu.load(
			() => api.account.eligibility(),
			(policy) => {
				restricted = policy.state === 'restricted';
			}
		);
	}
	$effect(() => {
		if (!section) loadMenu();
	});
	onDestroy(() => menu.dispose());
	const groups = [
		{
			title: 'Preferences',
			items: [
				['appearance', 'Appearance'],
				['customisation', 'Profile customisation'],
				['languages', 'Languages'],
				['notifications', 'Notifications'],
				['messaging', 'Messaging'],
				['privacy', 'Privacy']
			]
		},
		{
			title: 'Connections',
			items: [
				['growth', 'Follower growth'],
				['followers', 'Followers'],
				['following', 'Following'],
				['muted', 'Muted accounts'],
				['blocked', 'Blocked accounts']
			]
		},
		{
			title: 'Discovery',
			items: [['instagram', 'Find your Instagram community']]
		},
		{
			title: 'Business',
			items: [
				['business', 'Business profile'],
				['products', 'Products'],
				['events', 'Events']
			]
		},
		{
			title: 'General',
			items: [
				['subscriptions', 'Subscriptions and billing'],
				['moderation', 'Account standing'],
				['account', 'Account'],
				['about', 'About CraftSky']
			]
		}
	];
	let shownGroups = $derived(
		groups
			.map((group) => ({
				...group,
				items: group.items.filter(
					([slug]) =>
						!restricted ||
						['account', 'moderation', 'muted', 'blocked', 'about'].includes(
							slug
						)
				)
			}))
			.filter((group) => group.items.length)
	);
</script>

<div class="settings-workspace">
	{#if section}<a class="back" href="/app/settings">← Settings</a>{/if}
	{#key `${session.did}:${section}`}
		{#if !section}
			<h1>Settings</h1>
			<Feedback work={menu} reload={loadMenu} />
			{#if !menu.loading && !menu.error}{#each shownGroups as group}<section
						class="panel"
					>
						<h2>{group.title}</h2>
						<nav class="nav-list" aria-label={group.title}>
							{#each group.items as [slug, label]}<a
									href={`/app/settings/${slug}`}>{label}</a
								>{/each}
						</nav>
					</section>{/each}{/if}
		{:else if ['appearance', 'customisation', 'languages', 'notifications', 'privacy', 'messaging'].includes(section)}
			<Preferences {api} {section} {session} />
		{:else if ['followers', 'following', 'muted', 'blocked', 'growth'].includes(section)}
			<Connections {api} {section} />
		{:else if ['account', 'moderation', 'about'].includes(section)}
			<Account {api} {session} {section} reference={parts[1]} />
		{:else if ['subscriptions', 'billing'].includes(section)}
			<Billing {api} {session} />
		{:else if section === 'instagram'}
			<Instagram {api} />
		{:else if ['business', 'products', 'events'].includes(section)}
			<Business {api} {session} {section} />
		{:else}<h1>Page not found</h1>
			<p>Choose a page in <a href="/app/settings">Settings</a>.</p>{/if}
	{/key}
</div>
