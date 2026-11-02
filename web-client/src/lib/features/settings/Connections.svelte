<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		newOperationKey,
		type Author,
		type CraftskyAPI
	} from '#lib/api/client.ts';
	import {
		growthSegments,
		record,
		text,
		title,
		type GrowthPoint
	} from './model.ts';
	import { safeURL } from '#lib/public.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	let { api, section }: { api: CraftskyAPI; section: string } = $props();
	const work = new Work();
	let items = $state<Author[]>([]),
		cursor = $state<string | null>(null),
		total = $state<number | null>(null);
	let period = $state<'7d' | '30d' | '1y'>('30d'),
		points = $state<GrowthPoint[]>([]),
		latest = $state<number | null>(null),
		change = $state<number | null>(null);
	let available = $state(''),
		allowsPlus = $state(false);
	let query = $state('');
	const kind = $derived(
		section === 'muted'
			? 'mutes'
			: section === 'blocked'
				? 'blocks'
				: (section as 'followers' | 'following')
	);
	let filtered = $derived(
		items.filter((p) =>
			`${p.handle} ${p.displayName || ''}`
				.toLowerCase()
				.includes(query.toLowerCase())
		)
	);
	function load() {
		if (section === 'growth')
			return work.load(
				async () => {
					const access = await api.billing.access();
					const data =
						access.givesAccess && access.effectiveTier !== 'free'
							? await api.profiles.followerGrowth(period)
							: {};
					return [data, access] as const;
				},
				([data, access]) => {
					const value = record(data);
					points = Array.isArray(value.points)
						? value.points.map((p) => {
								const point = record(p);
								return {
									date: text(point.date),
									count: typeof point.count === 'number' ? point.count : null
								};
							})
						: [];
					latest =
						typeof value.latestFollowerCount === 'number'
							? value.latestFollowerCount
							: null;
					change = typeof value.netChange === 'number' ? value.netChange : null;
					available = text(value.availableFrom);
					allowsPlus = access.givesAccess && access.effectiveTier !== 'free';
				}
			);
		return work.load(
			() => api.profiles.relationships(kind),
			(value) => {
				items = value.items;
				cursor = value.cursor || null;
				total = value.totalCount;
			}
		);
	}
	function more() {
		const next = cursor;
		if (!next) return;
		void work.run(
			() => api.profiles.relationships(kind, { cursor: next }),
			(value) => {
				items = [
					...items,
					...value.items.filter((p) => !items.some((old) => old.did === p.did))
				];
				cursor = value.cursor || null;
			},
			''
		);
	}
	function remove(profile: Author) {
		const id = profile.did;
		const key = newOperationKey();
		const request = async () => {
			if (kind === 'mutes') await api.profiles.unmute(id);
			else if (kind === 'blocks') await api.profiles.unblock(id, key);
			else await api.profiles.unfollow(id, key);
		};
		void work.run(
			request,
			() => {
				items = items.filter((p) => p.did !== id);
				if (total !== null) total = Math.max(0, total - 1);
			},
			kind === 'mutes'
				? 'Account unmuted.'
				: kind === 'blocks'
					? 'Account unblocked.'
					: 'Unfollowed.'
		);
	}
	onMount(load);
	onDestroy(() => work.dispose());
</script>

<h1>
	{section === 'growth'
		? 'Follower growth'
		: section === 'muted'
			? 'Muted accounts'
			: section === 'blocked'
				? 'Blocked accounts'
				: title(section)}
</h1>
<Feedback {work} reload={load} />
{#if section === 'growth'}
	<section class="panel">
		<div class="row">
			<h2>Your community</h2>
			<label
				><span class="muted">Period</span><select
					bind:value={period}
					disabled={work.busy}
					onchange={load}
					><option value="7d">7 days</option><option value="30d">30 days</option
					><option value="1y">1 year</option></select
				></label
			>
		</div>
		{#if !work.loading && allowsPlus}<div class="two-col">
				<div>
					<strong class="metric">{latest ?? '—'}</strong>
					<p class="muted">Latest follower count</p>
				</div>
				<div>
					<strong class="metric"
						>{change === null
							? '—'
							: `${change > 0 ? '+' : ''}${change}`}</strong
					>
					<p class="muted">Change in this period</p>
				</div>
			</div>
			{#if points.some((p) => p.count !== null)}<svg
					viewBox="0 0 600 210"
					role="img"
					aria-label="Follower growth over the selected period. Exact values appear in the table below."
					><path
						d="M20,190 L580,190"
						stroke="var(--hairline)"
						fill="none"
					/>{#each growthSegments(points) as path}<path
							d={path}
							fill="none"
							stroke="var(--primary)"
							stroke-width="3"
							stroke-linecap="round"
						/>{/each}</svg
				>
				<p class="muted">
					Gaps indicate an unavailable snapshot.{available
						? ` History is available from ${available}.`
						: ''}
				</p>
				<details>
					<summary>View daily counts</summary>
					<table>
						<thead><tr><th>Date</th><th>Followers</th></tr></thead><tbody
							>{#each points as point}<tr
									><td>{point.date}</td><td>{point.count ?? 'Unavailable'}</td
									></tr
								>{/each}</tbody
						>
					</table>
				</details>{:else}<p class="empty">
					Your history will appear after follower snapshots have been captured.
				</p>{/if}
		{:else if !work.loading}<p>
				Follower growth is included with CraftSky Plus and Business.
			</p>
			<a class="action" href="/app/settings/subscriptions"
				>View subscription access</a
			>{/if}
	</section>
{:else if !work.loading}
	<section class="panel">
		<label
			>Search these accounts<input
				type="search"
				bind:value={query}
				placeholder="Name or handle"
			/></label
		>{#if total !== null}<p class="muted">
				{total}
				{total === 1 ? 'account' : 'accounts'}
			</p>{/if}
		{#each filtered as profile (profile.did)}<div class="row">
				<div class="identity">
					{#if safeURL(profile.avatar || undefined)}<img
							class="avatar"
							src={safeURL(profile.avatar || undefined)}
							alt=""
							loading="lazy"
						/>{:else}<span class="avatar" aria-hidden="true"></span>{/if}<a
						href={`/app/profile/${encodeURIComponent(profile.did)}`}
						><strong
							>{profile.displayName ||
								profile.handle ||
								'Unavailable account'}</strong
						><small>@{profile.handle}</small></a
					>
				</div>
				{#if kind !== 'followers'}<button
						disabled={work.busy || !!work.retry}
						onclick={() => {
							if (
								kind === 'blocks' &&
								!confirm(
									profile.handle
										? `Unblock @${profile.handle}?`
										: 'Unblock this account?'
								)
							)
								return;
							remove(profile);
						}}
						>{kind === 'mutes'
							? 'Unmute'
							: kind === 'blocks'
								? 'Unblock'
								: 'Unfollow'}</button
					>{/if}
			</div>{:else}<p class="empty">
				{query
					? 'No matching accounts.'
					: section === 'muted'
						? 'You have not muted any accounts.'
						: section === 'blocked'
							? 'You have not blocked any accounts.'
							: `No ${section} yet.`}
			</p>{/each}
		{#if cursor}<button disabled={work.busy} onclick={more}>Load more</button
			>{/if}
	</section>
{/if}

<style>
	.metric {
		font-size: 2rem;
	}
	svg {
		width: 100%;
		height: auto;
	}
	summary {
		cursor: pointer;
		font-weight: 600;
		padding: 0.8rem 0;
	}
</style>
