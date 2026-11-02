<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import type { CraftskyAPI, ChatMessage } from '#lib/api/client.ts';
	import { MessagesController, type MessagesState } from './controller';
	import MessagePreview from './MessagePreview.svelte';
	import {
		actorLabel,
		conversationURL,
		graphemeCount,
		messageText,
		messageLinks,
		reportReasons,
		safeAvatar,
		validateText
	} from './model';
	let {
		api,
		path,
		session
	}: {
		api: CraftskyAPI;
		path: string[];
		session: { id: string; did: string; handle: string };
	} = $props();
	let controller = $state<MessagesController>();
	let viewState = $state<MessagesState>();
	let search = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	let reportReason = $state<string>(reportReasons[0][1]);
	let reportExplanation = $state('');
	let authorizing = $state(false);
	let preselected = $state(false);
	let authorizationError = $state('');
	let leaveDialog: HTMLDialogElement;
	let deleteDialog: HTMLDialogElement;
	let deleteTarget = $state<ChatMessage>();
	const date = (value: string) =>
		new Intl.DateTimeFormat(undefined, {
			month: 'short',
			day: 'numeric',
			hour: 'numeric',
			minute: '2-digit'
		}).format(new Date(value));
	$effect(() => {
		const ownedApi = api,
			ownedSession = session.id,
			ownedDid = session.did;
		const next = new MessagesController(
			ownedApi,
			`${ownedSession}:${ownedDid}`,
			{
				getItem: (key) => window.localStorage.getItem(key),
				setItem: (key, value) => window.localStorage.setItem(key, value),
				removeItem: (key) => window.localStorage.removeItem(key)
			},
			(value) => (viewState = value)
		);
		controller = next;
		viewState = next.state;
		search = '';
		reportExplanation = '';
		reportReason = reportReasons[0][1];
		authorizing = false;
		authorizationError = '';
		return () => {
			void next.sync(false);
			next.dispose();
		};
	});
	$effect(() => {
		const report = viewState?.report;
		if (report) {
			reportReason = report.reasonType;
			reportExplanation = report.reason;
		}
	});
	$effect(() => {
		const current = controller;
		const currentPath = [...path];
		if (current) void current.navigate(currentPath);
	});
	$effect(() => {
		const recipient = page.url.searchParams.get('did');
		if (
			recipient &&
			/^did:[a-z]+:[^/\s]+$/.test(recipient) &&
			controller &&
			viewState?.view === 'new' &&
			!viewState.loading &&
			!viewState.busy &&
			!preselected
		) {
			preselected = true;
			void start(recipient);
		}
	});
	onMount(() => {
		const sync = () =>
			void controller?.sync(document.visibilityState === 'visible');
		const timer = setInterval(sync, 10_000);
		document.addEventListener('visibilitychange', sync);
		window.addEventListener('focus', sync);
		return () => {
			clearInterval(timer);
			clearTimeout(searchTimer);
			document.removeEventListener('visibilitychange', sync);
			window.removeEventListener('focus', sync);
		};
	});
	function visibleMessage(node: HTMLElement, id: string) {
		const observer = new IntersectionObserver(
			(entries) => {
				if (
					entries.some(
						(entry) => entry.isIntersecting && entry.intersectionRatio >= 0.7
					) &&
					document.visibilityState === 'visible' &&
					document.hasFocus()
				)
					void controller?.presented(id);
			},
			{ threshold: 0.7 }
		);
		observer.observe(node);
		return {
			destroy() {
				observer.disconnect();
			}
		};
	}
	function searchRecipients(value: string) {
		search = value;
		clearTimeout(searchTimer);
		searchTimer = setTimeout(() => void controller?.search(value), 250);
	}
	async function start(did: string) {
		const id = await controller?.start(did);
		if (id) await goto(conversationURL(id));
	}
	async function authorize() {
		if (authorizing) return;
		authorizing = true;
		authorizationError = '';
		const account = session.id;
		try {
			const response = await fetch('/api/auth/start', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'X-Craftsky-Account-Id': account
				},
				body: JSON.stringify({ kind: 'chat', returnTo: '/app/messages' })
			});
			if (!response.ok) throw new Error('Authorization unavailable');
			const result = (await response.json()) as { authUrl: string };
			const target = new URL(result.authUrl);
			if (
				target.protocol !== 'https:' &&
				!(target.protocol === 'http:' && target.hostname === '127.0.0.1')
			)
				throw new Error('Invalid authorization target');
			if (account === session.id) window.location.assign(target.href);
		} catch {
			if (account === session.id) {
				authorizationError =
					'Messaging permissions could not be opened. Try again.';
				authorizing = false;
			}
		}
	}
	function openDelete(message: ChatMessage) {
		deleteTarget = message;
		deleteDialog.showModal();
	}
	async function deleteMessage() {
		if (deleteTarget) {
			await controller?.deleteMessage(deleteTarget);
			deleteDialog.close();
			deleteTarget = undefined;
		}
	}
	async function leave() {
		await controller?.conversationAction(
			'leave',
			() => void goto('/app/messages')
		);
		leaveDialog.close();
	}
