<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	let {
		title,
		detail,
		icon = 'home',
		actions
	}: {
		title: string;
		detail: string;
		icon?: string;
		actions?: Snippet;
	} = $props();
</script>

<section class="social-empty">
	<div class="empty-icon"><Icon name={icon} /></div>
	<h2>{title}</h2>
	<p>{detail}</p>
	{#if actions}<div class="empty-actions">{@render actions()}</div>{/if}
</section>

<style>
	.social-empty {
		max-width: 400px;
		min-height: 260px;
		margin: 0 auto;
		padding: 40px 24px;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		text-align: center;
		gap: 12px;
	}
	.empty-icon {
		color: var(--primary);
		width: 48px;
		height: 48px;
	}
	.empty-icon :global(svg) {
		width: 48px;
		height: 48px;
		stroke-width: 1.6;
	}
	h2 {
		font: inherit;
		font-weight: 600;
		font-size: 22px;
		margin: 0;
	}
	p {
		margin: 0;
		font-size: 15px;
		line-height: 1.5;
		color: var(--ink-2);
	}
	.empty-actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 8px 16px;
		margin-top: 4px;
	}
	.empty-actions :global(a) {
		font-size: 14px;
		font-weight: 600;
	}
</style>
