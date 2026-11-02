<script lang="ts">
	import { page } from '$app/state';
</script>

<svelte:head
	><title
		>{page.status === 404 ? 'Page not found' : 'Unable to load page'} · CraftSky</title
	><meta name="robots" content="noindex,nofollow" /></svelte:head
>
<section class="intro">
	<p class="eyebrow">CraftSky</p>
	<h1>
		{page.status === 404
			? 'This page isn’t available.'
			: 'We couldn’t load this page.'}
	</h1>
	<p>
		{page.status === 404
			? 'It may have been removed, or the link may be incorrect.'
			: 'Please try again in a moment.'}
	</p>
	<a href="/">Back to CraftSky</a>
</section>
