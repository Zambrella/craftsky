<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head
	><title
		>{page.status === 403
			? 'Sign in required'
			: page.status === 404
				? 'Page not found'
				: 'Unable to load page'} · CraftSky</title
	><meta name="robots" content="noindex,nofollow" /></svelte:head
>
<section class="intro">
	<p class="eyebrow">CraftSky</p>
	<h1>
		{page.status === 403
			? 'Sign in to view this content.'
			: page.status === 404
				? 'This page isn’t available.'
				: 'We couldn’t load this page.'}
	</h1>
	<p>
		{page.status === 403
			? 'Sign in to view this content.'
			: page.status === 404
				? 'It may have been removed, or the link may be incorrect.'
				: 'Please try again in a moment.'}
	</p>
	{#if page.status === 403}<a
			href={'/sign-in?' + new URLSearchParams({ returnTo: page.url.pathname })}
			>Sign in / Register</a
		>{:else}<a href="/">Back to CraftSky</a>{/if}
</section>
