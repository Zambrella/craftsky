<script lang="ts">
	import { contextMenu } from './context-menu';
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import { newOperationKey } from '#lib/api/client.ts';
	import type {
		CraftskyAPI,
		AppNotification,
		NotificationCategory
	} from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import { compactDate } from '#lib/public.ts';
	import { visible, postRoute, postAddress, profileStyle } from './model';
	import { notificationCopy, notificationCategories } from './presentation';
	import EmptyState from './EmptyState.svelte';
	import Icon from '../shell/Icon.svelte';
	let { api, did = '' }: { api: CraftskyAPI; did?: string } = $props();
	const work = new Work();
	let items = $state<AppNotification[]>([]),
		cursor = $state<string | null>(null);
	const category =
		(page.url.searchParams.get('category') as NotificationCategory) ||
		undefined;
	async function load(more = false) {
		await work.run(
			() =>
				api.notifications.list({ cursor: more ? cursor : undefined, category }),
			(value) => {
				items = more ? [...items, ...value.items] : value.items;
				cursor = value.cursor || null;
			}
		);
		if (!more && !work.error) await api.notifications.seen().catch(() => {});
	}
	function destination(item: AppNotification) {
		if (!notificationCopy(item).interactive) return undefined;
		if (item.caseReference)
			return (
				'/app/settings/moderation/' + encodeURIComponent(item.caseReference)
			);
		if (item.subjectPost && visible(item.subjectPost))
			return (
				postRoute(item.subjectPost) +
				(item.reply ? '?focus=' + encodeURIComponent(item.reply.uri) : '')
			);
		const ref = postAddress(item.uri || '');
		if (ref)
			return (
				'/app/posts/' +
				encodeURIComponent(ref.did) +
				'/' +
				encodeURIComponent(ref.rkey)
			);
		return item.actor
			? '/app/profile/' + encodeURIComponent(item.actor.did)
			: '/app/settings/account';
	}
	function follow(item: AppNotification) {
		const actor = item.actor;
		if (!actor) return;
		const key = newOperationKey();
		void work.run(
			async () => {
				const current =
					typeof actor.viewerIsFollowing === 'boolean'
						? actor
						: await api.profiles.get(actor.did);
				return current.viewerIsFollowing
					? api.profiles.unfollow(actor.did, key)
					: api.profiles.follow(actor.did, key);
			},
			(profile) => {
				items = items.map((old) =>
					old.actor?.did === profile.did
						? { ...old, actor: { ...old.actor, ...profile } }
						: old
				);
			}
		);
	}
	onMount(() => load());
	onDestroy(() => work.dispose());
</script>

