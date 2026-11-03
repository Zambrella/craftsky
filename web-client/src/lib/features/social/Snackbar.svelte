<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	let {
		message,
		warning = false,
		onDismiss,
		children
	}: {
		message: string;
		warning?: boolean;
		onDismiss: () => void;
		children?: Snippet;
	} = $props();
	$effect(() => {
		if (!message || warning) return;
		const timer = setTimeout(onDismiss, 4000);
		return () => clearTimeout(timer);
	});
</script>

<div class="snackbar" class:warning role={warning ? 'alert' : 'status'}>
	<span class="severity"><Icon name={warning ? 'warning' : 'info'} /></span>
	<p>{message}</p>
	{#if children || warning}<div class="snackbar-actions">
			{#if children}{@render children()}{/if}
			{#if warning}<button aria-label="Dismiss" onclick={onDismiss}
					><Icon name="close" /></button
				>{/if}
		</div>{/if}
</div>

<style>
	.snackbar {
		position: fixed;
		z-index: 95;
		left: 16px;
		right: 16px;
		bottom: calc(16px + env(safe-area-inset-bottom));
		padding: 14px 16px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		background: var(--primary-soft);
		color: var(--ink);
		box-shadow: 3px 3px var(--shadow);
		display: grid;
		grid-template-columns: 20px minmax(0, 1fr) auto;
		gap: 12px;
		align-items: center;
	}
	.severity {
		color: var(--primary);
		flex-shrink: 0;
	}
	.severity :global(svg) {
		width: 20px;
		height: 20px;
	}
	.snackbar.warning {
		background: var(--butter);
	}
	p {
		margin: 0;
		flex: 1;
		overflow-wrap: anywhere;
	}
	.snackbar-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		align-items: center;
	}
	button {
		padding: 8px;
		background: transparent;
		box-shadow: none;
		border: 0;
		color: var(--primary);
	}
	@media (max-width: 900px) {
		.snackbar {
			bottom: calc(80px + env(safe-area-inset-bottom));
		}
	}
	@media (max-width: 380px) {
		.snackbar-actions {
			grid-column: 2 / -1;
			justify-content: flex-end;
		}
	}
</style>
