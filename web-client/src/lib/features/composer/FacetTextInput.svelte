<script lang="ts">
	import { onDestroy, tick, untrack } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import {
		activeFacetToken,
		replaceFacetToken,
		type ActiveFacetToken
	} from './facet-input.ts';
	let {
		api,
		label,
		value,
		onchange,
		rows = 1,
		placeholder = '',
		disabled = false,
		allowed = ['mention', 'hashtag'],
		className = '',
		hint = '',
		onsubmit = () => {}
	}: {
		api: CraftskyAPI;
		label: string;
		value: string;
		onchange: (value: string) => void;
		rows?: number;
		placeholder?: string;
		disabled?: boolean;
		allowed?: ActiveFacetToken['kind'][];
		className?: string;
		hint?: string;
		onsubmit?: () => void;
	} = $props();
	const uid = $props.id();
	let editor = $state<HTMLTextAreaElement>();
	let focused = $state(false);
	let composing = $state(false);
	let token = $state<ActiveFacetToken>();
	let choices = $state<{ value: string; name: string; description?: string }[]>(
		[]
	);
	let active = $state(0);
	let loading = $state(false);
	let failed = $state(false);
	let timer: ReturnType<typeof setTimeout> | undefined;
	let controller: AbortController | undefined;
	let generation = 0;
	let dead = false;
	let dismissed = '';
	const open = $derived(
		focused &&
			!disabled &&
			!composing &&
			!!token &&
			(loading || choices.length > 0 || failed)
	);
	function close() {
		generation++;
		clearTimeout(timer);
		controller?.abort();
		choices = [];
		loading = false;
		failed = false;
		token = undefined;
	}
	onDestroy(() => {
		dead = true;
		close();
	});
	$effect(() => {
		if (disabled) untrack(close);
	});
	$effect(() => {
		void api;
		untrack(close);
	});
	function update() {
		if (!editor || disabled || composing) return;
		const current = activeFacetToken(
			editor.value,
			editor.selectionStart,
			editor.selectionEnd
		);
		if (!current || !current.query || !allowed.includes(current.kind)) {
			close();
			dismissed = '';
			return;
		}
		const identity = `${current.kind}:${current.start}:${current.end}:${current.query}`;
		if (identity === dismissed) return;
		if (
			token &&
			current.kind === token.kind &&
			current.start === token.start &&
			current.end === token.end &&
			current.query === token.query
		)
			return;
		close();
		token = current;
		active = 0;
		loading = true;
		const revision = ++generation;
		controller = new AbortController();
		const signal = controller.signal;
		timer = setTimeout(() => {
			void (
				current.kind === 'mention'
					? api.search
							.mentionSuggestions(current.query, 10, signal)
							.then((result) =>
								result.items.map((person) => ({
									value: person.handle,
									name: person.displayName || `@${person.handle}`,
									description: `@${person.handle}`
								}))
							)
					: api.search
							.hashtagSuggestions(current.query, 10, signal)
							.then((result) =>
								result.items.map((tag) => ({
									value: tag.tag,
									name: `#${tag.tag}`
								}))
							)
			)
				.then((items) => {
					if (!dead && revision === generation) {
						choices = items;
						loading = false;
					}
				})
				.catch(() => {
					if (!dead && revision === generation && !signal.aborted) {
						loading = false;
						failed = true;
					}
				});
		}, 220);
	}
	async function choose(index: number) {
		if (!editor || !choices[index] || !token) return;
		const current = activeFacetToken(
			editor.value,
			editor.selectionStart,
			editor.selectionEnd
		);
		if (
			!current ||
			current.start !== token.start ||
			current.query !== token.query
		)
			return;
		const next = replaceFacetToken(editor.value, current, choices[index].value);
		close();
		onchange(next.text);
		await tick();
		if (dead || !editor) return;
		editor.focus();
		editor.setSelectionRange(next.caret, next.caret);
	}
	function keydown(event: KeyboardEvent) {
		if (composing || event.isComposing) return;
		if (
			open &&
			choices.length &&
			['ArrowDown', 'ArrowUp'].includes(event.key)
		) {
			event.preventDefault();
			active =
				(active + (event.key === 'ArrowDown' ? 1 : -1) + choices.length) %
				choices.length;
			return;
		}
		if (open && choices.length && event.key === 'Enter') {
			event.preventDefault();
			void choose(active);
			return;
		}
		if (rows === 1 && event.key === 'Enter') {
			event.preventDefault();
			onsubmit();
		}
		if (event.key === 'Escape' && token) {
			event.preventDefault();
			dismissed = `${token.kind}:${token.start}:${token.end}:${token.query}`;
			close();
		}
	}
