<script lang="ts">
	import { onMount } from 'svelte';
	let {
		title,
		detail,
		confirmLabel,
		busy = false,
		error = '',
		onRetry,
		onConfirm,
		onCancel
	}: {
		title: string;
		detail: string;
		confirmLabel: string;
		busy?: boolean;
		error?: string;
		onRetry?: () => unknown;
		onConfirm: () => void;
		onCancel: () => void;
	} = $props();
	let dialog = $state<HTMLDialogElement>(null!);
	onMount(() => dialog.showModal());
</script>

<dialog
	bind:this={dialog}
	aria-labelledby="confirm-action-title"
	oncancel={(event) => {
		event.preventDefault();
		if (!busy) onCancel();
	}}
>
	<h2 id="confirm-action-title">{title}</h2>
	<p>{detail}</p>
	{#if error}<p class="error" role="alert">{error}</p>{/if}
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
		width: min(440px, calc(100vw - 32px));
		padding: 24px;
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
		align-items: center;
		justify-content: flex-end;
		gap: 12px;
	}
	.error {
		color: var(--red-ink);
	}
</style>
