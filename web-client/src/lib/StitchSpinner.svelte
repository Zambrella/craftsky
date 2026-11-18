<script lang="ts">
	let { label = 'Loading' }: { label?: string } = $props();
</script>

<span class="stitch-spinner" role="status" aria-label={label}>
	<svg viewBox="0 0 36 36" width="36" height="36" aria-hidden="true">
		<circle
			cx="18"
			cy="18"
			r="16.5"
			fill="none"
			stroke="currentColor"
			stroke-width="3"
			pathLength="28"
			stroke-dasharray="1 1"
		/>
	</svg>
</span>

<style>
	.stitch-spinner {
		display: inline-flex;
		color: var(--primary);
		width: 36px;
		height: 36px;
	}
	svg {
		animation: stitch-spin 1.4s linear infinite;
	}
	@keyframes stitch-spin {
		to {
			transform: rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		svg {
			animation: none;
		}
	}
</style>