</script>

<div class="facet-input">
	<label for={uid}
		>{label}{#if hint}<span
				class="facet-hint"
				class:required={hint.toLowerCase() === 'required'}
				id={uid + '-hint'}
				aria-hidden="true">{hint}</span
			>{/if}</label
	>
	<textarea
		id={uid}
		bind:this={editor}
		class={className}
		{value}
		{rows}
		wrap={rows === 1 ? 'off' : 'soft'}
		{placeholder}
		{disabled}
		aria-describedby={hint ? uid + '-hint' : undefined}
		aria-autocomplete="list"
		aria-controls={open ? `${uid}-suggestions` : undefined}
		aria-activedescendant={open && choices.length
			? `${uid}-option-${active}`
			: undefined}
		aria-haspopup="listbox"
		autocomplete="off"
		oninput={(e) => {
			dismissed = '';
			onchange(e.currentTarget.value);
			update();
		}}
		onfocus={() => {
			focused = true;
			update();
		}}
		onblur={() => {
			focused = false;
			close();
		}}
		onselect={update}
		onclick={update}
		onkeyup={(e) => {
			if (!['ArrowUp', 'ArrowDown', 'Escape', 'Enter'].includes(e.key))
				update();
		}}
		onkeydown={keydown}
		oncompositionstart={() => {
			composing = true;
			close();
		}}
		oncompositionend={() => {
			composing = false;
			update();
		}}></textarea>
	{#if open}<div class="facet-popup">
			<div
				id={`${uid}-suggestions`}
				role="listbox"
				aria-label={token?.kind === 'mention'
					? 'Account suggestions'
					: 'Hashtag suggestions'}
			>
				{#each choices as choice, index}<div
						role="option"
						tabindex="-1"
						id={`${uid}-option-${index}`}
						aria-selected={active === index}
						class:current={active === index}
						onpointerdown={(e) => {
							e.preventDefault();
							void choose(index);
						}}
					>
						<strong>{choice.name}</strong>{#if choice.description}<span
								>{choice.description}</span
							>{/if}
					</div>{/each}
			</div>
			<p class="facet-hint" role="status">
				{loading
					? 'Finding suggestions…'
					: failed
						? 'Suggestions could not load. You can keep typing.'
						: 'Use ↑ ↓ and Enter to choose, or Escape to close.'}
			</p>
		</div>{/if}
</div>

<style>
	.facet-input {
		position: relative;
		display: grid;
		gap: 6px;
		min-width: 0;
	}
	.facet-input label {
		display: flex;
		flex-direction: row;
		flex-wrap: wrap;
		align-items: baseline;
		gap: 8px;
	}
	.facet-input label .facet-hint {
		font-size: 12px;
		font-weight: 400;
	}

	.facet-input label .facet-hint.required {
		font-weight: 600;
		letter-spacing: 1.68px;
	}
	.facet-input textarea {
		width: 100%;
		font: inherit;
		font-size: 16px;
		resize: vertical;
	}
	.facet-input textarea[rows='1'] {
		min-height: 44px;
		resize: none;
		overflow: hidden;
	}
	.facet-popup {
		position: absolute;
		top: 100%;
		left: 0;
		right: 0;
		z-index: 30;
		border: 1px solid var(--ink);
		background: var(--surface);
		border-radius: 10px;
		box-shadow: 2px 3px 0 var(--shadow);
		padding: 5px;
		max-height: 280px;
		overflow: auto;
	}
	.facet-popup [role='option'] {
		padding: 9px 12px;
		display: grid;
		border-radius: 6px;
		cursor: pointer;
		gap: 2px;
	}
	.facet-popup [role='option'].current,
	.facet-popup [role='option']:hover {
		background: var(--primary-soft);
	}
	.facet-popup span,
	.facet-hint {
		font-size: 0.85rem;
		color: var(--ink-3);
	}
	.facet-popup p {
		padding: 7px 12px;
	}
</style>
