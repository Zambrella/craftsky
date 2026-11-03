<script lang="ts">
	import { onDestroy, tick, untrack } from 'svelte';
	import { safeURL, type Image } from './public';
	let {
		images,
		detail = false,
		interactive = false,
		initialIndex = 0,
		onChange
	}: {
		images: Image[];
		detail?: boolean;
		interactive?: boolean;
		initialIndex?: number;
		onChange?: (index: number) => void;
	} = $props();
	const pictures = $derived(
		images.filter((image) => !!safeURL(image.fullsize))
	);
	let gallery = $state<HTMLDivElement>();
	let description = $state<HTMLDialogElement>();
	let active = $state(0);
	let signature = '';
	let requested: number | undefined;
	let previousDistance = Infinity;
	let settled: ReturnType<typeof setTimeout> | undefined;
	function clearRequest() {
		requested = undefined;
		clearTimeout(settled);
		settled = undefined;
	}
	function report(index: number) {
		const original = images.indexOf(pictures[index]);
		if (original >= 0) onChange?.(original);
	}
	$effect(() => {
		const next = JSON.stringify(
			pictures.map((image) => [images.indexOf(image), image.fullsize])
		);
		if (next !== signature) {
			signature = next;
			clearRequest();
			active = Math.max(
				0,
				pictures.indexOf(images[untrack(() => initialIndex)])
			);
			untrack(() => report(active));
			description?.close();
			const selected = active;
			void tick().then(() => {
				const left = targetOffset(selected);
				if (left !== undefined)
					gallery?.scrollTo({ left, behavior: 'instant' });
			});
		}
	});
	onDestroy(() => {
		clearRequest();
		description?.close();
	});
	function targetOffset(index: number) {
		const target = gallery?.children[index];
		return gallery && target
			? gallery.scrollLeft +
					target.getBoundingClientRect().left -
					gallery.getBoundingClientRect().left
			: undefined;
	}
	function finish() {
		const index = requested;
		const left = index === undefined ? undefined : targetOffset(index);
		clearRequest();
		// Complete an interrupted programmatic animation when no user gesture
		// replaced it; a menu opening must not change the requested photo.
		if (left !== undefined) gallery?.scrollTo({ left, behavior: 'instant' });
		sync();
	}
	function interrupt() {
		if (requested === undefined) return;
		clearRequest();
		gallery?.scrollTo({ left: gallery.scrollLeft, behavior: 'instant' });
		sync();
	}
	function sync() {
		if (!gallery) return;
		const left = gallery.getBoundingClientRect().left;
		let nearest = 0,
			distance = Infinity;
		for (const [index, node] of [...gallery.children].entries()) {
			const delta = Math.abs(node.getBoundingClientRect().left - left);
			if (delta < distance) {
				nearest = index;
				distance = delta;
			}
		}
		if (requested !== undefined) {
			const target = gallery.children[requested];
			const remaining = target
				? Math.abs(target.getBoundingClientRect().left - left)
				: 0;
			if (remaining > previousDistance + 2) {
				// Also recognise a direct scroll-position change which moves away
				// from the requested page (for example assistive scrolling).
				clearRequest();
			} else if (remaining <= 1) {
				clearRequest();
			} else {
				previousDistance = remaining;
				clearTimeout(settled);
				settled = setTimeout(finish, 120);
				return;
			}
		}
		if (active !== nearest) {
			active = nearest;
			report(nearest);
		}
	}
	function select(index: number, focus = false) {
		if (!gallery) return;
		const target = gallery.children[index] as HTMLAnchorElement | undefined;
		if (!target) return;
		clearRequest();
		const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
		if (!reduced) {
			requested = index;
			previousDistance = Math.abs(
				target.getBoundingClientRect().left -
					gallery.getBoundingClientRect().left
			);
			settled = setTimeout(finish, 120);
		}
		active = index;
		report(index);
		gallery.scrollTo({
			left:
				gallery.scrollLeft +
				target.getBoundingClientRect().left -
				gallery.getBoundingClientRect().left,
			behavior: reduced ? 'instant' : 'smooth'
		});
		if (focus) target.focus({ preventScroll: true });
	}
</script>

