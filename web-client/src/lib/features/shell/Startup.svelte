<div class="startup" role="status" aria-label="Loading your CraftSky">
	<img src="/startup-logo.png" width="176" height="176" alt="" />
</div>

<style>
	.startup {
		min-height: 100svh;
		display: grid;
		place-items: center;
	}
	img {
		display: block;
		animation: breathe 2.8s ease-in-out infinite;
	}
	@keyframes breathe {
		0%,
		100% {
			transform: scale(1);
			opacity: 1;
		}
		50% {
			transform: scale(1.04);
			opacity: 0.85;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		img {
			animation: none;
		}
	}
</style>
