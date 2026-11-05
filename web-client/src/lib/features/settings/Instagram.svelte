<script lang="ts">
	import { fieldError } from '#lib/field-validation.ts';
	import { usePrompt } from '../social/prompt';
	const ask = usePrompt();
	import { onMount, onDestroy } from 'svelte';
	import {
		newOperationKey,
		type CraftskyAPI,
		type Page
	} from '#lib/api/client.ts';
	import {
		dateTime,
		parseFollowing,
		record,
		text,
		title,
		type ParsedFollowing
	} from './model.ts';
	import { safeURL } from '#lib/public.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import SettingsIcon from './SettingsIcon.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	let { api }: { api: CraftskyAPI } = $props();
	const work = new Work();
	let loaded = $state(false);
	let disposed = false;
	const archiveAbort = new AbortController();
	type Account = {
		state: string;
		username: string;
		discoverable: boolean;
		conflictPending?: boolean;
		reactivationRequired?: boolean;
		verifiedAt?: string;
	};
	type Verification = {
		verificationId: string;
		state: string;
		expiresAt: string;
		challenge?: string;
		dmUrl?: string;
		candidateUsername?: string;
		retryCode?: string;
	};
	type Import = {
		importId: string;
		state: string;
		sourceType: string;
		followingCount: number;
		createdAt: string;
	};
	type Suggestion = {
		suggestionId: string;
		target: {
			did: string;
			handle: string;
			displayName?: string;
			avatar?: string;
		};
	};
	type Data = {
		account: { integrationAvailable: boolean; account: Account | null };
		verification: { verification: Verification | null };
		imports: Page<Import>;
		suggestions: Page<Suggestion>;
	};
	let available = $state(true),
		account = $state<Account | null>(null),
		verification = $state<Verification | null>(null),
		imports = $state<Import[]>([]),
		suggestions = $state<Suggestion[]>([]);
	let importCursor = $state<string | null>(null),
		suggestionCursor = $state<string | null>(null),
		discoverable = $state(true),
		manual = $state(''),
		parsed = $state<ParsedFollowing | null>(null),
		sourceType = $state('manual'),
		fileName = $state('');
	let inputKind = $state<'export' | 'manual'>('export');
	let selectedImport = $state<Record<string, unknown> | null>(null);
	const transport = $derived(api.transport);
	async function fetchData(): Promise<Data> {
		const [a, v, i, s] = await Promise.all([
			transport.get<Data['account']>('/v1/migrations/instagram/account'),
			transport.get<Data['verification']>(
				'/v1/migrations/instagram/verifications/current'
			),
			transport.page<Import>('/v1/migrations/instagram/imports'),
			transport.page<Suggestion>('/v1/migrations/instagram/suggestions')
		]);
		return { account: a, verification: v, imports: i, suggestions: s };
	}
	function apply(data: Data) {
		loaded = true;
		available = data.account.integrationAvailable;
		account = data.account.account;
		const previous = verification;
		verification = data.verification.verification;
		if (
			previous?.verificationId === verification?.verificationId &&
			verification
		) {
			verification = {
				...verification,
				challenge: previous?.challenge,
				dmUrl: previous?.dmUrl
			};
		}
		if (
			verification?.state === 'pendingConfirmation' &&
			previous?.verificationId !== verification.verificationId
		)
			discoverable = true;
		else discoverable = account?.discoverable ?? discoverable;
		imports = data.imports.items;
		importCursor = data.imports.cursor || null;
		suggestions = data.suggestions.items;
		suggestionCursor = data.suggestions.cursor || null;
	}
	function load() {
		return work.load(fetchData, apply);
	}
	function update(action: () => Promise<unknown>, notice: string) {
		void work.run(
			async () => {
				await action();
				return fetchData();
			},
			apply,
			notice
		);
	}
	async function setDiscoverable(discoverable: boolean) {
		if (!account || work.busy || work.retry) return;
		const previous = account;
		account = { ...account, discoverable };
		await work.run(
			async () => {
				await api.instagram.settings({ discoverable });
				return fetchData();
			},
			apply,
			'Discovery preference saved.'
		);
		if (!disposed && work.error) account = previous;
	}
	function verify() {
		void work.run(
			() =>
				transport.request<Verification>(
					'POST',
					'/v1/migrations/instagram/verifications',
					{ body: {} }
				),
			(v) => {
				verification = v;
			},
			'Verification started. Send the exact challenge below from your Instagram account.'
		);
	}
	function parseManual(event: SubmitEvent) {
		event.preventDefault();
		try {
			parsed = parseFollowing(manual);
			sourceType = 'manual';
			fileName = '';
			work.error = '';
		} catch (error) {
			parsed = null;
			fieldError(
				'Instagram usernames',
				error instanceof Error ? error.message : 'Could not read usernames.'
			);
		}
	}
	async function readFile(event: Event) {
		const file = (event.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		await work.run(
			async () => {
				const { parseInstagramExport } = await import('./archive.ts');
				return parseInstagramExport(file, archiveAbort.signal);
			},
			(value) => {
				parsed = value;
				sourceType = 'instagramJson';
				fileName = file.name;
			},
			'File ready to review.'
		);
	}
	function importFollowing() {
		if (!parsed?.entries.length) return;
		const body = {
			sourceType,
			entries: parsed.entries.map((entry) => ({ ...entry }))
		};
		void work.run(
			async () => {
				await api.instagram.createImport(body);
				return fetchData();
			},
			(data) => {
				apply(data);
				parsed = null;
				manual = '';
				fileName = '';
			},
			'Following list imported. New matches will appear here.'
		);
	}
	function accept(suggestion: Suggestion) {
		const id = suggestion.suggestionId,
			key = newOperationKey();
		void work.run(
			() => api.instagram.acceptSuggestion(id, key),
			() => {
				suggestions = suggestions.filter((s) => s.suggestionId !== id);
			},
			'Following this maker.'
		);
	}
	function moreImports() {
		const next = importCursor;
		if (next)
			void work.run(
				() =>
					transport.page<Import>('/v1/migrations/instagram/imports', {
						cursor: next
					}),
				(page) => {
					imports = [...imports, ...page.items];
					importCursor = page.cursor || null;
				},
				''
			);
	}
	function moreSuggestions() {
		const next = suggestionCursor;
		if (next)
			void work.run(
				() =>
					transport.page<Suggestion>('/v1/migrations/instagram/suggestions', {
						cursor: next
					}),
				(page) => {
					suggestions = [...suggestions, ...page.items];
					suggestionCursor = page.cursor || null;
				},
				''
			);
	}
	onMount(load);
	onDestroy(() => {
		disposed = true;
		work.dispose();
		archiveAbort.abort();
	});
</script>

<h1 class="workspace-heading">Find people from Instagram</h1>
<Feedback {work} reload={load} />
{#if !work.loading && loaded}
	{#if !available}<p class="notice">
			Instagram verification is temporarily unavailable. Your saved account and
			imports remain below.
		</p>{/if}
	<section class="panel stack">
		<h2 class="section-heading">
			<SettingsIcon name="instagram" />{account
				? 'Instagram account'
				: 'Verify your Instagram account'}
		</h2>
		{#if account}<p>
				Verified as <strong>@{account.username}</strong>
			</p>
			{#if account.verifiedAt}<p class="muted">
					Verified {dateTime(account.verifiedAt)}
				</p>{/if}{#if account.conflictPending}<p class="notice">
					This account has a pending ownership conflict. Verification may be
					required again.
				</p>{/if}
			{#if account.reactivationRequired || account.state === 'membershipInactive'}<p
				>
					Reactivation keeps discovery off until you choose to turn it on again.
				</p>
				<button
					disabled={work.busy || !!work.retry}
					onclick={() =>
						update(
							() => api.instagram.settings({ reactivate: true }),
							'Instagram account reactivated.'
						)}>Reactivate Instagram account</button
				>
			{:else}<label class="switch-row"
					><span class="tile-copy"
						>Let others find me by my Instagram username</span
					><input
						type="checkbox"
						role="switch"
						checked={account.discoverable}
						disabled={work.busy || !!work.retry}
						onchange={(event) => setDiscoverable(event.currentTarget.checked)}
					/></label
				>{/if}
		{:else}<p>
				Send a one-time challenge to CraftSky's official Instagram account. You
				will confirm the username here before it is verified.
			</p>{/if}
		{#if verification && ['pendingDm', 'processing', 'pendingConfirmation'].includes(verification.state)}<div
				class="verification stack"
			>
				<h3>
					{verification.state === 'pendingConfirmation'
						? 'Confirm your username'
						: verification.state === 'processing'
							? 'Checking verification'
							: 'Send your verification message'}
				</h3>
				<p class="muted">Expires {dateTime(verification.expiresAt)}</p>
				{#if verification.state === 'pendingDm'}{#if verification.challenge}<p>
							Send this exact message to the CraftSky account on Instagram:
						</p>
						<code class="challenge">{verification.challenge}</code>
						<div class="actions">
							<button
								disabled={work.busy}
								onclick={() =>
									work.run(
										() =>
											navigator.clipboard.writeText(
												verification?.challenge || ''
											),
										() => {},
										'Challenge copied.'
									)}>Copy message</button
							>{#if safeURL(verification.dmUrl)}<a
									class="action primary"
									href={safeURL(verification.dmUrl)}
									target="_blank"
									rel="noreferrer">Open Instagram ↗</a
								>{/if}
						</div>{:else}<p>
							Send the challenge shown when you started verification, then check
							the status below. If you no longer have it, cancel and start
							again.
						</p>{/if}{/if}{#if verification.state === 'pendingConfirmation'}<p>
						Instagram username: <strong
							>@{verification.candidateUsername}</strong
						>
					</p>
					<div class="segmented" aria-label="Instagram discovery">
						<button
							type="button"
							aria-pressed={discoverable}
							disabled={work.busy || !!work.retry}
							onclick={() => (discoverable = true)}>Allow discovery</button
						><button
							type="button"
							aria-pressed={!discoverable}
							disabled={work.busy || !!work.retry}
							onclick={() => (discoverable = false)}>Keep private</button
						>
					</div>
					<p class="helper">
						{discoverable
							? 'When enabled, eligible CraftSky members who imported your Instagram username may see a private suggestion to follow you.'
							: 'Your Instagram account remains verified, but CraftSky will not match it with people who imported your username.'}
					</p>
					<p class="helper">Confirm only if this is your Instagram username.</p>
					<button
						class="primary"
						disabled={work.busy}
						onclick={() => {
							const id = verification!.verificationId;
							update(
								() => api.instagram.confirmVerification(id, discoverable),
								'Instagram account verified.'
							);
						}}>Confirm this is my Instagram account</button
					>{/if}
				<div class="actions">
					<button disabled={work.busy} onclick={load}
						>Check verification status</button
					><button
						disabled={work.busy}
						onclick={() => {
							const id = verification!.verificationId;
							update(
								() => api.instagram.cancelVerification(id),
								'Verification cancelled.'
							);
						}}>Cancel verification</button
					>
				</div>
			</div>
		{:else}{#if verification}<p class="muted">
					Previous verification: {title(
						verification.state
					)}{verification.retryCode
						? ` · ${title(verification.retryCode)}`
						: ''}
				</p>{/if}{#if available && !account}<button
					class="primary"
					disabled={work.busy}
					onclick={verify}>Create verification challenge</button
				>{/if}{/if}
	</section>
	<a
		class="import-help"
		href="https://craftsky.social/instagram-import"
		target="_blank"
		rel="noopener noreferrer"
		>How to export your Instagram following <SettingsIcon
			name="external"
			size={16}
		/></a
	>
	{#if !account}<p class="muted">
			Complete verification to import the accounts you follow.
		</p>{:else}
		<section class="panel stack">
			<h2 class="section-heading">
				<SettingsIcon name="followers" />Import accounts you follow
			</h2>
			<div class="segmented" aria-label="Import method">
				<button
					aria-pressed={inputKind === 'export'}
					disabled={work.busy}
					onclick={() => (inputKind = 'export')}>Instagram export</button
				><button
					aria-pressed={inputKind === 'manual'}
					disabled={work.busy}
					onclick={() => (inputKind = 'manual')}>Enter handles</button
				>
			</div>
			{#if inputKind === 'export'}<p class="helper">
					Choose JSON format and All time when exporting from Instagram, then
					select the downloaded ZIP here. CraftSky processes it on this device
					and uploads only the usernames of accounts you follow. Everything else
					stays on your device.
				</p>
				<label
					>Choose your Instagram ZIP export or following.json<input
						type="file"
						accept="application/json,application/zip,.json,.zip"
						disabled={work.busy}
						onchange={readFile}
					/></label
				>
			{:else}<p class="helper">
					Enter the Instagram handles of accounts you follow, one per line.
				</p>
				<form class="stack" onsubmit={parseManual}>
					<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
						<label
							>Instagram usernames<textarea
								name="manual"
								bind:value={manual}
								placeholder={'@maker_one\nmaker_two'}></textarea></label
						><button disabled={work.busy}>Review usernames</button>
					</fieldset>
				</form>{/if}
			<p class="helper">
				Importing creates private suggestions only. You choose whether to follow
				each account.
			</p>
			{#if parsed}<div class="notice">
					<h3>Review import{fileName ? `: ${fileName}` : ''}</h3>
					<p>
						{parsed.entries.length} valid usernames · {parsed.duplicates} duplicates
						removed · {parsed.ignored} invalid entries ignored
					</p>
					<p class="muted">
						{parsed.entries
							.slice(0, 8)
							.map((e) => `@${e.username}`)
							.join(', ')}{parsed.entries.length > 8 ? '…' : ''}
					</p>
					<div class="actions">
						<button
							class="primary"
							disabled={work.busy || !parsed.entries.length}
							onclick={importFollowing}
							>Import {parsed.entries.length} usernames</button
						><button disabled={work.busy} onclick={() => (parsed = null)}
							>Cancel</button
						>
					</div>
				</div>{/if}
		</section>
		<section class="panel">
			<h2 class="section-heading">
				<SettingsIcon name="document" />Your imports
			</h2>
			{#each imports as item}<div class="import-row">
					<div class="row">
						<div>
							<h3>{item.followingCount} usernames</h3>
							<small class="muted"
								>{title(item.sourceType)} · {title(item.state)} · {dateTime(
									item.createdAt
								)}</small
							>
						</div>
						<div class="actions">
							<button
								disabled={work.busy}
								onclick={() =>
									work.run(
										() => api.instagram.import(item.importId),
										(detail) => {
											selectedImport = record(detail);
										},
										''
									)}>Details</button
							>{#if ['inactive', 'membershipInactive'].includes(item.state)}<button
									disabled={work.busy}
									onclick={() =>
										update(
											() =>
												api.instagram.updateImport(item.importId, {
													reactivate: true
												}),
											'Import reactivated.'
										)}>Reactivate</button
								>{/if}<button
								class="danger"
								disabled={work.busy}
								onclick={async () => {
									if (
										await ask({
											title: 'Delete import?',
											detail: 'Delete this imported following list?',
											confirmLabel: 'Delete',
											danger: true
										})
									)
										update(
											() => api.instagram.deleteImport(item.importId),
											'Import deleted.'
										);
								}}>Delete</button
							>
						</div>
					</div>
				</div>{:else}<p class="empty">
					No following lists imported yet.
				</p>{/each}{#if importCursor}<button
					disabled={work.busy}
					onclick={moreImports}>Load more imports</button
				>{/if}{#if selectedImport}{@const detail = record(
					selectedImport.import || selectedImport
				)}
				<div class="notice">
					<h3>Import details</h3>
					<p>Status: {title(text(detail.state))}</p>
					<p>
						{typeof detail.followingCount === 'number'
							? detail.followingCount
							: '—'} usernames · {dateTime(detail.createdAt)}
					</p>
					<button onclick={() => (selectedImport = null)}>Close details</button>
				</div>{/if}
		</section>
		<section class="panel">
			<h2 class="section-heading">
				<SettingsIcon name="followers" />Possible CraftSky accounts
			</h2>
			<p class="muted">
				Your imports find possible CraftSky accounts privately. Nobody is
				followed until you choose Follow.
			</p>
			{#each suggestions as suggestion (suggestion.suggestionId)}<div
					class="row"
				>
					<a
						class="identity instagram-identity"
						href={`/profiles/${encodeURIComponent(suggestion.target.did)}`}
						><Avatar author={suggestion.target} /><CompactIdentity
							author={suggestion.target}
						/></a
					>
					<div class="actions">
						<button
							class="primary"
							disabled={work.busy || !!work.retry}
							onclick={() => accept(suggestion)}>Follow</button
						><button
							disabled={work.busy || !!work.retry}
							onclick={() => {
								const id = suggestion.suggestionId;
								void work.run(
									() => api.instagram.dismissSuggestion(id),
									() => {
										suggestions = suggestions.filter(
											(s) => s.suggestionId !== id
										);
									},
									'Suggestion dismissed.'
								);
							}}>Dismiss</button
						>
					</div>
				</div>{:else}<p class="empty">
					No possible CraftSky accounts yet.
				</p>{/each}{#if suggestionCursor}<button
					disabled={work.busy}
					onclick={moreSuggestions}>Load more matches</button
				>{/if}
		</section>{/if}
	{#if account}<button
			class="danger revoke-verification"
			disabled={work.busy || !!work.retry}
			onclick={async () => {
				if (
					await ask({
						title: 'Revoke Instagram verification?',
						detail:
							'This removes your Instagram verification and deletes your imported handles. Existing CraftSky follows will not be affected.',
						confirmLabel: 'Revoke',
						danger: true
					})
				)
					update(
						() => api.instagram.revoke(),
						'Instagram verification revoked.'
					);
			}}
			><SettingsIcon name="link" size={18} />Revoke Instagram verification</button
		>{/if}
{/if}

<style>
	.instagram-identity {
		flex: 1;
		color: inherit;
		text-decoration: none;
	}
	.revoke-verification {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-top: 24px;
	}
	.import-help {
		display: flex;
		align-items: center;
		gap: 8px;
		margin: 16px 0;
		font-size: 14px;
		font-weight: 600;
		text-decoration: none;
	}
	:global(.settings-workspace) .segmented {
		margin: 0;
	}

	.verification {
		padding: 1rem;
		background: var(--paper);
		border-radius: 10px;
	}
	.challenge {
		font-size: 1rem;
		padding: 1rem;
		border: 1px dashed var(--ink);
		border-radius: 6px;
	}
</style>