{#if pictures.length}<div class="post-images" class:interactive>
		<div
			class="gallery"
			bind:this={gallery}
			role="region"
			aria-label="Post images"
			onscroll={sync}
			onscrollend={finish}
			onpointerdown={interrupt}
			onwheel={interrupt}
		>
			{#each pictures as picture, index}<a
					href={safeURL(picture.fullsize)}
					aria-label={picture.alt || `View image ${index + 1}`}
					onkeydown={(event) => {
						if (
							!interactive ||
							!['ArrowLeft', 'ArrowRight'].includes(event.key)
						)
							return;
						event.preventDefault();
						select(
							Math.min(
								pictures.length - 1,
								Math.max(0, active + (event.key === 'ArrowRight' ? 1 : -1))
							),
							true
						);
					}}
					><img
						src={safeURL(
							detail ? picture.fullsize : picture.thumb || picture.fullsize
						)}
						alt={picture.alt}
						width={picture.aspectRatio?.width}
						height={picture.aspectRatio?.height}
						loading={detail && index === 0 ? 'eager' : 'lazy'}
						fetchpriority={detail && index === 0 ? 'high' : 'auto'}
					/></a
				>{/each}
		</div>
		{#if interactive}
			{#if pictures.length > 1}<span
					class="image-counter"
					aria-live="polite"
					aria-atomic="true">{active + 1}/{pictures.length}</span
				>
				<div class="image-pages" role="group" aria-label="Choose a post image">
					{#each pictures as _, index}<button
							type="button"
							aria-label={`Show image ${index + 1} of ${pictures.length}`}
							aria-pressed={index === active}
							onclick={() => select(index)}><span></span></button
						>{/each}
				</div>{/if}
			{#if pictures[active]?.alt}<button
					class="image-description"
					type="button"
					aria-label="Image description"
					title="Image description"
					onclick={() => description?.showModal()}
					><span aria-hidden="true">T</span></button
				>{/if}
		{/if}
	</div>{/if}
{#if interactive}<dialog
		bind:this={description}
		class="image-description-dialog"
		aria-label="Image description"
	>
		<h2>Image description</h2>
		<p>{pictures[active]?.alt}</p>
		<button type="button" onclick={() => description?.close()}>Close</button>
	</dialog>{/if}

<style>
	.post-images {
		position: relative;
	}
	.interactive .gallery {
		scrollbar-width: none;
	}
	.interactive .gallery::-webkit-scrollbar {
		display: none;
	}
	.image-counter,
	.image-pages,
	.image-description {
		position: absolute;
		z-index: 1;
		background: #161210cc;
		color: #fff;
		border: 0;
		box-shadow: none;
	}
	.image-counter {
		top: 8px;
		right: 8px;
		border-radius: 10px;
		padding: 3px 8px;
		font-size: 12px;
	}
	.image-pages {
		bottom: 10px;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		border-radius: 14px;
		padding: 0 4px;
	}
	.image-pages button {
		width: 18px;
		height: 24px;
		min-height: 24px;
		min-width: 0;
		padding: 0;
		border: 0;
		border-radius: 50%;
		background: transparent;
		box-shadow: none;
		display: grid;
		place-items: center;
	}
	.image-pages button span {
		display: block;
		width: 7px;
		height: 7px;
		background: #aaa;
		border-radius: 50%;
	}
	.image-pages button[aria-pressed='true'] span {
		background: #fff;
	}
	.image-pages button:focus-visible {
		outline: 2px solid white;
		outline-offset: 0;
	}
	.image-description {
		right: 8px;
		bottom: 8px;
		width: 28px;
		height: 28px;
		min-height: 28px;
		padding: 0;
		border-radius: 8px;
		font-family: Georgia, serif;
		font-size: 20px;
		font-weight: 400;
	}
	.image-description:hover {
		background: #161210;
		color: #fff;
	}
	.image-description-dialog {
		width: min(440px, calc(100vw - 32px));
		background: var(--paper);
		color: var(--ink);
	}
	.image-description-dialog h2 {
		margin-bottom: 16px;
		font-size: 20px;
	}
	.image-description-dialog p {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		margin-bottom: 20px;
		font-size: 16px;
	}
</style>
