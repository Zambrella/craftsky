<script lang="ts">
	import { getContext, onMount } from 'svelte';
	import { page } from '$app/state';
	import { label } from '#lib/public.ts';
	import Filters from './Filters.svelte';
	import Icon from '../shell/Icon.svelte';
	const returnTo =
		getContext<(destination: string) => Promise<void>>('craftsky-return');
	const craft = page.url.searchParams.get('craft') || '';
	const origin = '/feed' + page.url.search;
	let filterFields = $state<{ clearAll: () => void }>(null!);
	let backButton: HTMLButtonElement;
	onMount(() => backButton?.focus({ preventScroll: true }));
	const close = () => returnTo(origin);
	async function clearAll() {
		filterFields.clearAll();
		const params = new URLSearchParams({
			craft,
			sort: page.url.searchParams.get('sort') || 'chronological'
		});
		await close();
		await returnTo('/feed?' + params);
	}
	async function apply(event: SubmitEvent) {
		event.preventDefault();
		const params = new URLSearchParams(
			[...new FormData(event.currentTarget as HTMLFormElement)].map(
				([key, value]) => [key, String(value)]
			)
		);
		await close();
		const destination = '/feed?' + params;
		if (destination !== page.url.pathname + page.url.search)
			await returnTo(destination);
	}
</script>

<svelte:window
	onkeydown={(event) => {
		if (event.key === 'Escape' && !event.defaultPrevented) {
			event.preventDefault();
			void close();
		}
	}}
/>
<section class="project-filter-page" aria-labelledby="filter-title">
	<form method="get" action="/feed" onsubmit={apply}>
		<header>
			<button
				type="button"
				class="icon-button"
				aria-label="Back to projects"
				bind:this={backButton}
				onclick={close}><Icon name="back" /></button
			>
			<h1 id="filter-title">
				{#if ['knitting', 'crochet', 'sewing', 'embroidery', 'quilting'].includes(craft
						.split('#')
						.at(-1)
						?.toLowerCase() || '')}<span
						class="feed-craft-icon"
						style:--craft-icon={`url('/craft_icons/${craft.split('#').at(-1)?.toLowerCase()}.svg')`}
						aria-hidden="true"
					></span>{/if}
				Filter {label(craft)} projects
			</h1>
		</header>
		<input type="hidden" name="craft" value={craft} /><input
			type="hidden"
			name="sort"
			value={page.url.searchParams.get('sort') || 'chronological'}
		/>
		<div class="filter-scroll">
			<Filters bind:this={filterFields} {craft} expanded />
		</div>
		<footer>
			<button type="button" class="clear-all-filters" onclick={clearAll}
				>Clear all</button
			><button class="button">Apply filters</button>
		</footer>
	</form>
</section>

<style>
	.project-filter-page {
		position: fixed;
		z-index: 25;
		top: 0;
		left: var(--app-content-left, 0px);
		width: var(--app-content-width, 100vw);
		height: 100dvh;
		background: var(--paper);
	}
	.project-filter-page form {
		display: flex;
		flex-direction: column;
		height: 100%;
	}
	.filter-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
	}
	.project-filter-page header,
	.project-filter-page footer {
		padding: 16px;
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: 16px;
	}
	.project-filter-page header {
		border-bottom: 1.5px solid var(--ink);
		flex: none;
		min-height: 56px;
		padding: 4px 8px;
		justify-content: flex-start;
		gap: 8px;
	}
	.project-filter-page footer {
		position: sticky;
		bottom: 0;
		background: var(--paper);
		border: 0;
		margin: 0;
		width: 100%;
		flex: none;
	}
	.project-filter-page footer > .button {
		box-shadow: none;
		border: 0;
		font-size: 16px;
	}
	.clear-all-filters {
		font-size: 16px;
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--primary);
	}
	.project-filter-page h1 {
		display: flex;
		align-items: center;
		gap: 6px;
		font-size: 20px;
		margin: 0;
	}

	.feed-craft-icon {
		width: 20px;
		height: 20px;
		flex: none;
		background: currentColor;
		mask: var(--craft-icon) center / contain no-repeat;
	}
</style>
