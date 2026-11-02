<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import type {
		CraftskyAPI,
		AppNotification,
		NotificationCategory
	} from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import { compactDate } from '#lib/public.ts';
	import { visible, postRoute, postAddress } from './model';
	let { api }: { api: CraftskyAPI } = $props();
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
		if (['instagramMatch', 'instagram_match'].includes(item.type))
			return '/app/settings/instagram';
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
	onMount(() => load());
	onDestroy(() => work.dispose());
</script>

<div class="workspace-heading">
	<h1>Activity</h1>
	<a href="/app/settings/notifications">Preferences</a><button
		onclick={() => load()}>Refresh</button
	>
</div>
<form class="panel" method="get" action="/app/activity">
	<label
		>Category<select name="category" value={category || ''}
			><option value="">All activity</option
			>{#each ['follow', 'like', 'reply', 'mention', 'quote', 'repost', 'instagramMatch', 'moderation'] as kind}<option
					value={kind}>{kind}</option
				>{/each}</select
		></label
	><button>Apply</button>
</form>
<Status {work} />{#each items as item}<a
		class="panel notification"
		class:unread={item.isNew}
		href={destination(item)}
		>{#if item.actor}<Avatar
				author={{
					did: item.actor.did,
					handle: item.actor.handle,
					displayName: item.actor.displayName || undefined,
					avatar: item.actor.avatar || undefined
				}}
			/>{/if}
		<div>
			<strong
				>{item.actor?.displayName || item.actor?.handle || 'CraftSky'}</strong
			>
			<p>{item.type.replaceAll('_', ' ')}</p>
			{#if item.preview}<p>
					{item.preview.projectTitle || item.preview.text}
				</p>{/if}<time datetime={item.createdAt}
				>{compactDate(item.createdAt)}</time
			>
		</div></a
	>{:else}{#if !work.busy}<p class="empty">
			You’re all caught up.
		</p>{/if}{/each}{#if cursor}<button
		class="button next"
		onclick={() => load(true)}>Load more</button
	>{/if}
