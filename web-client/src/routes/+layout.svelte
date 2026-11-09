<script lang="ts">
	import '../style.css';
	import { authenticatedContext } from '#lib/features/shell/page-context.ts';
	import GuestRuntime from '#lib/features/browsing/GuestRuntime.svelte';
	import { guestRoute } from '#lib/routes.ts';

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
	onMount(installFieldValidation);
	import DiagnosticsHost from '#lib/features/diagnostics/DiagnosticsHost.svelte';
	import { page } from '$app/state';
	let { children } = $props();
	let authenticatedData = $derived(authenticatedContext(page.data));
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
	{:else if guestRoute(page.url.pathname)}<GuestRuntime
			>{@render children()}</GuestRuntime
		>
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
				<a href="/safety">Safety and support</a>
				<a href="https://craftsky.social/privacy">Privacy</a><a
					href="https://craftsky.social/terms">Terms</a
				><a href="https://craftsky.social/reporting">Report a concern</a>
			</nav>
		</footer>
	{/if}
</div>
