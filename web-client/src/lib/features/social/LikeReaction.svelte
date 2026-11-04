<script lang="ts">
	import { onMount } from 'svelte';
	import { loadRiveAnimation, type RiveAnimation } from '#lib/rive.ts';
	import animationURL from '../../../../../app/assets/animations/like.riv?url&no-inline';
	import Icon from '../shell/Icon.svelte';
	import { reactionBounds, type ReactionRect } from './reaction-bounds';
	let { host }: { host: HTMLElement | undefined } = $props();
	let source = $state<RiveAnimation>();
	let sequence = 0;
	let timer: ReturnType<typeof setTimeout> | undefined;
	let playback = $state<{
		sequence: number;
		source: RiveAnimation | undefined;
		reduced: boolean;
	}>();
	let position = $state<{ left: number; top: number; size: number }>();
	let canvas = $state<HTMLCanvasElement>();
	let ready = $state(false);
	let failed = $state(false);
	export function cancel() {
		clearTimeout(timer);
		playback = undefined;
	}
	export function play() {
		cancel();
		const reduced = matchMedia('(prefers-reduced-motion: reduce)').matches;
		playback = { sequence: ++sequence, source, reduced };
		timer = setTimeout(cancel, reduced ? 400 : 1400);
	}
	onMount(() => {
		let disposed = false;
		void loadRiveAnimation(animationURL)
			.then((value) => {
				if (!disposed) source = value;
			})
			.catch(() => {});
		return () => {
			disposed = true;
			cancel();
		};
	});
	$effect(() => {
		if (!playback || !host) return;
		const card = host;
		let frame = 0;
		const measure = () => {
			const rect = card.getBoundingClientRect();
			const viewport = window.visualViewport;
			const clips: ReactionRect[] = [
				{
					left: viewport?.offsetLeft || 0,
					top: viewport?.offsetTop || 0,
					right: (viewport?.offsetLeft || 0) + (viewport?.width || innerWidth),
					bottom: (viewport?.offsetTop || 0) + (viewport?.height || innerHeight)
				}
			];
			for (
				let parent = card.parentElement;
				parent && parent !== document.body;
				parent = parent.parentElement
			) {
				const style = getComputedStyle(parent),
					bounds = parent.getBoundingClientRect();
				const horizontal = /(hidden|clip|auto|scroll)/.test(style.overflowX),
					vertical = /(hidden|clip|auto|scroll)/.test(style.overflowY);
				if (horizontal || vertical)
					clips.push({
						left: horizontal ? bounds.left : -Infinity,
						right: horizontal ? bounds.right : Infinity,
						top: vertical ? bounds.top : -Infinity,
						bottom: vertical ? bounds.bottom : Infinity
					});
			}
			const bounds = reactionBounds(rect, clips),
				style = getComputedStyle(card);
			position = bounds
				? {
						left: bounds.left - rect.left - parseFloat(style.borderLeftWidth),
						top: bounds.top - rect.top - parseFloat(style.borderTopWidth),
						size: bounds.size
					}
				: undefined;
		};
		const schedule = () => {
			cancelAnimationFrame(frame);
			frame = requestAnimationFrame(measure);
		};
		const motion = matchMedia('(prefers-reduced-motion: reduce)');
		motion.addEventListener('change', cancel);
		const resize = new ResizeObserver(schedule);
		resize.observe(card);
		window.addEventListener('scroll', schedule, true);
		window.addEventListener('resize', schedule);
		visualViewport?.addEventListener('scroll', schedule);
		visualViewport?.addEventListener('resize', schedule);
		measure();
		return () => {
			cancelAnimationFrame(frame);
			resize.disconnect();
			motion.removeEventListener('change', cancel);
			window.removeEventListener('scroll', schedule, true);
			window.removeEventListener('resize', schedule);
			visualViewport?.removeEventListener('scroll', schedule);
			visualViewport?.removeEventListener('resize', schedule);
		};
	});
	$effect(() => {
		ready = false;
		failed = false;
		if (!playback?.source || !canvas) return;
		const surface = canvas;
		const { runtime, buffer } = playback.source,
			reduced = playback.reduced;
		let disposed = false;
		const resize = () => {
			player.resizeDrawingSurfaceToCanvas();
			if (reduced) {
				player.drawFrame();
				player.stopRendering();
			}
		};
		const observer = new ResizeObserver(() => {
			if (ready && !disposed) resize();
		});
		const player = new runtime.Rive({
			canvas: surface,
			buffer,
			artboard: 'Like',
			stateMachine: reduced ? undefined : 'Like',
			animations: reduced ? 'Like reaction' : undefined,
			autoplay: !reduced,
			enableRiveAssetCDN: false,
			layout: new runtime.Layout({
				fit: runtime.Fit.Contain,
				alignment: runtime.Alignment.Center
			}),
			onLoadError: () => {
				if (!disposed) failed = true;
			},
			onLoad: () => {
				if (disposed) return;
				if (reduced) player.scrub('Like reaction', 0.65);
				resize();
				ready = true;
				observer.observe(surface);
			}
		});
		return () => {
			disposed = true;
			observer.disconnect();
			ready = false;
			player.cleanup();
		};
	});
</script>

<div class="like-reaction-host" aria-hidden="true" data-asset-ready={!!source}>
	{#if playback}
		<div
			class="like-reaction"
			aria-hidden="true"
			hidden={!position}
			data-ready={ready}
			data-sequence={playback.sequence}
			data-reduced-motion={playback.reduced}
			style:left={`${position?.left || 0}px`}
			style:top={`${position?.top || 0}px`}
			style:width={`${position?.size || 160}px`}
			style:height={`${position?.size || 160}px`}
		>
			{#key playback.sequence}
				{#if playback.source}<canvas
						bind:this={canvas}
						width="320"
						height="320"
						style:opacity={failed ? 0 : 1}
					></canvas>{/if}
				{#if !playback.source || failed}<span class="fallback"
						><Icon name="heart" size={72} filled /></span
					>{/if}
			{/key}
		</div>
	{/if}
</div>

<style>
	.like-reaction-host {
		display: contents;
	}
	.like-reaction {
		position: absolute;
		z-index: 5;
		pointer-events: none;
		display: grid;
		place-items: center;
	}
	.like-reaction[hidden] {
		display: none;
	}
	canvas {
		grid-area: 1 / 1;
		display: block;
		width: 100%;
		height: 100%;
	}
	.fallback {
		grid-area: 1 / 1;
		display: grid;
		place-items: center;
		color: #f03a2e;
		width: 100%;
		height: 100%;
	}
	.fallback :global(svg) {
		display: block;
		width: 45%;
		height: 45%;
	}
</style>
