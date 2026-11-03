<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { CraftskyAPI, Profile, Author } from '#lib/api/client.ts';
	import Avatar from '#lib/Avatar.svelte';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	import { profileStyle } from '../social/model';
	import { craftFeeds, craftKey } from '../social/craft-feeds';
	let {
		api,
		profile,
		onProfile
	}: {
		api: CraftskyAPI;
		profile?: Profile;
		onProfile: (profile: Profile) => void;
	} = $props();
	let members = $state<Author[]>([]),
		loading = $state(true),
		error = $state(false),
		retryingCrafts = $state(false);
	let dead = false;
	async function loadMembers() {
		loading = true;
		error = false;
		try {
			const result = await api.profiles.recent();
			if (!dead) members = result.items;
		} catch {
			if (!dead) error = true;
		} finally {
			if (!dead) loading = false;
		}
	}
	async function loadCrafts() {
		retryingCrafts = true;
		try {
			const result = await api.profiles.me();
			if (!dead) onProfile(result);
		} catch {
			/* Keep the local retry control available. */
		} finally {
			if (!dead) retryingCrafts = false;
		}
	}
	onMount(() => {
		void loadMembers();
	});
	onDestroy(() => {
		dead = true;
	});
</script>

<section aria-labelledby="discovery-crafts-title">
	<h2 id="discovery-crafts-title">Your crafts</h2>
	{#if !profile}<div class="section-message">
			<p>Couldn't load your crafts.</p>
			<button disabled={retryingCrafts} onclick={loadCrafts}
				>{retryingCrafts ? 'Loading…' : 'Retry'}</button
			>
		</div>
	{:else}{#each craftFeeds(profile.crafts, true) as craft}<a
				class="craft-link"
				href={'/app/home?craft=' + craftKey(craft.id)}
				><span
					class="craft-icon"
					style:--craft-icon={`url('/craft_icons/${craftKey(craft.id)}.svg')`}
					aria-hidden="true"
				></span>{craft.labels.en}</a
			>
		{:else}<a class="craft-link" href="/app/profile/edit">Choose your crafts</a
			>{/each}{/if}
</section>
<section class="members" aria-labelledby="discovery-members-title">
	<h2 id="discovery-members-title">New to CraftSky</h2>
	{#if loading}<p class="section-message" role="status">Loading new members…</p>
	{:else if error}<div class="section-message" role="status">
			<p>Couldn't load new members.</p>
			<button onclick={loadMembers}>Retry</button>
		</div>
	{:else}{#each members as author (author.did)}<a
				class="member"
				href={'/app/profile/' + encodeURIComponent(author.did)}
				style={profileStyle(author.customisation)}
				><Avatar
					author={{
						did: author.did,
						handle: author.handle,
						displayName: author.displayName || undefined,
						avatar: author.avatar || undefined
					}}
				/><CompactIdentity {author} /></a
			>
		{:else}<p class="section-message">
				No new members to show yet.
			</p>{/each}{/if}
</section>

<style>
	section + section {
		margin-top: 24px;
	}
	h2 {
		font-size: 16px;
		line-height: 1.5;
		font-weight: 600;
		margin: 0;
		padding: 16px;
	}
	.members {
		border: 1px solid var(--hairline);
		border-radius: 14px;
		overflow: hidden;
	}
	.section-message {
		padding: 16px;
		margin: 0;
		font-size: 14px;
	}
	.section-message p {
		margin: 0 0 8px;
	}
	.craft-link,
	.member {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
		text-decoration: none;
	}
	.craft-link {
		color: var(--primary);
		font-size: 14px;
		font-weight: 600;
	}
	.craft-icon {
		width: 20px;
		height: 20px;
		background: currentColor;
		mask: var(--craft-icon) center / contain no-repeat;
	}
	.member {
		color: var(--ink);
		min-width: 0;
	}
	.member:hover,
	.craft-link:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
</style>
