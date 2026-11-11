<script lang="ts">
	import type { Snippet } from 'svelte';
	import SettingsIcon from './SettingsIcon.svelte';
	import './settings.css';
	let {
		section = 'index',
		backHref,
		backLabel = 'Back',
		onBack,
		children
	}: {
		section?: string;
		backHref?: string;
		backLabel?: string;
		onBack?: (event: MouseEvent) => void;
		children: Snippet;
	} = $props();
</script>

<div class="settings-workspace" data-settings-page={section}>
	{#if backHref}<a
			class="back settings-back"
			href={backHref}
			aria-label={backLabel}
			onclick={onBack}><SettingsIcon name="back" /><span>{backLabel}</span></a
		>{/if}
	{@render children()}
</div>
