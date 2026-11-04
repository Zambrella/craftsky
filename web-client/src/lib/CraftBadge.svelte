<script lang="ts">
	import { label } from './public';
	let { craft }: { craft: string } = $props();
	let key = $derived(craft.split('#').at(-1)?.toLowerCase() || '');
	let icon = $derived(
		['sewing', 'knitting', 'crochet', 'quilting', 'embroidery'].includes(key)
			? `url('/crafts/${key}.svg')`
			: undefined
	);
</script>

<span class="craft-badge"
	>{#if icon}<span
			class="craft-icon"
			style:--craft-icon={icon}
			aria-hidden="true"
		></span>{/if}{label(craft)}</span
>
