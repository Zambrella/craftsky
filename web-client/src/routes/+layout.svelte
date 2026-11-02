<script lang="ts">
	import '../style.css';
	let { children } = $props();
</script>

<a class="skip" href="#main">Skip to content</a>
<header class="site-header">
	<div class="header-content">
		<a class="brand" href="/" aria-label="CraftSky home"
			><img src="/favicon.svg" width="36" height="36" alt="" />CraftSky</a
		><a class="about-link" href="https://craftsky.social">About CraftSky ↗</a>
	</div>
</header>
<main id="main">{@render children()}</main>
<footer>
	<p>A home for textile crafters. Your craft, your community, your data.</p>
	<nav aria-label="Information">
		<a href="https://craftsky.social/privacy">Privacy</a><a
			href="https://craftsky.social/terms">Terms</a
		><a href="https://craftsky.social/reporting">Report a concern</a>
	</nav>
</footer>
