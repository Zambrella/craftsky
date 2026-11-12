<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { Profile } from '#lib/api/client.ts';
	import { label } from '#lib/public.ts';
	import Icon from '../shell/Icon.svelte';
	import { Work } from './work.svelte';
	import { shareCardEffects } from './share-card-effects';
	import {
		renderProfileShareCard,
		captureProfileShareCardEffects,
		shareProfileCard
	} from './profile-share-card';
	let {
		profile,
		onClose,
		returnFocus
	}: { profile: Profile; onClose: () => void; returnFocus?: HTMLElement } =
		$props();
	let dialog: HTMLDialogElement;
	let file = $state<File>();
	let url = $state('');
	let exportedURL = $state('');
	const work = new Work('social.Profile.action');
	const id = $props.id();
	function prepare() {
		return work.run(
			async () => {
				const preview = await renderProfileShareCard(profile);
				return {
					preview,
					exported: await captureProfileShareCardEffects(preview)
				};
			},
			(value) => {
				file = value.exported;
				url = URL.createObjectURL(value.preview);
				exportedURL = URL.createObjectURL(value.exported);
			}
		);
	}
	onMount(() => {
		dialog.showModal();
		void prepare();
		return () => {
			dialog.close();
			returnFocus?.focus({ preventScroll: true });
		};
	});
	onDestroy(() => {
		work.dispose();
		if (url) URL.revokeObjectURL(url);
		if (exportedURL) URL.revokeObjectURL(exportedURL);
	});
</script>

<dialog
	bind:this={dialog}
	aria-labelledby={id + '-title'}
	oncancel={(event) => {
		event.preventDefault();
		if (!work.busy || !file) onClose();
	}}
	onclick={(event) => {
		if (event.target === dialog && (!work.busy || !file)) {
			const bounds = dialog.getBoundingClientRect();
			if (
				event.clientX < bounds.left ||
				event.clientX > bounds.right ||
				event.clientY < bounds.top ||
				event.clientY > bounds.bottom
			)
				onClose();
		}
	}}
>
	<header>
		<h2 id={id + '-title'}>Your share card</h2>
		<button
			class="close-card"
			aria-label="Close share card"
			disabled={work.busy && !!file}
			onclick={onClose}><Icon name="close" /></button
		>
	</header>
	<div class="card-stage" use:shareCardEffects={!!url && !work.busy}>
		<div class="artwork" aria-busy={work.busy && !file}>
			{#if url}<img
					src={url}
					width="1200"
					height="1500"
					alt={`${profile.displayName?.trim() || profile.handle}, @${profile.handle}. ${profile.crafts.map(label).join(', ')}. Scan to open this CraftSky profile.`}
				/>
			{:else}<div class="preparing" role="status">
					{work.error ? 'Couldn’t prepare your card.' : 'Loading...'}
				</div>{/if}
		</div>
	</div>
	<button
		class="share-card-button"
		disabled={work.busy}
		onclick={() => {
			if (file) void work.run(() => shareProfileCard(file!, exportedURL));
			else void prepare();
		}}
		aria-busy={work.busy}
	>
		<Icon name="share" /><span
			>{work.busy
				? 'Loading...'
				: !file && work.error
					? 'Try again'
					: 'Share'}</span
		>
	</button>
	{#if work.error}<p class="message" role="alert">{work.error}</p>{/if}
</dialog>

<style>
	dialog {
		width: min(560px, calc(100vw - 48px));
		max-width: none;
		max-height: calc(100svh - 48px);
		margin: auto;
		padding: 20px;
		border: 0;
		border-radius: 0;
		background: transparent;
		box-shadow: none;
		color: #f5efe4;
		overflow-y: auto;
		overflow-x: hidden;
		color-scheme: light;
	}
	dialog::backdrop {
		background: rgb(22 18 16 / 80%);
	}
	header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		margin-bottom: 20px;
	}
	h2 {
		font:
			600 24px Outfit,
			sans-serif;
		margin: 0;
		color: #f5efe4;
	}
	.close-card {
		width: 40px;
		height: 40px;
		min-height: 40px;
		min-width: 40px;
		flex: 0 0 40px;
		aspect-ratio: 1;
		padding: 8px;
		border-radius: 50%;
		background: #f5efe4;
		color: #161210;
		border: 0;
		box-shadow: none;
		display: grid;
		place-items: center;
	}
	.card-stage {
		aspect-ratio: 4 / 5;
		position: relative;
	}
	.card-stage:global(.motion-enabled) {
		touch-action: none;
		cursor: grab;
	}
	.card-stage:global(.dragging) {
		cursor: grabbing;
	}
	.artwork {
		position: relative;
		width: 100%;
		height: 100%;
		transform-origin: center;
		box-shadow:
			inset 1px 1px 2px #ffffff77,
			inset -1px -1px 2px #07186199,
			6px 7px #161210;
		aspect-ratio: 4 / 5;
		border-radius: 4.58% / 3.67%;
		background: #1535d6;
		overflow: hidden;
	}
	.artwork img {
		user-select: none;
		pointer-events: none;
		display: block;
		width: 100%;
		height: 100%;
	}
	.preparing {
		height: 100%;
		display: grid;
		place-items: center;
		font-size: 18px;
	}
	.share-card-button {
		margin: 28px 0 0;
		width: 100%;
		min-height: 48px;
		padding: 12px 24px;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		background: #f5efe4;
		color: #161210;
		border: 2px solid #161210;
		border-radius: 999px;
		box-shadow: 4px 5px #161210;
		font:
			700 16px/20px Outfit,
			sans-serif;
		opacity: 1;
	}
	.share-card-button:disabled,
	.close-card:disabled {
		background: #f5efe4;
		color: #161210;
		opacity: 1;
		cursor: wait;
	}
	.message {
		margin: 14px 0 0;
		color: #f5efe4;
		text-align: center;
		font:
			400 14px/1.5 Outfit,
			sans-serif;
	}
</style>
