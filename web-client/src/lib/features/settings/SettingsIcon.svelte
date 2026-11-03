<script lang="ts">
	import Icon from '../shell/Icon.svelte';
	let { name, size = 24 }: { name: string; size?: number } = $props();
</script>

<Icon {name} {size} filled={name === 'subscription'} />
