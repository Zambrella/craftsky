<script lang="ts">
	import '../style.css';
	import { onMount } from 'svelte';
	import StatusHost from '#lib/features/status/StatusHost.svelte';
	let statusBlocking = $state(false);
	onMount(() => {
		try {
			const theme = localStorage.getItem('craftsky.theme');
			if (theme === 'light' || theme === 'dark')
				document.documentElement.dataset.theme = theme;
			else delete document.documentElement.dataset.theme;
		} catch {}
	});
	import DiagnosticsHost from '#lib/features/diagnostics/DiagnosticsHost.svelte';
	import { page } from '$app/state';
	let { children } = $props();
</script>

<DiagnosticsHost /><StatusHost
	onBlocking={(value) => (statusBlocking = value)}
/>
<div inert={statusBlocking}>
	<a class="skip" href="#main">Skip to content</a>
	{#if page.url.pathname.startsWith('/app/') || page.url.pathname === '/app'}
		{@render children()}
	{:else if page.url.pathname === '/sign-in'}
		<main id="main" class="welcome-main">{@render children()}</main>
	{:else}
		<header class="site-header">
			<div class="header-content">
				<a class="brand" href="/" aria-label="CraftSky home"
					><img src="/favicon.svg" width="36" height="36" alt="" />CraftSky</a
				><a class="about-link" href="/sign-in">Sign in</a>
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
	{/if}
</div>
