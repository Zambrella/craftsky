<script lang="ts">
	import { authorName, safeURL, type Author } from './public';
	let { author, large = false }: { author: Author; large?: boolean } = $props();
	let initial = $derived(
		Array.from(authorName(author).trim())[0]?.toUpperCase() || '?'
	);
	let swatch = $derived(
		['butter', 'clay', 'moss', 'sky', 'lilac', 'paper'][
			initial.charCodeAt(0) % 6
		]
	);
</script>

<span
	class:profile-avatar={large}
	class="avatar"
	data-swatch={swatch}
	aria-hidden="true"
>
	<span>{initial}</span>
	{#if safeURL(author.avatar)}<img
			src={safeURL(author.avatar)}
			alt=""
			width={large ? 96 : 36}
			height={large ? 96 : 36}
			loading={large ? 'eager' : 'lazy'}
		/>{/if}
</span>
