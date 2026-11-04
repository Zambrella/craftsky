<script lang="ts">
	import { onDestroy, tick, type Snippet } from 'svelte';
	import { pushState } from '$app/navigation';
	import { page } from '$app/state';
	import { mountComposerPage } from './page-layout';
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
	let trigger: HTMLButtonElement;
	let closing = false;
	let disposed = false;
	async function openPage() {
		await pushState('', { ...page.state, projectDetail: uid });
	}
	async function close() {
		if (closing || disposed) return;
		closing = true;
		if (!(await beforeClose()) || disposed) {
			closing = false;
			return;
		}
		history.back();
	}
	$effect(() => {
		if (page.state.projectDetail === uid) {
			if (!open) onopen();
		} else if (open) void finishBack();
	});
	async function finishBack() {
		const accepted = closing || (await beforeClose());
		if (disposed) return;
		if (!accepted) {
			await pushState('', { ...page.state, projectDetail: uid });
			return;
		}
		onclose();
		closing = false;
		await tick();
		requestAnimationFrame(() => {
			if (!disposed && !open) trigger?.focus({ preventScroll: true });
		});
	}

	onDestroy(() => {
		disposed = true;
	});
</script>

<button
	bind:this={trigger}
	type="button"
	class="project-detail-action"
	aria-label={title}
	aria-describedby={uid + '-description'}
	aria-controls={uid + '-page'}
	aria-expanded={open}
	{disabled}
	onclick={openPage}
>
	<Icon name={icon} /><span
		><strong>{title}</strong><small id={uid + '-description'}
			>{description}</small
		></span
	><Icon name="chevron" />
</button>

<svelte:window
	onkeydown={(event) => {
		if (
			open &&
			event.key === 'Enter' &&
			event.target instanceof HTMLInputElement &&
			event.target.closest('.project-detail-page')
		)
			event.preventDefault();
		if (
			open &&
			event.key === 'Escape' &&
			!event.defaultPrevented &&
			!document.querySelector('.app-main[inert]') &&
			!document.querySelector('dialog[open]')
		) {
			event.preventDefault();
			void close();
		}
	}}
/>
{#if open}
	<div class="project-detail-host">
		<section
			id={uid + '-page'}
			class="project-detail-page composer"
			aria-labelledby={uid + '-title'}
			use:mountComposerPage
		>
			<header class="workspace-header project-detail-heading">
				<button
					type="button"
					class="icon-button project-detail-back"
					aria-label="Back to project"
					onclick={close}><Icon name="back" /></button
				>
				<h1 id={uid + '-title'}>{title}</h1>
			</header>
			<div class="project-detail-scroll">
				<fieldset class="project-detail-fields" {disabled}>
					{@render children()}
				</fieldset>
			</div>
		</section>
	</div>
{/if}

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
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.project-detail-action:disabled {
		opacity: 0.5;
		cursor: default;
	}

	.project-detail-page {
		position: fixed;
		z-index: 25;
		top: 0;
		left: var(--app-content-left, 0px);
		width: var(--app-content-width, 100vw);
		max-width: none;
		min-height: 100dvh;
		height: 100dvh;
		margin: 0;
		background: var(--paper);
	}
	.project-detail-heading {
		flex: none;
		justify-content: flex-start;
		gap: 16px;
		padding: 3px 8px;
		height: 56px;
	}
	.project-detail-heading h1 {
		margin: 0;
		font:
			600 20px/1.25 Outfit,
			sans-serif;
	}
	.project-detail-back {
		color: var(--ink);
		width: 48px;
		height: 48px;
		border: 0;
		border-radius: 50%;
		padding: 12px;
		background: transparent;
		box-shadow: none;
	}
	.project-detail-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
		padding: 24px 16px calc(24px + env(safe-area-inset-bottom));
		overscroll-behavior: contain;
	}
	.project-detail-fields {
		display: grid;
		gap: 16px;
		border: 0;
		padding: 0;
		margin: 0;
		min-width: 0;
	}
</style>
