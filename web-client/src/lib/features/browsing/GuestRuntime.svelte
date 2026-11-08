<script lang="ts">
	import { onMount, onDestroy, setContext, type Snippet } from 'svelte';
	import { page, navigating } from '$app/state';
	import {
		goto,
		replaceState,
		beforeNavigate,
		afterNavigate
	} from '$app/navigation';
	import { PageHistory, pageHistoryContext } from '../shell/page-history';
	import { PublicCraftskyAPI } from '#lib/api/public-client.ts';
	import { GuestSession } from './guest-session.svelte';
	import { guestRoute } from '#lib/routes.ts';
	import BrowsingShell from './BrowsingShell.svelte';
	import SignInContent from './SignInContent.svelte';
	import { signInContext, type RequestSignIn } from './account-action';
	let { children }: { children: Snippet } = $props();
	const api = new PublicCraftskyAPI();
	setContext('craftsky-api', api);
	const pageHistory = new PageHistory();
	setContext(pageHistoryContext, pageHistory);
	onDestroy(() => pageHistory.dispose());
	let depth = 0;
	let entries: string[] = [];
	let returning: (() => void) | undefined;
	let replacing = false;
	beforeNavigate((navigation) => {
		if (
			!navigation.to ||
			!navigation.from ||
			navigation.to.url.href === navigation.from.url.href
		)
			return;
		const from = navigation.from.url,
			to = navigation.to.url;
		const tab =
			(/^\/feed(?:\/filters)?\/?$/.test(from.pathname) &&
				/^\/feed(?:\/filters)?\/?$/.test(to.pathname)) ||
			(from.pathname === '/search' &&
				to.pathname === '/search' &&
				from.searchParams.get('q') === to.searchParams.get('q'));
		pageHistory.leave(
			depth,
			depth + (navigation.type === 'popstate' ? navigation.delta : 1),
			navigation.type === 'popstate',
			tab
		);
	});
	afterNavigate((navigation) => {
		if (navigation.type === 'enter') {
			depth = 0;
			entries = [];
		} else if (navigation.type === 'popstate')
			depth =
				page.state.navigationIndex ?? Math.max(0, depth + navigation.delta);
		else if (navigation.from && !navigation.willUnload && !replacing) {
			depth++;
			entries = entries.slice(0, depth);
		}
		entries[depth] = page.url.href;
		replaceState('', { ...page.state, navigationIndex: depth });
		pageHistory.arrive(depth, navigation.type === 'popstate');
		returning?.();
		returning = undefined;
		if (ready) consumeSignInEntry();
	});
	setContext('craftsky-return', async (destination: string) => {
		if (navigating.to) await navigating.complete;
		const target = new URL(destination, location.href),
			previous = entries[depth - 1];
		if (
			previous &&
			new URL(previous).pathname === target.pathname &&
			(!target.search || new URL(previous).search === target.search)
		) {
			await new Promise<void>((resolve) => {
				returning = resolve;
				history.back();
			});
		} else {
			replacing = true;
			try {
				await goto(destination, { replace: true });
			} finally {
				replacing = false;
			}
		}
	});
	setContext('craftsky-back', async (event: MouseEvent) => {
		if (
			!event.metaKey &&
			!event.ctrlKey &&
			!event.shiftKey &&
			!event.altKey &&
			(depth > 0 || navigating.to)
		) {
			event.preventDefault();
			const fallback = (event.currentTarget as HTMLAnchorElement).href;
			if (navigating.to) await navigating.complete;
			if (depth > 0) history.back();
			else await goto(fallback);
		}
	});
	let explicitReturnTo = $state<string>();
	function publicDestination(
		raw: string | null | undefined
	): string | undefined {
		if (
			!raw ||
			raw.length > 2048 ||
			!raw.startsWith('/') ||
			raw.startsWith('//')
		)
			return;
		const target = new URL(raw, page.url.origin);
		if (
			target.origin !== page.url.origin ||
			target.hash ||
			!guestRoute(target.pathname)
		)
			return;
		const allowed = new Set(
			/^\/(feed|projects)(?:\/|$)/.test(target.pathname)
				? [
						'craft',
						'sort',
						'cursor',
						'status',
						'projectType',
						'projectSubtype',
						'patternDifficulty',
						'color',
						'designTag',
						'yarnWeight',
						'piecingTechnique',
						'quiltingMethod',
						'selfDrafted'
					]
				: target.pathname.startsWith('/search')
					? ['q', 'tab', 'tag', 'cursor']
					: target.pathname.startsWith('/profiles/')
						? ['tab', 'cursor']
						: target.pathname.startsWith('/posts/')
							? ['focus', 'show-comments', 'imageCid', 'cursor']
							: []
		);
		for (const key of [...target.searchParams.keys()])
			if (!allowed.has(key)) target.searchParams.delete(key);
		return target.pathname + target.search;
	}
	const authReturn = $derived.by(() => {
		const retained = publicDestination(
			explicitReturnTo ??
				(page.url.searchParams.get('auth') === '1'
					? page.url.searchParams.get('returnTo')
					: undefined)
		);
		const url = new URL(page.url.href);
		for (const flag of ['auth', 'error', 'returnTo'])
			url.searchParams.delete(flag);
		return retained ?? publicDestination(url.pathname + url.search) ?? '/feed';
	});
	let ready = $state(false);
	const welcomePending = $derived(
		!ready && /^\/(?:feed(?:\/filters)?)?\/?$/.test(page.url.pathname)
	);
	let bannerHeight = $state(150);
	let guest = $state(new GuestSession());
	const bannerVisible = $derived(!welcomePending && !guest.promptOpen);
	let dialog = $state<HTMLDialogElement>();
	let signInTrigger = $state<HTMLElement>();
	setContext<RequestSignIn>(signInContext, (trigger) => {
		explicitReturnTo = publicDestination(page.url.pathname + page.url.search);
		signInTrigger = trigger;
		guest.signIn();
	});
	function consumeSignInEntry() {
		if (page.url.searchParams.get('auth') === '1') {
			explicitReturnTo = publicDestination(
				page.url.searchParams.get('returnTo')
			);
			guest.signIn();
			const url = new URL(page.url.href);
			url.searchParams.delete('auth');
			url.searchParams.delete('returnTo');
			replaceState(url.pathname + url.search, page.state);
		}
	}

	onMount(() => {
		let storage: Storage | undefined;
		try {
			storage = sessionStorage;
		} catch {}
		guest = new GuestSession(storage);
		guest.enter(page.url.pathname);
		ready = true;
		consumeSignInEntry();
	});
	$effect(() => {
		if (!ready) return;
		guest.enter(page.url.pathname);
	});
	$effect(() => {
		if (!dialog) return;
		if (guest.promptOpen && !dialog.open) {
			dialog.showModal();
			// Native dialog autofocus can select the explainer while the auth
			// controls are still disabled during startup. Start without a focused
			// control; keyboard navigation will focus controls normally.
			if (document.activeElement instanceof HTMLElement)
				document.activeElement.blur();
		} else if (!guest.promptOpen && dialog.open) {
			dialog.close();
			// Safari touch activation does not focus buttons before opening a
			// dialog, so native dialog restoration alone loses the trigger.
			queueMicrotask(() => {
				const target =
					signInTrigger?.isConnected && signInTrigger.getClientRects().length
						? signInTrigger
						: [
								...document.querySelectorAll<HTMLElement>(
									'.app-sidebar a[href="/feed"], .bottom-nav a[href="/feed"]'
								)
							].find((element) => element.getClientRects().length);
				target?.focus({ preventScroll: true });
			});
		}
	});
