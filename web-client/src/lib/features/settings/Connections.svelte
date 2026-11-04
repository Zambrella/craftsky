<script lang="ts">
	import { appendPage } from '#lib/pagination.ts';
	import { usePrompt } from '../social/prompt';
	const ask = usePrompt();
	import { onMount, onDestroy } from 'svelte';
	import {
		newOperationKey,
		type Author,
		type CraftskyAPI
	} from '#lib/api/client.ts';
	import { record, text, title, type GrowthPoint } from './model.ts';
	import Avatar from '#lib/Avatar.svelte';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import SettingsIcon from './SettingsIcon.svelte';
	import { followerChart } from './native-settings.ts';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	import EmptyState from '../social/EmptyState.svelte';
	let { api, section }: { api: CraftskyAPI; section: string } = $props();
	const work = new Work();
	let loaded = $state(false);
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
	let chartWidth = $state(600);
	let chart = $derived(followerChart(points, period, chartWidth));
	function chartDate(value: string) {
		const date = new Date(value + 'T00:00:00Z');
		return Number.isFinite(date.getTime())
			? new Intl.DateTimeFormat(undefined, {
					month: 'short',
					...(period === '1y'
						? { year: 'numeric' as const }
						: { day: 'numeric' as const }),
					timeZone: 'UTC'
				}).format(date)
			: value;
	}
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
					loaded = true;
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
				loaded = true;
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
				items = appendPage(items, value.items, (item) => item.did);
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

<h1 class="workspace-heading">
	{section === 'growth'
		? 'Growth'
		: section === 'muted'
			? 'Muted accounts'
			: section === 'blocked'
				? 'Blocked accounts'
				: `${title(section)} (${total ?? 0})`}
</h1>
<Feedback {work} reload={load} />
{#if section === 'growth'}
	<section class="growth-page">
		{#if !work.loading && loaded && allowsPlus}<div class="follower-total">
				<strong class="metric"
					>{latest === null
						? '—'
						: new Intl.NumberFormat().format(latest)}</strong
				>
				<p class="muted">followers</p>
			</div>
			<h2>Trend</h2>
			<p class="muted">Craftsky followers. Updated daily. Dates are UTC.</p>
			<div class="segmented" aria-label="Growth period">
				{#each [['7d', '7 days'], ['30d', '30 days'], ['1y', '1 year']] as [value, label]}<button
						aria-pressed={period === value}
						disabled={work.busy || work.loading}
						onclick={() => {
							period = value as typeof period;
							load();
						}}>{label}</button
					>{/each}
			</div>
			<p class="growth-change">
				{change === null
					? 'Not enough history'
					: change > 0
						? `Up ${change}`
						: change < 0
							? `Down ${Math.abs(change)}`
							: 'No change'}
			</p>
			{#if chart}<div class="growth-chart" bind:clientWidth={chartWidth}>
					<svg
						viewBox={`0 0 ${chartWidth} 240`}
						role="img"
						aria-label={`Follower growth: ${latest ?? 'Unavailable'} followers. ${change === null ? 'Not enough history.' : 'Net change ' + change + '.'} Exact daily values are available below.`}
					>
						{#each chart.ticks as tick}<line
								x1="58"
								x2={chartWidth - 22}
								y1={tick.y}
								y2={tick.y}
								stroke="var(--hairline)"
							/><text x="46" y={tick.y + 4} text-anchor="end"
								>{new Intl.NumberFormat().format(tick.count)}</text
							>{/each}
						<rect
							x="58"
							y="20"
							width={Math.max(40, chartWidth - 80)}
							height="184"
							stroke="var(--hairline)"
							fill="none"
						/>
						{#each chart.paths as path}<path
								d={path}
								fill="none"
								stroke="var(--primary)"
								stroke-width="3"
								stroke-linecap="round"
							/>{/each}
						{#each chart.dots as point}<circle
								cx={point.x}
								cy={point.y}
								r="4"
								fill="var(--surface)"
								stroke="var(--primary)"
								stroke-width="2"
							/>{/each}
						{#each chart.dates as point, index}<text
								x={point.x}
								y="228"
								text-anchor={index === 0
									? 'start'
									: index === chart.dates.length - 1
										? 'end'
										: 'middle'}>{chartDate(point.date)}</text
							>{/each}
					</svg>
				</div>
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
					<SettingsIcon name="growth" /><strong>Followers</strong>No follower
					history yet
				</p>{/if}
		{:else if !work.loading && loaded}<p>
				Follower growth is included with CraftSky Plus and Business.
			</p>
			<a class="action" href="/app/settings/subscriptions"
				>View subscription access</a
			>{/if}
	</section>
{:else if !work.loading && loaded}
	<section class="native-list relationship-list">
		{#each filtered as profile (profile.did)}<div class="row identity-row">
				<a
					class="identity profile-list-identity"
					href={`/app/profile/${encodeURIComponent(profile.did)}`}
					><Avatar author={profile} /><CompactIdentity author={profile} /></a
				>
				{#if kind === 'mutes' || kind === 'blocks'}<button
						disabled={work.busy || !!work.retry}
						onclick={async () => {
							if (
								kind === 'blocks' &&
								!(await ask({
									title: 'Unblock account?',
									detail: profile.handle
										? `Unblock @${profile.handle}?`
										: 'Unblock this account?',
									confirmLabel: 'Unblock'
								}))
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
			</div>{:else}<EmptyState
				fill
				icon={section === 'muted'
					? 'muted'
					: section === 'blocked'
						? 'blocked'
						: 'followers'}
				title={query
					? 'No matching accounts.'
					: section === 'muted'
						? 'You have not muted any accounts.'
						: section === 'blocked'
							? 'You have not blocked any accounts.'
							: section === 'followers'
								? 'No one follows you yet'
								: 'You are not following anyone'}
				detail="Nothing here yet."
			/>{/each}
		{#if cursor}<button disabled={work.busy} onclick={more}>Load more</button
			>{/if}
	</section>
{/if}

<style>
	.metric {
		font-size: 30px;
		line-height: 1.15;
		font-weight: 600;
		color: var(--primary);
	}
	.follower-total {
		text-align: center;
		padding: 8px 0 32px;
	}
	.growth-page {
		padding-top: 8px;
	}
	.growth-change {
		display: inline-block;
		padding: 4px 10px;
		border-radius: 999px;
		background: color-mix(in srgb, var(--primary) 10%, var(--paper));
		color: var(--primary);
		font-size: 14px;
	}
	.relationship-list > .row {
		padding: 4px 16px;
		min-height: 56px;
		border: 0;
	}
	.relationship-list > .row button {
		border: 0;
		background: transparent;
		color: var(--primary);
		padding: 8px 12px;
	}

	.growth-chart {
		width: 100%;
		height: 240px;
	}
	.growth-chart svg {
		width: 100%;
		height: 240px;
	}
	.growth-chart text {
		font:
			400 12px Outfit,
			sans-serif;
		fill: var(--ink-2);
	}
	.profile-list-identity {
		color: inherit;
		text-decoration: none;
		flex: 1;
		min-height: 40px;
	}
	.relationship-list :global(.avatar) {
		width: 41px;
		height: 41px;
	}

	summary {
		cursor: pointer;
		font-weight: 600;
		padding: 0.8rem 0;
	}
</style>
