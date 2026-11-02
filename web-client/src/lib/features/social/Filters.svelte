<script lang="ts">
	import { page } from '$app/state';
	import { label } from '#lib/public.ts';
	import { filterFamilies, filterOptions } from './filter-model';
	let { craft = '' }: { craft?: string } = $props();
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
	let feedback = $state('');
	const maximum = $derived(craft ? 49 : 50);
	function options(family: string) {
		const values = filterOptions(family, craft, selected.projectType);
		return [
			...values,
			...(selected[family] || [])
				.filter((value) => !values.some((option) => option.id === value))
				.map((value) => ({ id: value, labels: { en: label(value) } }))
		];
	}
	function toggle(family: string, id: string, event: Event) {
		const input = event.currentTarget as HTMLInputElement,
			before = selected[family] || [];
		if (
			input.checked &&
			(before.length >= 10 ||
				Object.values(selected).reduce(
					(sum, values) => sum + values.length,
					0
				) >= maximum)
		) {
			input.checked = false;
			feedback = `Choose up to 10 options in each filter and ${maximum} options across these filters.`;
			return;
		}
		const next = input.checked
			? [...before, id]
			: before.filter((value) => value !== id);
		selected = { ...selected, [family]: next };
		feedback = '';
		if (family === 'projectType') {
			const allowed = new Set(
				filterOptions('projectSubtype', craft, next).map((option) => option.id)
			);
			selected = {
				...selected,
				projectSubtype: selected.projectSubtype.filter((value) =>
					allowed.has(value)
				)
			};
		}
	}
	function clear(family: string) {
		selected = { ...selected, [family]: [] };
		feedback = '';
	}
</script>

<details class="filter-panel">
	<summary>Filter projects</summary>
	<p class="filter-help">Choose one or more options in each filter.</p>
	<div class="filter-grid">
		{#each filterFamilies as [family, title]}{@const values =
				options(family)}{#if values.length}<details
					class="filter-family"
					data-filter={family}
				>
					<summary
						><span>{title}</span><span
							class:selected-count={selected[family].length > 0}
							>{selected[family].length
								? `${selected[family].length} selected`
								: 'Any'}</span
						></summary
					>
					<fieldset>
						<legend class="visually-hidden">{title}</legend>
						<div class="filter-options">
							{#each values as option (option.id)}<label
									class="filter-option"
									class:chosen={selected[family].includes(option.id)}
									><input
										type="checkbox"
										name={family}
										value={option.id}
										checked={selected[family].includes(option.id)}
										onchange={(event) => toggle(family, option.id, event)}
									/><span>{option.labels.en}</span></label
								>{/each}
						</div>
					</fieldset>
					{#if selected[family].length}<button
							class="clear-filter"
							type="button"
							onclick={() => clear(family)}>Clear {title.toLowerCase()}</button
						>{/if}
				</details>{/if}{/each}
		<label class="self-drafted"
			><input
				type="checkbox"
				name="selfDrafted"
				value="true"
				checked={page.url.searchParams.get('selfDrafted') === 'true'}
			/><span>Self drafted patterns only</span></label
		>
	</div>
	{#if feedback}<p class="filter-feedback" role="status">{feedback}</p>{/if}
</details>

<style>
	.filter-help {
		font-size: 14px;
		color: var(--ink-3);
		margin: 12px 0 16px;
	}
	.filter-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(240px, 100%), 1fr));
		gap: 12px;
		align-items: start;
	}
	.filter-family {
		border: 1px solid var(--hairline);
		border-radius: 10px;
		background: var(--surface);
		overflow: clip;
	}
	.filter-family > summary {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 12px 14px;
		cursor: pointer;
		list-style: none;
		font-weight: 600;
	}
	.filter-family > summary:after {
		content: '⌄';
		color: var(--ink-3);
	}
	.filter-family[open] > summary:after {
		transform: rotate(180deg);
	}
	.filter-family > summary > span:nth-child(2) {
		margin-left: auto;
		font-size: 12px;
		font-weight: 400;
		color: var(--ink-3);
	}
	.filter-family > summary .selected-count {
		background: var(--primary-soft);
		color: var(--ink);
		padding: 2px 8px;
		border-radius: 20px;
	}
	.filter-family fieldset {
		padding: 0;
		margin: 0;
		border: 0;
		min-width: 0;
	}
	.filter-options {
		max-height: 250px;
		overflow: auto;
		padding: 4px 10px 10px;
		display: grid;
		gap: 4px;
	}
	.filter-option,
	.self-drafted {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 10px;
		border-radius: 8px;
		font-weight: 400;
		cursor: pointer;
	}
	.filter-option input,
	.self-drafted input {
		width: 18px;
		height: 18px;
		flex: none;
		accent-color: var(--primary);
		margin: 0;
	}
	.filter-option:hover {
		background: var(--paper);
	}
	.filter-option.chosen {
		background: var(--primary-soft);
	}
	.clear-filter {
		display: block;
		padding: 10px 14px;
		background: none;
		border: 0;
		color: var(--primary);
		font: inherit;
		font-size: 14px;
		cursor: pointer;
		text-decoration: underline;
	}
	.filter-feedback {
		font-size: 14px;
		color: var(--red-ink);
		margin-top: 12px;
	}
	.self-drafted {
		min-height: 48px;
		border: 1px solid var(--hairline);
	}
	.filter-family > summary:focus-visible,
	.filter-option input:focus-visible,
	.self-drafted input:focus-visible,
	.clear-filter:focus-visible {
		outline: 3px solid var(--primary);
		outline-offset: 3px;
	}
</style>