</script>

<div
	inert={guest.promptOpen}
	aria-hidden={guest.promptOpen}
	class="guest-runtime"
	style:--guest-banner-height={(bannerVisible ? bannerHeight : 0) + 'px'}
>
	<BrowsingShell>
		{@render children()}
		{#snippet banner()}<section
				bind:clientHeight={bannerHeight}
				class="guest-banner"
				hidden={!bannerVisible}
				aria-label="Join CraftSky"
			>
				<div>
					<strong>Your craft. Your community.</strong>
					<p>Join CraftSky to share what you make and join the conversation.</p>
				</div>
				<form
					class="guest-actions"
					method="get"
					action="/sign-in"
					onsubmit={(event) => {
						if (!ready) return;
						event.preventDefault();
						explicitReturnTo = undefined;
						signInTrigger = event.submitter ?? undefined;
						guest.signIn();
					}}
				>
					<input type="hidden" name="returnTo" value={authReturn} />
					<button class="guest-sign-in">Sign in</button>
					<button class="button">Register</button>
				</form>
			</section>{/snippet}
	</BrowsingShell>
</div>
<dialog
	bind:this={dialog}
	aria-label="Join CraftSky"
	oncancel={(event) => {
		event.preventDefault();
		guest.explore();
	}}
	onclose={() => guest.explore()}
>
	<SignInContent
		embedded
		onExplore={() => guest.explore()}
		returnTo={authReturn}
	/>
</dialog>

<style>
	.guest-runtime {
		--guest-rail-width: 256px;
		--guest-navigation-height: 0px;
	}
	@media (max-width: 900px) {
		.guest-runtime {
			--guest-rail-width: 0px;
			--guest-navigation-height: calc(64px + env(safe-area-inset-bottom));
		}
	}

	.guest-runtime :global(.filter-trigger) {
		bottom: calc(
			var(--guest-banner-height) + var(--guest-navigation-height) + 16px
		);
	}
	.guest-runtime :global(.project-filter-page) {
		bottom: calc(var(--guest-banner-height) + var(--guest-navigation-height));
		height: calc(
			100dvh - var(--guest-banner-height) - var(--guest-navigation-height)
		);
	}

	.guest-banner {
		position: fixed;
		bottom: var(--guest-navigation-height);
		left: var(--guest-rail-width);
		right: 0;
		z-index: 30;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 24px;
		padding: 16px max(20px, calc((100vw - 800px) / 2));
		background: var(--paper-2);
		border-top: 1.5px solid var(--ink);
		box-shadow: 0 -3px 12px var(--shadow);
	}
	.guest-banner[hidden] {
		display: none;
	}
	.guest-banner p {
		font-size: 14px;
		margin: 4px 0 0;
	}
	.guest-actions {
		display: flex;
		gap: 12px;
		flex-shrink: 0;
	}
	.guest-sign-in {
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--primary);
	}
	.guest-sign-in:hover {
		background: transparent;
		text-decoration: underline;
	}
	dialog {
		padding: 0;
		border: 0;
		background: transparent;
		box-shadow: none;
		width: 620px;
		max-width: calc(100vw - 24px);
		max-height: calc(100dvh - 32px);
		overflow: auto;
	}
	dialog::backdrop {
		background: rgb(0 0 0 / 0.45);
	}
	@media (max-width: 600px) {
		.guest-banner {
			display: block;
			padding: 12px 16px;
		}
		.guest-banner p {
			font-size: 13px;
		}
		.guest-actions {
			margin-top: 10px;
		}
		.guest-actions button {
			flex: 1;
			text-align: center;
		}
	}
</style>
