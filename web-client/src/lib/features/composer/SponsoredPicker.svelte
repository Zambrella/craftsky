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
	onclick={() => dialog.showModal()}><Icon name="currency" /></button
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
		margin: auto;
		width: 400px;
		max-width: calc(100vw - 32px);
		padding: 24px;
		background: var(--paper);
		color: var(--ink);
		border: 1.5px solid var(--ink);
		border-radius: 20px;
		box-shadow: 3px 3px 0 var(--ink);
	}
	.sponsored-dialog::backdrop {
		background: #0006;
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
	.sponsored-dialog input {
		appearance: none;
		position: relative;
		width: 44px;
		height: 26px;
		margin: 0;
		flex: none;
		border: 1.5px solid var(--ink);
		border-radius: 20px;
		background: var(--paper-2);
		cursor: pointer;
	}
	.sponsored-dialog input::before {
		content: '';
		display: block;
		position: absolute;
		width: 18px;
		height: 18px;
		left: 2.5px;
		top: 2.5px;
		border: 1px solid var(--ink);
		border-radius: 50%;
		background: var(--paper);
		transition: transform 150ms;
	}
	.sponsored-dialog input:checked {
		background: var(--primary);
	}
	.sponsored-dialog input:checked::before {
		transform: translateX(17px);
	}
	.sheet-handle {
		display: none;
	}
	@media (max-width: 700px) {
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
	@media (prefers-reduced-motion: reduce) {
		.sponsored-dialog input::before {
			transition: none;
		}
	}
</style>
