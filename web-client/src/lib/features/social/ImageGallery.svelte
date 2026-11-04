<script lang="ts">
	import { onMount, tick, untrack } from 'svelte';
	import { preloadAdjacent } from '#lib/adjacent-image-preloader.ts';
	import Icon from '../shell/Icon.svelte';
	import { safeLink } from './model';
	let {
		images,
		initialIndex = 0,
		onclose
	}: {
		images: { fullsize?: string; thumb?: string; alt: string }[];
		initialIndex?: number;
		onclose: () => void;
	} = $props();
	let dialog: HTMLDialogElement;
	let viewport: HTMLDivElement;
	let caption: HTMLParagraphElement;
	let index = $state(untrack(() => initialIndex)),
		expanded = $state(false),
		overflowing = $state(false),
		failed = $state(false);
	let scale = $state(1),
		x = $state(0),
		y = $state(0);
	const current = $derived(images[index]);
	const displayURLs = $derived(
		images.map((image) => safeLink(image.fullsize || image.thumb || ''))
	);
	const points = new Map<number, { x: number; y: number }>();
	let start = { x: 0, y: 0, panX: 0, panY: 0 },
		pinchDistance = 0,
		pinchScale = 1,
		pinched = false;
	function resetZoom() {
		scale = 1;
		x = y = 0;
	}
	function select(next: number) {
		index = Math.max(0, Math.min(images.length - 1, next));
		expanded = false;
		failed = false;
		resetZoom();
	}
	function measure() {
		if (caption)
			overflowing =
				caption.scrollHeight >
				parseFloat(getComputedStyle(caption).lineHeight) * 2 + 1;
	}
	$effect(() => {
		void current.alt;
		void expanded;
		void tick().then(measure);
	});
	function zoom(next: number) {
		scale = Math.max(1, Math.min(4, next));
		if (scale === 1) x = y = 0;
		else constrain();
	}
	function constrain() {
		const maxX = (viewport.clientWidth * (scale - 1)) / 2,
			maxY = (viewport.clientHeight * (scale - 1)) / 2;
		x = Math.max(-maxX, Math.min(maxX, x));
		y = Math.max(-maxY, Math.min(maxY, y));
	}
	function distance() {
		const [a, b] = [...points.values()];
		return Math.hypot(a.x - b.x, a.y - b.y);
	}
	function down(event: PointerEvent) {
		if (event.button !== 0) return;
		viewport.setPointerCapture(event.pointerId);
		points.set(event.pointerId, { x: event.clientX, y: event.clientY });
		if (points.size === 1) {
			pinched = false;
			start = { x: event.clientX, y: event.clientY, panX: x, panY: y };
		}
		if (points.size === 2) {
			pinched = true;
			pinchDistance = distance();
			pinchScale = scale;
		}
	}
	function move(event: PointerEvent) {
		if (!points.has(event.pointerId)) return;
		points.set(event.pointerId, { x: event.clientX, y: event.clientY });
		if (points.size === 2) {
			if (pinchDistance > 0) zoom((pinchScale * distance()) / pinchDistance);
		} else if (scale > 1 && !pinched) {
			x = start.panX + event.clientX - start.x;
			y = start.panY + event.clientY - start.y;
			constrain();
		}
	}
	function up(event: PointerEvent) {
		const tracked = points.delete(event.pointerId);
		if (!tracked || pinched || scale > 1 || event.type === 'pointercancel')
			return;
		const dx = event.clientX - start.x,
			dy = event.clientY - start.y;
		if (Math.abs(dx) > 50 && Math.abs(dx) > Math.abs(dy))
			select(index + (dx < 0 ? 1 : -1));
		else if (Math.abs(dy) > Math.min(150, viewport.clientHeight * 0.18))
			onclose();
	}
	function key(event: KeyboardEvent) {
		if (event.key === 'ArrowLeft' || event.key === 'ArrowRight') {
			event.preventDefault();
			select(index + (event.key === 'ArrowLeft' ? -1 : 1));
		}
		if (event.key === '+' || event.key === '=') {
			event.preventDefault();
			zoom(scale + 0.5);
		}
		if (event.key === '-') {
			event.preventDefault();
			zoom(scale - 0.5);
		}
		if (event.key === '0') resetZoom();
	}
	onMount(() => {
		const previous = document.activeElement,
			overflow = document.documentElement.style.overflow;
		document.documentElement.style.overflow = 'hidden';
		dialog.showModal();
		const observer = new ResizeObserver(measure);
		observer.observe(caption);
		const wheel = (event: WheelEvent) => {
			if (event.ctrlKey || event.metaKey) {
				event.preventDefault();
				zoom(scale * Math.exp(-event.deltaY * 0.01));
			}
		};
		viewport.addEventListener('wheel', wheel, { passive: false });
		return () => {
			observer.disconnect();
			viewport.removeEventListener('wheel', wheel);
			dialog.close();
			document.documentElement.style.overflow = overflow;
			if (previous instanceof HTMLElement && previous.isConnected)
				previous.focus({ preventScroll: true });
		};
	});
</script>

