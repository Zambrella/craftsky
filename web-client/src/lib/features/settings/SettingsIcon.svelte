<script lang="ts">
	import Icon from '../shell/Icon.svelte';
	let {
		name,
		size = 24,
		bold = false
	}: { name: string; size?: number; bold?: boolean } = $props();
</script>

<Icon
	name={name === 'business' ? 'businessIdentity' : name}
	{size}
	{bold}
	filled={name === 'subscription'}
/>
