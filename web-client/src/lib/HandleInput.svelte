<script lang="ts">
	import { onDestroy } from 'svelte';
	import Avatar from './Avatar.svelte';
	import type { Author } from './api/types';
	let {
		value = $bindable(''),
		disabled = false
	}: { value?: string; disabled?: boolean } = $props();
	const uid = $props.id();
	let input: HTMLInputElement;
	let focused = $state(false),
		composing = $state(false),
		active = $state(0);
	let choices = $state<Author[]>([]);
	let timer: ReturnType<typeof setTimeout>;
	let controller: AbortController | undefined;
	let revision = 0;
	function clear() {
		revision++;
		clearTimeout(timer);
		controller?.abort();
		choices = [];
	}
	function search() {
		clear();
		const query = value.trim().replace(/^@/, '');
		if (!focused || disabled || composing || query.length < 2) return;
		const current = revision;
		controller = new AbortController();
		const signal = controller.signal;
		timer = setTimeout(async () => {
			try {
				const url = new URL(
					'https://typeahead.waow.tech/xrpc/tech.waow.typeahead.searchActors'
				);
				url.searchParams.set('q', query);
				url.searchParams.set('limit', '5');
				const response = await fetch(url, {
					signal,
					credentials: 'omit',
					redirect: 'error',
					headers: { 'X-Client': 'craftsky.social' }
				});
				if (!response.ok) return;
				const data = await response.json();
				if (current !== revision || !Array.isArray(data.actors)) return;
				const seen = new Set<string>();
				choices = data.actors
					.flatMap((actor: Record<string, unknown>) => {
						if (!actor || typeof actor.handle !== 'string') return [];
						const handle = actor.handle.trim().toLowerCase();
						if (
							handle === 'handle.invalid' ||
							!/^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z][a-z0-9-]*[a-z0-9]$/.test(
								handle
							) ||
							seen.has(handle)
						)
							return [];
						seen.add(handle);
						let avatar: string | undefined;
						try {
							const url = new URL(String(actor.avatar));
							if (url.protocol === 'https:' && !url.username && !url.password)
								avatar = url.href;
						} catch {}
						return [
							{
								did: typeof actor.did === 'string' ? actor.did : '',
								handle,
								displayName:
									typeof actor.displayName === 'string'
										? actor.displayName
										: undefined,
								avatar
							}
						];
					})
					.slice(0, 5);
				active = 0;
			} catch {
				/* Discovery failure never prevents manual sign-in. */
			}
		}, 300);
	}
	function choose(index: number) {
		if (!choices[index]) return;
		value = choices[index].handle;
		clear();
		input.focus();
	}
	onDestroy(clear);
	$effect(() => {
		if (disabled) clear();
	});
</script>

<div class="handle-field">
	<div class="handle-heading">
		<label for={uid}>Your Atmosphere Handle</label>
		<div class="atmosphere-logos">
			{#each ['Bluesky', 'mu', 'Blacksky', 'Witchsky'] as name}
				<span
					class="atmosphere-logo"
					role="img"
					aria-label={name}
					title={name}
					style={'--logo: url(/atmosphere/' + name.toLowerCase() + '.svg)'}
				></span>
			{/each}
		</div>
	</div>
	<input
		id={uid}
		required
		bind:this={input}
		bind:value
		placeholder="your-handle.bsky.social"
		autocomplete="username"
		autocapitalize="none"
		spellcheck="false"
		{disabled}
		role="combobox"
		aria-autocomplete="list"
		aria-expanded={choices.length > 0}
		aria-controls={choices.length ? uid + '-options' : undefined}
		aria-activedescendant={choices.length
			? uid + '-option-' + active
			: undefined}
		oninput={search}
		onfocus={() => {
			focused = true;
			search();
		}}
		onblur={() => {
			focused = false;
			clear();
		}}
		oncompositionstart={() => {
			composing = true;
			clear();
		}}
		oncompositionend={() => {
			composing = false;
			search();
		}}
		onkeydown={(event) => {
			if (event.isComposing || !choices.length) return;
			if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
				event.preventDefault();
				active =
					(active + (event.key === 'ArrowDown' ? 1 : -1) + choices.length) %
					choices.length;
			}
			if (event.key === 'Enter') {
				event.preventDefault();
				choose(active);
			}
			if (event.key === 'Escape') {
				event.preventDefault();
				clear();
			}
		}}
	/>
	{#if choices.length}<div
			class="handle-options"
			role="listbox"
			id={uid + '-options'}
			aria-label="Account suggestions"
		>
			{#each choices as author, index}<div
					role="option"
					tabindex="-1"
					id={uid + '-option-' + index}
					aria-selected={active === index}
					class:current={active === index}
					onpointerdown={(event) => {
						event.preventDefault();
						choose(index);
					}}
				>
					<Avatar {author} /><span
						><strong>{author.displayName || author.handle}</strong><small
							>@{author.handle}</small
						></span
					>
				</div>{/each}
		</div>{/if}
</div>

<style>
	.handle-field {
		position: relative;
		display: grid;
		gap: 10px;
	}
	.handle-heading {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 6px 12px;
	}
	.atmosphere-logos {
		display: flex;
		align-items: center;
		gap: 8px;
		color: var(--ink-2);
	}
	.atmosphere-logo {
		display: block;
		width: 18px;
		height: 18px;
		background: currentColor;
		mask: var(--logo) center / contain no-repeat;
	}
	input {
		min-height: 56px;
	}
	label {
		font-size: 16px;
		font-weight: 700;
		pointer-events: none;
	}
	.handle-options {
		position: absolute;
		top: calc(100% + 4px);
		left: 0;
		right: 0;
		z-index: 5;
		background: var(--surface);
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		padding: 4px;
		box-shadow: 3px 3px var(--shadow);
	}
	[role='option'] {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 10px 12px;
		border-radius: 10px;
		cursor: pointer;
	}
	[role='option']:hover,
	.current {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	span {
		min-width: 0;
		display: grid;
	}
	strong,
	small {
		overflow: hidden;
		text-overflow: ellipsis;
	}
	small {
		color: var(--ink-2);
	}
</style>
