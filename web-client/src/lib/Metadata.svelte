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
	// Anonymous responses can omit images pending safety clearance. Always give
	// crawlers a public image without bypassing that filtering.
	const shareImage = $derived(safeURL(image) || origin + '/social-share.png');
	const shareImageAlt = $derived(safeURL(image) ? imageAlt : 'CraftSky');
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
	<meta name="twitter:card" content="summary_large_image" />
	<meta name="twitter:title" content={title} />
	<meta name="twitter:description" content={description} />
	<meta property="og:image" content={shareImage} /><meta
		property="og:image:alt"
		content={shareImageAlt}
	/>
	<meta name="twitter:image" content={shareImage} /><meta
		name="twitter:image:alt"
		content={shareImageAlt}
	/>
</svelte:head>