<div class="workspace-heading notification-heading">
	<h1>Notifications</h1>
	<details class="notification-categories" use:contextMenu>
		<summary
			class="category-trigger"
			aria-label={notificationCategories.find(
				([value]) => value === (category || '')
			)?.[1] || 'All categories'}
		>
			<Icon name="filter" size={18} />
			{notificationCategories.find(
				([value]) => value === (category || '')
			)?.[1] || 'All categories'}
		</summary>
		<div class="menu-items">
			{#each notificationCategories as [value, label, icon]}
				<a
					href={'/app/activity' + (value ? '?category=' + value : '')}
					aria-current={(category || '') === value ? 'page' : undefined}
				>
					<Icon name={icon} /><span>{label}</span>
					{#if (category || '') === value}<Icon name="check" />{/if}
				</a>
			{/each}
		</div>
	</details>
</div>
<Status
	{work}
	skeleton="activity"
	reload={() => load()}
/>{#each items as item}{@const copy = notificationCopy(item)}
	<article
		class="notification"
		class:unread={item.isNew}
		class:system={!item.actor}
		style={profileStyle(item.actor?.customisation)}
	>
		<a class="notification-main" href={destination(item)}
			>{#if item.actor}<div class="notification-avatar">
					<Avatar
						author={copy.available
							? {
									did: item.actor.did,
									handle: item.actor.handle,
									displayName: item.actor.displayName || undefined,
									avatar: item.actor.avatar || undefined
								}
							: {
									did: 'unavailable',
									handle: '',
									displayName: 'Account unavailable'
								}}
					/>
					<span
						class="notification-kind"
						class:like={copy.icon === 'heart' || !copy.available}
						class:repost={copy.icon === 'repost'}
						class:accent={copy.icon === 'at' || copy.icon === 'quotes'}
						class:generic={copy.icon === 'activity' && copy.available}
						><Icon name={copy.icon} filled /></span
					>
				</div>{:else}<span
					class="system-icon"
					class:generic={copy.icon === 'activity'}
					><Icon name={copy.icon} /></span
				>{/if}
			<div class="notification-body">
				<p class="notification-title">
					{#if item.actor && copy.available}<strong
							>{item.actor.displayName ||
								item.actor.handle ||
								'Account unavailable'}</strong
						>
					{/if}{' '}{copy.action}
				</p>
				{#if copy.context}<p class="notification-context">
						{copy.context}
					</p>{/if}
				{#if copy.preview}<div class="notification-preview">
						{#if copy.previewTitle}<strong>{copy.previewTitle}</strong
							>{/if}{#if copy.previewText}<p>{copy.previewText}</p>{/if}
					</div>{/if}
				{#if copy.quotedPreview}<div
						class="notification-preview quoted-preview"
					>
						{#if copy.quotedPreview.projectTitle}<strong
								>{copy.quotedPreview.projectTitle}</strong
							>{/if}
						<p>{copy.quotedPreview.text}</p>
					</div>{/if}<time datetime={item.createdAt}
					>{compactDate(item.createdAt)}</time
				>
			</div></a
		>
		{#if copy.canFollow && item.actor?.did !== did}<button
				class="follow-notification"
				class:following={item.actor?.viewerIsFollowing}
				disabled={work.busy || !!work.retry}
				onclick={() => follow(item)}
				>{item.actor?.viewerIsFollowing ? 'Unfollow' : 'Follow'}</button
			>{/if}
	</article>{:else}{#if !work.busy && !work.error}<EmptyState
			title={category
				? notificationCategories.find(([value]) => value === category)?.[1] ||
					'Notifications'
				: 'Notifications'}
			detail={category
				? 'No notifications in this category yet.'
				: 'You’re all caught up.'}
			icon="activity"
			>{#snippet actions()}{#if category}<a href="/app/activity"
						>Show all notifications</a
					>{/if}{/snippet}</EmptyState
		>{/if}{/each}{#if cursor}<button
		class="button next"
		disabled={work.busy}
		onclick={() => load(true)}>Load more</button
	>{/if}

<style>
	.notification-categories {
		margin-left: auto;
	}
	.category-trigger {
		display: flex;
		align-items: center;
		gap: 8px;
		min-height: 40px;
		padding: 8px 12px;
		border: 1.5px solid var(--hairline);
		border-radius: 8px;
		font-size: 15px;
		font-weight: 600;
		cursor: pointer;
	}
	.category-trigger::-webkit-details-marker {
		display: none;
	}
	.category-trigger :global(svg) {
		color: var(--primary);
	}
	.notification-categories .menu-items span {
		flex: 1;
	}
	.notification-categories .menu-items [aria-current='page'] {
		background: color-mix(in srgb, var(--primary) 4%, var(--surface));
	}
	.notification {
		position: relative;
		display: block;
		border: 0;
		padding: 12px 16px;
		margin: 0;
		background: transparent;
	}
	.notification.unread {
		background: color-mix(in srgb, var(--primary) 7%, var(--paper));
	}
	.notification-main {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		text-decoration: none;
		color: var(--ink);
	}
	.notification-avatar {
		position: relative;
		padding: 0 4px 4px 0;
		flex-shrink: 0;
	}
	.notification-avatar :global(.avatar) {
		width: 55px;
		height: 55px;
		border-width: 3.5px;
		font-size: 24px;
		box-shadow: none;
	}
	.notification-kind {
		position: absolute;
		bottom: 0;
		right: 0;
		width: 26px;
		height: 26px;
		border: 2px solid var(--paper);
		display: grid;
		place-items: center;
		border-radius: 50%;
		color: var(--primary);
		background: var(--primary-soft);
	}
	.notification-kind.like {
		color: var(--red-ink);
		background: var(--rose, #fce5e7);
	}
	.notification-kind :global(svg) {
		width: 14px;
		height: 14px;
	}
	.system-icon {
		width: 24px;
		height: 24px;
		color: var(--primary);
		flex: none;
	}
	.system-icon.generic {
		color: var(--ink-3);
	}
	.system .notification-main {
		align-items: center;
	}
	.notification.unread::after {
		content: '';
		position: absolute;
		top: 8px;
		right: 8px;
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--primary);
	}

	.notification-body {
		min-width: 0;
		flex: 1;
	}
	.notification .notification-title {
		font-size: 16px;
		line-height: 1.45;
		margin: 0 0 4px;
	}
	.notification-preview {
		color: var(--ink-2);
		line-height: 1.4;
		margin: 0 0 4px;
	}
	.notification-context {
		margin: 0 0 4px;
		font-size: 13px;
		color: var(--ink-2);
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
	}
	.quoted-preview {
		margin-top: 8px;
		padding-left: 12px;
		border-left: 2px solid var(--hairline);
	}
	.notification-preview strong {
		display: block;
		color: var(--ink);
		font-weight: 600;
	}
	.notification-preview p {
		margin: 4px 0 0;
		white-space: pre-wrap;
		display: -webkit-box;
		-webkit-line-clamp: 3;
		line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.follow-notification {
		margin: 8px 0 0 71px;
		min-height: 36px;
		padding: 6px 16px;
		border: 1.5px solid var(--ink);
		box-shadow: 3px 3px 0 var(--ink);
		color: var(--on-primary);
		background: var(--primary);
	}
	.follow-notification.following {
		background: var(--surface);
		color: var(--ink);
	}
	.notification time {
		font-size: 13px;
	}

	.notification-heading {
		flex-wrap: nowrap;
		gap: 8px;
	}
	.notification-heading h1 {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.notification-categories {
		flex-shrink: 0;
	}
	@media (max-width: 600px) {
		.notification-heading {
			gap: 4px;
		}
		.category-trigger {
			max-width: 170px;
		}
	}
	.notification-kind {
		background: var(--primary);
		color: var(--on-primary);
	}
	.notification-kind.like {
		background: #f03a2e;
		color: white;
	}
	.notification-kind.repost {
		background: var(--butter);
		color: var(--ink);
	}
	.notification-kind.accent {
		background: var(--accent, #f03a2e);
		color: white;
	}
	.notification-kind.generic {
		background: var(--ink-3);
		color: white;
	}
</style>
