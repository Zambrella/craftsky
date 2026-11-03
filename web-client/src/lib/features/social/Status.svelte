<script lang="ts">
	import type { Work } from './work.svelte';
	import Skeleton from './Skeleton.svelte';
	import Icon from '../shell/Icon.svelte';
	let {
		work,
		skeleton,
		reload
	}: {
		work: Work;
		skeleton?: 'posts' | 'people' | 'profile';
		reload?: () => void;
	} = $props();
</script>

{#if work.busy}{#if skeleton && !work.hasLoaded}<Skeleton
			kind={skeleton}
		/>{:else}<p role="status" class="social-loading">
			<span aria-hidden="true"></span>Loading…
		</p>{/if}{/if}
{#if work.error}<div class="notice error" role="alert">
		<p>{work.error}</p>
		<Icon name="shield" />
		{#if work.retry || reload}<button
				disabled={work.busy}
				onclick={() => (work.retry ? void work.retry() : reload?.())}
				>Retry</button
			>{/if}
	</div>{/if}

<style>
	.social-loading {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 18px;
		gap: 10px;
		color: var(--ink-2);
		font-size: 14px;
	}
	.social-loading span {
		width: 16px;
		height: 16px;
		border: 2px dashed var(--primary);
		border-radius: 50%;
		animation: spin 1.2s linear infinite;
	}
	.error {
		margin: 16px;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.social-loading span {
			animation: none;
		}
	}
</style>
