<script lang="ts">
	import type { Post } from '#lib/api/types.ts';
	import { safeLink } from './model';
	import { parseYouTubeExternal } from './youtube';
	import YouTubePlayer from './YouTubePlayer.svelte';
	let {
		external,
		compact = false
	}: {
		external: NonNullable<Post['external']>;
		compact?: boolean;
	} = $props();
	const video = $derived(parseYouTubeExternal(external.uri));
	const href = $derived(safeLink(external.uri));
	const host = $derived(href ? new URL(href).host : '');
	const consentKey = 'external_players.youtube.always_allow';
	let playing = $state(false),
		failed = $state(false);
	let dialog = $state<HTMLDialogElement>();
	let trigger = $state<HTMLButtonElement>();
	const id = $props.id();
	function play() {
		try {
			if (localStorage.getItem(consentKey) === 'true') {
				playing = true;
				return;
			}
		} catch {
			/* Ask each time when storage is unavailable. */
		}
		dialog?.showModal();
	}
	function allow(always: boolean) {
		if (always) {
			try {
				localStorage.setItem(consentKey, 'true');
			} catch {
				/* This playback is still allowed. */
			}
		}
		dialog?.close();
		playing = true;
	}
</script>

{#snippet copy()}
	<span class="preview-copy">
		<strong>{external.title}</strong>
		{#if !compact && external.description.trim()}<span class="description"
				>{external.description}</span
			>{/if}
		<span class="host">{host}</span>
	</span>
{/snippet}

{#if href}
	{#if video}
		<div class="external-card youtube-card" class:compact class:playing>
			{#if playing}
				{#if failed}<p class="playback-error" role="status">
						This video can’t be played here. It may be private, unavailable, or
						restricted from embedded playback.
					</p>
				{:else}<YouTubePlayer
						{video}
						title={external.title}
						onerror={() => (failed = true)}
					/>{/if}
				{@render copy()}
				<a class="external-open" {href} rel="ugc nofollow noreferrer"
					>Open in YouTube</a
				>
			{:else}
				<button
					bind:this={trigger}
					class="youtube-preview"
					aria-label={'Play YouTube video: ' + external.title}
					onclick={play}
				>
					<span class="preview-media">
						{#if safeLink(external.thumb?.url || '')}<img
								src={safeLink(external.thumb!.url)}
								alt=""
								loading="lazy"
							/>{/if}
						<span class="play" aria-hidden="true">
							<svg
								width="56"
								height="56"
								viewBox="0 0 56 56"
								fill="none"
								focusable="false"
							>
								<circle
									cx="28"
									cy="28"
									r="24"
									stroke="currentColor"
									stroke-width="5"
								/>
								<path
									d="M23 18.5a2 2 0 0 1 3-1.7l13 9.5a2 2 0 0 1 0 3.4l-13 9.5a2 2 0 0 1-3-1.7z"
									fill="currentColor"
									transform="translate(-2 -1)"
								/>
							</svg>
						</span>
					</span>
					{@render copy()}
				</button>
			{/if}
			<dialog
				bind:this={dialog}
				aria-labelledby={id + '-title'}
				aria-describedby={id + '-description'}
				onclose={() => trigger?.focus({ preventScroll: true })}
			>
				<h2 id={id + '-title'}>Play video from YouTube?</h2>
				<p id={id + '-description'}>
					Playing this video connects to YouTube. YouTube may receive your IP
					address and device information.
				</p>
				<div class="actions">
					<button class="text-action" onclick={() => dialog?.close()}
						>Cancel</button
					><button class="text-action" onclick={() => allow(false)}
						>Allow once</button
					><button class="primary-button" onclick={() => allow(true)}
						>Always allow YouTube</button
					>
				</div>
			</dialog>
		</div>
	{:else}
		<a class="external-card" class:compact {href} rel="ugc nofollow noreferrer">
			{#if compact}
				{#if safeLink(external.thumb?.url || '')}<span class="preview-media"
						><img
							src={safeLink(external.thumb!.url)}
							alt=""
							loading="lazy"
						/></span
					>{/if}
				{@render copy()}
			{:else}
				<strong>{external.title}</strong>
				{#if safeLink(external.thumb?.url || '')}<img
						src={safeLink(external.thumb!.url)}
						alt=""
						loading="lazy"
					/>{/if}
				<p>{external.description}</p>
			{/if}</a
		>
	{/if}
{/if}

<style>
	.compact {
		margin: 0;
		padding: 0;
		overflow: hidden;
		border: 1px solid var(--ink-4);
		border-radius: 6px;
		background: var(--image-frame);
	}
	a.compact {
		display: flex;
		align-items: center;
		min-width: 0;
	}
	.compact:not(.playing) .youtube-preview {
		display: flex;
		align-items: center;
	}
	.compact .preview-media {
		width: 72px;
		height: 72px;
		aspect-ratio: 1;
		flex: none;
	}
	.compact .preview-copy {
		flex: 1;
		min-width: 0;
		padding: 10px;
	}
	.compact .preview-copy strong {
		display: block;
		font-size: 14px;
		white-space: nowrap;
		text-overflow: ellipsis;
	}
	.compact .play svg {
		width: 40px;
		height: 40px;
	}
	img {
		max-height: 240px;
		width: 100%;
		object-fit: cover;
		margin-top: 8px;
	}
	.youtube-card {
		overflow: hidden;
		padding: 0;
		border: 1px solid var(--ink-4);
		border-radius: 6px;
		background: var(--image-frame);
	}
	.youtube-preview {
		display: block;
		width: 100%;
		padding: 0;
		border: 0;
		border-radius: 0;
		box-shadow: none;
		text-align: start;
		color: inherit;
		background: transparent;
	}
	.youtube-preview:hover {
		transform: none;
		background: transparent;
	}
	.youtube-preview:focus-visible {
		outline-offset: -3px;
	}
	.preview-media {
		display: block;
		position: relative;
		aspect-ratio: 16 / 9;
		overflow: hidden;
		background: var(--ink-3);
	}
	.preview-media img {
		display: block;
		width: 100%;
		height: 100%;
		max-height: none;
		margin: 0;
		object-fit: cover;
	}
	.play {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
		color: white;
		background: #0003;
	}
	.preview-copy {
		display: block;
		padding: 12px;
	}
	.preview-copy strong,
	.description {
		display: -webkit-box;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		overflow: hidden;
		overflow-wrap: anywhere;
		line-height: 1.4;
	}
	.preview-copy strong {
		font-size: 16px;
		font-weight: 600;
	}
	.description {
		margin-top: 4px;
		color: var(--ink-2);
		font-size: 14px;
		font-weight: 400;
	}
	.host {
		display: block;
		margin-top: 6px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		color: var(--ink-3);
		font-size: 12px;
		font-weight: 600;
		letter-spacing: 0.12em;
	}
	.external-open {
		display: inline-block;
		margin: 0 12px 12px;
	}
	.playback-error {
		padding: 12px;
	}

	dialog {
		width: min(440px, 80vw);
		padding: 24px;
		border: 0;
		border-radius: 28px;
		box-shadow: none;
		background: color-mix(in srgb, var(--surface) 92%, var(--ink));
	}
	dialog::backdrop {
		background: #0008;
	}
	h2 {
		margin: 0 0 16px;
		font-size: 22px;
		font-weight: 600;
		line-height: 1.3;
	}
	dialog > p {
		margin: 0 0 24px;
		color: var(--ink);
		font-size: 14px;
		line-height: 1.5;
	}
	.actions {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: 8px;
	}
	.actions button {
		max-width: 100%;
		min-height: 40px;
		border: 0;
		box-shadow: none;
		font-size: 16px;
		line-height: 1.5;
	}
	.text-action {
		padding: 8px 12px;
		background: transparent;
		color: var(--primary);
	}
	.text-action:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
		color: var(--primary);
	}
	.primary-button {
		padding: 8px 24px;
		border-radius: 999px;
		background: var(--primary);
		color: var(--on-primary);
	}
	.primary-button:hover {
		background: var(--primary-hover);
		color: var(--on-primary);
	}
</style>
