<script lang="ts">
	import { onMount } from 'svelte';
	import { loadRiveAnimation, type RiveAnimation } from '#lib/rive.ts';
	import animationURL from '../../../../../app/assets/animations/publishing.riv?url&no-inline';
	let { active }: { active: boolean } = $props();
	let canvas: HTMLCanvasElement;
	let source = $state<RiveAnimation>();
	let ready = $state(false);
	let reduced = $state(false);
	let color = $state('#1535d6');
	onMount(() => {
		let disposed = false;
		const motion = matchMedia('(prefers-reduced-motion: reduce)');
		const scheme = matchMedia('(prefers-color-scheme: dark)');
		const sync = () => {
			reduced = motion.matches;
			color = getComputedStyle(document.documentElement)
				.getPropertyValue('--primary')
				.trim();
		};
		sync();
		motion.addEventListener('change', sync);
		scheme.addEventListener('change', sync);
		const observer = new MutationObserver(sync);
		observer.observe(document.documentElement, {
			attributes: true,
			attributeFilter: ['data-theme']
		});
		void loadRiveAnimation(animationURL)
			.then((value) => {
				if (!disposed) source = value;
			})
			.catch(() => {}); // Retain the stitched fallback when decorative assets are unavailable.

		return () => {
			disposed = true;
			observer.disconnect();
			motion.removeEventListener('change', sync);
			scheme.removeEventListener('change', sync);
		};
	});
	$effect(() => {
		ready = false;
		if (!active || !source || !canvas) return;
		const reduceMotion = reduced;
		const threadColor = color;
		const { Rive, Layout, Fit, Alignment } = source.runtime;
		let disposed = false;
		const player = new Rive({
			canvas,
			buffer: source.buffer,
			artboard: 'Publishing',
			stateMachine: reduceMotion ? undefined : 'Publishing',
			animations: reduceMotion ? 'Publishing loop' : undefined,
			autoplay: !reduceMotion,
			autoBind: true,
			enableRiveAssetCDN: false,
			layout: new Layout({ fit: Fit.Contain, alignment: Alignment.Center }),
			onLoad: () => {
				if (disposed) return;
				const rgb = threadColor.match(/^#([\da-f]{6})$/i);
				if (rgb) {
					const value = parseInt(rgb[1], 16);
					player.viewModelInstance
						?.color('threadColor')
						?.rgb((value >> 16) & 255, (value >> 8) & 255, value & 255);
				}
				player.resizeDrawingSurfaceToCanvas();
				if (reduceMotion) {
					player.scrub('Publishing loop', 0.2);
					player.drawFrame();
					player.stopRendering();
				}
				ready = true;
			}
		});
		return () => {
			disposed = true;
			ready = false;
			player.cleanup();
		};
	});
</script>

<div
	class="publishing-animation"
	aria-hidden="true"
	data-ready={ready}
	data-reduced-motion={reduced}
>
	<canvas bind:this={canvas} width="96" height="96" class:ready></canvas>
	{#if !ready}<span class="stitches"></span>{/if}
</div>

<style>
	.publishing-animation {
		width: 48px;
		height: 48px;
		position: relative;
	}
	canvas {
		width: 48px;
		height: 48px;
		opacity: 0;
		display: block;
	}
	canvas.ready {
		opacity: 1;
	}
	.stitches {
		position: absolute;
		inset: 5px;
		border: 3px dashed var(--primary);
		border-top-color: var(--danger-fill);
		border-radius: 50%;
		animation: publishing-spin 2.4s linear infinite;
	}
	@keyframes publishing-spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.stitches {
			animation: none;
		}
	}
</style>
