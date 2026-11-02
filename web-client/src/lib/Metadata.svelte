<script lang="ts">
	import { origin, safeURL } from './public';
	let {
		title,
		description,
		path,
		image,
		imageAlt = ''
	}: {
		title: string;
		description: string;
		path: string;
		image?: string;
		imageAlt?: string;
	} = $props();
</script>

<svelte:head>
	<title>{title} · CraftSky</title>
	<meta name="description" content={description} />
	<link rel="canonical" href={origin + path} />
	<meta property="og:site_name" content="CraftSky" />
	<meta property="og:type" content="website" />
	<meta property="og:title" content={title} />
	<meta property="og:description" content={description} />
	<meta property="og:url" content={origin + path} />
	<meta
		name="twitter:card"
		content={safeURL(image) ? 'summary_large_image' : 'summary'}
	/>
	<meta name="twitter:title" content={title} />
	<meta name="twitter:description" content={description} />
	{#if safeURL(image)}
		<meta property="og:image" content={safeURL(image)} /><meta
			property="og:image:alt"
			content={imageAlt}
		/>
		<meta name="twitter:image" content={safeURL(image)} /><meta
			name="twitter:image:alt"
			content={imageAlt}
		/>
	{/if}
</svelte:head>
