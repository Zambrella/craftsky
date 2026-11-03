<script lang="ts">
	import { page } from '$app/state';
	import { label } from '#lib/public.ts';
	import MultiSelect from '#lib/MultiSelect.svelte';
	import { filterFamilies, filterOptions } from './filter-model';
	let { craft = '', expanded = false }: { craft?: string; expanded?: boolean } =
		$props();
	let selected = $state<Record<string, string[]>>(
		Object.fromEntries(
			filterFamilies.map(([family]) => [
				family,
				[
					...new Set(
						page.url.searchParams.getAll(family).filter((value) => value.trim())
					)
				]
			])
		)
	);
	let selfDrafted = $state(page.url.searchParams.get('selfDrafted') === 'true');
	let feedback = $state('');
	const maximum = $derived(craft ? 49 : 50);
	const order = [
		'projectType',
		'projectSubtype',
		'patternDifficulty',
		'color',
		'designTag',
		'yarnWeight',
		'piecingTechnique',
		'quiltingMethod',
		'status'
	];
	const titles: Record<string, string> = {
		color: 'Color',
		designTag: 'Design'
	};
	function options(family: string) {
		const values = filterOptions(family, craft, selected.projectType);
		return [
			...values.map((option) => ({ id: option.id, label: option.labels.en })),
			...(selected[family] || [])
				.filter((value) => !values.some((option) => option.id === value))
				.map((value) => ({ id: value, label: label(value) }))
		];
	}
	function update(family: string, next: string[]) {
		if (
			next.length > 10 ||
			Object.entries(selected).reduce(
				(sum, [key, values]) =>
					sum + (key === family ? next.length : values.length),
				0
			) > maximum
		) {
			feedback = `Choose up to 10 options in each filter and ${maximum} options across these filters.`;
			return false;
		}
		selected = { ...selected, [family]: next };
		if (family === 'projectType') {
			const allowed = new Set(
				next.length
					? filterOptions('projectSubtype', craft, next).map(
							(option) => option.id
						)
					: []
			);
			selected = {
				...selected,
				projectSubtype: selected.projectSubtype.filter((value) =>
					allowed.has(value)
				)
			};
		}
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

<div class="filter-panel" class:embedded={expanded}>
	<div class="filter-grid">
		{#each order as family}{@const values = options(family)}{@const title =
				titles[family] ||
				filterFamilies.find(([key]) => key === family)?.[1] ||
				family}
			{#if values.length || family === 'projectSubtype'}<div
					data-filter={family}
				>
					<MultiSelect
						label={title}
						options={values}
						values={selected[family]}
						disabled={family === 'projectSubtype' &&
							!selected.projectType.length}
						onchange={(next) => update(family, next)}
					/>
					{#each selected[family] as value}<input
							type="hidden"
							name={family}
							{value}
						/>{/each}
				</div>{/if}
		{/each}
		<label class="self-drafted"
			><span>Self drafted</span><input
				type="checkbox"
				role="switch"
				name="selfDrafted"
				value="true"
				bind:checked={selfDrafted}
			/></label
		>
	</div>
	{#if feedback}<p class="filter-feedback" role="status">{feedback}</p>{/if}
</div>

<style>
	.filter-panel {
		padding: 16px;
		border: 0;
		border-radius: 0;
		box-shadow: none;
		background: transparent;
	}
	.filter-grid {
		display: grid;
		gap: 24px;
	}
	.self-drafted {
		display: flex;
		flex-direction: row;
		align-items: center;
		justify-content: space-between;
		min-height: 48px;
		font-size: 16px;
		font-weight: 400;
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
		margin-top: 16px;
		font-size: 14px;
	}
</style>
