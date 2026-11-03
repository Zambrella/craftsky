<script lang="ts">
	import { tick } from 'svelte';
	import Icon from './features/shell/Icon.svelte';
	let {
		label,
		values,
		options,
		disabled = false,
		maximum = 10,
		placeholder = '',
		onchange
	}: {
		label: string;
		values: string[];
		options: { id: string; label: string }[];
		disabled?: boolean;
		maximum?: number;
		placeholder?: string;
		onchange: (values: string[]) => boolean | void;
	} = $props();
	const uid = $props.id();
	let root: HTMLDivElement, input: HTMLInputElement;
	let open = $state(false),
		query = $state(''),
		above = $state(false),
		available = $state(280);
	let suppressFocus = false;
	const matches = $derived(
		options.filter((option) =>
			option.label
				.toLocaleLowerCase()
				.includes(query.trim().toLocaleLowerCase())
		)
	);
	const selected = $derived(
		values.map((id) => ({
			id,
			label: options.find((option) => option.id === id)?.label || id
		}))
	);
	function close(restore = false) {
		open = false;
		query = '';
		if (restore && input && document.activeElement !== input) {
			suppressFocus = true;
			input.focus({ preventScroll: true });
		}
	}
	function position() {
		if (!root) return;
		const box = root.getBoundingClientRect(),
			viewport = window.visualViewport;
		let top = viewport?.offsetTop || 0,
			bottom = top + (viewport?.height || innerHeight);
		for (
			let parent = root.parentElement;
			parent && parent !== document.body;
			parent = parent.parentElement
		) {
			if (
				/(auto|scroll|hidden|clip)/.test(getComputedStyle(parent).overflowY)
			) {
				const bounds = parent.getBoundingClientRect();
				top = Math.max(top, bounds.top);
				bottom = Math.min(bottom, bounds.bottom);
			}
		}
		const below = bottom - box.bottom - 12,
			before = box.top - top - 12;
		above = below < 220 && before > below;
		available = Math.max(100, Math.min(280, above ? before : below));
	}
	$effect(() => {
		if (open) void tick().then(position);
	});
	function outside(node: HTMLElement) {
		const dismiss = (event: Event) => {
			if (!node.contains(event.target as Node)) close();
		};
		document.addEventListener('pointerdown', dismiss, true);
		document.addEventListener('focusin', dismiss, true);
		window.addEventListener('resize', position);
		document.addEventListener('scroll', position, true);
		window.visualViewport?.addEventListener('resize', position);
		return {
			destroy() {
				document.removeEventListener('pointerdown', dismiss, true);
				document.removeEventListener('focusin', dismiss, true);
				window.removeEventListener('resize', position);
				document.removeEventListener('scroll', position, true);
				window.visualViewport?.removeEventListener('resize', position);
			}
		};
	}
</script>

<div
	class="native-multi-select"
	class:open
	class:disabled
	bind:this={root}
	use:outside
