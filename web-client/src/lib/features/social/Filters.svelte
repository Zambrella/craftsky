<script lang="ts">
	import { page } from '$app/state';
	import { untrack } from 'svelte';
	import { label } from '#lib/public.ts';
	import MultiSelect from '#lib/MultiSelect.svelte';
	import {
		craftChoice,
		sortedProjectOptions
	} from '../composer/project-options';
	import {
		browseFilters,
		filterFamilies,
		filterOptions,
		filterValueLabel,
		projectItemFilterChoices,
		selectedProjectItems,
		sameFilterValue
	} from './filter-model';
	import { craftFilterFamilies } from './craft-detail-filters';
	let { craft = '' }: { craft?: string } = $props();
	const initial = untrack(() => browseFilters(page.url.searchParams, craft));
	let selected = $state<Record<string, string[]>>(
		Object.fromEntries(
			filterFamilies.map(([family]) => [
				family,
				Array.isArray(initial[family]) ? [...initial[family]] : []
			])
		)
	);
	let selfDrafted = $state(page.url.searchParams.get('selfDrafted') === 'true');
	let feedback = $state('');
	const maximum = $derived(craft ? 49 : 50);
	const fieldOrder = [
		'status',
		'patternDifficulty',
		'selfDrafted',
		'projectItem',
		...craftFilterFamilies.map(([key]) => key)
	];
	const itemValues = $derived(selectedProjectItems(craft, selected));
	const itemOptions = $derived(projectItemFilterChoices(craft, itemValues));
	function options(family: string, values: ReturnType<typeof filterOptions>) {
		const active = selected[family] || [];
		return sortedProjectOptions([
			...values.map((option) => ({
				id:
					active.find((value) => sameFilterValue(option.id, value)) ??
					option.id,
				label: option.labels.en,
				aliases: option.aliases
			})),
			...active
				.filter(
					(value) => !values.some((option) => sameFilterValue(option.id, value))
				)
				.map((value) => ({
					id: value,
					label: filterValueLabel(family, craft, value) ?? label(value)
				}))
		]);
	}
	function update(family: string, next: string[]) {
		const nextSelections = { ...selected, [family]: next };
		if (family === 'projectItem') {
			nextSelections.projectType = [];
			nextSelections.projectSubtype = [];
		}
		if (
			next.length > 10 ||
			Object.values(nextSelections).reduce(
				(sum, values) => sum + values.length,
				0
			) > maximum
		) {
			feedback = `Choose up to 10 options in each filter and ${maximum} options across these filters.`;
			return false;
		}
		selected = nextSelections;
		feedback = '';
		return true;
	}
	export function clearAll() {
		selected = Object.fromEntries(
			filterFamilies.map(([family]) => [family, []])
		);
		selfDrafted = false;
		feedback = '';
	}
</script>

<div class="filter-fields">
	{#each fieldOrder as family (family)}
		{#if family === 'selfDrafted'}
			<label class="self-drafted" data-filter={family}
				><span>Self drafted</span><input
					type="checkbox"
					role="switch"
					name="selfDrafted"
					value="true"
					bind:checked={selfDrafted}
				/></label
			>
		{:else if family === 'projectItem'}
			{#if itemOptions.length}
				<div data-filter={family}>
					<MultiSelect
						label={craftChoice(craft)?.profileId === 'dyeing'
							? 'What did you dye?'
							: 'Project type'}
						options={itemOptions}
						values={itemValues}
						onchange={(next) => update(family, next)}
					/>
				</div>
			{/if}
		{:else}
			{@const choices = options(family, filterOptions(family, craft))}
			{@const title =
				filterFamilies.find(([key]) => key === family)?.[1] || family}
			{#if choices.length}
				<div data-filter={family}>
					<MultiSelect
						label={title}
						options={choices}
						values={selected[family]}
						onchange={(next) => update(family, next)}
					/>
				</div>
			{/if}
		{/if}
	{/each}
	{#each Object.entries(selected) as [family, values] (family)}
		{#each values as value (value)}<input
				type="hidden"
				name={family}
				{value}
			/>{/each}
	{/each}
	{#if feedback}<p class="filter-feedback" role="status">{feedback}</p>{/if}
</div>

<style>
	.filter-fields {
		padding: 16px;
		display: grid;
		gap: 16px;
	}
	.self-drafted {
		display: flex;
		flex-direction: row;
		align-items: center;
		justify-content: space-between;
		min-height: 48px;
		font-size: 16px;
		font-weight: 700;
	}
	.self-drafted input {
		appearance: none;
		width: 52px;
		height: 32px;
		border: 1px solid var(--ink-3);
		border-radius: 20px;
		background: var(--paper-2);
		position: relative;
		margin: 0;
	}
	.self-drafted input::after {
		content: '';
		position: absolute;
		left: 4px;
		top: 4px;
		width: 22px;
		height: 22px;
		border-radius: 50%;
		background: var(--ink-3);
	}
	.self-drafted input:checked {
		background: var(--primary);
		border-color: var(--primary);
	}
	.self-drafted input:checked::after {
		left: 24px;
		background: var(--on-primary);
	}
	.filter-feedback {
		color: var(--red-ink);
		margin: 0;
		font-size: 14px;
	}
</style>
