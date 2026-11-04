<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	let {
		owner,
		value,
		disabled = false,
		onchange
	}: {
		owner: string;
		value: boolean;
		disabled?: boolean;
		onchange: (value: boolean) => void;
	} = $props();
	let dialog: HTMLDialogElement;
	let trigger = $state<HTMLButtonElement>();
	let disposed = false;
	$effect(() => {
		owner;
		untrack(() => dialog?.close());
	});
	onDestroy(() => {
		disposed = true;
		dialog?.close();
	});
	function show() {
		const rect = trigger?.getBoundingClientRect();
		if (rect && innerWidth > 900) {
			const width = Math.min(280, innerWidth - 32);
			dialog.style.left = `${Math.max(8, Math.min(innerWidth - width - 8, rect.left))}px`;
			dialog.style.top =
				rect.top > innerHeight / 2 ? 'auto' : `${rect.bottom + 4}px`;
			dialog.style.bottom =
				rect.top > innerHeight / 2 ? `${innerHeight - rect.top + 4}px` : 'auto';
		} else {
			dialog.style.removeProperty('left');
			dialog.style.removeProperty('top');
			dialog.style.removeProperty('bottom');
		}
		dialog.showModal();
	}
	function close() {
		dialog.close();
		if (!disposed) trigger?.focus({ preventScroll: true });
	}
</script>

<button
	bind:this={trigger}
	type="button"
	class="sponsored-control"
	class:enabled={value}
	aria-label="Sponsored options"
	aria-pressed={value}
	title="Sponsored"
	{disabled}
	onclick={show}><Icon name="currency" /></button
>
<dialog
	bind:this={dialog}
	class="sponsored-dialog"
	aria-label="Sponsored"
	onclick={(event) => {
		if (event.target === dialog) close();
	}}
	oncancel={(event) => {
		event.preventDefault();
		close();
	}}
>
	<div class="sheet-handle" aria-hidden="true"></div>
	<label
		><Icon name="currency" /><span
			><strong>Sponsored</strong><small
				>This post includes sponsorship or other commercial consideration</small
			></span
		><input
			type="checkbox"
			role="switch"
			aria-label="Sponsored content"
			checked={value}
			{disabled}
			onchange={(event) => {
				onchange(event.currentTarget.checked);
				close();
			}}
		/></label
	>
	<button type="button" class="visually-hidden" onclick={close}
		>Close sponsored options</button
	>
</dialog>

<style>
	.sponsored-dialog > .visually-hidden {
		top: 0;
		left: 0;
		min-width: 0;
		min-height: 0;
		margin: 0;
		padding: 0;
		width: 1px;
		height: 1px;
	}
	.sponsored-dialog > .visually-hidden:focus-visible {
		position: static !important;
		width: auto;
		height: auto;
		margin: 16px 0 0;
		padding: 8px 12px;
		clip-path: none;
		white-space: normal;
	}
	.sponsored-dialog label:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}

	.sponsored-control {
		display: grid;
		place-items: center;
		width: 44px;
		height: 44px;
		padding: 8px;
		border: 0;
		background: transparent;
		box-shadow: none;
	}
	.sponsored-control.enabled {
		color: var(--primary);
	}
	.sponsored-dialog {
		position: fixed;
		inset: auto;
		margin: 0;
		overflow: hidden;
		width: 280px;
		max-width: calc(100vw - 32px);
		padding: 16px;
		background: var(--surface);
		color: var(--ink);
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		box-shadow: 3px 3px 0 var(--ink);
	}
	.sponsored-dialog::backdrop {
		background: transparent;
	}
	.sponsored-dialog label {
		display: flex;
		flex-direction: row;
		align-items: center;
		gap: 12px;
		cursor: pointer;
	}
	.sponsored-dialog span {
		display: grid;
		gap: 4px;
		flex: 1;
		min-width: 0;
	}
	.sponsored-dialog strong {
		font-size: 16px;
		font-weight: 600;
	}
	.sponsored-dialog small {
		color: var(--ink-3);
		font-size: 13px;
		font-weight: 400;
		line-height: 1.5;
	}
	.sheet-handle {
		display: none;
	}
	@media (max-width: 900px) {
		.sponsored-dialog::backdrop {
			background: #0006;
		}
		.sponsored-dialog {
			position: fixed;
			inset: auto 0 0;
			margin: 0;
			max-width: none;
			width: 100%;
			padding: 12px 24px calc(28px + env(safe-area-inset-bottom));
			border: 0;
			border-radius: 20px 20px 0 0;
			box-shadow: none;
		}
		.sheet-handle {
			display: block;
			width: 36px;
			height: 4px;
			margin: 0 auto 24px;
			background: var(--ink-3);
			border-radius: 4px;
		}
	}
</style>
