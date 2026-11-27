<script lang="ts">
	import { clearFieldError } from './field-validation';
	import { tick } from 'svelte';
	import Icon from './features/shell/Icon.svelte';
	import {
		matchingSelectOptions,
		selectedOptionLabel,
		selectOptionMatches,
		type SelectOption
	} from './select-options';
	let {
		label,
		helperText,
		value = '',
		options,
		disabled = false,
		required = false,
		clearable = true,
		alwaysSearchable = false,
		placeholder = '',
		showRequired = required,
		onchange
	}: {
		label: string;
		helperText?: string;
		value?: string;
		options: SelectOption[];
		disabled?: boolean;
		required?: boolean;
		clearable?: boolean;
		alwaysSearchable?: boolean;
		placeholder?: string;
		showRequired?: boolean;
		onchange: (value: string) => void;
	} = $props();
	const uid = $props.id();
	let root: HTMLDivElement, input: HTMLInputElement, menu: HTMLDivElement;
	let open = $state(false),
		query = $state(''),
		highlighted = $state(0);
	const searchable = $derived(alwaysSearchable || options.length > 5);
	const choices = $derived(
		searchable ? matchingSelectOptions(options, query) : options
	);
	const parentIds = $derived(new Set(choices.map(({ parentId }) => parentId)));
	const hasHierarchy = $derived(options.some(({ parentId }) => parentId));
	const selectedOption = $derived(
		options.find((option) => option.id === value)
	);
	const selected = $derived(
		selectedOption ? selectedOptionLabel(selectedOption) : value
	);
	function search(value: string) {
		query = value;
		highlighted = Math.max(
			0,
			choices.findIndex((option) => selectOptionMatches(option, query))
		);
	}
	function close() {
		open = false;
		query = '';
		menu?.hidePopover();
	}
	function position() {
		if (!open || !menu || !root) return;
		const rect = input.getBoundingClientRect(),
			viewport = window.visualViewport;
		const top = viewport?.offsetTop || 0,
			bottom = top + (viewport?.height || innerHeight);
		const below = bottom - rect.bottom - 8,
			above = rect.top - top - 8;
		const up = below < 220 && above > below;
		menu.style.width = rect.width + 'px';
		menu.style.left = rect.left + 'px';
		menu.style.maxHeight =
			Math.max(56, Math.min(280, up ? above : below)) + 'px';
		menu.style.top = up ? 'auto' : rect.bottom + 4 + 'px';
		menu.style.bottom = up ? innerHeight - rect.top + 4 + 'px' : 'auto';
	}
	async function show() {
		if (disabled || open) return;
		open = true;
		query = '';
		highlighted = 0;
		await tick();
		if (!open || !menu?.isConnected) return;
		menu.showPopover();
		position();
	}
	function choose(next: string) {
		if (disabled) return;
		onchange(next);
		clearFieldError(input);
		close();
		input.focus({ preventScroll: true });
	}
	function key(event: KeyboardEvent) {
		if (event.key === 'Escape' && open) {
			event.preventDefault();
			event.stopPropagation();
			close();
		} else if (event.key === 'Tab') close();
		else if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
			event.preventDefault();
			if (!open) void show();
			else {
				highlighted = Math.max(
					0,
					Math.min(
						choices.length - 1,
						highlighted + (event.key === 'ArrowDown' ? 1 : -1)
					)
				);
				void tick().then(() =>
					menu
						.querySelector(`[data-index="${highlighted}"]`)
						?.scrollIntoView({ block: 'nearest' })
				);
			}
		} else if (event.key === 'Enter' || (!searchable && event.key === ' ')) {
			event.preventDefault();
			if (!open) void show();
			else if (choices[highlighted]) choose(choices[highlighted].id);
		}
	}
	$effect(() => {
		if (disabled) close();
	});
	function observe(node: HTMLElement) {
		const outside = (event: Event) => {
			if (!node.contains(event.target as Node)) close();
		};
		document.addEventListener('pointerdown', outside, true);
		document.addEventListener('focusin', outside, true);
		document.addEventListener('scroll', position, true);
		window.addEventListener('resize', position);
		window.visualViewport?.addEventListener('resize', position);
		return {
			destroy() {
				document.removeEventListener('pointerdown', outside, true);
				document.removeEventListener('focusin', outside, true);
				document.removeEventListener('scroll', position, true);
				window.removeEventListener('resize', position);
				window.visualViewport?.removeEventListener('resize', position);
			}
		};
	}
</script>

<div
	class="native-single-select"
	class:disabled
	class:open
	bind:this={root}
	use:observe
