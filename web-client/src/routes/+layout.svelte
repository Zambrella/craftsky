<script lang="ts">
	import '../style.css';
	import AppShell from '#lib/features/shell/AppShell.svelte';
	import { installFieldValidation } from '#lib/field-validation.ts';
	import { onMount, setContext } from 'svelte';
	import ToastHost from '#lib/features/social/ToastHost.svelte';
	import {
		ToastHub,
		toastContext
	} from '#lib/features/social/toasts.svelte.ts';
	const toasts = new ToastHub();
	setContext(toastContext, toasts);
	$effect(() => {
		const session = page.data.session || page.data.authenticated?.session;
		toasts.scope(session ? session.id + ':' + session.did : 'public');
	});
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
	onMount(installFieldValidation);
	import DiagnosticsHost from '#lib/features/diagnostics/DiagnosticsHost.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	let authenticatedData = $derived(
		page.data.authenticated || (page.data.session ? page.data : undefined)
	);
	let authenticated = $derived(!!authenticatedData);
</script>

<ToastHost hub={toasts} fullWidth={!authenticated} /><DiagnosticsHost
/><StatusHost onBlocking={(value) => (statusBlocking = value)} />
<div inert={statusBlocking}>
	<a class="skip" href="#main">Skip to content</a>
	{#if authenticatedData}
		{#key authenticatedData.session.id}<AppShell
				session={authenticatedData.session}
				accounts={authenticatedData.accounts}>{@render children()}</AppShell
			>{/key}
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
