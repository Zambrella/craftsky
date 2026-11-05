<script lang="ts">
	import { authorName, safeURL } from './public';
	type AvatarAuthor = {
		did: string;
		handle: string;
		displayName?: string | null;
		avatar?: string | null;
	};
	let { author, large = false }: { author: AvatarAuthor; large?: boolean } =
		$props();
	let initial = $derived(
		Array.from(
			authorName({
				...author,
				displayName: author.displayName ?? undefined,
				avatar: author.avatar ?? undefined
			}).trim()
		)[0]?.toUpperCase() || '?'
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
	{#if safeURL(author.avatar ?? undefined)}<img
			src={safeURL(author.avatar ?? undefined)}
			alt=""
			width={large ? 96 : 36}
			height={large ? 96 : 36}
			loading={large ? 'eager' : 'lazy'}
		/>{/if}
</span>
