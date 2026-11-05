<script lang="ts">
	import Snackbar from '../social/Snackbar.svelte';
	import SingleSelect from '#lib/SingleSelect.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount, tick, untrack } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	import {
		historyScrollMode,
		inboxTime,
		messageBackURL,
		startsTimeSection,
		timeSectionLabel
	} from './presentation';
	import type { CraftskyAPI, ChatMessage } from '#lib/api/client.ts';
	import { MessagesController, type MessagesState } from './controller';
	import MessagePreview from './MessagePreview.svelte';
	import ChatIdentity from './ChatIdentity.svelte';
	import { contextMenu } from '../social/context-menu';
	import {
		actorLabel,
		conversationURL,
		graphemeCount,
		messageText,
		messageLinks,
		reportReasons,
		safeAvatar,
		validateText,
		sharedCraftskyLink
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
	const settingsGate = $derived(
		viewState?.view === 'settings' ||
			['disabled', 'authorizationNeeded'].includes(
				viewState?.status?.state || ''
			)
	);
	const contentReady = $derived(
		!!viewState?.status?.enabled &&
			viewState.status.capabilities.canRead &&
			['ready', 'unavailable'].includes(viewState.status.state)
	);
	// Inbox navigation is known on the first render, before status/history load.
	const inboxRoute = $derived(!(path[0] === 'messages' ? path[1] : path[0]));
	const inboxActionsVisible = $derived(
		inboxRoute && (!viewState?.status || contentReady)
	);
	const requestCount = $derived(viewState?.counts.requests ?? 0);
	let search = $state('');
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	let reportReason = $state<string>(reportReasons[0][1]);
	let reportExplanation = $state('');
	let authorizing = $state(false);
	let preselected = $state(false);
	let authorizationError = $state('');
	let leaveDialog: HTMLDialogElement;
	let deleteDialog: HTMLDialogElement;
	let blockDialog: HTMLDialogElement;
	let ignoreRequest = $state(false);
	let clipboardNotice = $state('');
	let clipboardError = $state(false);
	let deleteTarget = $state<ChatMessage>();
	const sharedLink = $derived(
		sharedCraftskyLink(page.url.searchParams.get('link'))
	);
	let historyNode = $state<HTMLElement>();
	let historySnapshot:
		{ id: string; first?: string; last?: string } | undefined;
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
		preselected = false;
		clipboardNotice = '';
		deleteTarget = undefined;
		leaveDialog?.close();
		deleteDialog?.close();
		blockDialog?.close();
		reportExplanation = '';
		reportReason = reportReasons[0][1];
		authorizing = false;
		authorizationError = '';
		return () => {
			void next.sync(false);
			next.dispose();
		};
	});
	$effect.pre(() => {
		const state = viewState,
			node = historyNode,
			owner = controller;
		if (!node || !state?.id || state.view !== 'conversation') return;
		const next = {
			id: session.id + ':' + state.id,
			first: state.messages[0]?.id,
			last: state.draft.pending?.operationId || state.messages.at(-1)?.id
		};
		const mode = historyScrollMode(
			historySnapshot,
			next,
			node.scrollHeight - node.scrollTop - node.clientHeight < 80
		);
		const height = node.scrollHeight,
			top = node.scrollTop;
		historySnapshot = next;
		void tick().then(() => {
			if (owner !== controller || node !== historyNode) return;
			if (mode === 'end') node.scrollTop = node.scrollHeight;
			else if (mode === 'preserve')
				node.scrollTop = top + node.scrollHeight - height;
		});
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
		// A new share/recipient query is a new intent even when the route path is unchanged.
		page.url.search;
		if (current) {
			untrack(() => {
				preselected = false;
				deleteTarget = undefined;
				clipboardNotice = '';
				leaveDialog?.close();
				deleteDialog?.close();
				blockDialog?.close();
			});
			void current.navigate(currentPath, true);
		}
	});
	$effect(() => {
		const recipient = page.url.searchParams.get('did');
		if (
			recipient &&
			contentReady &&
			viewState?.status?.capabilities.canSend &&
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
	function anchorHistory(node: HTMLElement) {
		let pinned = true;
		let userScrolled = false;
		const interact = () => {
			userScrolled = true;
		};
		const scroll = () => {
			if (!userScrolled) return;
			pinned = node.scrollHeight - node.scrollTop - node.clientHeight < 80;
		};
		const resize = new ResizeObserver(() => {
			if (pinned) node.scrollTop = node.scrollHeight;
		});
		const observe = () => {
			resize.disconnect();
			resize.observe(node);
			for (const child of node.children) resize.observe(child);
		};
		const mutations = new MutationObserver(observe);
		mutations.observe(node, { childList: true });
		node.addEventListener('scroll', scroll, { passive: true });
		for (const type of ['wheel', 'touchstart', 'pointerdown', 'keydown'])
			node.addEventListener(type, interact, { passive: true });
		observe();
		return {
			destroy() {
				resize.disconnect();
				mutations.disconnect();
				node.removeEventListener('scroll', scroll);
				for (const type of ['wheel', 'touchstart', 'pointerdown', 'keydown'])
					node.removeEventListener(type, interact);
			}
		};
	}

	function visibleMessage(node: HTMLElement, id: string) {
		const owner = controller,
			conversation = viewState?.id;
		const observer = new IntersectionObserver(
			(entries) => {
				if (
					entries.some(
						(entry) => entry.isIntersecting && entry.intersectionRatio >= 0.7
					) &&
					owner === controller &&
					owner?.state.id === conversation &&
					document.visibilityState === 'visible' &&
					document.hasFocus()
				)
					void owner?.presented(id);
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
		const owner = controller,
			location = page.url.href;
		const id = await owner?.start(
			did,
			page.url.searchParams.get('link') ?? undefined
		);
		if (owner === controller && page.url.href === location && id)
			await goto(conversationURL(id));
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
				body: JSON.stringify({ kind: 'chat', returnTo: '/messages' })
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
		const owner = controller;
		if (deleteTarget && owner) {
			await owner.deleteMessage(deleteTarget);
			if (owner === controller && !owner.state.error) {
				deleteDialog.close();
				deleteTarget = undefined;
			}
		}
	}
	function askLeave(ignore = false) {
		ignoreRequest = ignore;
		leaveDialog.showModal();
	}
	async function leave() {
		const owner = controller;
		await owner?.conversationAction('leave', () => void goto('/messages'));
		if (owner === controller && !owner?.state.error) leaveDialog.close();
	}
	async function blockPeer() {
		const owner = controller;
		await owner?.blockPeer();
		if (owner === controller && owner?.state.peerBlocked) blockDialog.close();
	}
	async function copyMessage(text: string) {
		const owner = controller;
		try {
			await navigator.clipboard.writeText(text);
			if (owner === controller) {
				clipboardError = false;
				clipboardNotice = 'Message copied.';
			}
		} catch {
			if (owner === controller) {
				clipboardError = true;
				clipboardNotice =
					'Could not copy this message. Select the text to copy it.';
			}
		}
	}
</script>

<section
	class="messages-workspace"
	class:conversation-view={!settingsGate && viewState?.view === 'conversation'}
	aria-label="Direct messages"
	aria-busy={viewState?.loading || viewState?.busy}
>
	<header class="message-header workspace-heading">
		<div class="heading-group">
			{#if !inboxRoute}<a
					class="back"
					href={messageBackURL(viewState?.view || 'inbox', viewState?.id)}
					aria-label="Back to messages"><Icon name="back" /></a
				>{/if}
			{#if contentReady && viewState?.view === 'conversation' && viewState.conversation}<a
					class="header-peer-avatar avatar"
					href={`/profiles/${encodeURIComponent(viewState.conversation.peer.did)}`}
					aria-label="View profile"
					>{#if safeAvatar(viewState.conversation.peer.avatar)}<img
							src={safeAvatar(viewState.conversation.peer.avatar)}
							alt=""
							referrerpolicy="no-referrer"
						/>{:else}{actorLabel(viewState.conversation.peer).slice(
							0,
							1
						)}{/if}</a
				>{/if}
			<div class="heading-copy">
				{#if contentReady && viewState?.view === 'conversation' && viewState.conversation}<h1
						class="peer-heading"
						aria-label={actorLabel(viewState.conversation.peer)}
					>
						<ChatIdentity {api} peer={viewState.conversation.peer} />
					</h1>{:else}<h1>
						{settingsGate
							? 'Direct messages'
							: !contentReady
								? 'Messages'
								: viewState?.view === 'new'
									? 'New message'
									: viewState?.view === 'report'
										? viewState.reportMessage
											? 'Report message'
											: 'Report conversation'
										: viewState?.conversation
											? actorLabel(viewState.conversation.peer)
											: viewState?.bucket === 'request'
												? 'Requests'
												: 'Messages'}
					</h1>{/if}
			</div>
		</div>
		{#if inboxActionsVisible || (contentReady && !settingsGate)}<nav
				aria-label="Messaging actions"
			>
				{#if inboxActionsVisible}<a
						class="icon-link"
						href="/messages/requests"
						aria-label="Requests"
						><Icon name="mail" />{#if requestCount}<span class="badge"
								>{requestCount >= 100 ? '99+' : requestCount}</span
							>{/if}</a
					><a
						class="icon-link"
						href="/settings/messaging"
						aria-label="Direct message settings"><Icon name="settings" /></a
					>{:else if viewState?.view === 'conversation' && viewState.conversation && viewState.id}<details
						class="conversation-actions"
						use:contextMenu
					>
						<summary class="icon-link" aria-label="Conversation actions"
							><Icon name="more" /></summary
						>
						<div class="menu-items">
							<button
								disabled={viewState.busy}
								onclick={() => controller?.conversationAction('mute')}
								><Icon name="muted" />{viewState.conversation.muted
									? 'Unmute conversation'
									: 'Mute conversation'}</button
							>
							{#if !viewState.peerBlocked}<button
									class="destructive"
									disabled={viewState.busy}
									onclick={() => blockDialog.showModal()}
									><Icon name="shield" />Block account</button
								>{/if}
							{#if viewState.status?.capabilities.canReport}<a
									href={conversationURL(viewState.id) + '/report'}
									><Icon name="flag" />Report conversation</a
								>{/if}
							<button
								class="destructive"
								disabled={viewState.busy}
								onclick={() => askLeave()}
								><Icon name="signOut" />Leave conversation</button
							>
						</div>
					</details>{/if}
			</nav>{/if}
	</header>
	{#if viewState?.error}<Snackbar
			message={viewState.error}
			warning
			onDismiss={() => {
				if (viewState) viewState.error = '';
			}}
			><button
				type="button"
				disabled={viewState.busy}
				onclick={() => controller?.refresh()}>Try again</button
			></Snackbar
		>{/if}
	{#if clipboardNotice}<Snackbar
			message={clipboardNotice}
			warning={clipboardError}
			onDismiss={() => (clipboardNotice = '')}
		/>{/if}
	{#if viewState?.notice}<Snackbar
			message={viewState.notice}
			onDismiss={() => {
				if (viewState) viewState.notice = '';
			}}
		/>{/if}
	{#if authorizationError}<Snackbar
			message={authorizationError}
			warning
			onDismiss={() => (authorizationError = '')}
		/>{/if}
	{#if !viewState || viewState.loading}<div
			class="loading"
			class:settings-loading={viewState?.view === 'settings'}
			role="status"
		>
			<span class="visually-hidden"
				>{viewState?.view === 'settings'
					? 'Loading message settings…'
					: 'Loading messages…'}</span
			>
			{#if viewState?.view === 'settings'}<span
					class="loading-spinner"
					aria-hidden="true"
				></span>
			{:else if viewState?.view === 'conversation'}<div
					class="history-skeleton"
					aria-hidden="true"
				>
					{#each Array(12) as _}<i></i>{/each}
				</div>
			{:else}<div class="inbox-skeleton" aria-hidden="true">
					{#each Array(12) as _}<div><i></i><span></span></div>{/each}
				</div>{/if}
		</div>
	{:else if settingsGate}
		<div class="settings-body">
			{#if !viewState.status?.enabled}<p>
					This is your shared Bluesky inbox, showing conversations with CraftSky
					members, including those started in other apps. Read, accept, mute and
					leave actions also affect those apps.
				</p>
				<p>
					Messages are stored by Bluesky, outside public PDS records. They are
					not end-to-end encrypted.
				</p>
				<p>
					Message reports go to Bluesky only. CraftSky moderators do not receive
					these reports.
				</p>{/if}
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
					{#if viewState.policy && viewState.status.capabilities.canSetPolicy}<SingleSelect
							label="Allow incoming messages"
							value={viewState.policy.allowIncoming}
							required
							showRequired={false}
							options={[
								{ id: 'everyone', label: 'Everyone' },
								{ id: 'following', label: 'People I follow' },
								{ id: 'nobody', label: 'Nobody' }
							]}
							disabled={viewState.busy}
							onchange={(value) =>
								controller?.setPolicy(
									value as 'everyone' | 'following' | 'nobody'
								)}
						/>{:else}<p>
							Reconnect with incoming-message settings permission to change this
							shared setting.
						</p>
						<button disabled={authorizing} onclick={authorize}
							>Review permissions</button
						>{/if}
				</div>
			{/if}
		</div>
	{:else if !viewState.status || !contentReady}
		<div class="empty">
			<span class="empty-icon"><Icon name="warning" size={48} /></span>
			<h2>Messages</h2>
			<p>
				{viewState.status?.state === 'restricted'
					? 'Messaging is restricted by Bluesky. Other CraftSky features are still available.'
					: 'Messaging is temporarily unavailable. Try again.'}
			</p>
			<button
				class="button"
				disabled={viewState.busy}
				onclick={() => controller?.refresh()}>Refresh</button
			>
		</div>
	{:else if viewState.view === 'inbox'}
		{#if viewState.bucket === 'request'}<p class="disclosure">
				Accept a shared message request before sending a reply. Reading it does
				not accept it.
			</p>{/if}
		{#if !viewState.conversations.length && !viewState.error}<div class="empty">
				<span class="empty-icon"
					><Icon
						name={viewState.bucket === 'request' ? 'mail' : 'comment'}
						size={48}
					/></span
				>
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
								><ChatIdentity {api} peer={conversation.peer} />
								{#if conversation.lastMessage}<time
										datetime={conversation.lastMessage.sentAt}
										title={date(conversation.lastMessage.sentAt)}
										>{inboxTime(conversation.lastMessage.sentAt)}</time
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
				href="/messages/new"><Icon name="edit" />New message</a
			>{/if}
	{:else if viewState.view === 'new'}
		{#if page.url.searchParams.has('link')}<p
				class="disclosure"
				class:error={!sharedLink}
			>
				{sharedLink
					? 'Choose someone to send this link to. It will be added to your draft for review.'
					: 'Open the original CraftSky content to share this link.'}
			</p>{/if}
		<form
			class="recipient-search"
			onsubmit={(event) => {
				event.preventDefault();
				controller?.search(search);
			}}
		>
			<label
				><span class="visually-hidden">Search CraftSky profiles</span><input
					type="search"
					value={search}
					oninput={(event) => searchRecipients(event.currentTarget.value)}
					placeholder="Search by name or handle"
					autocomplete="off"
				/></label
			><button class="search-submit" type="submit" aria-label="Search"
				><Icon name="search" /></button
			>
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
						{#if sharedLink}<button
								class="conversation-row recipient"
								disabled={viewState.busy}
								onclick={() => start(conversation.peer.did)}
								><span class="avatar"
									>{#if safeAvatar(conversation.peer.avatar)}<img
											src={safeAvatar(conversation.peer.avatar)}
											alt=""
											referrerpolicy="no-referrer"
										/>{:else}{actorLabel(conversation.peer).slice(
											0,
											1
										)}{/if}</span
								><strong>{actorLabel(conversation.peer)}</strong></button
							>{:else}<a
								class="conversation-row"
								href={conversationURL(conversation.id)}
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
							>{/if}
					</li>{/each}
			</ul>
			<p class="disclosure">Search for someone to send a message to.</p>{/if}
	{:else if viewState.conversation && viewState.id}
		{#if viewState.view === 'report'}<form
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
				<SingleSelect
					label="Reason"
					value={reportReason}
					required
					showRequired={false}
					options={reportReasons.map(([label, id]) => ({ id, label }))}
					disabled={viewState.busy || !!viewState.report}
					onchange={(value) => (reportReason = value)}
				/><label
					>Explanation (optional)<textarea
						name="reportExplanation"
						aria-invalid={!!validateText(reportExplanation, true)}
						aria-describedby={validateText(reportExplanation, true)
							? 'report-explanation-error'
							: undefined}
						rows="5"
						bind:value={reportExplanation}
						readonly={!!viewState.report}
						disabled={viewState.busy}></textarea><small
						>{graphemeCount(reportExplanation)} / 2,000</small
					></label
				>{#if validateText(reportExplanation, true)}<p
						id="report-explanation-error"
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
			<div
				class="message-history"
				bind:this={historyNode}
				use:anchorHistory
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
				{#each viewState.messages as message, index (session.id + ':' + viewState.id + ':' + message.id)}{#if startsTimeSection(message.sentAt, viewState.messages[index - 1]?.sentAt)}<p
							class="time-section"
						>
							<time datetime={message.sentAt}
								>{timeSectionLabel(message.sentAt)}</time
							>
						</p>{/if}
					<article
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
							{#if message.kind === 'text'}<MessagePreview
									{api}
									text={message.text || ''}
									accepted={viewState.conversation.status === 'accepted'}
								/>{/if}
						</div>
						<div class="message-meta">
							<time class="visually-hidden" datetime={message.sentAt}
								>{date(message.sentAt)}</time
							>{#if message.kind === 'text'}<details use:contextMenu>
									<summary aria-label="Message actions">•••</summary>
									<div class="menu-items">
										{#if message.senderDid === session.did}<button
												disabled={viewState.busy}
												onclick={() => openDelete(message)}
												>Delete for me</button
											>{:else if viewState.status?.capabilities.canReport}<a
												href={conversationURL(viewState.id) +
													'/report/' +
													encodeURIComponent(message.id)}>Report message</a
											>{/if}<button
											onclick={() => copyMessage(message.text || '')}
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
			{#if viewState.conversation.status === 'request'}<div
					class="request-panel"
				>
					<p class="visually-hidden" id="request-disclosure">
						Accept this shared message request before sending a reply. Reading
						it does not accept it.
					</p>
					<div class="action-row">
						<button disabled={viewState.busy} onclick={() => askLeave(true)}
							>Ignore</button
						><button
							class="button"
							disabled={viewState.busy}
							aria-describedby="request-disclosure"
							onclick={() => controller?.conversationAction('accept')}
							>Accept request</button
						>
					</div>
				</div>{/if}

			{#if !viewState.draftSaved}<div class="feedback error" role="alert">
					<p>
						Could not save this draft on your device. Your text is kept here.
						Try saving again before leaving.
					</p>
					<button onclick={() => controller?.saveDraft()}>Save draft</button>
				</div>{/if}
			{#if !viewState.peerBlocked && viewState.conversation.canSend && viewState.status.capabilities.canSend}<form
					class="message-composer"
					onsubmit={(event) => {
						event.preventDefault();
						controller?.send();
					}}
				>
					<label class="visually-hidden" for="chat-compose">Message</label
					><textarea
						id="chat-compose"
						aria-invalid={!!viewState.draft.text &&
							!!validateText(viewState.draft.text)}
						aria-describedby={viewState.draft.text &&
						validateText(viewState.draft.text)
							? 'chat-validation'
							: undefined}
						value={viewState.draft.text}
						rows="1"
						placeholder="Message"
						autocapitalize="sentences"
						enterkeyhint="enter"
						oninput={(event) => controller?.edit(event.currentTarget.value)}
						onkeydown={(event) => {
							if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
								event.preventDefault();
								controller?.send();
							}
						}}></textarea>
					<div class="composer-actions">
						<small
							class:visually-hidden={graphemeCount(viewState.draft.text) < 900}
							class:invalid={!!validateText(viewState.draft.text) &&
								!!viewState.draft.text}
							>{graphemeCount(viewState.draft.text)} / 1,000</small
						><button
							class="button send-button"
							aria-label={viewState.busy ? 'Sending…' : 'Send message'}
							disabled={viewState.busy ||
								!!viewState.draft.pending ||
								!!validateText(viewState.draft.text)}
							><Icon name="send" filled={true} /></button
						>
					</div>
					{#if viewState.draft.text && validateText(viewState.draft.text)}<p
							class="validation-message"
							id="chat-validation"
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
			<a href="/messages">Return to messages</a>
		</div>{/if}
</section>

<dialog
	bind:this={leaveDialog}
	class="chat-dialog"
	aria-labelledby="leave-chat-title"
	oncancel={(event) => {
		if (viewState?.busy) event.preventDefault();
	}}
>
	<h2 id="leave-chat-title">
		{ignoreRequest ? 'Ignore message request?' : 'Leave conversation?'}
	</h2>
	<p>
		This removes the conversation from your shared inbox, including in other
		apps. Messages already received by the other person remain available to
		them.
	</p>
	<div class="action-row">
		<button onclick={() => leaveDialog.close()}>Cancel</button><button
			class="danger"
			disabled={viewState?.busy}
			onclick={leave}
			>{ignoreRequest ? 'Ignore request' : 'Leave conversation'}</button
		>
	</div>
</dialog>
<dialog
	bind:this={deleteDialog}
	class="chat-dialog"
	aria-labelledby="delete-message-title"
	oncancel={(event) => {
		if (viewState?.busy) event.preventDefault();
	}}
>
	<h2 id="delete-message-title">Delete message for me?</h2>
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

<dialog
	bind:this={blockDialog}
	class="chat-dialog"
	aria-labelledby="block-peer-title"
	oncancel={(event) => {
		if (viewState?.busy) event.preventDefault();
	}}
>
	<h2 id="block-peer-title">Block this account?</h2>
	<p>
		Blocking is public on the AT Protocol. You will no longer see or interact
		with each other's content.
	</p>
	<div class="action-row">
		<button disabled={viewState?.busy} onclick={() => blockDialog.close()}
			>Cancel</button
		><button class="danger" disabled={viewState?.busy} onclick={blockPeer}
			>{viewState?.busy ? 'Blocking…' : 'Block account'}</button
		>
	</div>
</dialog>

<style>
	.messages-workspace {
		position: relative;
		min-height: 100dvh;
		max-width: 800px;
		margin: auto;
		background: var(--paper);
	}
	.messages-workspace.conversation-view {
		height: 100dvh;
		min-height: 420px;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}
	.message-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		padding: 8px 16px;
		min-height: 56px;
		background: var(--paper);
		flex: none;
		z-index: 3;
	}
	.message-header h1 {
		font-size: 20px;
		font-family: inherit;
		font-weight: 700;
		line-height: 1.35;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.heading-group,
	.message-header nav,
	.action-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}
	.heading-group {
		flex: 1;
		min-width: 0;
	}
	.back {
		display: grid;
		place-items: center;
		flex: none;
		width: 40px;
		height: 40px;
		margin-left: -8px;
		border-radius: 50%;
		text-decoration: none;
		color: var(--ink);
	}
	.icon-link {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 4px;
		min-width: 40px;
		height: 40px;
		color: var(--ink);
		border: 0;
		border-radius: 50%;
		background: none;
		text-decoration: none;
		white-space: nowrap;
		font: inherit;
		padding: 8px;
		cursor: pointer;
		position: relative;
	}
	.icon-link:hover,
	.back:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 20px;
		height: 20px;
		padding: 0 5px;
		border-radius: 16px;
		background: #ef3b32;
		color: #fff;
		font-size: 11px;
		font-weight: 700;
		flex: none;
	}
	.icon-link .badge {
		position: absolute;
		top: 0;
		right: -2px;
		border: 2px solid var(--paper);
		height: 21px;
	}
	.feedback {
		margin: 12px 16px;
		padding: 12px 14px;
		background: var(--primary-soft);
		border-radius: 10px;
		display: flex;
		align-items: center;
		gap: 12px;
		justify-content: space-between;
		flex: none;
		font-size: 14px;
		line-height: 1.5;
	}
	.feedback.error {
		background: var(--red-soft);
		color: var(--red-ink);
	}
	.empty {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 12px;
		text-align: center;
		padding: 60px 24px;
		min-height: 55vh;
	}
	.loading {
		padding: 16px 12px;
		min-height: calc(100dvh - 56px);
		flex: 1;
		min-width: 0;
		overflow: hidden;
		display: flex;
		align-items: flex-start;
	}
	.settings-loading {
		align-items: center;
		justify-content: center;
	}
	.loading-spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--primary-soft);
		border-top-color: var(--primary);
		border-radius: 50%;
		animation: spinner 800ms linear infinite;
	}
	@keyframes spinner {
		to {
			transform: rotate(360deg);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.loading-spinner {
			animation: none;
		}
	}
	.conversation-actions summary {
		list-style: none;
		cursor: pointer;
	}
	.conversation-actions summary::-webkit-details-marker {
		display: none;
	}
	.chat-message:has(details[open]) {
		z-index: 50;
	}
	.chat-message:has(details[open]) .message-meta {
		opacity: 1;
	}
	.inbox-skeleton {
		width: 100%;
		display: grid;
		gap: 24px;
	}
	.inbox-skeleton > div {
		display: flex;
		align-items: center;
		gap: 12px;
	}
	.inbox-skeleton i {
		width: 44px;
		height: 44px;
		flex: none;
		border-radius: 50%;
		background: var(--paper-2);
	}
	.inbox-skeleton span {
		width: 65%;
		height: 32px;
		border-radius: 6px;
		background: var(--paper-2);
	}
	.history-skeleton {
		width: 100%;
		display: grid;
		gap: 24px;
	}
	.history-skeleton i {
		display: block;
		height: 48px;
		width: 60%;
		border-radius: 12px;
		background: var(--paper-2);
	}
	.history-skeleton i:nth-child(even) {
		justify-self: end;
		width: 45%;
		background: var(--primary-soft);
	}

	.empty-icon {
		color: var(--primary);
		display: grid;
		place-items: center;
	}
	.empty h2 {
		font-size: 22px;
		font-weight: 600;
	}
	.empty p {
		color: var(--ink-3);
		max-width: 360px;
		line-height: 1.6;
	}
	.empty > a {
		color: var(--primary);
		padding: 8px;
	}
	.settings-body {
		display: flex;
		flex-direction: column;
		gap: 24px;
		max-width: 688px;
		padding: 24px;
		margin: auto;
	}
	.settings-body p {
		color: var(--ink-3);
		font-size: 14px;
		line-height: 1.6;
	}
	.switch-row {
		display: flex;
		flex-direction: row;
		align-items: center;
		justify-content: space-between;
		gap: 24px;
		padding: 8px 0;
	}
	.switch-row span {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.switch-row strong {
		font-weight: 500;
		font-size: 16px;
	}
	.switch-row input {
		appearance: none;
		flex: none;
		width: 50px;
		height: 28px;
		border: 1.5px solid var(--ink);
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
		border-color: var(--ink);
	}
	.switch-row input:checked:before {
		background: var(--on-primary);
		transform: translateX(21px);
	}
	.switch-row input:disabled {
		opacity: 0.5;
	}
	.settings-body small,
	.setting-section p {
		color: var(--ink-3);
		font-size: 13px;
		line-height: 1.5;
	}
	.setting-section {
		display: grid;
		gap: 12px;
	}
	.setting-section h2 {
		font-size: 16px;
		font-weight: 600;
	}
	.settings-body label:not(.switch-row) {
		display: grid;
		gap: 8px;
		font-size: 14px;
	}
	.settings-body textarea,
	.recipient-search input,
	.message-composer textarea {
		font: inherit;
		color: var(--ink);
		background: var(--paper);
		border: 1.5px solid var(--ink);
		border-radius: 10px;
		padding: 13px 12px;
		width: 100%;
		resize: vertical;
	}
	.permission-panel,
	.request-panel {
		padding: 16px;
		background: var(--primary-soft);
		border-radius: 12px;
		display: grid;
		gap: 16px;
		font-size: 14px;
		line-height: 1.6;
	}
	.request-panel {
		margin: 12px;
		flex: none;
	}
	.conversation-list {
		list-style: none;
		padding: 0 0 96px;
		margin: 0;
	}
	.conversation-row {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 16px;
		min-height: 74px;
		text-decoration: none;
		color: var(--ink);
		width: 100%;
		text-align: left;
	}
	.conversation-row:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.avatar {
		font-family: 'DM Serif Display', serif;
		width: 40px;
		height: 40px;
		border: 2px solid var(--primary);
		border-radius: 50%;
		flex: none;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--lilac);
		font-weight: 600;
		font-size: 19px;
		overflow: hidden;
	}
	.avatar img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.avatar.header-peer-avatar {
		width: 40px;
		height: 40px;
		font-size: 16px;
		text-decoration: none;
		color: var(--ink);
	}
	.conversation-copy {
		flex: 1;
		min-width: 0;
	}
	.identity {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: 8px;
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
		color: var(--ink-2);
		margin-top: 2px;
		font-size: 14px;
		line-height: 1.4;
	}
	.muted-label {
		font-size: 12px;
	}
	.load-more {
		display: block;
		margin: 12px auto;
		padding: 10px 16px;
		background: var(--paper);
		color: var(--primary);
		border: 0;
		border-radius: 20px;
		cursor: pointer;
		font: inherit;
		font-size: 14px;
	}
	.compose-message {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		position: fixed;
		left: calc(var(--app-content-left) + var(--app-content-width) - 16px);
		transform: translateX(-100%);
		bottom: calc(24px + env(safe-area-inset-bottom));
		z-index: 8;
		white-space: nowrap;
		text-decoration: none;
		min-height: 56px;
		padding: 0 24px;
		border-radius: 999px;
	}
	.compose-message:hover {
		transform: translate(calc(-100% - 1px), -1px);
	}
	.compose-message:active {
		transform: translate(calc(-100% + 3px), 3px);
	}
	@media (max-width: 900px) {
		.compose-message {
			bottom: calc(96px + env(safe-area-inset-bottom));
		}
	}

	.disclosure {
		padding: 16px;
		font-size: 13px;
		color: var(--ink-3);
		line-height: 1.6;
	}
	.recipient-search {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 16px;
		position: relative;
	}
	.recipient-search label {
		display: grid;
		flex: 1;
	}
	.recipient-search input {
		padding-right: 48px;
	}
	.search-submit {
		position: absolute;
		right: 22px;
		width: 40px;
		height: 40px;
		background: none;
		color: var(--ink-3);
		border: 0;
		display: grid;
		place-items: center;
		cursor: pointer;
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
		font-size: 12px;
		margin-top: 4px;
	}
	.recipient-action {
		margin-left: auto;
		color: var(--primary);
		font-size: 12px;
	}
	.list-heading {
		font-size: 16px;
		font-weight: 600;
		padding: 8px 16px;
	}
	.danger {
		color: var(--red-ink);
	}
	.message-history {
		padding: 16px 12px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		min-height: 0;
		flex: 1;
		overflow: auto;
		overscroll-behavior: contain;
		overflow-anchor: none;
	}
	.time-section {
		text-align: center;
		color: var(--ink-3);
		font-size: 12px;
		font-weight: 400;
		padding: 0 12px 12px;
		flex: none;
	}
	.history-empty {
		color: var(--ink-3);
		text-align: center;
		padding: 30px;
		font-size: 14px;
	}
	.chat-message {
		position: relative;
		align-self: flex-start;
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 4px;
		max-width: 85%;
		flex: none;
		min-width: 60px;
	}
	.chat-message.own {
		align-self: flex-end;
		align-items: flex-end;
	}
	.message-bubble {
		padding: 12px;
		background: var(--surface);
		border-radius: 16px;
		max-width: 100%;
		font-size: 16px;
		line-height: 1.5;
	}
	.message-bubble p {
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
	.message-bubble a {
		color: inherit;
		text-decoration: none;
	}
	.own .message-bubble {
		background: var(--primary-soft);
		color: var(--ink);
	}
	.deleted .message-bubble {
		font-style: italic;
		color: var(--ink-3);
	}
	.message-meta {
		opacity: 0;
		position: absolute;
		top: 8px;
		right: -26px;
		display: flex;
		align-items: center;
		font-size: 11px;
		color: var(--ink-3);
		min-height: 0;
	}
	.chat-message:hover .message-meta,
	.chat-message:focus-within .message-meta {
		opacity: 1;
	}
	@media (hover: none) {
		.message-meta {
			opacity: 1;
		}
	}
	.own .message-meta {
		left: -26px;
		right: auto;
	}
	.message-meta details {
		position: relative;
	}
	.message-meta summary {
		cursor: pointer;
		list-style: none;
		padding: 2px 8px;
		border-radius: 12px;
	}
	.message-meta summary::-webkit-details-marker {
		display: none;
	}
	.message-meta summary:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.message-composer {
		position: relative;
		padding: 12px 12px calc(12px + env(safe-area-inset-bottom));
		display: grid;
		grid-template-columns: minmax(0, 1fr) 48px;
		align-items: start;
		column-gap: 8px;
		row-gap: 6px;
		background: var(--paper);
		flex: none;
	}
	.message-composer textarea {
		background: var(--surface);
		border-color: var(--ink-3);
		border-radius: 14px;
		min-height: 48px;
		max-height: 180px;
		resize: vertical;
		grid-column: 1;
		grid-row: 1;
		padding: 12px;
		font-size: 16px;
	}
	.composer-actions {
		display: contents;
	}
	.composer-actions small {
		grid-column: 1;
		grid-row: 2;
		font-size: 11px;
		color: var(--ink-3);
		text-align: right;
	}
	.composer-actions .invalid,
	.validation-message {
		color: var(--red-ink);
	}

	.composer-actions .send-button:disabled {
		background: var(--ink-4);
		color: var(--surface);
		border-color: var(--ink-3);
		opacity: 1;
		box-shadow: 3px 3px 0 color-mix(in srgb, var(--ink) 25%, transparent);
	}
	.composer-actions .send-button {
		grid-column: 2;
		grid-row: 1;
		min-width: 48px;
		width: 48px;
		height: 48px;
		padding: 12px;
		display: grid;
		place-items: center;
		border: 1.5px solid var(--ink);
		border-radius: 50%;
		background: var(--primary);
		color: var(--on-primary);
		box-shadow: 3px 3px 0 var(--ink);
	}
	.pending-message {
		opacity: 0.85;
	}
	.pending-viewState {
		font-size: 12px;
		color: var(--ink-3);
	}
	.pending-message .action-row {
		flex-wrap: wrap;
	}
	.action-row button,
	.feedback button {
		font: inherit;
		font-size: 14px;
		padding: 10px 12px;
		border: 0;
		border-radius: 8px;
		background: transparent;
		color: var(--primary);
		font-weight: 600;
		cursor: pointer;
	}
	.action-row .danger {
		color: var(--red-ink);
	}
	.report-form textarea {
		min-height: 150px;
	}
	.chat-dialog {
		max-width: min(420px, calc(100vw - 32px));
		padding: 24px;
		border: 1.5px solid var(--ink);
		border-radius: 20px;
		background: var(--paper);
		color: var(--ink);
		box-shadow: 4px 4px 0 var(--ink);
	}
	.chat-dialog::backdrop {
		background: #0006;
	}
	.chat-dialog h2 {
		font-size: 22px;
		font-family: inherit;
	}
	.chat-dialog p {
		margin: 16px 0 24px;
		font-size: 14px;
		line-height: 1.6;
		color: var(--ink-3);
	}
	.chat-dialog .action-row {
		justify-content: flex-end;
		flex-wrap: wrap;
	}
	.chat-dialog .action-row button {
		color: var(--primary);
	}
	.chat-dialog .action-row button.danger {
		color: var(--red-ink);
	}
	button:disabled {
		cursor: default;
		opacity: 0.5;
	}
	button:focus-visible,
	input:focus-visible,
	textarea:focus-visible {
		outline: 3px solid var(--primary);
		outline-offset: 3px;
	}
	.validation-message {
		grid-column: 1 / -1;
		font-size: 12px;
	}
	.message-header h1.peer-heading {
		font-weight: 400;
	}
	.heading-copy {
		flex: 1;
		min-width: 0;
	}
	.message-history::before {
		content: '';
		flex: 1 0 0;
	}
	.request-panel {
		padding: 12px 12px calc(16px + env(safe-area-inset-bottom));
		margin: 0;
		background: transparent;
		border-radius: 0;
	}
	.request-panel .action-row {
		gap: 12px;
	}
	.request-panel button {
		flex: 1;
		min-height: 48px;
		border-radius: 24px;
		border: 1.5px solid var(--ink);
		box-shadow: 3px 3px 0 var(--ink);
		background: var(--surface);
		color: var(--ink);
		font-size: 16px;
		font-weight: 700;
	}
	.request-panel button.button {
		background: var(--primary);
		color: var(--on-primary);
	}

	@media (max-width: 600px) {
		.message-header {
			gap: 4px;
		}
		.message-header nav {
			gap: 0;
		}
		.settings-body {
			padding: 24px 16px;
		}
		.recipient-action {
			display: none;
		}
		.chat-message {
			max-width: 88%;
		}
		.message-header h1 {
			font-size: 20px;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.switch-row input:before {
			transition: none;
		}
	}
</style>
