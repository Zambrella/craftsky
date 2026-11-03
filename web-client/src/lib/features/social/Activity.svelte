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
		if (!notificationCopy(item).available) return undefined;
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
	<form
		class="notification-tools"
		method="get"
		action="/app/activity"
		onchange={(event) => event.currentTarget.requestSubmit()}
	>
		<label
			><Icon name="filter" size={18} /><span class="visually-hidden"
				>Category</span
			><select aria-label="Category" name="category" value={category || ''}
				>{#each notificationCategories as [value, label]}<option {value}
						>{label}</option
					>{/each}</select
			></label
		>
	</form>
	<details class="notification-menu" use:contextMenu>
		<summary class="icon-button" aria-label="Notification actions"
			><Icon name="more" /></summary
		>
		<div class="menu-items">
			<a href="/app/settings/notifications">Notification preferences</a><button
				disabled={work.busy}
				onclick={() => load()}>Refresh</button
			>
		</div>
	</details>
</div>
<Status
	{work}
	skeleton="people"
	reload={() => load()}
/>{#each items as item}{@const copy = notificationCopy(item)}
	<article
		class="notification"
		class:unread={item.isNew}
		style={profileStyle(item.actor?.customisation)}
	>
		<a class="notification-main" href={destination(item)}
			><div class="notification-avatar">
				{#if item.actor && item.actor.available !== false}<Avatar
						author={{
							did: item.actor.did,
							handle: item.actor.handle,
							displayName: item.actor.displayName || undefined,
							avatar: item.actor.avatar || undefined
						}}
					/>{:else}<span class="system-avatar"><Icon name="shield" /></span
					>{/if}<span
					class="notification-kind"
					class:like={copy.icon === 'heart'}
					class:repost={copy.icon === 'repost'}><Icon name={copy.icon} /></span
				>
			</div>
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
	.notification-tools {
		padding: 0;
		margin-left: auto;
		display: flex;
		justify-content: flex-end;
	}
	.notification-tools label {
		display: flex;
		align-items: center;
		gap: 6px;
		border: 1.5px solid var(--hairline);
		border-radius: 8px;
		padding-left: 10px;
	}
	.notification-tools select {
		border: 0;
		margin: 0;
		min-height: 42px;
		width: auto;
		font-size: 14px;
		background: transparent;
	}
	.notification {
		display: block;
		border: 0;
		border-bottom: 1px solid var(--hairline);
		padding: 12px 16px;
		margin: 0;
		background: transparent;
	}
	.notification.unread {
		background: color-mix(in srgb, var(--primary-soft) 55%, transparent);
		border-bottom-color: var(--hairline);
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
		width: 25px;
		height: 25px;
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
	.system-avatar {
		width: 44px;
		height: 44px;
		background: var(--primary-soft);
		display: grid;
		place-items: center;
		border-radius: 50%;
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
		display: -webkit-box;
		-webkit-line-clamp: 3;
		line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
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
	}
	.follow-notification {
		margin: 8px 0 0 71px;
		min-height: 36px;
		padding: 6px 16px;
		color: var(--on-primary);
		background: var(--primary);
	}
	.follow-notification.following {
		background: transparent;
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
	.notification-tools {
		flex-shrink: 0;
	}
	.notification-tools select {
		appearance: none;
		padding-right: 10px;
		max-width: 160px;
	}
	.notification-menu {
		position: relative;
	}
	.notification-menu summary {
		list-style: none;
		cursor: pointer;
	}
	.notification-menu .icon-button {
		min-width: 32px;
		padding: 4px;
	}
	@media (max-width: 600px) {
		.notification-heading {
			gap: 4px;
		}
		.notification-tools select {
			max-width: 104px;
			font-size: 14px;
		}
	}

	.notification-tools label {
		flex-direction: row;
	}
	.notification-tools select {
		box-shadow: none;
		border: 0;
		background: transparent;
		padding-left: 0;
	}
	.notification-kind {
		background: var(--primary);
		color: var(--on-primary);
	}
	.notification-kind.like {
		background: #f33232;
		color: white;
	}
	.notification-kind.repost {
		background: var(--butter);
		color: var(--ink);
	}
</style>
