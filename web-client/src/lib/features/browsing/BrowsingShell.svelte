<script lang="ts">
	import { onMount, type Snippet } from 'svelte';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import Icon from '../shell/Icon.svelte';
	import RailNavigation from '../shell/RailNavigation.svelte';
	import NavigationFooter from '../shell/NavigationFooter.svelte';
	import { contentBounds } from '../shell/content-bounds';
	let { children, banner }: { children: Snippet; banner?: Snippet } = $props();
	const nav = [
		['home', 'Home', '/feed'],
		['search', 'Search', '/search']
	];
	let drawer = $state(false);
	let menuButton = $state<HTMLButtonElement>();
	let rail = $state<HTMLElement>();
	const shellRoot = $derived(['/feed', '/search'].includes(page.url.pathname));
	function active(key: string) {
		return page.url.pathname.startsWith(key === 'home' ? '/feed' : '/search');
	}
	function closeDrawer() {
		drawer = false;
		menuButton?.focus({ preventScroll: true });
	}
	function select(event: MouseEvent, key: string) {
		drawer = false;
		if (
			key === 'home' &&
			page.url.pathname === '/feed' &&
			!page.url.search &&
			!event.metaKey &&
			!event.ctrlKey &&
			!event.shiftKey &&
			!event.altKey
		) {
			event.preventDefault();
			window.scrollTo({
				top: 0,
				behavior: matchMedia('(prefers-reduced-motion: reduce)').matches
					? 'instant'
					: 'smooth'
			});
			window.dispatchEvent(new Event('craftsky-refresh'));
		}
	}
	afterNavigate(() => {
		drawer = false;
	});
	$effect(() => {
		if (drawer && rail) rail.querySelector<HTMLElement>('a')?.focus();
	});
	onMount(() => {
		const keyboard = (event: KeyboardEvent) => {
			if (!drawer) return;
			if (event.key === 'Escape') {
				event.preventDefault();
				closeDrawer();
			}
			if (event.key === 'Tab' && rail) {
				const items = [...rail.querySelectorAll<HTMLElement>('a,button')];
				const first = items[0],
					last = items.at(-1);
				if (event.shiftKey && document.activeElement === first) {
					event.preventDefault();
					last?.focus();
				} else if (!event.shiftKey && document.activeElement === last) {
					event.preventDefault();
					first?.focus();
				}
			}
		};
		document.addEventListener('keydown', keyboard);
		return () => document.removeEventListener('keydown', keyboard);
	});
</script>

<header class="app-topbar" class:visible={shellRoot} inert={drawer}>
	<button
		bind:this={menuButton}
		class="icon-button mobile-only"
		aria-label="Open navigation"
		aria-expanded={drawer}
		onclick={() => (drawer = true)}><Icon name="menu" /></button
	>
</header>
{#if drawer}<button
		class="drawer-scrim"
		aria-label="Close navigation"
		onclick={closeDrawer}
	></button>{/if}
<aside
	bind:this={rail}
	class="app-sidebar"
	class:open={drawer}
	aria-label="Navigation"
	role={drawer ? 'dialog' : undefined}
	aria-modal={drawer ? true : undefined}
>
	<RailNavigation {nav} {active} onSelect={select} />
	<NavigationFooter />
</aside>
<main
	id="main"
	class="app-main guest-main"
	class:shell-root={shellRoot}
	use:contentBounds
	inert={drawer}
>
	{@render children()}
</main>
<nav class="bottom-nav" aria-label="Main tabs" inert={drawer}>
	{#each nav as [key, label, href]}<a
			{href}
			aria-label={label}
			aria-current={active(key) ? 'page' : undefined}
			onclick={(event) => select(event, key)}
			><Icon name={key} filled={active(key)} /><span class="visually-hidden"
				>{label}</span
			></a
		>{/each}
</nav>
<div inert={drawer}>
	{#if banner}{@render banner()}{/if}
</div>

<style>
	.guest-main {
		padding-bottom: var(--guest-banner-height, 0px);
	}
</style>
