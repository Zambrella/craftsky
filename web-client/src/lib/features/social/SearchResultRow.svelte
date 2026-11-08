<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	let {
		href,
		label,
		hashtag = false,
		postCount,
		inset = false,
		divider = false,
		trailing
	}: {
		href: string;
		label: string;
		hashtag?: boolean;
		postCount?: number;
		inset?: boolean;
		divider?: boolean;
		trailing?: Snippet;
	} = $props();
</script>

<div class="search-result-row" class:inset class:divider>
	<a {href}>
		<span class="search-result-icon" aria-hidden="true">
			{#if hashtag}#{:else}<Icon name="search" />{/if}
		</span>
		<span class="search-result-label">
			<strong>{label}</strong>
			{#if postCount !== undefined}<small
					>{postCount} {postCount === 1 ? 'post' : 'posts'}</small
				>{/if}
		</span>
	</a>
	{#if trailing}{@render trailing()}{/if}
</div>

<style>
	.search-result-row {
		display: flex;
		align-items: center;
		gap: 8px;
		min-width: 0;
		min-height: 68px;
		padding: 12px 0;
	}
	.inset {
		padding-inline: 16px;
	}
	.divider {
		border-bottom: 1px solid var(--hairline);
	}
	a {
		display: flex;
		align-items: center;
		gap: 12px;
		flex: 1;
		min-width: 0;
		text-decoration: none;
		color: var(--ink);
	}
	.search-result-icon {
		width: 41px;
		height: 41px;
		flex-shrink: 0;
		display: grid;
		place-items: center;
		background: var(--primary-soft);
		color: var(--ink);
		border-radius: 50%;
		font-size: 24px;
	}
	.search-result-label {
		min-width: 0;
	}
	strong {
		overflow-wrap: anywhere;
	}
	small {
		display: block;
		font-size: 14px;
		margin-top: 2px;
		color: var(--ink-3);
	}
</style>
