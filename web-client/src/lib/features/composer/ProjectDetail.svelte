<script lang="ts">
	import { onDestroy, type Snippet } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	let {
		title,
		description,
		icon,
		open,
		disabled = false,
		onopen,
		onclose,
		beforeClose = () => true,
		children
	}: {
		title: string;
		description: string;
		icon: string;
		open: boolean;
		disabled?: boolean;
		onopen: () => void;
		onclose: () => void;
		beforeClose?: () => boolean | Promise<boolean>;
		children: Snippet;
	} = $props();
	const uid = $props.id();
	let dialog = $state<HTMLDialogElement>();
	let trigger: HTMLButtonElement;
	let disposed = false;
	let closing = false;
	$effect(() => {
		const node = dialog;
		if (!node) return;
		if (open && !node.open) node.showModal();
		else if (!open && node.open) node.close();
	});
	onDestroy(() => {
		disposed = true;
		dialog?.close();
	});
	async function close() {
		if (disposed || closing) return;
		closing = true;
		if (!(await beforeClose()) || disposed) {
			closing = false;
			return;
		}
		// Close while the focused child still exists, then release its snippet.
		// Return explicitly to the action row, including browsers that do not focus clicked buttons.
		dialog?.close();
		onclose();
		if (trigger?.isConnected && !trigger.disabled)
			trigger.focus({ preventScroll: true });
		closing = false;
	}
</script>

<button
	bind:this={trigger}
	type="button"
	class="project-detail-action"
	aria-label={title}
	aria-describedby={uid + '-description'}
	aria-haspopup="dialog"
	aria-controls={uid + '-dialog'}
	aria-expanded={open}
	{disabled}
	onclick={onopen}
>
	<Icon name={icon} /><span
		><strong>{title}</strong><small id={uid + '-description'}
			>{description}</small
		></span
	><Icon name="chevron" />
</button>
<dialog
	id={uid + '-dialog'}
	bind:this={dialog}
	class="project-detail-dialog"
	aria-labelledby={uid + '-title'}
	aria-modal="true"
	oncancel={(event) => {
		event.preventDefault();
		close();
	}}
	onclose={() => {
		if (open) close();
	}}
>
	{#if open}
		<header class="project-detail-heading">
			<button
				type="button"
				class="project-detail-back"
				aria-label="Back to project"
				onclick={close}><Icon name="back" /></button
			>
			<h2 id={uid + '-title'}>{title}</h2>
		</header>
		<div class="project-detail-scroll">
			<fieldset class="project-detail-fields" {disabled}>
				{@render children()}
			</fieldset>
		</div>
	{/if}
</dialog>

<style>
	.project-detail-action {
		display: flex;
		align-items: center;
		gap: 16px;
		width: 100%;
		padding: 16px 8px;
		min-height: 80px;
		background: transparent;
		border: 0;
		border-bottom: 1px solid var(--hairline);
		border-radius: 0;
		box-shadow: none;
		color: var(--ink);
		text-align: left;
		font: inherit;
		cursor: pointer;
	}
	.project-detail-action > span {
		flex: 1;
		min-width: 0;
		display: grid;
		gap: 4px;
	}
	.project-detail-action strong {
		font-size: 16px;
		font-weight: 500;
	}
	.project-detail-action small {
		color: var(--ink-3);
		font-size: 14px;
		font-weight: 400;
		line-height: 1.5;
	}
	.project-detail-action:hover {
		background: var(--paper-2);
	}
	.project-detail-action:disabled {
		opacity: 0.5;
		cursor: default;
	}
	.project-detail-dialog {
		margin: auto;
		padding: 0;
		width: min(640px, calc(100vw - 48px));
		height: min(760px, calc(100dvh - 48px));
		max-width: none;
		max-height: none;
		overflow: hidden;
		border: 1.5px solid var(--ink);
		border-radius: 16px;
		background: var(--paper);
		color: var(--ink);
		box-shadow: 4px 4px 0 var(--ink);
	}
	.project-detail-dialog[open] {
		display: flex;
		flex-direction: column;
	}
	.project-detail-dialog::backdrop {
		background: #0006;
	}
	.project-detail-heading {
		display: flex;
		align-items: center;
		gap: 12px;
		min-height: 56px;
		height: 56px;
		padding: 8px 16px;
		border-bottom: 1.5px solid var(--ink);
		background: var(--paper);
		flex: none;
	}
	.project-detail-heading h2 {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-size: 20px;
		font-weight: 700;
		font-family: inherit;
	}
	.project-detail-back {
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		min-height: 40px;
		padding: 8px;
		margin-left: -8px;
		border: 0;
		border-radius: 50%;
		background: transparent;
		color: var(--ink);
		box-shadow: none;
	}
	.project-detail-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
		overscroll-behavior: contain;
		padding: 24px 16px calc(24px + env(safe-area-inset-bottom));
	}
	.project-detail-fields {
		display: grid;
		gap: 16px;
		border: 0;
		border-radius: 0;
		padding: 0;
		margin: 0;
		min-width: 0;
	}
	@media (max-width: 700px) {
		.project-detail-dialog {
			inset: 0;
			margin: 0;
			width: 100vw;
			height: 100dvh;
			border: 0;
			border-radius: 0;
			box-shadow: none;
		}
	}
</style>
