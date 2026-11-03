<script lang="ts">
	import Icon from '../shell/Icon.svelte';
	import { contextMenu } from './context-menu';
	let {
		label,
		value,
		options,
		disabled = false,
		onchange,
		url
	}: {
		label: string;
		value: string;
		options: { value: string; label: string; icon: string }[];
		disabled?: boolean;
		onchange?: (value: string) => void;
		url?: { href: string };
	} = $props();
	const selected = $derived(
		options.find((option) => option.value === value) || options[0]
	);
	function href(value: string) {
		const target = new URL(url!.href);
		target.searchParams.set('sort', value);
		return target.pathname + target.search + target.hash;
	}
</script>

<details class="choice-menu" use:contextMenu>
	<summary
		aria-label={label + ': ' + selected.label}
		aria-disabled={disabled}
		onclick={(event) => {
			if (disabled) event.preventDefault();
		}}><Icon name={selected.icon} size={20} />{selected.label}</summary
	>
	<div class="menu-items">
		{#each options as option}
			{#if url}<a
					href={href(option.value)}
					aria-current={value === option.value ? 'page' : undefined}
					><Icon name={option.icon} /><span>{option.label}</span
					>{#if value === option.value}<Icon name="check" />{/if}</a
				>
			{:else}<button
					type="button"
					{disabled}
					aria-pressed={value === option.value}
					onclick={() => onchange?.(option.value)}
					><Icon name={option.icon} /><span>{option.label}</span
					>{#if value === option.value}<Icon name="check" />{/if}</button
				>{/if}
		{/each}
	</div>
</details>

<style>
	summary {
		list-style: none;
		min-height: 40px;
		display: flex;
		align-items: center;
		gap: 8px;
		border: 1.5px solid var(--hairline);
		border-radius: 8px;
		padding: 8px 12px;
		font-size: 15px;
		font-weight: 600;
		white-space: nowrap;
	}
	summary::-webkit-details-marker {
		display: none;
	}
	summary :global(svg) {
		color: var(--primary);
	}
	summary[aria-disabled='true'] {
		opacity: 0.5;
		cursor: default;
	}
	.menu-items span {
		flex: 1;
	}
	.menu-items :is([aria-current='page'], [aria-pressed='true']) {
		background: color-mix(in srgb, var(--primary) 4%, var(--surface));
	}
</style>
