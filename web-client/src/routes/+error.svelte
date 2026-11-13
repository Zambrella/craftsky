<script lang="ts">
	import { page } from '$app/state';

	const copy = $derived.by(() => {
		switch (page.status) {
			case 403:
				return {
					title: 'Sign in required',
					heading: 'Sign in to view this content.',
					message: 'Sign in to view this content.'
				};
			case 404:
				return {
					title: 'Page not found',
					heading: 'This page isn’t available.',
					message: 'It may have been removed, or the link may be incorrect.'
				};
			case 400:
				return {
					title: 'Invalid link',
					heading: 'This link isn’t valid.',
					message: 'Check the address or return to CraftSky.'
				};
			case 429:
				return {
					title: 'Too many requests',
					heading: 'Too many requests.',
					message: 'Please wait a moment, then reload this page.'
				};
			default:
				return {
					title: 'Unable to load page',
					heading: 'We couldn’t load this page.',
					message:
						page.status >= 500
							? 'CraftSky is having trouble right now. Please try again.'
							: 'Please try again in a moment.'
				};
		}
	});
</script>

<svelte:head
	><title>{copy.title} · CraftSky</title><meta
		name="robots"
		content="noindex,nofollow"
	/></svelte:head
>
<section class="intro">
	<p class="eyebrow">CraftSky</p>
	<h1>{copy.heading}</h1>
	<p>{copy.message}</p>
	{#if page.status === 403}<a
			href={'/sign-in?' + new URLSearchParams({ returnTo: page.url.pathname })}
			>Sign in / Register</a
		>{:else}<a href="/">Back to CraftSky</a>{/if}
</section>
