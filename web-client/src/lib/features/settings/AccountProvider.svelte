<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import type { WhoAmI } from '#lib/api/types.ts';
	import type { Session } from './model.ts';
	import SettingsIcon from './SettingsIcon.svelte';
	import Feedback from './Feedback.svelte';
	import { Work } from './work.svelte.ts';

	let { api, session }: { api: CraftskyAPI; session: Session } = $props();
	const work = new Work('settings.Account.load');
	let info = $state<WhoAmI | null>(null);
	const current = $derived(info?.did === session.did ? info : null);
	function safeLink(value?: string): string | undefined {
		if (!value) return;
		try {
			const url = new URL(value);
			if (
				url.protocol === 'https:' &&
				url.hostname &&
				!url.username &&
				!url.password
			)
				return url.href;
		} catch {
			/* Unavailable provider links stay hidden. */
		}
	}
	const manage = $derived(safeLink(current?.accountManagementUrl));
	const password = $derived(safeLink(current?.passwordChangeUrl));
	function load() {
		return work.load(
			() => api.account.whoami(),
			(value) => {
				info = value;
			},
			'settings.Account.load'
		);
	}
	function copy(value: string) {
		void work.run(
			() => navigator.clipboard.writeText(value),
			() => {},
			'Copied.'
		);
	}
	onMount(load);
	onDestroy(() => work.dispose());
</script>

<section aria-label="Bluesky / account provider">
	<h2 class="section-label">Bluesky / account provider</h2>
	<p>
		Your handle, password and email belong to your shared AT Protocol account.
		Changes made with your provider also affect other apps that use this
		account.
	</p>
	<dl>
		<div>
			<dt>Handle</dt>
			<dd>{current?.handle || session.handle}</dd>
			<button
				class="copy"
				aria-label="Copy handle"
				onclick={() => copy(current?.handle || session.handle)}>Copy</button
			>
		</div>
		<div>
			<dt>Account ID (DID)</dt>
			<dd>{session.did}</dd>
			<button
				class="copy"
				aria-label="Copy account ID"
				onclick={() => copy(session.did)}>Copy</button
			>
		</div>
		{#if current?.hostingProvider}<div>
				<dt>Hosting provider</dt>
				<dd>{current.hostingProvider}<small>{current.pdsUrl || ''}</small></dd>
			</div>{/if}
	</dl>
	<Feedback {work} reload={load} />
	{#if work.loading}<p>
			Loading hosting details…
		</p>{:else if !work.error && !current?.pdsUrl}<p>
			Hosting details are unavailable.
		</p>
		<button onclick={load}>Try again</button>{/if}
	{#if manage}<a
			class="settings-tile"
			href={manage}
			target="_blank"
			rel="noopener noreferrer"
			><SettingsIcon name="account" /><span class="tile-copy"
				>Manage account with provider</span
			><SettingsIcon name="external" size={24} /></a
		>{/if}
	{#if password}<a
			class="settings-tile"
			href={password}
			target="_blank"
			rel="noopener noreferrer"
			><SettingsIcon name="lock" /><span class="tile-copy"
				>Change password in Bluesky</span
			><SettingsIcon name="external" size={24} /></a
		>{/if}
	{#if manage}<p>
			Available settings vary by provider. You may need to sign in again.{#if password}{' '}
				In Bluesky account settings, select Password to change it.{/if}
		</p>{/if}
</section>

<style>
	section {
		border-bottom: 1px solid var(--ink-3);
		padding-bottom: 16px;
		margin-bottom: 16px;
	}
	p,
	dl {
		margin: 12px 16px;
	}
	dl {
		display: block;
	}
	dl > div {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 4px 12px;
		margin: 16px 0;
	}
	dt {
		font-size: 0.85rem;
		color: var(--ink-3);
	}
	dd {
		grid-column: 1;
		margin: 0;
		overflow-wrap: anywhere;
		user-select: text;
	}
	small {
		display: block;
		overflow-wrap: anywhere;
	}
	.copy {
		grid-column: 2;
		grid-row: 1 / span 2;
		align-self: center;
		min-width: 48px;
		min-height: 48px;
	}
</style>
