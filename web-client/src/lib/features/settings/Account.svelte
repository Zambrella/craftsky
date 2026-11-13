<script lang="ts">
	import AccountProvider from './AccountProvider.svelte';
	import { reportFailure } from '../diagnostics/client.ts';
	import { APIError } from '#lib/api/client.ts';
	import { UserFacingError } from '#lib/user-facing-error.ts';
	import { usePrompt } from '../social/prompt';
	const ask = usePrompt();
	import { onMount, onDestroy, tick } from 'svelte';
	import type {
		CraftskyAPI,
		EligibilityStatus,
		ModerationHistoryItem,
		Profile
	} from '#lib/api/client.ts';
	import { dateTime, record, text, type Session } from './model.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import SettingsIcon from './SettingsIcon.svelte';
	import { moderationLabel } from './native-settings.ts';
	import { drafts } from '../drafts/store.ts';
	import { forgetPush } from '../push/client.ts';
	import { clearAccountChat } from './account-cleanup.ts';
	import { moderationSubject } from './navigation.ts';
	let {
		api,
		session,
		section,
		reference
	}: {
		api: CraftskyAPI;
		session: Session;
		section: string;
		reference?: string;
	} = $props();
	const work = new Work('settings.Account.action');
	const version = __CRAFTSKY_VERSION__.trim();
	let businessAccess = $state(false);
	let profile = $state<Profile | null>(null),
		eligibility = $state<EligibilityStatus | null>(null);
	let standing = $state<Record<string, unknown>>({}),
		history = $state<ModerationHistoryItem[]>([]),
		cursor = $state<string | null>(null);
	let deletionDialog = $state<HTMLDialogElement>();
	$effect(() => {
		if (deleteOpen) void tick().then(() => deletionDialog?.showModal());
	});
	let deleteOpen = $state(false),
		confirmation = $state(''),
		pending = $state<{ jobId: string; did: string } | null>(null);
	async function browser<T>(
		path: string,
		body?: unknown,
		method = body === undefined ? 'GET' : 'POST'
	): Promise<T> {
		if (!session.id)
			throw new UserFacingError('Reopen your account settings to continue.');
		const response = await fetch(`/api/auth/deletion/${path}`, {
			method,
			credentials: 'same-origin',
			headers: {
				'X-Craftsky-Account-Id': session.id,
				...(body === undefined ? {} : { 'Content-Type': 'application/json' })
			},
			redirect: 'manual',
			cache: 'no-store',
			body: body === undefined ? undefined : JSON.stringify(body)
		});
		if (!response.ok) {
			let value: unknown;
			try {
				value = await response.json();
			} catch {}
			throw new APIError(
				response.status,
				text(record(value).error) || 'request_failed',
				'Account deletion failed'
			);
		}
		return response.status === 204
			? (undefined as T)
			: ((await response.json()) as T);
	}
	function load() {
		if (section === 'about') {
			work.loading = false;
			return;
		}
		if (section === 'moderation')
			return work.load(
				() =>
					Promise.all([
						api.moderation.standing(),
						reference
							? api.moderation.entry(reference)
							: api.moderation.history()
					]),
				([state, page]) => {
					standing = record(state);
					history = page.items;
					cursor = reference ? null : page.cursor || null;
				},
				'settings.Account.load'
			);
		return work.load(
			async () => {
				const eligible = await api.account.eligibility();
				const [me, status, access] = await Promise.all([
					eligible.state === 'eligible'
						? api.profiles.me()
						: Promise.resolve(null),
					browser<{ pending: boolean; jobId?: string; did?: string }>('status'),
					eligible.state === 'eligible'
						? api.billing.access().catch((error) => {
								void reportFailure(error, 'settings.Account.access');
								return null;
							})
						: Promise.resolve(null)
				]);
				return [me, eligible, status, access] as const;
			},
			([me, eligible, status, access]) => {
				businessAccess =
					!!access?.givesAccess && access.effectiveTier === 'business';
				profile = me;
				eligibility = eligible;
				if (status.pending && status.did === session.did && status.jobId) {
					pending = { jobId: status.jobId, did: status.did };
					deleteOpen = true;
				} else {
					pending = null;
					try {
						sessionStorage.removeItem('craftsky.deletion');
					} catch {}
				}
			},
			'settings.Account.load'
		);
	}
	async function beginDeletion() {
		if (
			await ask({
				title: 'Delete CraftSky account?',
				detail: `Deleting the CraftSky account identified by ${session.did} permanently removes all your CraftSky data from your PDS and all private data held by CraftSky. It won’t delete your PDS, DID, or wider AT Protocol account.\n\nTo continue, you’ll need to authenticate with your PDS again.`,
				confirmLabel: 'Continue',
				danger: true
			})
		)
			startDeletion();
	}
	function startDeletion() {
		void work.run(
			() =>
				browser<{
					jobId: string;
					authUrl: string;
					confirmationDid: string;
					expiresAt: string;
				}>('start', {}),
			(intent) => {
				try {
					sessionStorage.setItem(
						'craftsky.deletion',
						JSON.stringify({
							jobId: intent.jobId,
							did: session.did,
							confirmationDid: intent.confirmationDid,
							expiresAt: intent.expiresAt
						})
					);
				} catch {}
				window.location.assign(intent.authUrl);
			},
			'Opening your account provider for fresh verification…',
			'settings.Account.startDeletion'
		);
	}
	function cancelDeletion() {
		if (!pending) {
			deleteOpen = false;
			return;
		}
		const jobId = pending.jobId;
		void work.run(
			() => browser('cancel', { jobId }),
			() => {
				pending = null;
				deleteOpen = false;
				confirmation = '';
				try {
					sessionStorage.removeItem('craftsky.deletion');
				} catch {}
			},
			'Account deletion cancelled.',
			'settings.Account.cancelDeletion'
		);
	}
	function confirmDeletion(event: SubmitEvent) {
		event.preventDefault();
		if (!pending || pending.did !== session.did || confirmation !== session.did)
			return;
		const jobId = pending.jobId;
		void work.run(
			async () => {
				await browser('confirm', { jobId, confirmationDid: confirmation });
				// Acceptance is final. Local cleanup cannot turn it into a retryable deletion.
				await Promise.allSettled([
					drafts.clear(session.did),
					session.id ? forgetPush(session.id) : Promise.resolve(),
					Promise.resolve().then(() => {
						if (session.id)
							clearAccountChat(localStorage, session.id, session.did);
					})
				]);
			},
			() => {
				try {
					sessionStorage.removeItem('craftsky.deletion');
				} catch {}
				window.location.assign('/sign-in');
			},
			'Account deletion requested.',
			'settings.Account.confirmDeletion'
		);
	}
	function more() {
		const next = cursor;
		if (!next) return;
		void work.run(
			() => api.moderation.history({ cursor: next }),
			(page) => {
				history = [...history, ...page.items];
				cursor = page.cursor || null;
			},
			'',
			'settings.Account.more'
		);
	}
	function appeal(reference: string) {
		return `mailto:moderation@craftsky.social?subject=${encodeURIComponent(`CraftSky moderation appeal ${reference}`)}&body=${encodeURIComponent(`Please explain why you are appealing ${reference}.`)}`;
	}
	onMount(load);
	onDestroy(() => work.dispose());
