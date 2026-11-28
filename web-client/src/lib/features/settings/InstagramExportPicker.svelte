<script lang="ts">
	import StitchSpinner from '#lib/StitchSpinner.svelte';
	import SettingsIcon from './SettingsIcon.svelte';

	let {
		accept,
		disabled = false,
		loading = false,
		onSelect
	}: {
		accept: string;
		disabled?: boolean;
		loading?: boolean;
		onSelect: (file: File) => void;
	} = $props();
	let input = $state<HTMLInputElement>();
	const blocked = $derived(disabled || loading);
	const label = 'Select Instagram export';

	function select(event: Event) {
		const picker = event.currentTarget as HTMLInputElement;
		const file = picker.files?.[0];
		if (!file) return;
		picker.value = '';
		onSelect(file);
	}
</script>

<input
	bind:this={input}
	type="file"
	{accept}
	hidden
	aria-label={label}
	disabled={blocked}
	onchange={select}
/>
<button
	type="button"
	class="primary filled-pill"
	disabled={blocked}
	aria-busy={loading || undefined}
	onclick={() => input?.click()}
>
	{#if loading}
		<span class="picker-progress" aria-hidden="true"><StitchSpinner /></span>
	{:else}
		<SettingsIcon name="openFile" size={18} bold />
	{/if}
	{label}
</button>

<style>
	.picker-progress {
		display: flex;
		flex: none;
	}
	.picker-progress :global(.stitch-spinner),
	.picker-progress :global(svg) {
		width: 18px;
		height: 18px;
		color: inherit;
	}
</style>
