<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	let { children, banner }: { children: Snippet; banner?: Snippet } = $props();
</script>

<header class="browsing-header">
	<a class="brand" href="/feed"
		><img src="/favicon.svg" width="36" height="36" alt="" />CraftSky</a
	>
	<nav aria-label="Main navigation">
		<a href="/feed"><Icon name="home" />Home</a><a href="/search"
			><Icon name="search" />Search</a
		>
	</nav>
</header>
<main id="main" class="browsing-main">{@render children()}</main>
{#if banner}{@render banner()}{/if}

<style>
	.browsing-header {
		position: sticky;
		top: 0;
		z-index: 20;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 24px;
		padding: 12px 24px;
		border-bottom: 1px solid var(--hairline);
		background: var(--paper);
	}
	.brand {
		display: flex;
		align-items: center;
		gap: 10px;
		text-decoration: none;
		font-weight: 700;
		font-size: 22px;
	}
	nav {
		display: flex;
		gap: 24px;
	}
	nav a {
		display: flex;
		align-items: center;
		gap: 8px;
		text-decoration: none;
	}
	.browsing-main {
		width: min(100%, 800px);
		margin: 0 auto;
		padding: 24px 16px calc(var(--guest-banner-height, 150px) + 24px);
	}
	@media (max-width: 600px) {
		.browsing-header {
			padding: 10px 16px;
			gap: 12px;
		}
		.brand {
			font-size: 18px;
		}
		nav {
			gap: 16px;
		}
		nav a {
			gap: 4px;
			font-size: 14px;
		}
		.browsing-main {
			padding: 12px 12px calc(var(--guest-banner-height, 150px) + 24px);
		}
	}
</style>
