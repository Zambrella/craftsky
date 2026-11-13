<script lang="ts">
	import StitchSpinner from './StitchSpinner.svelte';
	let {
		loading = false,
		disabled = false,
		label = 'Load more',
		loadingLabel = 'Loading more',
		autoLoad = false,
		pageKey,
		onclick
	}: {
		loading?: boolean;
		disabled?: boolean;
		label?: string;
		loadingLabel?: string;
		autoLoad?: boolean;
		pageKey?: string;
		onclick: () => void;
	} = $props();
	let element = $state<HTMLDivElement>();
	// Failed pages wait for an explicit retry, even if an error notice is dismissed.
	let attemptedPage: string | undefined;
	function load() {
		if (loading || disabled) return;
		attemptedPage = pageKey;
		onclick();
	}
	$effect(() => {
		if (
			!autoLoad ||
			!element ||
			!pageKey ||
			loading ||
			disabled ||
			pageKey === attemptedPage ||
			typeof IntersectionObserver === 'undefined'
		)
			return;
		const key = pageKey;
		let active = true;
		const observer = new IntersectionObserver(
			(entries) => {
				if (
					active &&
					autoLoad &&
					!loading &&
					!disabled &&
					pageKey === key &&
					attemptedPage !== key &&
					entries.some((entry) => entry.isIntersecting)
				) {
					observer.disconnect();
					load();
				}
			},
			{ rootMargin: '0px 0px 400px 0px' }
		);
		observer.observe(element);
		return () => {
			active = false;
			observer.disconnect();
		};
	});
</script>

<div class="load-more" bind:this={element}>
	<button
		type="button"
		disabled={loading || disabled}
		aria-label={label}
		aria-busy={loading}
		onclick={load}
		>{#if loading}<StitchSpinner
				label={loadingLabel}
			/>{:else}{label}{/if}</button
	>
</div>

<style>
	.load-more {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 80px;
		padding: 16px;
	}
	.load-more button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--primary);
		padding: 6px 16px;
		height: 48px;
		min-width: 112px;
		border-radius: 999px;
		font:
			600 16px/1.5 Outfit,
			sans-serif;
	}
	.load-more button:disabled {
		opacity: 1;
	}
	.load-more button:hover:not(:disabled) {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
</style>
