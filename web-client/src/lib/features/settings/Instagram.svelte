<script lang="ts">
	import { goto } from '$app/navigation';
	import LoadMore from '#lib/LoadMore.svelte';
	import StitchSpinner from '#lib/StitchSpinner.svelte';
	import type { InstagramImport, InstagramSuggestion } from '#lib/api/types.ts';
	import { InputError } from '#lib/user-facing-error.ts';
	import { usePrompt } from '../social/prompt';
	const ask = usePrompt();
	import { onMount, onDestroy } from 'svelte';
	import {
		newOperationKey,
		type CraftskyAPI,
		type Page
	} from '#lib/api/client.ts';
	import { dateTime, title } from './model.ts';
	import { safeURL } from '#lib/public.ts';
	import { reportFailure } from '../diagnostics/client.ts';
	import { notify } from '../social/toasts.svelte';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import SettingsIcon from './SettingsIcon.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import CompactIdentity from '../social/CompactIdentity.svelte';
	import InstagramExportPicker from './InstagramExportPicker.svelte';
	import InstagramPosts from './InstagramPosts.svelte';
	import { clearInstagramPostSession } from './instagram-post-session';
	import type { Session } from './model';
	let {
		api,
		session,
		view
	}: { api: CraftskyAPI; session: Session; view?: string } = $props();
	const instagramPath = '/settings/instagram';
	const tools = [
		{
			view: 'following',
			title: 'Import accounts you follow',
			icon: 'following'
		},
		{ view: 'posts', title: 'Import posts', icon: 'image' },
		{ view: 'suggestions', title: 'Suggested accounts', icon: 'people' }
	];
	let viewTitle = $derived(
		view === 'account'
			? 'Manage account'
			: tools.find((tool) => tool.view === view)?.title
	);
	const work = new Work('settings.Instagram.action');
	let loaded = $state(false);
	let loadingImports = $state(false),
		loadingSuggestions = $state(false);
	let importingFollowing = $state(false);
	let disposed = false;
	const archiveAbort = new AbortController();
	type Account = {
		state: string;
		username: string;
		discoverable: boolean;
		conflictPending?: boolean;
		reactivationRequired?: boolean;
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
	type SuggestionAction = 'follow' | 'dismiss';
	type Data = {
		account: { integrationAvailable: boolean; account: Account | null };
		verification: { verification: Verification | null };
		imports: Page<InstagramImport>;
		suggestions: Page<InstagramSuggestion>;
	};
	let available = $state(true),
		account = $state<Account | null>(null),
		verification = $state<Verification | null>(null),
		imports = $state<InstagramImport[]>([]),
		suggestions = $state<InstagramSuggestion[]>([]);
	let pendingSuggestion = $state<{
		id: string;
		action: SuggestionAction;
	} | null>(null);
	let importCursor = $state<string | null>(null),
		suggestionCursor = $state<string | null>(null),
		discoverable = $state(true);
	let pollFailed = $state(false);
	let awaitingDm = $derived(
		verification?.state === 'pendingDm' || verification?.state === 'processing'
	);
	let dmUrl = $derived(safeURL(verification?.dmUrl));
	let needsReactivation = $derived(
		account?.reactivationRequired || account?.state === 'membershipInactive'
	);
	let canImport = $derived(
		account?.state === 'active' &&
			!needsReactivation &&
			!account.conflictPending
	);
	const transport = $derived(api.transport);
	async function fetchData(): Promise<Data> {
		const a = await transport.get<Data['account']>(
			'/v1/migrations/instagram/account'
		);
		const [v, i, s] = await Promise.all([
			!a.account && !view
				? transport.get<Data['verification']>(
						'/v1/migrations/instagram/verifications/current'
					)
				: { verification: null },
			a.account && view === 'following'
				? api.instagram.imports()
				: { items: [] },
			a.account && (!view || view === 'suggestions')
				? loadSuggestionPage()
				: { items: [] }
		]);
		return { account: a, verification: v, imports: i, suggestions: s };
	}
	async function loadSuggestionPage(): Promise<Page<InstagramSuggestion>> {
		try {
			return await api.instagram.suggestions();
		} catch (error) {
			if (view) throw error;
			// A count is optional on the landing page.
			void reportFailure(error, 'settings.Instagram.load');
			return { items: [] };
		}
	}
	function apply(data: Data) {
		loaded = true;
		available = data.account.integrationAvailable;
		account = data.account.account;
		if (!canImport) clearInstagramPostSession();
		if (!account && view) void goto(instagramPath, { replaceState: true });
		discoverable = account?.discoverable ?? discoverable;
		applyVerification(data.verification.verification);
		imports = data.imports.items;
		importCursor = data.imports.cursor || null;
		suggestions = data.suggestions.items;
		suggestionCursor = data.suggestions.cursor || null;
	}
	function applyVerification(next: Verification | null) {
		const previous = verification;
		if (
			next?.state === 'pendingConfirmation' &&
			(previous?.state !== next.state ||
				previous.verificationId !== next.verificationId)
		)
			discoverable = true;
		verification =
			next &&
			previous?.verificationId === next.verificationId &&
			(next.state === 'pendingDm' || next.state === 'processing')
				? {
						...next,
						challenge: next.challenge ?? previous.challenge,
						dmUrl: next.dmUrl ?? previous.dmUrl
					}
				: next;
		pollFailed = false;
	}
	$effect(() => {
		const id = verification?.verificationId;
		if (!id || !awaitingDm || !available) return;
		const controller = new AbortController();
		let checking = false;
		const timer = window.setInterval(async () => {
			if (checking || work.busy || work.loading || work.retry) return;
			checking = true;
			try {
				const next = await transport.get<Verification>(
					`/v1/migrations/instagram/verifications/${encodeURIComponent(id)}`,
					undefined,
					controller.signal
				);
				if (
					!controller.signal.aborted &&
					!work.busy &&
					!work.loading &&
					awaitingDm &&
					verification?.verificationId === id &&
					next.verificationId === id
				)
					applyVerification(next);
			} catch (error) {
				if (controller.signal.aborted || work.busy || work.loading) return;
				if (!pollFailed) void reportFailure(error, 'settings.Instagram.load');
				pollFailed = true;
			} finally {
				checking = false;
			}
		}, 2000);
		return () => {
			window.clearInterval(timer);
			controller.abort();
		};
	});
	function load() {
		return work.load(fetchData, apply, 'settings.Instagram.load');
	}
	function update(
		action: () => Promise<unknown>,
		notice: string,
		afterUpdate?: () => void
	) {
		void work.run(
			async () => {
				await action();
				return fetchData();
			},
			(data) => {
				apply(data);
				afterUpdate?.();
			},
			notice,
			'settings.Instagram.update'
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
			'Discovery preference saved.',
			'settings.Instagram.setDiscoverable'
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
			applyVerification,
			'',
			'settings.Instagram.verify'
		);
	}
	function cancelVerification() {
		const id = verification?.verificationId;
		if (id)
			update(
				() => api.instagram.cancelVerification(id),
				'Verification cancelled.'
			);
	}
	async function importFollowing(file: File) {
		if (!canImport || work.loading || work.busy || work.retry) return;
		await work.run(
			async () => {
				importingFollowing = true;
				try {
					const { readFollowingExport } =
						await import('./read-following-export');
					const { entries } = await readFollowingExport(
						file,
						archiveAbort.signal
					);
					archiveAbort.signal.throwIfAborted();
					if (!entries.length)
						throw new InputError(
							'No Instagram usernames were found in this export.'
						);
					return await api.instagram.createImport({
						sourceType: 'instagramJson',
						entries
					});
				} finally {
					importingFollowing = false;
				}
			},
			({ import: item }) => {
				imports = [
					item,
					...imports.filter((existing) => existing.importId !== item.importId)
				];
			},
			'Instagram import created',
			'settings.Instagram.importFollowing'
		);
	}
	function importSourceLabel(source: string) {
		if (source === 'instagramJson') return 'Instagram export';
		if (source === 'manual') return 'Manual handles';
		return 'Instagram import';
	}
	function deleteFollowingImport(id: string) {
		void work.run(
			() => api.instagram.deleteImport(id),
			() => {
				imports = imports.filter((item) => item.importId !== id);
			},
			'Import deleted.'
		);
	}
	function changeSuggestion(id: string, action: SuggestionAction) {
		const operationKey = action === 'follow' ? newOperationKey() : null;
		void work.run(
			async () => {
				pendingSuggestion = { id, action };
				try {
					if (operationKey)
						await api.instagram.acceptSuggestion(id, operationKey);
					else await api.instagram.dismissSuggestion(id);
				} finally {
					pendingSuggestion = null;
				}
			},
			() => {
				suggestions = suggestions.filter((s) => s.suggestionId !== id);
			},
			action === 'follow' ? 'Following this maker.' : 'Suggestion dismissed.',
			action === 'follow'
				? 'settings.Instagram.accept'
				: 'settings.Instagram.action'
		);
	}
	function moreImports() {
		const next = importCursor;
		if (next && !work.busy && !work.retry)
			void work.run(
				async () => {
					loadingImports = true;
					try {
						return await api.instagram.imports({ cursor: next });
					} finally {
						loadingImports = false;
					}
				},
				(page) => {
					imports = [...imports, ...page.items];
					importCursor = page.cursor || null;
				},
				'',
				'settings.Instagram.moreImports'
			);
	}
	function moreSuggestions() {
		const next = suggestionCursor;
		if (next && !work.busy && !work.retry)
			void work.run(
				async () => {
					loadingSuggestions = true;
					try {
						return await api.instagram.suggestions({ cursor: next });
					} finally {
						loadingSuggestions = false;
					}
				},
				(page) => {
					suggestions = [...suggestions, ...page.items];
					suggestionCursor = page.cursor || null;
				},
				'',
				'settings.Instagram.moreSuggestions'
			);
	}
	onMount(load);
	onDestroy(() => {
		disposed = true;
		work.dispose();
		archiveAbort.abort();
	});
</script>

{#snippet accountWarnings()}
	{#if account?.conflictPending || account?.state === 'disputed'}<p
			class="notice"
		>
			This account has a pending ownership conflict. Verification may be
			required again.
		</p>{/if}
	{#if needsReactivation}<p class="notice">
			Reactivate your Instagram account to start a new import.
		</p>{/if}
{/snippet}

{#snippet instagramEmpty(icon: string, heading: string, detail: string)}
	<div class="empty instagram-empty">
		<SettingsIcon name={icon} size={48} />
		<strong>{heading}</strong>
		<p class="helper">{detail}</p>
	</div>
{/snippet}

{#snippet suggestionLabel(label: string, pending: boolean)}
	<span class:pending>{label}</span>
	{#if pending}<span class="suggestion-spinner" aria-hidden="true">
			<StitchSpinner />
		</span>{/if}
{/snippet}

<h1 class="workspace-heading">
	{viewTitle ?? (view ? 'Page not found' : 'Instagram')}
</h1>
<Feedback {work} reload={load} />
{#if !work.loading && loaded}
	{#if !available && account}<p class="notice">
			Instagram verification is temporarily unavailable. Your saved account and
			imports are still available.
		</p>{/if}
	{#if (!view && !account) || (view === 'account' && account)}
		<section
			class="panel stack"
			class:unverified={!account}
			class:manage-account={!!account}
		>
			<h2 class="section-heading">
				<SettingsIcon name={account ? 'link' : 'verifiedAccount'} />{account
					? 'Instagram account'
					: 'Verify your Instagram account'}
			</h2>
			{#if account}<p class="account-handle">
					Verified as <strong>@{account.username}</strong>
				</p>
				{@render accountWarnings()}
				{#if needsReactivation}<p>
						Reactivation keeps discovery off until you choose to turn it on
						again.
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
			{:else if !available}<p>
					Instagram verification is unavailable right now.
				</p>
				<p class="helper">
					Imports become available after Instagram verification is configured
					and your account is verified.
				</p>
			{:else if !awaitingDm}<p>
					Send a one-time challenge to CraftSky's official Instagram account.
					You will confirm the username here before it is verified.
				</p>{/if}
			{#if available && verification && awaitingDm}
				<div class="challenge-controls">
					<p>Send this exact one-time challenge in an Instagram DM:</p>
					<p class="challenge">
						<span class="visually-hidden">Instagram verification challenge</span
						>
						<span>{verification.challenge ?? 'Checking your message…'}</span>
					</p>
					{#if !verification.challenge && verification.state === 'pendingDm'}
						<p class="helper">
							Use the challenge from the tab where you started verification, or
							cancel and start again.
						</p>
					{/if}
					<button
						class="challenge-copy"
						disabled={work.busy || !verification.challenge}
						onclick={() => {
							const challenge = verification?.challenge;
							if (challenge)
								void work.run(
									() => navigator.clipboard.writeText(challenge),
									() => {},
									'Challenge copied.'
								);
						}}
						><SettingsIcon name="copyChallenge" size={18} bold />Copy challenge</button
					>
					{#if dmUrl}
						<a
							class="action primary challenge-open"
							href={dmUrl}
							target="_blank"
							rel="noreferrer"
							data-external-confirmed
							><SettingsIcon name="external" size={18} bold />Open Instagram DM</a
						>
					{:else}
						<button class="primary challenge-open" disabled
							><SettingsIcon name="external" size={18} bold />Open Instagram DM</button
						>
					{/if}
					<button
						class="challenge-cancel"
						disabled={work.busy}
						onclick={cancelVerification}>Cancel verification</button
					>
					{#if pollFailed}<p role="status">
							Couldn’t check verification. Trying again…
						</p>{/if}
				</div>
			{:else if (available || account) && verification?.state === 'pendingConfirmation'}<div
					class="verification stack"
				>
					<h3>Confirm your username</h3>
					<p class="muted">Expires {dateTime(verification.expiresAt)}</p>
					<p>
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
					>
					<button disabled={work.busy} onclick={cancelVerification}
						>Cancel verification</button
					>
				</div>
			{:else if available || account}{#if verification}<p class="muted">
						Previous verification: {title(
							verification.state
						)}{verification.retryCode
							? ` · ${title(verification.retryCode)}`
							: ''}
					</p>{/if}{#if available && !account}<button
						class="primary verification-start"
						disabled={work.busy}
						onclick={verify}
						><SettingsIcon name="verifiedAccount" size={18} bold />
						Create verification challenge</button
					>{/if}{/if}
		</section>
	{/if}
	{#if !view && !account}
		<p>Complete verification to import the accounts you follow.</p>
	{:else if !view && account}
		<section class="panel stack linked-summary">
			<div class="linked-account">
				<p>
					<SettingsIcon name="verifiedAccount" size={20} />
					<span>Verified as <strong>@{account.username}</strong></span>
				</p>
				<a href={`${instagramPath}/account`}>Manage account</a>
			</div>
			{@render accountWarnings()}
		</section>
		<nav class="instagram-tools" aria-label="Instagram tools">
			{#each tools as tool (tool.view)}
				<a class="settings-tile" href={`${instagramPath}/${tool.view}`}>
					<SettingsIcon name={tool.icon} />
					<span class="tile-copy">{tool.title}</span>
					{#if tool.view === 'suggestions' && suggestions.length}
						<span class="suggestion-count"
							>{suggestions.length}{suggestionCursor ? '+' : ''}</span
						>
					{/if}
					<SettingsIcon name="chevron" size={20} />
				</a>
			{/each}
		</nav>
	{:else if !viewTitle && view}
		<p>Choose a page in <a href={instagramPath}>Instagram settings</a>.</p>
	{:else if view === 'posts' && account}
		{#if canImport}
			<InstagramPosts
				{api}
				owner={session.did}
				sessionId={session.id || session.did}
			/>
		{:else}
			<p>
				Resolve the issue shown in <a href={`${instagramPath}/account`}
					>Manage account</a
				> before importing posts.
			</p>
		{/if}
	{:else if view === 'following' && account}
		{#if canImport}
			<a
				class="button import-help"
				href="https://craftsky.social/instagram-import"
				target="_blank"
				rel="noopener noreferrer"
				aria-label="How to export your Instagram following"
			>
				<SettingsIcon name="help" size={20} />
				<span>How to export</span>
				<SettingsIcon name="external" size={16} />
			</a>
			<section
				class="panel stack instagram-export following-export"
				aria-label="Instagram following export"
			>
				<p class="helper">
					Choose JSON format and All time when exporting from Instagram, then
					select the downloaded ZIP here. CraftSky processes it on this device
					and uploads only the usernames of accounts you follow. Everything else
					stays on your device.
				</p>
				<InstagramExportPicker
					accept="application/json,application/zip,.json,.zip"
					disabled={work.busy || !!work.retry}
					loading={importingFollowing}
					onSelect={importFollowing}
				/>
				<p class="helper import-disclosure">
					Importing creates private suggestions only. You choose whether to
					follow each account.
				</p>
			</section>
		{:else}
			<p>
				Resolve the issue shown in <a href={`${instagramPath}/account`}
					>Manage account</a
				> before starting a new import.
			</p>
		{/if}
		<section class="panel following-history">
			<h2 class="section-heading">
				<SettingsIcon name="projectCount" />Your imports
			</h2>
			{#each imports as item (item.importId)}
				<div class="import-row">
					<div>
						<h3>{importSourceLabel(item.sourceType)}</h3>
						<p>
							{item.followingCount}
							{item.followingCount === 1 ? 'account' : 'accounts'} imported
						</p>
						{#if ['inactive', 'membershipInactive'].includes(item.state)}
							<p>
								This import paused when your CraftSky membership changed.
								Reactivate it to resume matching.
							</p>
							<button
								disabled={work.busy || !!work.retry}
								onclick={() =>
									update(
										() =>
											api.instagram.updateImport(item.importId, {
												reactivate: true
											}),
										'Import reactivated.'
									)}>Reactivate import</button
							>
						{/if}
					</div>
					<button
						type="button"
						class="danger import-delete"
						aria-label="Delete import"
						title="Delete import"
						disabled={work.busy || !!work.retry}
						onclick={() => deleteFollowingImport(item.importId)}
						><SettingsIcon name="trash" size={24} bold /></button
					>
				</div>
			{:else}{@render instagramEmpty(
					'history',
					'No Instagram imports yet.',
					'0 accounts imported'
				)}{/each}{#if importCursor}<LoadMore
					loading={loadingImports}
					disabled={work.busy || !!work.retry}
					label="Load more imports"
					loadingLabel="Loading more imports"
					onclick={moreImports}
				/>{/if}
		</section>
	{:else if view === 'suggestions' && account}
		<section class="panel suggested-accounts" aria-label="Suggested accounts">
			<p class="helper">
				Your imports find possible CraftSky accounts privately. Nobody is
				followed until you choose Follow.
			</p>
			{#each suggestions as suggestion (suggestion.suggestionId)}
				{@const id = suggestion.suggestionId}
				{@const activeAction =
					pendingSuggestion?.id === id ? pendingSuggestion.action : null}
				<div class="suggestion-row">
					<a
						class="identity instagram-identity"
						href={`/profiles/${encodeURIComponent(suggestion.target.did)}`}
						><Avatar author={suggestion.target} /><CompactIdentity
							author={suggestion.target}
						/></a
					>
					<div class="suggestion-actions">
						<button
							type="button"
							class="primary filled-pill suggestion-follow"
							disabled={work.busy || !!work.retry}
							aria-busy={activeAction === 'follow'}
							onclick={() => changeSuggestion(id, 'follow')}
							>{@render suggestionLabel(
								'Follow',
								activeAction === 'follow'
							)}</button
						><button
							type="button"
							class="suggestion-dismiss"
							disabled={work.busy || !!work.retry}
							aria-busy={activeAction === 'dismiss'}
							onclick={() => changeSuggestion(id, 'dismiss')}
							>{@render suggestionLabel(
								'Dismiss',
								activeAction === 'dismiss'
							)}</button
						>
					</div>
				</div>
			{:else}
				{@render instagramEmpty(
					'instagramMatch',
					'No possible CraftSky accounts yet.',
					'Import the accounts you follow on Instagram to look for matches.'
				)}
			{/each}{#if suggestionCursor}<LoadMore
					loading={loadingSuggestions}
					disabled={work.busy || !!work.retry}
					loadingLabel="Loading more suggestions"
					onclick={moreSuggestions}
				/>{/if}
		</section>
	{/if}
	{#if account && view === 'account'}<button
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
						'',
						() => notify('Instagram verification revoked.')
					);
			}}
			><SettingsIcon name="unlink" size={18} bold />Revoke Instagram
			verification</button
		>{/if}
{/if}

<style>
	.linked-account,
	.linked-account p {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.linked-account {
		justify-content: space-between;
		flex-wrap: wrap;
	}
	.linked-summary .linked-account p {
		margin: 0;
		overflow-wrap: anywhere;
	}
	.linked-account p :global(svg) {
		flex: none;
		color: var(--primary);
	}
	.instagram-tools .settings-tile + .settings-tile {
		border-top: 1px solid var(--hairline);
	}
	.suggestion-count {
		color: var(--ink-2);
		font-variant-numeric: tabular-nums;
	}
	.panel.manage-account {
		gap: 8px;
		margin-bottom: 0;
	}
	.manage-account :is(.section-heading, .account-handle) {
		margin: 0;
	}
	.manage-account .account-handle {
		font-size: 14px;
		overflow-wrap: anywhere;
	}
	.account-handle strong,
	.manage-account .switch-row {
		font-weight: 600;
	}
	.manage-account .switch-row {
		line-height: 1.3;
	}
	.manage-account input[role='switch']:not(:checked) {
		--discovery-switch-outline: color-mix(
			in srgb,
			var(--ink) 46%,
			var(--surface)
		);
		border-color: var(--discovery-switch-outline);
		background: color-mix(in srgb, var(--ink) 10%, var(--surface));
	}
	.manage-account input[role='switch']:not(:checked)::after {
		background: var(--discovery-switch-outline);
	}
	.panel.unverified {
		gap: 0;
	}
	.unverified .section-heading {
		display: flex;
		align-items: center;
		gap: 8px;
		margin: 0 0 8px;
		padding: 0;
	}
	.unverified .section-heading :global(svg) {
		color: var(--primary);
		flex: none;
	}
	.unverified .verification-start.primary {
		min-height: 40px;
		padding-block: 8px;
		border: 0;
		border-radius: 999px;
		box-shadow: none;
		text-align: center;
	}
	.verification-start :global(svg) {
		flex: none;
	}
	.suggestion-row {
		padding-bottom: 8px;
		margin-bottom: 12px;
		border-bottom: 1px solid var(--hairline);
	}
	.instagram-identity {
		color: inherit;
		text-decoration: none;
		border-radius: 8px;
	}
	.instagram-identity :global(.avatar) {
		width: 41px;
		height: 41px;
	}
	@media (hover: hover) {
		.instagram-identity:hover {
			background: color-mix(in srgb, var(--primary) 8%, var(--surface));
		}
	}
	.suggestion-actions {
		display: grid;
		grid-template-columns: minmax(0, 1fr) auto;
		gap: 8px;
		margin-top: 8px;
	}
	.suggestion-actions button {
		position: relative;
	}
	button.suggestion-dismiss {
		min-height: 40px;
		padding: 8px 12px;
		border: 0;
		border-radius: 999px;
		background: transparent;
		color: var(--primary);
	}
	button.suggestion-dismiss:not(:disabled):hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	button.suggestion-follow.primary[aria-busy='true'] {
		background: color-mix(in srgb, var(--ink) 12%, var(--surface));
	}
	.suggestion-actions button[aria-busy='true'] {
		opacity: 1;
	}
	.pending {
		opacity: 0;
	}
	.suggestion-spinner {
		position: absolute;
		inset: 0;
		display: grid;
		place-items: center;
		pointer-events: none;
	}
	.suggestion-spinner :global(.stitch-spinner),
	.suggestion-spinner :global(svg) {
		width: 18px;
		height: 18px;
	}
	button.revoke-verification {
		display: flex;
		width: fit-content;
		max-width: 100%;
		margin: 32px auto 0;
		padding: 8px 12px;
		border: 0;
		border-radius: 999px;
		background: transparent;
	}
	.revoke-verification :global(svg) {
		flex: none;
	}
	button.revoke-verification:not(:disabled):hover {
		background: color-mix(in srgb, currentColor 8%, transparent);
	}
	button.revoke-verification:not(:disabled):active {
		background: color-mix(in srgb, currentColor 12%, transparent);
	}
	.import-help {
		display: flex;
		justify-content: flex-start;
		gap: 8px;
		width: fit-content;
		max-width: 100%;
		margin-bottom: 16px;
		padding: 14px 28px;
		background: var(--lilac);
		color: var(--ink);
		font-weight: 700;
		line-height: 1.2;
		letter-spacing: 0.2px;
	}
	.import-help :global(svg) {
		flex: none;
	}
	.import-help:hover {
		background: color-mix(in srgb, var(--lilac), black 8%);
		color: var(--ink);
	}
	.import-help:active {
		background: color-mix(in srgb, var(--lilac), black 18%);
	}
	.following-export.instagram-export {
		gap: 12px;
	}
	.following-export .import-disclosure {
		font-size: 13px;
	}
	.following-history .section-heading {
		margin-bottom: 8px;
	}
	.import-row {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: 12px;
		padding: 8px 0 12px;
		margin-bottom: 14px;
		border-bottom: 1px solid var(--hairline);
	}
	.import-row h3 {
		font-weight: 600;
	}
	.import-row p {
		font-size: 14px;
		margin: 0;
	}
	.import-row p + p {
		margin: 4px 0 8px;
	}
	button.import-delete {
		width: 48px;
		min-height: 48px;
		padding: 0;
		border: 0;
		border-radius: 50%;
		background: transparent;
		flex: none;
	}
	button.import-delete:not(:disabled):hover {
		background: color-mix(in srgb, currentColor 8%, transparent);
	}
	.instagram-empty {
		padding: 20px 16px;
	}
	.instagram-empty strong {
		color: var(--ink);
		font-weight: 600;
		line-height: 1.25;
	}
	.suggested-accounts .instagram-empty {
		max-width: 400px;
		margin-inline: auto;
		padding-inline: 20px;
	}
	.suggested-accounts .helper {
		letter-spacing: 0.25px;
	}
	.suggested-accounts :global(.load-more) {
		min-height: 48px;
		padding: 0;
	}

	.verification {
		padding: 1rem;
		background: var(--paper);
		border-radius: 10px;
	}
	.challenge-controls {
		display: grid;
		gap: 8px;
		min-width: 0;
	}
	.challenge-controls p {
		margin: 0;
	}
	.challenge-controls .challenge {
		font-size: 22px;
		font-weight: 600;
		line-height: 1.3;
		text-align: center;
		overflow-wrap: anywhere;
	}
	.challenge-controls :is(.challenge-copy, .challenge-open, .challenge-cancel) {
		min-height: 40px;
		padding-block: 8px;
		border-radius: 999px;
		box-shadow: none;
		text-align: center;
	}
	.challenge-controls .challenge-copy {
		border: 1px solid var(--ink-4);
		color: var(--primary);
	}
	.challenge-controls .challenge-open.primary {
		border: 0;
	}
	.challenge-controls .challenge-cancel {
		border: 0;
		background: transparent;
		color: var(--primary);
	}
	.challenge-controls :global(svg) {
		flex: none;
	}
</style>
