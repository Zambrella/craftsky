<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';
	let {
		nav,
		active,
		onSelect,
		badge,
		icon,
		trailing
	}: {
		nav: string[][];
		active: (key: string) => boolean;
		onSelect: (event: MouseEvent, key: string) => void;
		badge?: Snippet<[string]>;
		icon?: Snippet<[string]>;
		trailing?: Snippet<[string]>;
	} = $props();
</script>

<nav aria-label="App navigation">
	{#each nav as [key, label, href]}<div class="rail-row">
			<a
				{href}
				aria-current={active(key) ? 'page' : undefined}
				onclick={(event) => onSelect(event, key)}
				><span class="rail-icon"
					>{#if icon}{@render icon(key)}{:else}<Icon
							name={key}
							filled={active(key)}
						/>{/if}</span
				><span>{label === 'Activity' ? 'Notifications' : label}</span
				>{#if badge}{@render badge(key)}{/if}</a
			>
			{#if trailing}{@render trailing(key)}{/if}
		</div>{/each}
</nav>
