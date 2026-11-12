<script lang="ts">
	import { mountProfileShader } from './profile-background-shader.ts';
	let { style }: { style: number } = $props();
	let canvas = $state<HTMLCanvasElement>();
	$effect(() => {
		if (canvas) return mountProfileShader(canvas, style);
	});
</script>

<span class="shader-background" aria-hidden="true">
	{#key style}
		<canvas bind:this={canvas} hidden></canvas>
	{/key}
</span>

<style>
	.shader-background,
	canvas {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		pointer-events: none;
	}
	.shader-background {
		background: linear-gradient(120deg, #00000005, #00000033);
	}
	canvas {
		display: block;
	}
	.shader-background:has(canvas:not([hidden])) {
		background: transparent;
	}
	canvas[hidden] {
		display: none;
	}
</style>