>
	<label for={uid}
		>{label}{#if showRequired}
			<small class="required-note">required</small>{/if}</label
	>
	<div class="single-field">
		<input
			bind:this={input}
			id={uid}
			data-value={value}
			role="combobox"
			type="text"
			autocomplete="off"
			aria-autocomplete={searchable ? 'list' : 'none'}
			aria-expanded={open}
			aria-controls={uid + '-choices'}
			aria-activedescendant={open && choices[highlighted]
				? uid + '-option-' + highlighted
				: undefined}
			aria-required={required}
			{disabled}
			readonly={!searchable}
			value={open && searchable ? query : selected}
			placeholder={placeholder || (open && selected) || `Select ${label}`}
			onclick={() => void show()}
			ontransitionend={position}
			onkeydown={key}
			oninput={(event) => {
				const entered = event.currentTarget.value;
				if (!open) {
					void show().then(() => search(entered));
				} else search(entered);
			}}
		/>
		<div class="single-adornments">
			{#if value && !required && clearable}<button
					type="button"
					aria-label={'Clear ' + label}
					{disabled}
					onclick={() => choose('')}><Icon name="clear" size={24} /></button
				>{:else}
				<span class:expanded={open && !searchable}
					><Icon name={searchable ? 'search' : 'down'} size={24} /></span
				>{/if}
		</div>
	</div>
	<div
		bind:this={menu}
		id={uid + '-choices'}
		popover="manual"
		role="listbox"
		aria-label={label + ' options'}
		class="single-choices"
	>
		{#if open}{#each choices as option, index (option.id)}
				{@const contextId = uid + '-context-' + index}
				{@const descriptionId = uid + '-description-' + index}
				<button
					type="button"
					role="option"
					id={uid + '-option-' + index}
					data-index={index}
					tabindex="-1"
					aria-selected={option.id === value}
					aria-label={option.label}
					aria-describedby={[
						option.selectionLabel && contextId,
						option.description && descriptionId
					]
						.filter(Boolean)
						.join(' ') || undefined}
					class:highlighted={highlighted === index}
					class:select-option-top-level={hasHierarchy && !option.parentId}
					class:select-option-parent={parentIds.has(option.id)}
					class:select-option-child={Boolean(option.parentId)}
					onpointerdown={(event) => event.preventDefault()}
					onclick={() => choose(option.id)}
					>{option.label}{#if option.selectionLabel}<span id={contextId} hidden
							>{option.selectionLabel}</span
						>{/if}{#if option.description}<small
							id={descriptionId}
							class="option-description">{option.description}</small
						>{/if}</button
				>
			{:else}<p>No results</p>{/each}{/if}
	</div>
	{#if helperText}<p class="field-helper">{helperText}</p>{/if}
</div>

<style>
	.native-single-select > label {
		display: flex;
		flex-direction: row;
		align-items: baseline;
		gap: 8px;
		pointer-events: none;
	}
	.field-helper {
		margin: 0;
		font-size: 14px;
		line-height: 1.5;
		color: var(--ink-2);
	}
	.option-description {
		display: block;
		font-size: 14px;
		font-weight: 400;
		line-height: 1.5;
		color: var(--ink-2);
	}

	.native-single-select {
		min-width: 0;
		display: grid;
		align-content: start;
		gap: var(--field-label-gap);
	}
	label {
		margin: 0;
		font-size: 16px;
		font-weight: 700;
		line-height: 1.3;
	}
	.single-field {
		position: relative;
		padding: 0 3px 3px 0;
	}
	.native-single-select .single-field input[type='text'][role='combobox'] {
		width: 100%;
		margin: 0;
		min-height: var(--field-min-height);
		border-radius: 14px;
		padding: var(--field-padding-block) 52px var(--field-padding-block)
			var(--field-padding-inline);
		background: var(--surface);
		transform: translate(3px, 3px);
		transition: transform 120ms ease-out;
	}
	.single-field input:read-only {
		cursor: pointer;
	}
	.native-single-select
		.single-field:focus-within
		input[type='text'][role='combobox'] {
		transform: translate(0, 0);
	}
	.single-field input:focus:not([aria-invalid='true']) {
		border-color: var(--ink);
		outline: 2px solid var(--ink);
		outline-offset: -2px;
		box-shadow: 3px 3px 0 var(--shadow);
	}
	.single-adornments {
		position: absolute;
		right: calc(var(--field-padding-inline) + 4px);
		top: 0;
		height: calc(100% - 3px);
		transform: translate(3px, 3px);
		transition: transform 120ms ease-out;
		display: flex;
		align-items: center;
		pointer-events: none;
	}
	.single-field:focus-within .single-adornments {
		transform: none;
	}
	@media (prefers-reduced-motion: reduce) {
		.single-field input,
		.single-adornments {
			transition: none;
		}
	}
	.single-adornments span {
		display: grid;
		place-items: center;
		width: 24px;
	}
	.single-adornments span.expanded {
		transform: rotate(180deg);
	}
	.single-adornments button {
		pointer-events: auto;
		display: grid;
		place-items: center;
		width: 40px;
		margin-right: -8px;
		min-height: 44px;
		padding: 0;
		border: 0;
		background: transparent;
		box-shadow: none;
	}
	.disabled {
		opacity: 0.5;
	}
	.single-choices {
		position: fixed;
		inset: auto;
		margin: 0;
		padding: 8px;
		border: 1px solid var(--ink-4);
		border-radius: 14px;
		background: var(--surface);
		color: var(--ink);
		box-shadow: 0 8px 18px rgb(0 0 0 / 15%);
		overflow-y: auto;
		overscroll-behavior: contain;
	}
	.single-choices button {
		display: block;
		width: 100%;
		min-height: 56px;
		padding: 12px 16px;
		text-align: left;
		border: 0;
		border-radius: 6px;
		background: transparent;
		box-shadow: none;
		font-size: 16px;
		font-weight: var(--select-option-weight, 400);
	}
	.single-choices button.highlighted {
		background: color-mix(in srgb, var(--primary-soft) 42%, var(--surface));
		color: var(--primary);
	}
	.single-choices button.select-option-child {
		padding-inline-start: 42px;
	}
	.single-choices button:hover {
		background: color-mix(in srgb, var(--primary) 8%, var(--surface));
	}
	.single-choices p {
		padding: 12px 16px;
	}
</style>
