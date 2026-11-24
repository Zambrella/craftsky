<script lang="ts">
	import MultiSelect from '#lib/MultiSelect.svelte';
	import SingleSelect from '#lib/SingleSelect.svelte';
	import type { Json, JsonObject } from '#lib/api/client.ts';
	import { str } from './model';
	import {
		craftChoice,
		label,
		projectDetailOptionsFor,
		projectMethodValues,
		sortedProjectOptions
	} from './project-options';

	let {
		craft,
		details,
		disabled,
		onchange
	}: {
		craft: string;
		details: JsonObject;
		disabled: boolean;
		onchange: (key: string, value: Json | undefined) => void;
	} = $props();

	const kind = $derived(craftChoice(craft)?.profileId);
	const multiFields = {
		knitting: ['knittingMethod', 'Knitting methods'],
		crochet: ['crochetMethod', 'Crochet methods'],
		felting: ['feltingMethod', 'Felting methods']
	} as const;
	const methodField = $derived(
		kind ? multiFields[kind as keyof typeof multiFields] : undefined
	);
</script>

{#snippet single(key: string, field: string, name: string)}
	{@const value = str(details[key])}
	{@const choices = projectDetailOptionsFor(field)}
	<SingleSelect
		label={name}
		{disabled}
		{value}
		options={sortedProjectOptions(
			value && !choices.some((choice) => choice.id === value)
				? [...choices, { id: value, label: label(value) }]
				: choices
		)}
		onchange={(next) => onchange(key, next || undefined)}
	/>
{/snippet}

{#if methodField}
	<MultiSelect
		label={methodField[1]}
		values={projectMethodValues(craft, details)}
		options={sortedProjectOptions(projectDetailOptionsFor(methodField[0]))}
		{disabled}
		onchange={(next) => onchange('methods', next)}
	/>
{:else if kind === 'embroidery'}
	{@render single('method', 'embroideryMethod', 'Method')}
	{@render single('style', 'embroideryStyle', 'Style')}
{:else if kind === 'weaving'}
	{@render single('loomType', 'loomType', 'Loom type')}
{/if}