>
	<label for={uid}>{label}</label>
	<div class="selection-field">
		{#if selected.length}<div class="selection-values">
				{#each selected as value (value.id)}<button
						type="button"
						class="selected-chip"
						aria-label={`Remove ${value.label} from ${label}`}
						{disabled}
						onclick={() => onchange(values.filter((id) => id !== value.id))}
						>{value.label}<Icon name="close" size={18} /></button
					>{/each}
			</div>{/if}
		<div class="selection-search">
			<input
				bind:this={input}
				id={uid}
				type="search"
				autocomplete="off"
				{placeholder}
				{disabled}
				bind:value={query}
				aria-controls={open ? uid + '-choices' : undefined}
				onfocus={() => {
					if (suppressFocus) suppressFocus = false;
					else open = true;
				}}
				oninput={() => {
					open = true;
				}}
				onclick={() => {
					open = true;
				}}
				onkeydown={async (event) => {
					if (event.key === 'Enter') event.preventDefault();
					if (event.key === 'Escape' && open) {
						event.preventDefault();
						event.stopPropagation();
						close();
					}
					if (event.key === 'ArrowDown') {
						event.preventDefault();
						open = true;
						await tick();
						root
							.querySelector<HTMLInputElement>(
								'.select-choices input:not(:disabled)'
							)
							?.focus();
					}
				}}
			/>
			<span class="search-icon" aria-hidden="true"><Icon name="search" /></span>
		</div>
	</div>
	{#if open && !disabled}<div
			class="select-choices"
			class:above
			style:max-height={available + 'px'}
			id={uid + '-choices'}
			role="group"
			aria-label={`${label} options`}
		>
			{#each matches as option (option.id)}<label
					class="select-option"
					class:chosen={values.includes(option.id)}
					><input
						type="checkbox"
						onkeydown={(event) => {
							if (event.key === 'Escape') {
								event.preventDefault();
								event.stopPropagation();
								close(true);
							}
						}}
						checked={values.includes(option.id)}
						disabled={!values.includes(option.id) && values.length >= maximum}
						onchange={(event) => {
							const checked = event.currentTarget.checked;
							const accepted = onchange(
								checked
									? [...values, option.id]
									: values.filter((id) => id !== option.id)
							);
							if (accepted === false) event.currentTarget.checked = !checked;
							else if (checked) query = '';
						}}
					/>{option.label}</label
				>{:else}<p>No matches found.</p>{/each}
		</div>{/if}
</div>

<style>
	.native-multi-select {
		position: relative;
		min-width: 0;
	}
	.native-multi-select.open {
		z-index: 10;
	}
	label {
		display: block;
		font-size: 16px;
		font-weight: 700;
		margin-bottom: 8px;
	}
	.selection-field {
		display: flex;
		flex-direction: column;
		align-items: stretch;
		gap: 8px;
		padding: 10px 16px;
		min-height: 48px;
		border: 1.5px solid var(--ink-3);
		border-radius: 14px;
		background: var(--surface);
	}
	.selection-field:focus-within {
		border-color: var(--ink);
		box-shadow: 3px 3px 0 var(--shadow);
	}
	.disabled {
		opacity: 0.45;
	}
	.selection-values {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 8px;
		min-width: 0;
	}
	.selection-search {
		display: flex;
		align-items: center;
		gap: 8px;
		min-width: 0;
	}
	.native-multi-select .selection-field .selection-search input[type='search'],
	.native-multi-select
		.selection-field
		.selection-search
		input[type='search']:focus {
		flex: 1;
		width: 100%;
		min-width: 0;
		min-height: 24px;
		padding: 0;
		border: 0;
		outline: none;
		border-radius: 0;
		box-shadow: none;
		background: transparent;
		font: inherit;
		font-size: 16px;
	}
	.selection-search input::-webkit-search-cancel-button {
		display: none;
	}
	.selected-chip {
		display: inline-flex;
		gap: 4px;
		max-width: 100%;
		overflow-wrap: anywhere;
		text-align: left;
		min-height: 36px;
		padding: 6px 12px;
		font-size: 14px;
		border: 1.5px solid var(--ink);
		background: var(--surface);
		border-radius: 999px;
		box-shadow: none;
	}
	.search-icon {
		flex: none;
		display: grid;
		place-items: center;
		width: 32px;
	}
	.select-choices {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		overflow: auto;
		border: 1.5px solid var(--hairline);
		border-radius: 14px;
		padding: 6px;
		background: var(--surface);
		box-shadow: 0 12px 24px #0002;
	}
	.select-choices.above {
		top: auto;
		bottom: calc(100% - 20px);
	}
	.select-option {
		display: flex;
		flex-direction: row;
		align-items: center;
		gap: 16px;
		min-height: 48px;
		margin: 0;
		padding: 10px 14px;
		border-radius: 8px;
		font-size: 16px;
		font-weight: 400;
		cursor: pointer;
	}
	.select-option:first-child {
		background: color-mix(in srgb, var(--primary-soft) 30%, var(--surface));
	}
	.select-option.chosen,
	.select-option:hover,
	.select-option:focus-within {
		background: var(--primary-soft);
	}
	.select-option input {
		margin: 0;
		width: 20px;
		height: 20px;
		flex: none;
		box-shadow: none;
	}
	.select-choices p {
		padding: 16px;
	}
</style>
