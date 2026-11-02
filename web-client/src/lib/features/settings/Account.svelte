<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type {
		CraftskyAPI,
		EligibilityStatus,
		ModerationHistoryItem,
		Profile
	} from '#lib/api/client.ts';
	import { dateTime, record, text, title, type Session } from './model.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
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
	const work = new Work();
	let profile = $state<Profile | null>(null),
		eligibility = $state<EligibilityStatus | null>(null);
	let standing = $state<Record<string, unknown>>({}),
		history = $state<ModerationHistoryItem[]>([]),
		cursor = $state<string | null>(null);
	let deleteOpen = $state(false),
		confirmation = $state(''),
		pending = $state<{ jobId: string; did: string } | null>(null);
	async function browser<T>(
		path: string,
		body?: unknown,
		method = body === undefined ? 'GET' : 'POST'
	): Promise<T> {
		if (!session.id)
			throw new Error('Reopen your account settings to continue.');
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
			throw new Error(
				text(record(value).message) ||
					'Could not complete account deletion. Please try again.'
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
				}
			);
		return work.load(
			async () => {
				const eligible = await api.account.eligibility();
				const [me, status] = await Promise.all([
					eligible.state === 'eligible'
						? api.profiles.me()
						: Promise.resolve(null),
					browser<{ pending: boolean; jobId?: string; did?: string }>('status')
				]);
				return [me, eligible, status] as const;
			},
			([me, eligible, status]) => {
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
			}
		);
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
			'Opening your account provider for fresh verification…'
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
			'Account deletion cancelled.'
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
			'Account deletion requested.'
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
			''
		);
	}
	function appeal(reference: string) {
		return `mailto:moderation@craftsky.social?subject=${encodeURIComponent(`CraftSky moderation appeal ${reference}`)}&body=${encodeURIComponent(`Please explain why you are appealing ${reference}.`)}`;
	}
	onMount(load);
	onDestroy(() => work.dispose());
</script>

<h1>
	{section === 'moderation'
		? 'Account standing'
		: section === 'about'
			? 'About CraftSky'
			: 'Account'}
