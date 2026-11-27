<script lang="ts">
	import { label } from './public';
	import { craftIconMask } from './features/social/craft-icons';
	let { craft }: { craft: string } = $props();
	const icon = $derived(craftIconMask(craft));
</script>

<span class="craft-badge"
	>{#if icon}<span
			class="craft-icon"
			style:--craft-icon={icon}
			aria-hidden="true"
		></span>{/if}{label(craft)}</span
>
