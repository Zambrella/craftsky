<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	import {
		addLanguage,
		languageChoices,
		languageName,
		removeLanguage
	} from './languages';
	let {
		owner,
		languages,
		disabled = false,
		onchange
	}: {
		owner: string;
		languages: string[];
		disabled?: boolean;
		onchange: (languages: string[]) => void;
	} = $props();
	const uid = $props.id();
	let dialog: HTMLDialogElement;
	let menu: HTMLElement;
	let trigger: HTMLButtonElement;
	let searchInput: HTMLInputElement;
	let search = $state('');
	let disposed = false;
	$effect(() => {
		owner;
		untrack(() => {
			menu?.hidePopover();
			dialog?.close();
		});
	});
	function positionMenu() {
		const bounds = trigger.getBoundingClientRect(),
			width = 280;
		menu.style.left = `${Math.max(16, Math.min(bounds.left, innerWidth - width - 16))}px`;
		menu.style.top = `${Math.max(16, Math.min(bounds.bottom + 8, innerHeight - Math.min(320, 64 + languages.length * 48) - 16))}px`;
	}
	const choices = $derived(languageChoices(search, languages));
	function openSearch() {
		if (disabled || languages.length >= 3) return;
		menu.hidePopover();
		search = '';
		dialog.showModal();
		searchInput.focus();
	}
	function close() {
		dialog.close();
		if (!disposed) trigger.focus({ preventScroll: true });
	}
	onDestroy(() => {
		disposed = true;
		menu?.hidePopover();
		dialog?.close();
	});
</script>

<button
	bind:this={trigger}
	type="button"
	class="language-control"
	aria-label="Post languages"
	title="Post languages"
	popovertarget={uid + '-menu'}
	onclick={positionMenu}
	{disabled}
>
	<Icon name="translate" /><span class="language-count">{languages.length}</span
	>
</button>
<div
	bind:this={menu}
	popover
	id={uid + '-menu'}
	class="language-menu"
	role="menu"
	aria-label="Post languages"
>
	{#each languages as tag}<button
			type="button"
			role="menuitemcheckbox"
			aria-checked="true"
			aria-label={`Remove ${languageName(tag)}`}
			disabled={disabled || languages.length < 2}
			onclick={() => {
				onchange(removeLanguage(languages, tag));
				menu.hidePopover();
			}}
			><Icon name="translate" /><span>{languageName(tag)}</span><Icon
				name="check"
				size={20}
			/></button
		>{/each}
	<hr />
	<button
		type="button"
		role="menuitem"
		disabled={disabled || languages.length >= 3}
		onclick={openSearch}><Icon name="plus" /><span>Add language</span></button
	>
</div>
<dialog
	bind:this={dialog}
	class="language-dialog"
	aria-labelledby={uid + '-title'}
	oncancel={(event) => {
		event.preventDefault();
		close();
	}}
>
	<h2 id={uid + '-title'}>Add post language</h2>
	<label class="language-search"
		>Search languages<input
			name="search"
			bind:this={searchInput}
			type="search"
			bind:value={search}
			autocomplete="off"
			placeholder="Search languages"
		/></label
	>
	<ul aria-label="Available languages">
		{#each choices as [tag, name]}<li>
				<button
					type="button"
					disabled={disabled || languages.length >= 3}
					onclick={() => {
						onchange(addLanguage(languages, tag));
						close();
					}}><span>{name}</span><small>{tag}</small></button
				>
			</li>{/each}
	</ul>
	{#if !choices.length}<p class="language-limit">No matching languages.</p>{/if}
	<div class="language-actions">
		<button type="button" onclick={close}>Cancel</button>
	</div>
</dialog>

<style>
	.language-control {
		position: relative;
		width: 44px;
		height: 44px;
		display: grid;
		place-items: center;
		border: 0;
		background: none;
		box-shadow: none;
		padding: 8px;
	}
	.language-count {
		position: absolute;
		top: 0;
		right: 0;
		min-width: 16px;
		height: 16px;
		background: var(--ink);
		color: var(--paper);
		border-radius: 10px;
		font-size: 10px;
		display: grid;
		place-items: center;
		padding: 0 3px;
	}
	.language-menu {
		position: fixed;
		inset: auto;
		margin: auto;
		width: 280px;
		max-width: calc(100vw - 32px);
		padding: 8px;
		background: var(--paper);
		color: var(--ink);
		border: 1.5px solid var(--ink);
		border-radius: 12px;
		box-shadow: 3px 3px 0 var(--ink);
	}
	.language-menu:popover-open {
		margin: 0;
	}
	.language-menu button {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		min-height: 48px;
		padding: 12px;
		text-align: left;
		border: 0;
		border-radius: 0;
		background: transparent;
		box-shadow: none;
		font-size: 16px;
		font-weight: 400;
	}
	.language-menu button span {
		flex: 1;
	}
	.language-menu hr {
		border: 0;
		border-top: 1px solid var(--hairline);
		margin: 8px -8px;
	}
	.language-dialog {
		width: min(420px, calc(100vw - 40px));
		max-height: calc(100dvh - 48px);
		border: 1.5px solid var(--ink);
		background: var(--paper);
		color: var(--ink);
		border-radius: 20px;
		padding: 24px;
		box-shadow: 4px 4px 0 var(--ink);
		overflow: auto;
	}
	.language-dialog::backdrop {
		background: #0006;
	}
	.language-dialog h2 {
		font-size: 22px;
		margin-bottom: 24px;
	}
	.language-search {
		display: grid;
		gap: 8px;
		font-size: 14px;
	}
	.language-search input {
		width: 100%;
		font: inherit;
		padding: 12px;
		border: 1.5px solid var(--ink);
		border-radius: 10px;
		background: var(--surface);
		color: var(--ink);
	}
	ul {
		padding: 8px 0;
		list-style: none;
		margin: 12px 0;
		height: clamp(180px, calc(100dvh - 400px), 420px);
		overflow: auto;
	}
	li > button {
		width: 100%;
		padding: 12px 16px;
		border: 0;
		background: transparent;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 4px;
		box-shadow: none;
		font-size: 16px;
		font-weight: 400;
		text-align: left;
	}
	li > button:hover,
	.language-menu button:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	small,
	.language-limit {
		color: var(--ink-3);
		font-size: 13px;
	}
	.language-limit {
		padding: 12px 0;
	}
	.language-actions {
		display: flex;
		justify-content: flex-end;
		padding-top: 12px;
	}
	.language-actions button {
		border: 0;
		color: var(--primary);
		background: transparent;
		box-shadow: none;
	}
	button:disabled {
		opacity: 0.5;
		cursor: default;
	}
</style>
