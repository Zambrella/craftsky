<script lang="ts">
	import Icon from '../shell/Icon.svelte';
	import type { ProfileCustomisation } from '../../api/types';
	import { profileStyle } from './model';
	let {
		size = 16,
		customisation
	}: { size?: number; customisation?: ProfileCustomisation } = $props();
</script>

<span
	class="business-identity-icon"
	style={profileStyle(customisation)}
	style:width={`${size}px`}
	style:height={`${size}px`}
	aria-hidden="true"
>
	<Icon name="business" {size} />
</span>

<style>
	.business-identity-icon {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		line-height: 0;
		color: light-dark(var(--profile-accent), var(--profile-dark));
	}
	/* Phosphor's inset storefront needs a larger drawing than full-canvas crafts. */
	.business-identity-icon :global(svg) {
		flex-shrink: 0;
		transform: scale(1.25);
	}
</style>