</script>

<h1 class="workspace-heading">
	{section === 'moderation'
		? 'Account standing'
		: section === 'about'
			? 'About'
			: 'Account'}
</h1>
<Feedback {work} reload={load} />
{#if section === 'about'}
	<nav class="nav-list native-list" aria-label="About and legal">
		<a
			class="settings-tile"
			href="https://craftsky.social/terms"
			target="_blank"
			rel="noopener noreferrer"
			><SettingsIcon name="document" /><span class="tile-copy">Terms</span
			><SettingsIcon name="external" size={24} /></a
		>
		<a
			class="settings-tile"
			href="https://craftsky.social/privacy"
			target="_blank"
			rel="noopener noreferrer"
			><SettingsIcon name="shield" /><span class="tile-copy"
				>Privacy policy</span
			><SettingsIcon name="external" size={24} /></a
		>
		{#if version}<div class="settings-tile">
				<SettingsIcon name="about" /><span class="tile-copy"
					><span>Version</span><small>{version}</small></span
				>
			</div>{/if}
	</nav>
{:else if !work.loading && section === 'account' && eligibility}
	<div class="native-list">
		{#key session.id ?? session.did}<AccountProvider {api} {session} />{/key}
		<h2 class="section-label">CraftSky account</h2>
		<p class="account-type">
			Deleting your CraftSky account removes your CraftSky data. It does not
			delete your Bluesky or AT Protocol account.
		</p>
		{#if profile}<h2 class="section-label">Account type</h2>
			<p class="account-type">
				{businessAccess ? 'Business' : 'Regular'}
			</p>{/if}

		<button
			class="settings-tile delete-account-row"
			disabled={work.busy}
			onclick={beginDeletion}
			><SettingsIcon name="delete" bold /><span class="tile-copy"
				>Delete CraftSky account</span
			></button
		>
	</div>
	{#if deleteOpen}<dialog
			class="settings-dialog deletion-confirmation"
			bind:this={deletionDialog}
			oncancel={(event) => {
				if (work.busy) event.preventDefault();
				else {
					event.preventDefault();
					cancelDeletion();
				}
			}}
		>
			<div class="stack">
				<Feedback {work} />
				<h2>Delete CraftSky account</h2>

				<p>
					Permanently remove your CraftSky records and private CraftSky data.
					This does not delete your identity or your account with your account
					provider. This action cannot be undone.
				</p>
				<p class="notice">
					Deleting your CraftSky account does not cancel subscriptions or
					provider billing. Manage subscriptions with your store before deleting
					your account.
				</p>
				{#if pending}<form class="stack" onsubmit={confirmDeletion}>
						<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
							<p>
								Your account provider has verified your identity. To confirm
								permanent deletion, type your complete account identifier
								exactly:
							</p>
							<code>{session.did}</code><label
								>Account identifier<input
									name="confirmation"
									required
									autocomplete="off"
									spellcheck="false"
									bind:value={confirmation}
								/></label
							>
							<div class="actions">
								<button
									type="submit"
									class="danger"
									disabled={work.busy || confirmation !== session.did}
									>Permanently delete CraftSky account</button
								><button
									type="button"
									disabled={work.busy}
									onclick={cancelDeletion}>Cancel deletion</button
								>
							</div>
						</fieldset>
					</form>
				{:else}<div class="stack">
						<p>
							First, verify your identity by signing in again with your account
							provider. You will return here to confirm deletion.
						</p>
						<div class="actions">
							<button
								class="danger"
								disabled={work.busy}
								onclick={startDeletion}>Verify identity to continue</button
							><button disabled={work.busy} onclick={cancelDeletion}
								>Cancel</button
							>
						</div>
					</div>{/if}
			</div>
		</dialog>{/if}
{:else if !work.loading && section === 'moderation'}
	<section class="panel standing-summary">
		<div
			class="standing-icon"
			class:standing-warning={!standing.suspended &&
				Number(standing.activeStrikeCount) > 0}
			class:standing-error={!!standing.suspended}
		>
			<SettingsIcon name={standing.suspended ? 'lock' : 'privacy'} size={30} />
		</div>
		<div>
			<h2>
				{standing.suspended
					? 'Account suspended'
					: Number(standing.activeStrikeCount) > 0
						? 'Action required'
						: 'Account in good standing'}
			</h2>
			{#if !standing.severeSuspended && !standing.thresholdSuspended}<p>
					Active strikes: {Number(standing.activeStrikeCount) || 0} of the {Number(
						standing.strikeThreshold
					) || '—'} strike suspension threshold.
				</p>{/if}
			{#if standing.severeSuspended}<p>
					Your account is suspended following a severe policy violation.
				</p>{:else if standing.thresholdSuspended}<p>
					Your account has reached the active strike threshold.
				</p>{/if}
		</div>
	</section>
	<section class="moderation-history">
		<h2>{reference ? 'Case history' : 'Moderation history'}</h2>
		{#each history as raw}{@const entry = record(raw)}{@const subject =
				moderationSubject(entry.safeSnapshot)}{@const ref = text(
				entry.caseReference
			)}
			<article class="panel history-entry">
				<div class="row">
					<SettingsIcon name="warning" />
					<div>
						<h3>{moderationLabel('reason', entry.reason)}</h3>
						<time>{dateTime(entry.occurredAt)}</time>
					</div>
				</div>
				<p>{text(entry.userSafeDetail)}</p>
				{#if subject.reference}<p class="muted">
						Affected content: <code>{subject.reference}</code>
					</p>{/if}
				<p class="muted">
					{ref} · {moderationLabel('appeal', entry.appealStatus)}
				</p>
				{#if Array.isArray(entry.effects) && entry.effects.length}<h4>
						Consequences
					</h4>
					<ul>
						{#each entry.effects as rawEffect}{@const effect =
								record(rawEffect)}
							<li>
								{moderationLabel('effect', effect.type)} · {moderationLabel(
									'action',
									effect.action
								)}{effect.dueAt ? ` (${dateTime(effect.dueAt)})` : ''}
							</li>{/each}
					</ul>{/if}
				<div class="actions">
					{#if subject.href}<a class="action" href={subject.href}
							>{subject.label}</a
						>{/if}
					<button
						disabled={work.busy}
						onclick={() =>
							work.run(
								() =>
									navigator.clipboard.writeText('moderation@craftsky.social'),
								() => {},
								'Email address copied.'
							)}>Copy email address</button
					>
					{#if !reference}<a
							class="action"
							href={`/settings/moderation/${encodeURIComponent(ref)}`}
							>Case details</a
						>{/if}{#if entry.eventType === 'decision' && entry.appealStatus === 'none'}<a
							class="action"
							href={appeal(ref)}>Appeal by email</a
						>{/if}<button
						disabled={work.busy}
						onclick={() =>
							work.run(
								() => navigator.clipboard.writeText(ref),
								() => {},
								'Case reference copied.'
							)}>Copy reference</button
					>
				</div>
			</article>{:else}<p class="empty">
				<SettingsIcon name="moderation" /><strong>No moderation history.</strong
				>There are no moderation decisions to show for this account.
			</p>{/each}{#if cursor}<button disabled={work.busy} onclick={more}
				>Load more</button
			>{/if}
	</section>
{/if}

<style>
	.account-type {
		padding: 0 16px 12px;
		margin: 0;
	}
	:global(.settings-workspace) .delete-account-row {
		color: var(--red-ink);
	}
	.deletion-confirmation {
		margin-top: 16px;
	}
	.standing-summary {
		display: flex;
		gap: 16px;
		align-items: flex-start;
		padding: 24px;
	}
	.standing-summary h2 {
		font-size: 16px;
	}
	.standing-summary p {
		font-size: 14px;
	}
	.standing-icon {
		width: 56px;
		height: 56px;
		border: 1.5px solid var(--ink);
		background: color-mix(in srgb, var(--moss) 20%, var(--surface));
		color: var(--ink);
		flex-shrink: 0;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.standing-warning {
		background: var(--butter);
		color: var(--ink);
	}
	.standing-error {
		background: var(--red-soft);
		color: var(--red-ink);
	}
	.moderation-history {
		margin-top: 24px;
	}
	.history-entry > .row {
		justify-content: flex-start;
		gap: 12px;
		align-items: flex-start;
	}
	.history-entry h4 {
		margin: 12px 0 6px;
		font-size: 14px;
	}
	.history-entry {
		padding: 16px;
	}

	time {
		font-size: 0.8rem;
		color: var(--ink-3);
	}
</style>