</script>

<section
	class="messages-workspace"
	aria-label="Direct messages"
	aria-busy={viewState?.loading || viewState?.busy}
>
	<header class="message-header">
		<div class="heading-group">
			{#if viewState?.view !== 'inbox'}<a
					class="back"
					href={viewState?.id ? conversationURL(viewState.id) : '/app/messages'}
					aria-label="Back to messages">←</a
				>{/if}
			<h1>
				{viewState?.view === 'settings'
					? 'Direct messages'
					: viewState?.view === 'new'
						? 'New message'
						: viewState?.view === 'report'
							? viewState.reportMessage
								? 'Report message'
								: 'Report conversation'
							: viewState?.view === 'details'
								? 'Conversation details'
								: viewState?.conversation
									? actorLabel(viewState.conversation.peer)
									: viewState?.bucket === 'request'
										? 'Requests'
										: 'Messages'}
			</h1>
		</div>
		<nav aria-label="Messaging actions">
			{#if viewState?.view === 'inbox'}<a
					class="icon-link"
					href="/app/messages/requests"
					>Requests{#if viewState.counts.requests}<span class="badge"
							>{viewState.counts.requests >= 100
								? '99+'
								: viewState.counts.requests}</span
						>{/if}</a
				><a
					class="icon-link"
					href="/app/messages/settings"
					aria-label="Direct message settings">⚙</a
				>{:else if viewState?.view === 'conversation' && viewState.id}<a
					class="icon-link"
					href={conversationURL(viewState.id) + '/details'}>Details</a
				>{/if}<button
				class="icon-link"
				disabled={viewState?.busy || viewState?.loading}
				onclick={() => controller?.refresh()}
				aria-label="Refresh messages">↻</button
			>
		</nav>
	</header>
	{#if viewState?.error}<div class="feedback error" role="alert">
			<p>{viewState.error}</p>
			<button disabled={viewState.busy} onclick={() => controller?.refresh()}
				>Try again</button
			>
		</div>{/if}
	{#if viewState?.notice}<p class="feedback" role="status">
			{viewState.notice}
		</p>{/if}
	{#if authorizationError}<p class="feedback error" role="alert">
			{authorizationError}
		</p>{/if}
	{#if !viewState || viewState.loading}<div class="loading" role="status">
			<span class="yarn">✳</span>
			<p>Loading messages…</p>
		</div>
	{:else if viewState.view === 'settings' || !viewState.status?.enabled}
		<div class="settings-body">
			<p>
				This is your shared Bluesky inbox, showing conversations with CraftSky
				members, including those started in other apps. Read, accept, mute and
				leave actions also affect those apps.
			</p>
			<p>
				Messages are stored by Bluesky, outside public PDS records. They are not
				end-to-end encrypted.
			</p>
			<label class="switch-row"
				><span
					><strong>Enable messaging</strong><small
						>Use messaging in CraftSky. Turning it off stops CraftSky message
						access and push notifications.</small
					></span
				><input
					type="checkbox"
					checked={viewState.status?.enabled || false}
					disabled={viewState.busy}
					onchange={(event) =>
						controller?.setEnabled(event.currentTarget.checked)}
				/></label
			>
			{#if viewState.status?.state === 'authorizationNeeded'}<div
					class="permission-panel"
				>
					<p>
						Approve messaging permissions to use direct messages in CraftSky.
					</p>
					<button
						class="button"
						disabled={authorizing || viewState.busy}
						onclick={authorize}
						>{authorizing
							? 'Opening permissions…'
							: 'Review permissions'}</button
					>
				</div>{/if}
			{#if viewState.status?.state === 'restricted'}<p class="feedback">
					Messaging is restricted by Bluesky. Other CraftSky features are still
					available.
				</p>{/if}
			{#if viewState.status?.enabled}
				<label class="switch-row"
					><span
						><strong>Push notifications</strong><small
							>Receive push notifications about new messages on your registered
							devices.</small
						></span
					><input
						type="checkbox"
						checked={viewState.push || false}
						disabled={viewState.busy || viewState.push === undefined}
						onchange={(event) =>
							controller?.setPush(event.currentTarget.checked)}
					/></label
				>
				<div class="setting-section">
					<h2>Who can message me</h2>
					<p>
						This setting is shared with Bluesky and other apps. Changes may take
						time to appear there and do not remove existing conversations.
					</p>
					{#if viewState.policy && viewState.status.capabilities.canSetPolicy}<label
							>Allow incoming messages<select
								value={viewState.policy.allowIncoming}
								disabled={viewState.busy}
								onchange={(event) =>
									controller?.setPolicy(
										event.currentTarget.value as
											'everyone' | 'following' | 'nobody'
									)}
								><option value="everyone">Everyone</option><option
									value="following">People I follow</option
								><option value="nobody">Nobody</option></select
							></label
						>{:else}<p>
							Reconnect with incoming-message settings permission to change this
							shared setting.
						</p>
						<button disabled={authorizing} onclick={authorize}
							>Review permissions</button
						>{/if}
				</div>
			{/if}
		</div>
	{:else if !viewState.status.capabilities.canRead}
		<div class="empty">
			<span class="empty-icon">✉</span>
			<h2>
				{viewState.status.state === 'restricted'
					? 'Messaging is restricted'
					: 'Connect your inbox'}
			</h2>
			<p>
				{viewState.status.state === 'restricted'
					? 'Messaging is restricted by Bluesky. Other CraftSky features are still available.'
					: 'Approve messaging permissions to use direct messages in CraftSky.'}
			</p>
			{#if viewState.status.state !== 'restricted'}<button
					class="button"
					disabled={authorizing}
					onclick={authorize}
					>{authorizing ? 'Opening permissions…' : 'Review permissions'}</button
				>{/if}<a href="/app/messages/settings">Direct message settings</a>
		</div>
	{:else if viewState.view === 'inbox'}
		{#if viewState.bucket === 'request'}<p class="disclosure">
				Accept a shared message request before sending a reply. Reading it does
				not accept it.
			</p>{/if}
		{#if !viewState.conversations.length && !viewState.error}<div class="empty">
				<span class="empty-icon">✉</span>
				<h2>{viewState.bucket === 'request' ? 'Requests' : 'Inbox'}</h2>
				<p>
					{viewState.bucket === 'request'
						? 'No message requests.'
						: 'No messages yet.'}
				</p>
			</div>{/if}
		<ul class="conversation-list">
			{#each viewState.conversations as conversation (conversation.id)}<li>
					<a class="conversation-row" href={conversationURL(conversation.id)}
						><span class="avatar"
							>{#if safeAvatar(conversation.peer.avatar)}<img
									src={safeAvatar(conversation.peer.avatar)}
									alt=""
									loading="lazy"
									referrerpolicy="no-referrer"
								/>{:else}{actorLabel(conversation.peer).slice(0, 1)}{/if}</span
						><span class="conversation-copy"
							><span class="identity"
								><strong>{actorLabel(conversation.peer)}</strong
								>{#if conversation.lastMessage}<time
										datetime={conversation.lastMessage.sentAt}
										>{date(conversation.lastMessage.sentAt)}</time
									>{/if}</span
							><span class="preview"
								>{conversation.lastMessage
									? messageText(conversation.lastMessage)
									: 'Start a conversation'}{#if conversation.muted}<span
										class="muted-label"
									>
										· Muted</span
									>{/if}</span
							></span
						>{#if conversation.unreadCount}<span
								class="badge"
								aria-label={`${conversation.unreadCount} unread messages`}
								>{conversation.unreadCount >= 100
									? '99+'
									: conversation.unreadCount}</span
							>{/if}</a
					>
				</li>{/each}
		</ul>
		{#if viewState.cursor}<button
				class="load-more"
				disabled={viewState.loadingMore}
				onclick={() => controller?.more()}
				>{viewState.loadingMore ? 'Loading…' : 'Load more'}</button
			>{/if}
		{#if viewState.bucket === 'accepted' && viewState.status.capabilities.canSend}<a
				class="compose-message button"
				href="/app/messages/new">＋ New message</a
			>{/if}
	{:else if viewState.view === 'new'}
		<form
			class="recipient-search"
			onsubmit={(event) => {
				event.preventDefault();
				controller?.search(search);
			}}
		>
			<label
				>Search CraftSky profiles<input
					type="search"
					value={search}
					oninput={(event) => searchRecipients(event.currentTarget.value)}
					placeholder="Search by name or handle"
					autocomplete="off"
				/></label
			><button class="button" type="submit">Search</button>
		</form>
		{#if search.trim()}<ul
				class="conversation-list"
				aria-label="Matching profiles"
			>
				{#each viewState.recipients.filter((person) => person.did !== session.did) as person (person.did)}<li
					>
						<button
							class="conversation-row recipient"
							disabled={viewState.busy}
							onclick={() => start(person.did)}
							><span class="avatar"
								>{#if safeAvatar(person.avatar)}<img
										src={safeAvatar(person.avatar)}
										alt=""
										loading="lazy"
										referrerpolicy="no-referrer"
									/>{:else}{actorLabel(person).slice(0, 1)}{/if}</span
							><span
								><strong>{actorLabel(person)}</strong><small
									>@{person.handle}</small
								></span
							><span class="recipient-action">Message →</span></button
						>
					</li>{/each}
			</ul>
			{#if !viewState.recipients.length}<p class="disclosure">
					No profiles found.
				</p>{/if}{#if viewState.recipientCursor}<button
					class="load-more"
					onclick={() => controller?.search(search, viewState?.recipientCursor)}
					>Load more profiles</button
				>{/if}
		{:else}<h2 class="list-heading">Recent conversations</h2>
			<ul class="conversation-list">
				{#each viewState.conversations as conversation (conversation.id)}<li>
						<a class="conversation-row" href={conversationURL(conversation.id)}
							><span class="avatar"
								>{#if safeAvatar(conversation.peer.avatar)}<img
										src={safeAvatar(conversation.peer.avatar)}
										alt=""
										referrerpolicy="no-referrer"
									/>{:else}{actorLabel(conversation.peer).slice(
										0,
										1
									)}{/if}</span
							><strong>{actorLabel(conversation.peer)}</strong></a
						>
					</li>{/each}
			</ul>
			<p class="disclosure">Search for someone to send a message to.</p>{/if}
	{:else if viewState.conversation && viewState.id}
		{#if viewState.view === 'details'}<div class="settings-body">
				<a
					class="peer-link"
					href={`/app/profile/${encodeURIComponent(viewState.conversation.peer.did)}`}
					><span class="avatar large"
						>{#if safeAvatar(viewState.conversation.peer.avatar)}<img
								src={safeAvatar(viewState.conversation.peer.avatar)}
								alt=""
								referrerpolicy="no-referrer"
							/>{:else}{actorLabel(viewState.conversation.peer).slice(
								0,
								1
							)}{/if}</span
					><strong>{actorLabel(viewState.conversation.peer)}</strong><span
						>@{viewState.conversation.peer.handle || 'CraftSky member'}</span
					><span>View profile →</span></a
				><button
					class="setting-action"
					disabled={viewState.busy}
					onclick={() => controller?.conversationAction('mute')}
					>{viewState.conversation.muted
						? 'Unmute conversation'
						: 'Mute conversation'}</button
				><a
					class="setting-action"
					href={conversationURL(viewState.id) + '/report'}
					>Report conversation</a
				><button
					class="setting-action danger"
					disabled={viewState.busy}
					onclick={() => leaveDialog.showModal()}>Leave conversation</button
				>
				<p>
					These actions also affect your shared Bluesky inbox in other apps.
				</p>
				<p>
					Message reports go to Bluesky only. CraftSky moderators do not receive
					these reports.
				</p>
			</div>
		{:else if viewState.view === 'report'}<form
				class="settings-body report-form"
				onsubmit={(event) => {
					event.preventDefault();
					controller?.report(reportReason, reportExplanation);
				}}
			>
				<p>
					Bluesky will receive a reference to this {viewState.reportMessage
						? 'message'
						: 'conversation'}, your selected reason and any explanation.
					CraftSky does not send a copied transcript.
				</p>
				<p>
					Message reports go to Bluesky only. CraftSky moderators do not receive
					these reports.
				</p>
				<label
					>Reason<select
						bind:value={reportReason}
						disabled={viewState.busy || !!viewState.report}
						>{#each reportReasons as [label, value]}<option {value}
								>{label}</option
							>{/each}</select
					></label
				><label
					>Explanation (optional)<textarea
						rows="5"
						bind:value={reportExplanation}
						readonly={!!viewState.report}
						disabled={viewState.busy}></textarea><small
						>{graphemeCount(reportExplanation)} / 2,000</small
					></label
				>{#if validateText(reportExplanation, true)}<p
						class="error"
						role="alert"
					>
						{validateText(reportExplanation, true)}
					</p>{/if}
				{#if viewState.report?.state === 'uncertain'}<p role="status">
						We could not confirm whether Bluesky received this report. Your
						explanation has been kept; it will not be sent again automatically.
					</p>
					<div class="action-row">
						<button
							type="button"
							disabled={viewState.busy}
							onclick={() => controller?.checkReport()}>Check status</button
						><button type="submit" disabled={viewState.busy}
							>Retry same report</button
						>
					</div>{:else if viewState.report?.state === 'failed'}<p role="status">
						Bluesky did not accept this report. Your explanation has been kept.
					</p>
					<button type="button" onclick={() => controller?.editFailedReport()}
						>Edit report</button
					>{:else}<button
						class="button"
						disabled={viewState.busy || !!validateText(reportExplanation, true)}
						>{viewState.busy ? 'Submitting…' : 'Report to Bluesky'}</button
					>{/if}
			</form>
		{:else}
			{#if viewState.conversation.status === 'request'}<div
					class="request-panel"
				>
					<p>
						Accept this shared message request before sending a reply. Reading
						it does not accept it.
					</p>
					<div class="action-row">
						<button
							class="button"
							disabled={viewState.busy}
							onclick={() => controller?.conversationAction('accept')}
							>Accept request</button
						><button
							disabled={viewState.busy}
							onclick={() => leaveDialog.showModal()}>Ignore</button
						>
					</div>
				</div>{/if}
			<div
				class="message-history"
				role="log"
				aria-label={`Messages with ${actorLabel(viewState.conversation.peer)}`}
				aria-live="polite"
				aria-relevant="additions text"
			>
				{#if viewState.historyCursor}<button
						class="load-more"
						disabled={viewState.loadingMore}
						onclick={() => controller?.more()}
						>{viewState.loadingMore
							? 'Loading older messages…'
							: 'Load older messages'}</button
					>{/if}
				{#if !viewState.messages.length}<p class="history-empty">
						This is the start of your conversation.
					</p>{/if}
				{#each viewState.messages as message (message.id)}<article
						class:own={message.senderDid === session.did}
						class:deleted={message.kind !== 'text'}
						class="chat-message"
						use:visibleMessage={message.id}
					>
						<div class="message-bubble">
							<p>
								{#each messageLinks(messageText(message)) as part}{#if part.href}<a
											href={part.href}
											rel="noreferrer nofollow noopener"
											target="_blank">{part.text}</a
										>{:else}{part.text}{/if}{/each}
							</p>
						</div>
						{#if message.kind === 'text'}<MessagePreview
								{api}
								text={message.text || ''}
								accepted={viewState.conversation.status === 'accepted'}
							/>{/if}
						<div class="message-meta">
							<time datetime={message.sentAt}>{date(message.sentAt)}</time
							>{#if message.kind === 'text'}<details>
									<summary aria-label="Message actions">•••</summary>
									<div class="message-menu">
										{#if message.senderDid === session.did}<button
												disabled={viewState.busy}
												onclick={() => openDelete(message)}
												>Delete for me</button
											>{:else if viewState.status.capabilities.canReport}<a
												href={conversationURL(viewState.id) +
													'/report/' +
													encodeURIComponent(message.id)}>Report message</a
											>{/if}<button
											onclick={() =>
												navigator.clipboard.writeText(message.text || '')}
											>Copy text</button
										>
									</div>
								</details>{/if}
						</div>
					</article>{/each}
				{#if viewState.draft.pending}<article
						class="chat-message own pending-message"
					>
						<div class="message-bubble">
							<p>{viewState.draft.pending.text}</p>
						</div>
						<p class="pending-viewState">
							{viewState.draft.pending.state === 'pending'
								? 'Sending…'
								: viewState.draft.pending.state === 'failed'
									? 'This message was not sent. Your draft is kept.'
									: 'Delivery has not been confirmed. Your text is kept.'}
						</p>
						{#if viewState.draft.pending.state === 'uncertain'}<div
								class="action-row"
							>
								<button
									disabled={viewState.busy}
									onclick={() => controller?.checkSend()}>Check status</button
								><button
									disabled={viewState.busy}
									onclick={() => controller?.send(true)}
									>Retry same message</button
								>
							</div>{:else if viewState.draft.pending.state === 'failed'}<button
								disabled={viewState.busy}
								onclick={() => controller?.dismissFailed()}
								>Keep draft and edit</button
							>{/if}
					</article>{/if}
			</div>
			{#if !viewState.draftSaved}<div class="feedback error" role="alert">
					<p>
						Could not save this draft on your device. Your text is kept here.
						Try saving again before leaving.
					</p>
					<button onclick={() => controller?.saveDraft()}>Save draft</button>
				</div>{/if}
			{#if viewState.conversation.canSend && viewState.status.capabilities.canSend}<form
					class="message-composer"
					onsubmit={(event) => {
						event.preventDefault();
						controller?.send();
					}}
				>
					<label class="visually-hidden" for="chat-compose">Message</label
					><textarea
						id="chat-compose"
						value={viewState.draft.text}
						rows="2"
						placeholder="Write a message…"
						oninput={(event) => controller?.edit(event.currentTarget.value)}
						onkeydown={(event) => {
							if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
								event.preventDefault();
								controller?.send();
							}
						}}></textarea>
					<div class="composer-actions">
						<small
							class:invalid={!!validateText(viewState.draft.text) &&
								!!viewState.draft.text}
							>{graphemeCount(viewState.draft.text)} / 1,000</small
						>{#if viewState.draft.text && !viewState.draft.pending}<button
								type="button"
								disabled={viewState.busy}
								onclick={() => controller?.discardDraft()}>Discard draft</button
							>{/if}<button
							class="button send-button"
							disabled={viewState.busy ||
								!!viewState.draft.pending ||
								!!validateText(viewState.draft.text)}
							>{viewState.busy ? 'Sending…' : 'Send message'}
							<span aria-hidden="true">↑</span></button
						>
					</div>
					{#if viewState.draft.text && validateText(viewState.draft.text)}<p
							class="validation-message"
						>
							{validateText(viewState.draft.text)}
						</p>{/if}
				</form>{:else if viewState.conversation.status !== 'request'}<p
					class="disclosure"
				>
					You cannot send to this conversation right now.
				</p>{/if}
		{/if}
	{:else if !viewState.error}<div class="empty">
			<h2>Conversation unavailable</h2>
			<a href="/app/messages">Return to messages</a>
		</div>{/if}
</section>

<dialog bind:this={leaveDialog} class="chat-dialog">
	<h2>Leave conversation?</h2>
	<p>
		This removes the conversation from your shared inbox, including in other
		apps. Messages already received by the other person remain available to
		them.
	</p>
	<div class="action-row">
		<button onclick={() => leaveDialog.close()}>Cancel</button><button
			class="danger"
			disabled={viewState?.busy}
			onclick={leave}>Leave conversation</button
		>
	</div>
</dialog>
<dialog bind:this={deleteDialog} class="chat-dialog">
	<h2>Delete message for me?</h2>
	<p>
		This removes the message from your view. The other person may still have a
		copy.
	</p>
	<div class="action-row">
		<button onclick={() => deleteDialog.close()}>Cancel</button><button
			class="danger"
			disabled={viewState?.busy}
			onclick={deleteMessage}>Delete for me</button
		>
	</div>
</dialog>

<style>
	.messages-workspace {
		position: relative;
		min-height: 65vh;
		background: var(--surface);
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		overflow: clip;
		padding-bottom: 1px;
	}
	.message-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		padding: 20px 24px;
		border-bottom: 1px solid var(--hairline);
		background: var(--paper);
	}
	.message-header h1 {
		font-size: 24px;
	}
	.heading-group,
	.message-header nav,
	.action-row {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.heading-group {
		min-width: 0;
	}
	.message-header h1 {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.back {
		font-size: 26px;
		text-decoration: none;
		color: var(--ink);
	}
	.icon-link {
		display: inline-flex;
		gap: 6px;
		align-items: center;
		color: var(--ink);
		border: 0;
		background: none;
		text-decoration: none;
		white-space: nowrap;
		font: inherit;
		padding: 8px;
		cursor: pointer;
	}
	.badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 24px;
		height: 24px;
		padding: 0 7px;
		border-radius: 20px;
		background: var(--primary);
		color: var(--on-primary);
		font-size: 12px;
		font-weight: 600;
	}
	.feedback {
		margin: 16px 20px;
		padding: 14px 16px;
		background: var(--primary-soft);
		border-radius: 10px;
		display: flex;
		align-items: center;
		gap: 16px;
		justify-content: space-between;
	}
	.feedback.error {
		background: var(--red-soft);
		color: var(--red-ink);
	}
	.loading,
	.empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 16px;
		text-align: center;
		padding: 70px 24px;
	}
	.loading {
		min-height: 300px;
	}
	.yarn {
		font-size: 40px;
		color: var(--primary);
	}
	.empty-icon {
		font-size: 56px;
		color: var(--ink-3);
	}
	.empty p {
		color: var(--ink-2);
		max-width: 420px;
	}
	.settings-body {
		display: flex;
		flex-direction: column;
		gap: 20px;
		max-width: 690px;
		padding: 28px;
		margin: auto;
	}
	.settings-body p {
		color: var(--ink-2);
	}
	.switch-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 30px;
		padding: 20px 0;
		border-block: 1px solid var(--hairline);
	}
	.switch-row span {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.switch-row input {
		appearance: none;
		flex: none;
		width: 50px;
		height: 28px;
		border: 1.5px solid var(--ink-3);
		background: var(--paper-2);
		border-radius: 18px;
		position: relative;
		cursor: pointer;
	}
	.switch-row input:before {
		content: '';
		display: block;
		background: var(--ink-3);
		width: 18px;
		height: 18px;
		border-radius: 50%;
		position: absolute;
		top: 3.5px;
		left: 4px;
		transition: transform 0.15s;
	}
	.switch-row input:checked {
		background: var(--primary);
		border-color: var(--primary);
	}
	.switch-row input:checked:before {
		background: var(--on-primary);
		transform: translateX(21px);
	}
	.switch-row input:focus-visible {
		outline: 3px solid var(--primary);
		outline-offset: 4px;
	}
	.switch-row input:disabled {
		opacity: 0.5;
	}
	.settings-body small,
	.setting-section p {
		color: var(--ink-3);
	}
	.setting-section {
		display: grid;
		gap: 16px;
	}
	.settings-body label:not(.switch-row) {
		display: grid;
		gap: 8px;
	}
	.settings-body select,
	.settings-body textarea,
	.recipient-search input,
	.message-composer textarea {
		font: inherit;
		color: var(--ink);
		background: var(--surface);
		border: 1.5px solid var(--hairline);
		border-radius: 10px;
		padding: 12px;
		width: 100%;
		resize: vertical;
	}
	.permission-panel,
	.request-panel {
		padding: 20px;
		background: var(--primary-soft);
		border-radius: 10px;
		display: grid;
		gap: 16px;
	}
	.request-panel {
		margin: 20px;
	}
	.conversation-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.conversation-list li + li {
		border-top: 1px solid var(--hairline);
	}
	.conversation-row {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 20px 24px;
		text-decoration: none;
		color: var(--ink);
		width: 100%;
		text-align: left;
	}
	.conversation-row:hover {
		background: var(--paper);
	}
	.avatar {
		width: 52px;
		height: 52px;
		border: 1.5px solid var(--ink);
		border-radius: 50%;
		flex: none;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--lilac);
		font-weight: 600;
		font-size: 22px;
		overflow: hidden;
	}
	.avatar img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.avatar.large {
		width: 88px;
		height: 88px;
		font-size: 36px;
	}
	.conversation-copy {
		flex: 1;
		min-width: 0;
	}
	.identity {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 16px;
	}
	.identity time {
		font-size: 12px;
		color: var(--ink-3);
		white-space: nowrap;
	}
	.preview {
		display: block;
		white-space: nowrap;
		text-overflow: ellipsis;
		overflow: hidden;
		color: var(--ink-3);
		margin-top: 3px;
		font-size: 14px;
	}
	.muted-label {
		font-size: 12px;
	}
	.load-more {
		display: block;
		margin: 20px auto;
		padding: 10px 20px;
		background: var(--paper);
		color: var(--ink);
		border: 1px solid var(--hairline);
		border-radius: 24px;
		cursor: pointer;
		font: inherit;
	}
	.compose-message {
		display: table;
		margin: 24px 24px 24px auto;
		text-decoration: none;
		border: 1.5px solid var(--ink);
		border-radius: 12px;
		box-shadow: 3px 3px 0 var(--ink);
	}
	.disclosure {
		padding: 20px 24px;
		font-size: 14px;
		color: var(--ink-3);
	}
	.recipient-search {
		display: flex;
		align-items: end;
		gap: 12px;
		padding: 24px;
	}
	.recipient-search label {
		display: grid;
		gap: 8px;
		flex: 1;
	}
	.recipient {
		background: none;
		border: 0;
		font: inherit;
		cursor: pointer;
	}
	.recipient small {
		display: block;
		color: var(--ink-3);
	}
	.recipient-action {
		margin-left: auto;
		color: var(--primary);
		font-size: 14px;
	}
	.list-heading {
		font-size: 18px;
		padding: 16px 24px;
	}
	.peer-link {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		padding-bottom: 16px;
		text-decoration: none;
		color: var(--ink);
	}
	.peer-link > span:not(.avatar) {
		font-size: 14px;
		color: var(--ink-3);
	}
	.setting-action {
		border: 0;
		border-bottom: 1px solid var(--hairline);
		text-decoration: none;
		text-align: left;
		background: none;
		padding: 14px 0;
		font: inherit;
		color: var(--ink);
		cursor: pointer;
	}
	.danger {
		color: var(--red-ink);
	}
	.message-history {
		padding: 24px;
		display: flex;
		flex-direction: column;
		gap: 18px;
		min-height: 250px;
	}
	.history-empty {
		color: var(--ink-3);
		text-align: center;
		padding: 30px;
	}
	.chat-message {
		align-self: flex-start;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 4px;
		max-width: min(82%, 520px);
		min-width: 100px;
	}
	.chat-message.own {
		align-self: flex-end;
		align-items: flex-end;
	}
	.message-bubble {
		padding: 12px 16px;
		background: var(--paper-2);
		border: 1px solid var(--hairline);
		border-radius: 16px 16px 16px 4px;
		max-width: 100%;
	}
	.message-bubble p {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.message-bubble a {
		color: inherit;
		text-decoration: underline;
	}
	.own .message-bubble {
		background: var(--primary);
		color: var(--on-primary);
		border-color: var(--primary);
		border-radius: 16px 16px 4px 16px;
	}
	.deleted .message-bubble {
		font-style: italic;
		background: var(--paper);
		color: var(--ink-3);
	}
	.message-meta {
		display: flex;
		align-items: center;
		gap: 10px;
		font-size: 11px;
		color: var(--ink-3);
	}
	.message-meta details {
		position: relative;
	}
	.message-meta summary {
		cursor: pointer;
		list-style: none;
		padding: 2px 8px;
	}
	.message-menu {
		position: absolute;
		right: 0;
		top: 100%;
		display: grid;
		gap: 6px;
		min-width: 140px;
		background: var(--surface);
		border: 1px solid var(--ink);
		box-shadow: 3px 3px 0 var(--hairline);
		border-radius: 8px;
		padding: 8px;
		z-index: 2;
	}
	.message-menu button,
	.message-menu a {
		font: inherit;
		font-size: 14px;
		text-align: left;
		border: 0;
		background: none;
		padding: 8px;
		color: var(--ink);
		text-decoration: none;
		cursor: pointer;
	}
	.message-composer {
		border-top: 1px solid var(--hairline);
		padding: 20px 24px;
		display: grid;
		gap: 10px;
		background: var(--paper);
	}
	.message-composer textarea {
		min-height: 80px;
		max-height: 300px;
	}
	.composer-actions {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: 14px;
	}
	.composer-actions small {
		margin-right: auto;
		color: var(--ink-3);
	}
	.composer-actions .invalid,
	.validation-message {
		color: var(--red-ink);
	}
	.composer-actions button:not(.button) {
		font: inherit;
		background: none;
		border: 0;
		color: var(--ink-2);
		cursor: pointer;
	}
	.send-button {
		border-radius: 24px !important;
		white-space: nowrap;
	}
	.pending-message {
		opacity: 0.8;
	}
	.pending-viewState {
		font-size: 12px;
		color: var(--ink-3);
	}
	.pending-message .action-row {
		flex-wrap: wrap;
	}
	.action-row button {
		font: inherit;
		cursor: pointer;
	}
	.report-form textarea {
		min-height: 150px;
	}
	.chat-dialog {
		max-width: min(480px, calc(100vw - 40px));
		padding: 28px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		background: var(--surface);
		color: var(--ink);
		box-shadow: 5px 5px 0 var(--ink);
	}
	.chat-dialog::backdrop {
		background: #0006;
	}
	.chat-dialog p {
		margin: 16px 0 24px;
	}
	.chat-dialog .action-row {
		justify-content: flex-end;
		flex-wrap: wrap;
	}
	button:disabled {
		cursor: default;
		opacity: 0.55;
	}
	button:focus-visible,
	input:focus-visible,
	select:focus-visible,
	textarea:focus-visible {
		outline: 3px solid var(--primary);
		outline-offset: 3px;
	}
	.validation-message {
		font-size: 13px;
	}
	@media (max-width: 600px) {
		.message-header {
			padding: 16px;
			gap: 4px;
		}
		.message-header h1 {
			font-size: 21px;
		}
		.message-header nav {
			gap: 2px;
		}
		.icon-link {
			padding: 6px;
			font-size: 14px;
		}
		.settings-body {
			padding: 20px;
		}
		.conversation-row {
			padding: 18px 16px;
			gap: 12px;
		}
		.identity {
			align-items: start;
			gap: 8px;
		}
		.identity time {
			font-size: 10px;
		}
		.avatar {
			width: 46px;
			height: 46px;
		}
		.message-history {
			padding: 20px 16px;
		}
		.message-composer {
			padding: 16px;
		}
		.recipient-search {
			padding: 20px 16px;
			flex-wrap: wrap;
		}
		.chat-message {
			max-width: 88%;
		}
		.composer-actions {
			gap: 8px;
		}
		.composer-actions button {
			font-size: 13px;
		}
		.settings-body,
		.settings-body p {
			overflow-wrap: anywhere;
		}
		.compose-message {
			margin: 24px 16px 24px auto;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.switch-row input:before {
			transition: none;
		}
	}
</style>
