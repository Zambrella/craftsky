<script lang="ts">
	import type { Draft } from './store';
	import Icon from '../shell/Icon.svelte';
	let { draft }: { draft: Draft } = $props();
	let url = $state('');
	let failed = $state(false);
	$effect(() => {
		const media = draft.images[0] || draft.externalImage;
		const blob = media?.blob;
		const remote = media?.remoteUrl;
		let next = '';
		if (blob) next = URL.createObjectURL(blob);
		else if (remote) {
			try {
				if (new URL(remote).protocol === 'https:') next = remote;
			} catch {
				/* unavailable preview */
			}
		}
		url = next;
		failed = false;
		return () => {
			if (blob && next) URL.revokeObjectURL(next);
		};
	});
</script>

<span
	class="draft-thumbnail"
	class:with-image={url && !failed}
	aria-hidden="true"
>
	{#if url && !failed}<img
			src={url}
			alt=""
			loading="lazy"
			referrerpolicy="no-referrer"
			onerror={() => (failed = true)}
		/>
	{:else}<Icon name={draft.video ? 'video' : 'document'} size={28} />{/if}
</span>

<style>
	.draft-thumbnail {
		display: grid;
		place-items: center;
		width: 48px;
		height: 48px;
		flex: none;
		color: var(--ink-2);
		border-radius: 8px;
		overflow: hidden;
	}
	.draft-thumbnail.with-image {
		background: var(--paper-2);
	}
	img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
</style>
