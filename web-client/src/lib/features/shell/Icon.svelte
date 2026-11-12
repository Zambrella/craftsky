<script lang="ts">
	import PlusTierIcon from './PlusTierIcon.svelte';
	import { nativeIconAliases, nativeIconPaths } from './native-icons.ts';
	let {
		name,
		size = 24,
		filled = false,
		bold = false
	}: {
		name: string;
		size?: number;
		filled?: boolean;
		bold?: boolean;
	} = $props();
	let icon = $derived(
		nativeIconPaths[nativeIconAliases[name] || name] || nativeIconPaths.info
	);
</script>

{#if name === 'subscription'}
	<PlusTierIcon {size} />
{:else}
	<svg
		width={size}
		height={size}
		viewBox="0 0 1024 1024"
		fill="currentColor"
		aria-hidden="true"
		focusable="false"
		><path
			d={bold
				? icon.bold || icon.regular
				: filled
					? icon.fill || icon.regular
					: icon.regular}
		/></svg
	>
{/if}
