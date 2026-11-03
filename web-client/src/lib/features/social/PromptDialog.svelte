<script lang="ts">
	import { onMount } from 'svelte';
	import type { PromptOptions } from './prompt';
	let {
		title,
		detail,
		confirmLabel,
		danger = false,
		cancelLabel = 'Cancel',
		href,
		target,
		onResult
	}: PromptOptions & { onResult: (accepted: boolean) => void } = $props();
	let dialog: HTMLDialogElement;
	const id = $props.id();
	onMount(() => {
		const previous = document.activeElement;
		dialog.showModal();
		return () => {
			dialog.close();
			if (previous instanceof HTMLElement && previous.isConnected)
				previous.focus({ preventScroll: true });
		};
	});
</script>

<dialog
	bind:this={dialog}
	aria-labelledby={id + '-title'}
	aria-describedby={id + '-detail'}
	oncancel={(event) => {
		event.preventDefault();
		onResult(false);
	}}
	onclick={(event) => {
		if (event.target === dialog) {
			const bounds = dialog.getBoundingClientRect();
			if (
				event.clientX < bounds.left ||
				event.clientX > bounds.right ||
				event.clientY < bounds.top ||
				event.clientY > bounds.bottom
			)
				onResult(false);
		}
	}}
>
	<h2 id={id + '-title'}>{title}</h2>
	<p id={id + '-detail'}>{detail}</p>
	{#if href}<p class="destination">{href}</p>{/if}
	<div class="actions">
		<button class="cancel" onclick={() => onResult(false)}>{cancelLabel}</button
		>
		{#if href}<a
				class="primary-button"
				{href}
				{target}
				rel="noopener noreferrer"
				onclick={() => onResult(true)}>{confirmLabel}</a
			>
		{:else}<button
				class:danger
				class:primary-button={!danger}
				onclick={() => onResult(true)}>{confirmLabel}</button
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
	.destination {
		font-size: 14px;
		font-weight: 600;
		overflow-wrap: anywhere;
		user-select: text;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		align-items: center;
		gap: 12px;
	}
	.cancel {
		background: transparent;
		border: 0;
		box-shadow: none;
		color: var(--primary);
	}
	.primary-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}
</style>