</h1>
<Feedback {work} reload={load} />
{#if section === 'about'}
	<section class="panel">
		<h2>A place for your craft</h2>
		<p>
			CraftSky is a community for textile crafters, built on the AT Protocol.
			Your creative work and your connections belong to you.
		</p>
		<nav class="nav-list" aria-label="About and legal">
			<a href="https://craftsky.social/terms" target="_blank" rel="noreferrer"
				>Terms of service</a
			><a
				href="https://craftsky.social/privacy"
				target="_blank"
				rel="noreferrer">Privacy policy</a
			><a
				href="https://docs.google.com/document/d/1BXqycv4IvnsVGWAhK1FZ94XF6hrNiW4FVFFtfjGyT94/edit?usp=sharing"
				target="_blank"
				rel="noreferrer">Community guidelines</a
			><a
				href="https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/3mpr5izppvt2k?lang=en"
				target="_blank"
				rel="noreferrer">Feedback and support</a
			>
		</nav>
	</section>
{:else if !work.loading && section === 'account' && eligibility}
	<section class="panel">
		<h2>@{session.handle}</h2>
		<p class="muted">Account identifier</p>
		<code>{session.did}</code>
		{#if profile}<div class="row">
				<span>Account type</span><strong
					>{title(profile.accountType || 'regular')}</strong
				>
			</div>{/if}
		<div class="row">
			<span>CraftSky access</span><strong
				>{title(eligibility?.state || 'unknown')}</strong
			>
		</div>
		{#if eligibility?.appealGuidance}<p>{eligibility.appealGuidance}</p>{/if}<a
			href="/app/settings/moderation">View account standing</a
		>
	</section>
	<section class="panel stack">
		<h2>Account management</h2>
		<p>
			CraftSky uses your account provider to sign in. Manage your password,
			recovery information and account provider settings with that provider.
		</p>
		{#if eligibility.state === 'eligible'}<a
				class="action"
				href="/app/settings/business">Manage business profile</a
			>{/if}
	</section>
	<section class="panel">
		<h2>Delete CraftSky account</h2>
		<p>
			Permanently remove your CraftSky records and private CraftSky data. This
			does not delete your identity or your account with your account provider.
			This action cannot be undone.
		</p>
		<p class="notice">
			Deleting your CraftSky account does not cancel subscriptions or provider
			billing. Manage subscriptions with your store before deleting your
			account.
		</p>
		{#if !deleteOpen}<button
				class="danger"
				onclick={() => {
					deleteOpen = true;
				}}>Delete CraftSky account…</button
			>
		{:else if pending}<form class="stack" onsubmit={confirmDeletion}>
				<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
					<p>
						Your account provider has verified your identity. To confirm
						permanent deletion, type your complete account identifier exactly:
					</p>
					<code>{session.did}</code><label
						>Account identifier<input
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
						><button type="button" disabled={work.busy} onclick={cancelDeletion}
							>Cancel deletion</button
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
					<button class="danger" disabled={work.busy} onclick={startDeletion}
						>Verify identity to continue</button
					><button disabled={work.busy} onclick={cancelDeletion}>Cancel</button>
				</div>
			</div>{/if}
	</section>
{:else if !work.loading && section === 'moderation'}
	<section class="panel">
		<h2>
			{standing.suspended
				? 'Account suspended'
				: Number(standing.activeStrikeCount) > 0
					? 'Action required'
					: 'Account in good standing'}
		</h2>
		<p>
			{Number(standing.activeStrikeCount) || 0} active {Number(
				standing.activeStrikeCount
			) === 1
				? 'strike'
				: 'strikes'} · Suspension threshold: {Number(
				standing.strikeThreshold
			) || '—'}
		</p>
		{#if standing.severeSuspended}<p>
				Your account is suspended following a severe policy violation.
			</p>{:else if standing.thresholdSuspended}<p>
				Your account has reached the active strike threshold.
			</p>{/if}
	</section>
	<section class="panel">
		<h2>{reference ? 'Case history' : 'Moderation history'}</h2>
		{#each history as raw}{@const entry = record(raw)}{@const subject =
				moderationSubject(entry.safeSnapshot)}{@const ref = text(
				entry.caseReference
			)}
			<article class="history-entry">
				<div class="row">
					<h3>{title(text(entry.reason) || text(entry.eventType))}</h3>
					<time>{dateTime(entry.occurredAt)}</time>
				</div>
				<p>{text(entry.userSafeDetail)}</p>
				{#if subject.reference}<p class="muted">
						Affected content: <code>{subject.reference}</code>
					</p>{/if}
				<p class="muted">
					{ref} · Appeal: {title(text(entry.appealStatus) || 'none')}
				</p>
				{#if Array.isArray(entry.effects)}<ul>
						{#each entry.effects as rawEffect}{@const effect =
								record(rawEffect)}
							<li>
								{title(text(effect.type))} — {title(
									text(effect.action)
								)}{effect.dueAt ? ` (${dateTime(effect.dueAt)})` : ''}
							</li>{/each}
					</ul>{/if}
				<div class="actions">
					{#if subject.href}<a class="action" href={subject.href}
							>{subject.label}</a
						>{/if}
					{#if !reference}<a
							class="action"
							href={`/app/settings/moderation/${encodeURIComponent(ref)}`}
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
				No moderation history.
			</p>{/each}{#if cursor}<button disabled={work.busy} onclick={more}
				>Load more</button
			>{/if}
	</section>
{/if}

<style>
	.history-entry {
		padding: 1rem 0;
		border-bottom: 1px solid var(--hairline);
	}
	.history-entry:last-child {
		border: 0;
	}
	time {
		font-size: 0.8rem;
		color: var(--ink-3);
	}
</style>
