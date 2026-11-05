<script lang="ts">
	import { onMount } from 'svelte';
	let {
		title,
		detail,
		confirmLabel,
		busy = false,
		onRetry,
		onConfirm,
		onCancel
	}: {
		title: string;
		detail: string;
		confirmLabel: string;
		busy?: boolean;
		onRetry?: () => unknown;
		onConfirm: () => void;
		onCancel: () => void;
	} = $props();
	let dialog = $state<HTMLDialogElement>(null!);
	const uid = $props.id();
	onMount(() => dialog.showModal());
</script>

<dialog
	bind:this={dialog}
	aria-labelledby={uid + '-title'}
	oncancel={(event) => {
		event.preventDefault();
		if (!busy) onCancel();
	}}
>
	<h2 id={uid + '-title'}>{title}</h2>
	<p>{detail}</p>

	<div class="actions">
		<button disabled={busy} onclick={onCancel}>Cancel</button
		>{#if onRetry}<button disabled={busy} onclick={() => onRetry?.()}
				>Retry the same action</button
			>{:else}<button class="danger" disabled={busy} onclick={onConfirm}
				>{confirmLabel}</button
			>{/if}
	</div>
</dialog>

<style>
	dialog {
		width: min(360px, calc(100vw - 48px));
		padding: 24px;
		border-radius: 14px;
		box-shadow: 10px 10px var(--shadow);
	}
	h2 {
		font-size: 20px;
		margin: 0 0 12px;
	}
	p {
		line-height: 1.5;
		margin: 0 0 20px;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: flex-end;
		gap: 12px;
	}
	button.danger {
		background: var(--danger-fill);
		color: white;
		border-color: var(--ink);
		box-shadow: 3px 3px var(--shadow);
	}
	button.danger:hover {
		background: color-mix(in srgb, var(--danger-fill) 88%, black);
	}
	.actions > button:first-child {
		background: transparent;
		border: 0;
		box-shadow: none;
		color: var(--primary);
	}
</style>
