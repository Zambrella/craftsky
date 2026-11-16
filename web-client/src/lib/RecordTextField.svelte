<script lang="ts">
	import {
		recordTextLimit,
		type TextLimit
	} from './features/composer/text-limit';
	import { graphemeCount } from './text';
	let {
		label,
		name,
		value = $bindable(''),
		limit,
		rows = 1,
		required = false,
		placeholder = ''
	}: {
		label: string;
		name: string;
		value?: string;
		limit: TextLimit;
		rows?: number;
		required?: boolean;
		placeholder?: string;
	} = $props();
	const id = $props.id();
</script>

<div class="record-field">
	<label for={id}
		>{label}{#if required}<small aria-hidden="true">required</small>{/if}</label
	>
	{#if rows > 1}<textarea
			{id}
			{name}
			aria-label={label}
			{placeholder}
			{rows}
			{required}
			aria-describedby={`${id}-count`}
			use:recordTextLimit={limit}
			bind:value></textarea>{:else}<input
			{id}
			{name}
			aria-label={label}
			{placeholder}
			{required}
			aria-describedby={`${id}-count`}
			use:recordTextLimit={limit}
			bind:value
		/>{/if}
	<span id={`${id}-count`} class="field-count"
		>{graphemeCount(value)}/{limit.graphemes}</span
	>
</div>

<style>
	.record-field {
		display: grid;
		gap: 8px;
	}
	.field-count {
		text-align: right;
		font-size: 14px;
		color: var(--ink-2);
	}

	.record-field label {
		display: flex;
		flex-direction: row;
		align-items: baseline;
		gap: 8px;
	}
	.record-field small {
		color: var(--ink-2);
		font-size: 12px;
		font-weight: 600;
		letter-spacing: 1.68px;
	}
</style>
