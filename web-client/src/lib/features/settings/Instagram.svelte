<script lang="ts">
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
	let { api }: { api: CraftskyAPI } = $props();
	const work = new Work();
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
		discoverable = account?.discoverable ?? discoverable;
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
			work.error =
				error instanceof Error ? error.message : 'Could not read usernames.';
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
		work.dispose();
		archiveAbort.abort();
	});
</script>

<h1>Find your Instagram community</h1>
<Feedback {work} reload={load} />
{#if !work.loading}
	{#if !available}<p class="notice">
			Instagram verification is temporarily unavailable. Your saved account and
			imports remain below.
		</p>{/if}
	<section class="panel stack">
		<h2>Your Instagram account</h2>
		{#if account}<p>
				<strong>@{account.username}</strong> · {title(account.state)}
			</p>
			{#if account.verifiedAt}<p class="muted">
					Verified {dateTime(account.verifiedAt)}
				</p>{/if}{#if account.conflictPending}<p class="notice">
					This account has a pending ownership conflict. Verification may be
					required again.
				</p>{/if}<label class="check"
				><input
					type="checkbox"
					checked={account.discoverable}
					disabled={work.busy || !!work.retry}
					onchange={(event) =>
						update(
							() =>
								api.instagram.settings({
									discoverable: event.currentTarget.checked
								}),
							'Discovery preference saved.'
						)}
				/>Let people who followed me on Instagram find me on CraftSky</label
			>{#if account.reactivationRequired}<button
					disabled={work.busy}
					onclick={() =>
						update(
							() => api.instagram.settings({ reactivate: true }),
							'Instagram account reactivated.'
						)}>Reactivate account</button
				>{/if}<button
				class="danger"
				disabled={work.busy || !!work.retry}
				onclick={() => {
					if (
						confirm(
							'Disconnect your Instagram account from CraftSky discovery?'
						)
					)
						update(
							() => api.instagram.revoke(),
							'Instagram account disconnected.'
						);
				}}>Disconnect Instagram account</button
			>
		{:else}<p>
				Verify your Instagram username so the people you already know can find
				you here. CraftSky will never ask for your Instagram password.
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
					<label class="check"
						><input type="checkbox" bind:checked={discoverable} />Allow people
						who followed me to find me here</label
					><button
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
				</p>{/if}<button
				class="primary"
				disabled={work.busy || !available}
				onclick={verify}
				>{account
					? 'Verify another username'
					: 'Verify Instagram username'}</button
			>{/if}
	</section>
	<section class="panel stack">
		<h2>Import your following list</h2>
		<p>
			Import usernames from Instagram’s JSON data export, or enter one username
			per line. Your imported list stays private.
		</p>
		<label
			>Choose your Instagram ZIP export or following.json<input
				type="file"
				accept="application/json,application/zip,.json,.zip"
				disabled={work.busy}
				onchange={readFile}
			/></label
		>
		<details>
			<summary>Enter usernames manually</summary>
			<form class="stack" onsubmit={parseManual}>
				<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
					<label
						>Instagram usernames<textarea
							bind:value={manual}
							placeholder={'@maker_one\nmaker_two'}></textarea></label
					><button disabled={work.busy}>Review usernames</button>
				</fieldset>
			</form>
		</details>
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
		<h2>Your imports</h2>
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
							onclick={() => {
								if (confirm('Delete this imported following list?'))
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
		<h2>Makers you know</h2>
		<p class="muted">
			Matches from your private following list who chose to be discoverable.
		</p>
		{#each suggestions as suggestion (suggestion.suggestionId)}<div class="row">
				<div class="identity">
					{#if safeURL(suggestion.target.avatar)}<img
							class="avatar"
							src={safeURL(suggestion.target.avatar)}
							alt=""
							loading="lazy"
						/>{/if}<a
						href={`/app/profile/${encodeURIComponent(suggestion.target.did)}`}
						><strong
							>{suggestion.target.displayName ||
								suggestion.target.handle}</strong
						><small>@{suggestion.target.handle}</small></a
					>
				</div>
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
				No new matches yet. Check back as more makers join CraftSky.
			</p>{/each}{#if suggestionCursor}<button
				disabled={work.busy}
				onclick={moreSuggestions}>Load more matches</button
			>{/if}
	</section>
{/if}

<style>
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
	summary {
		cursor: pointer;
		padding: 0.75rem 0;
		font-weight: 600;
	}
</style>