<dialog
	bind:this={dialog}
	class="image-dialog"
	aria-label="Image gallery"
	oncancel={(event) => {
		event.preventDefault();
		onclose();
	}}
	onkeydown={key}
>
	<div
		class="viewport"
		bind:this={viewport}
		role="presentation"
		onpointerdown={down}
		onpointermove={move}
		onpointerup={up}
		onpointercancel={up}
		ondblclick={() => zoom(scale > 1 ? 1 : 2)}
	>
		{#key index}<img
				src={displayURLs[index]}
				use:preloadAdjacent={{ urls: displayURLs, index }}
				decoding="async"
				alt={current.alt}
				draggable="false"
				style:transform={`translate(${x}px, ${y}px) scale(${scale})`}
				onerror={() => (failed = true)}
			/>{/key}
		{#if failed}<p role="alert" class="image-error">
				This image could not be loaded.
			</p>{/if}
	</div>
	<button class="close" aria-label="Close" onclick={onclose}
		><Icon name="close" size={24} /></button
	>
	{#if images.length > 1}<span class="count" aria-live="polite"
			>{index + 1}/{images.length}</span
		>
		<button
			class="previous"
			aria-label="Previous"
			disabled={index === 0}
			onclick={() => select(index - 1)}><Icon name="back" /></button
		>
		<button
			class="next"
			aria-label="Next"
			disabled={index === images.length - 1}
			onclick={() => select(index + 1)}><Icon name="chevron" /></button
		>{/if}
	<div class="bottom">
		{#if images.length > 1}<div class="dots" aria-label="Images">
				{#each images as _, i}<button
						aria-label={`Image ${i + 1} of ${images.length}`}
						aria-pressed={i === index}
						onclick={() => select(i)}><span></span></button
					>{/each}
			</div>{/if}
		<div class="description">
			<p bind:this={caption} class:expanded>{current.alt}</p>
			{#if overflowing}<button
					class="expand"
					aria-expanded={expanded}
					onclick={() => (expanded = !expanded)}
					>{expanded ? 'Show less' : 'Show more'}</button
				>{/if}
		</div>
	</div>
</dialog>

<style>
	.image-dialog {
		inset: 0;
		margin: 0;
		padding: 0;
		width: 100vw;
		max-width: none;
		height: 100dvh;
		max-height: none;
		border: 0;
		border-radius: 0;
		background: black;
		color: white;
		box-shadow: none;
		overflow: hidden;
	}
	.image-dialog::backdrop {
		background: black;
	}
	.viewport {
		position: absolute;
		inset: 0;
		touch-action: none;
		user-select: none;
		display: grid;
		place-items: center;
		overflow: hidden;
	}
	.viewport img {
		width: 100%;
		height: 100%;
		object-fit: contain;
		user-select: none;
	}
	button {
		color: white;
		background: transparent;
		border: 0;
		box-shadow: none;
		min-height: 44px;
		margin: 0;
	}
	.close,
	.previous,
	.next {
		position: absolute;
		display: grid;
		place-items: center;
		width: 44px;
		height: 44px;
		padding: 10px;
		border-radius: 50%;
		background: #0007;
	}
	.close {
		top: calc(8px + env(safe-area-inset-top));
		left: calc(8px + env(safe-area-inset-left));
	}
	.count {
		position: absolute;
		top: calc(16px + env(safe-area-inset-top));
		right: calc(16px + env(safe-area-inset-right));
		padding: 4px 12px;
		background: #0007;
		border-radius: 6px;
		font-size: 13px;
	}
	.previous,
	.next {
		top: calc(50% - 22px);
	}
	.previous {
		left: 8px;
	}
	.next {
		right: 8px;
	}
	.previous:disabled,
	.next:disabled {
		visibility: hidden;
	}
	.bottom {
		position: absolute;
		bottom: 0;
		width: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
	}
	.dots {
		display: flex;
		align-items: center;
		max-width: calc(100% - 24px);
		overflow-x: auto;
		border: 1px solid #ffffff40;
		border-radius: 999px;
		background: #0007;
		padding: 0 4px;
	}
	.dots button {
		padding: 4px;
		min-width: 16px;
		width: 16px;
		min-height: 20px;
		height: 20px;
		display: grid;
		place-items: center;
	}
	.dots span {
		display: block;
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background: #ffffff66;
	}
	.dots [aria-pressed='true'] span {
		background: white;
	}
	.description {
		width: 100%;
		padding: 12px max(12px, env(safe-area-inset-right))
			calc(12px + env(safe-area-inset-bottom))
			max(12px, env(safe-area-inset-left));
		background: #0009;
	}
	.description p {
		color: white;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
		font-size: 16px;
		line-height: 1.5;
		margin: 0;
		display: -webkit-box;
		-webkit-box-orient: vertical;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		overflow: hidden;
	}
	.description p.expanded {
		display: block;
		max-height: 40dvh;
		overflow-y: auto;
	}
	.expand {
		padding: 8px 12px;
		font-size: 16px;
	}
	.image-error {
		position: absolute;
		top: 45%;
		padding: 16px;
		background: black;
		color: white;
	}
	@media (pointer: coarse) {
		.previous,
		.next {
			display: none;
		}
	}
</style>
