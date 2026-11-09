<script lang="ts">
	import guide from '../safety/guide-data.json';
	import type { Snippet } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	let {
		message,
		warning = false,
		support = false,
		onDismiss,
		children
	}: {
		message: string;
		warning?: boolean;
		support?: boolean;
		onDismiss: () => void;
		children?: Snippet;
	} = $props();
	let hovered = $state(false);
	let focused = $state(false);
	const paused = $derived(hovered || focused);
	$effect(() => {
		if (!message || warning || paused) return;
		// Preserve the short-message timeout and allow longer confirmations to be read.
		const words = (message + (support ? ' ' + guide.reportSupport : ''))
			.trim()
			.split(/\s+/).length;
		const timer = setTimeout(
			onDismiss,
			Math.min(30000, Math.max(4000, words * 300))
		);
		return () => clearTimeout(timer);
	});
</script>

<div
	class="snackbar"
	class:warning
	role={warning ? 'alert' : 'status'}
	onpointerenter={() => (hovered = true)}
	onpointerleave={() => (hovered = false)}
	onfocusin={() => (focused = true)}
	onfocusout={() => (focused = false)}
>
	<span class="severity"><Icon name={warning ? 'warning' : 'info'} /></span>
	<div>
		<p>{message}</p>
		{#if support}<p>{guide.reportSupport}</p>
			<a href="/safety">Find support</a>{/if}
	</div>
	<div class="snackbar-actions">
		{#if children}{@render children()}{/if}
		<button type="button" aria-label="Dismiss notification" onclick={onDismiss}
			><Icon name="close" /></button
		>
	</div>
</div>

<style>
	.snackbar {
		position: fixed;
		z-index: 95;
		left: calc(var(--app-content-left, 0px) + 16px);
		width: calc(var(--app-content-width, 100vw) - 32px);
		bottom: var(--toast-bottom, calc(16px + env(safe-area-inset-bottom)));
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
		display: flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
	}
	.severity :global(svg) {
		display: block;
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
	button,
	.snackbar-actions :global(button),
	.snackbar-actions :global(a) {
		padding: 8px;
		background: transparent;
		box-shadow: none;
		border: 0;
		color: var(--ink);
	}
	@media (max-width: 900px) {
		.snackbar {
			bottom: var(--toast-bottom, calc(80px + env(safe-area-inset-bottom)));
		}
	}
	@media (max-width: 380px) {
		.snackbar-actions {
			grid-column: 2 / -1;
			justify-content: flex-end;
		}
	}
</style>
