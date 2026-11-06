<script lang="ts">
	import Snackbar from '../social/Snackbar.svelte';
	import PublishingOverlay from './PublishingOverlay.svelte';
	import {
		Publications,
		publicationsContext
	} from '../social/publications.svelte';
	import { notify } from '../social/toasts.svelte';
	import {
		findField,
		showFieldError,
		focusField,
		presentFieldErrors
	} from '#lib/field-validation.ts';
	import { getContext, onDestroy, onMount, untrack } from 'svelte';
	import { page } from '$app/state';
	import { beforeNavigate } from '$app/navigation';
	import {
		APIError,
		MutationOutcomeUnknownError,
		type CraftskyAPI,
		type JsonObject,
		type Post,
		type PostEditChanges,
		type PostPayload,
		type StrongRef
	} from '#lib/api/client.ts';
	import {
		drafts,
		newDraft,
		type Draft,
		type DraftMedia,
		type PendingWrite
	} from '../drafts/store.ts';
	import {
		buildFacets,
		projectFacets,
		defaultProject,
		editChanges,
		freezeWrite,
		postPayload,
		record,
		str,
		textLength,
		validateDraft,
		selectedQuoteImage,
		responseLabel,
		scheduledLocalDate,
		scheduledUTCDate
	} from './model.ts';
	import {
		attachmentSelection,
		inspectVideo,
		prepareImage,
		uploadVideo
	} from './media.ts';
	import ProjectEditor from './ProjectEditor.svelte';
	import MediaEditor from './MediaEditor.svelte';
	import PhotoAttachments from './PhotoAttachments.svelte';
	import VideoAttachment from './VideoAttachment.svelte';
	import FacetTextInput from './FacetTextInput.svelte';
	import LanguagePicker from './LanguagePicker.svelte';
	import ComposerTarget from './ComposerTarget.svelte';
	import SchedulePicker from './SchedulePicker.svelte';
	import SponsoredPicker from './SponsoredPicker.svelte';
	import ComposerLinkPreview from './ComposerLinkPreview.svelte';
	import {
		ComposerLinkPreviews,
		previewNavigation,
		type PreviewState
	} from './link-preview';
	import Icon from '../shell/Icon.svelte';
	import VideoPlayer from '../social/VideoPlayer.svelte';
	import { ComposerRecovery } from './recovery';
	import { reportFailure } from '../diagnostics/client.ts';
	import './composer.css';
	let {
		api,
		path,
		session
	}: {
		api: CraftskyAPI;
		path: string[];
		session: { id: string; did: string; handle: string };
	} = $props();
	let draft = $state<Draft>(newDraft('did:pending'));
	let loading = $state(true);
	let initialized = $state(false);
	let busy = $state(false);
	let publishing = $state(false);
	let scheduling = $state(false);
	const publications = getContext<Publications | undefined>(
		publicationsContext
	);
	const origin = getContext<(() => string | undefined) | undefined>(
		'craftsky-composer-origin'
	)?.();
	let mediaTasks = $state<string[]>([]);
	let mediaEdits = $state<string[]>([]);
	let error = $state('');
	let notice = $state('');
	let progress = $state('');
	let stored = $state('');
	let mounted = $state(false);
	let dirty = $state(false);
	let recovery = $state<ComposerRecovery>();
	let closeDialog: HTMLDialogElement;
	let closing = $state(false);
	let existingDraft = $state(false);
	let closeDestination = $state('/feed');
	let closeTarget = $state('/feed');
	let approvedNavigation = false;
	let editRevision = 0;
	let context = $state<Post>();
	let scheduleAt = $state('');
	let showSchedule = $state(false);
	let scheduleFull = $state(false);
	let previews = $state<ComposerLinkPreviews>();
	let externalFrozen = $state(false);
	let previewState = $state<PreviewState>({
		index: 0,
		total: 0,
		loading: false,
		dismissed: false,
		suppressed: false
	});
	let videoDetailsRequest = $state('');
	let generation = 0;
	let destroyed = false;
	let controller = new AbortController();
	const tier =
		getContext<
			() => import('#lib/api/client.ts').SubscriptionAccess | undefined
		>('craftsky-access');
	const locked = $derived(
		busy || closing || mediaTasks.length > 0 || !!draft.pending
	);
	// Compare the live immutable Blob identities: snapshots clone Blob instances.
	const changed = $derived(recovery?.hasChanges(draft) ?? false);
	function mediaBusy(id: string, value: boolean) {
		if (value && !mediaTasks.includes(id)) mediaTasks = [...mediaTasks, id];
		else if (!value && mediaTasks.includes(id))
			mediaTasks = mediaTasks.filter((item) => item !== id);
	}
	function mediaEditing(id: string, value: boolean) {
		if (value && !mediaEdits.includes(id)) mediaEdits = [...mediaEdits, id];
		else if (!value && mediaEdits.includes(id))
			mediaEdits = mediaEdits.filter((item) => item !== id);
	}
	const canSchedule = $derived(
		(!tier ||
			(tier()?.givesAccess && tier()?.effectiveTier !== 'free') ||
			!!draft.scheduleId) &&
			!draft.edit &&
			!draft.reply &&
			!draft.quote &&
			!draft.video
	);
	const canChooseSchedule = $derived(
		!draft.edit && !draft.reply && !draft.quote && !draft.video
	);
	const title = $derived(
		draft.edit
			? 'Edit post'
			: draft.reply
				? responseLabel(draft)
				: draft.quote
					? 'Quote'
					: draft.scheduleId
						? 'Edit scheduled post'
						: draft.kind === 'project'
							? 'Project post'
							: 'New post'
	);
	const canSaveDraft = $derived(
		!draft.reply && !draft.quote && !draft.edit && !draft.scheduleId
	);
	const draftSize = $derived(textLength(draft.text));
	const fixedEmbed = $derived(
		!!draft.edit &&
			!!record(draft.edit.editable.embed).$type &&
			!['app.bsky.embed.video', 'app.bsky.embed.external'].includes(
				str(record(draft.edit.editable.embed).$type)
			)
	);
	const retainedVideo = $derived(
		!!draft.edit?.post.video && draft.embedChanges !== null && !draft.video
	);
	const retainedExternal = $derived(externalFrozen && !!draft.external);
	const previewSuppressed = $derived(
		draft.kind === 'project' ||
			fixedEmbed ||
			retainedVideo ||
			retainedExternal ||
			!!draft.quote ||
			!!draft.video ||
			draft.images.length > 0 ||
			(!!draft.edit && draft.text === str(draft.edit.editable.text))
	);
	const canAddVideo = $derived(
		!fixedEmbed &&
			!draft.images.length &&
			!draft.quote &&
			!draft.reply &&
			!retainedExternal &&
			!showSchedule &&
			!draft.scheduleId &&
			!draft.video
	);
	let initializedRoute = '';
	$effect(() => {
		const identity = session.did;
		const route = path.join('/');
		const query = page.url.search;
		const key = JSON.stringify([session.id, identity, route, query]);
		if (mounted && key !== initializedRoute) {
			initializedRoute = key;
			untrack(() => {
				void initialize(identity, route, new URLSearchParams(query));
			});
		}
	});
	$effect(() => {
		const text = draft.text,
			suppressed = previewSuppressed,
			dismissed = draft.externalDismissed === true;
		if (
			mounted &&
			initialized &&
			!loading &&
			!busy &&
			!closing &&
			!draft.pending
		)
			untrack(() => previews?.update(text, suppressed, dismissed));
	});

	$effect(() => {
		const lifecycle = recovery;
		if (!mounted || loading || busy || closing || !lifecycle?.active) return;
		const snapshot = $state.snapshot(draft as unknown) as Draft;
		JSON.stringify(snapshot);
		const revision = ++editRevision;
		if (!changed && !snapshot.pending) {
			dirty = false;
			return;
		}
		dirty = true;
		const seq = generation;
		const timer = setTimeout(() => {
			void lifecycle
				.checkpoint(snapshot)
				.then(() => {
					if (seq === generation && revision === editRevision && !destroyed) {
						stored = 'Draft saved on this device';
						dirty = false;
					}
				})
				.catch(() => {
					if (seq === generation && !destroyed)
						stored = 'Could not save draft on this device';
				});
		}, 1000);
		return () => clearTimeout(timer);
	});
	onMount(() => {
		mounted = true;
	});
	beforeNavigate((nav) => {
		if (approvedNavigation || !initialized || nav.willUnload) return;
		if (publishing) {
			nav.cancel();
			return;
		}
		if (draft.pending) return; // Exact durable recovery survives leaving a pending action.
		if (changed || mediaEdits.length) {
			nav.cancel();
			requestClose(nav.to?.url.href || closeDestination);
		}
	});
	onDestroy(() => {
		preserveRecovery();
		destroyed = true;
		generation++;
		controller.abort();
		previews?.dispose();
		closeDialog?.close();
	});
	function preserveRecovery() {
		const lifecycle = recovery;
		if (
			!approvedNavigation &&
			initialized &&
			lifecycle?.active &&
			lifecycle.hasChanges(draft)
		) {
			void lifecycle
				.checkpoint($state.snapshot(draft as unknown) as Draft)
				.catch(() => {});
		}
		lifecycle?.abandon();
	}
	const returnTo =
		getContext<(destination: string) => Promise<void>>('craftsky-return');
	async function leaveEditor(destination: string) {
		approvedNavigation = true;
		try {
			await returnTo(destination);
		} finally {
			approvedNavigation = false;
		}
	}
	function requestClose(destination = closeDestination) {
		if (closing) return;
		if (draft.pending) {
			void leaveEditor(destination).catch((e) => {
				error = message(e);
			});
			return;
		}
		closeTarget = destination;
		if (!changed && !mediaEdits.length && !busy && !mediaTasks.length) {
			void leaveEditor(destination).catch((e) => {
				error = message(e);
			});
			return;
		}
		closeDialog.showModal();
	}
	function keepEditing() {
		if (!closing) closeDialog.close();
	}
	async function discardDraft() {
		const lifecycle = recovery,
			seq = generation;
		if (!lifecycle?.active || locked || mediaEdits.length) return;
		previews?.pause();
		closing = true;
		error = '';
		try {
			await lifecycle.discard($state.snapshot(draft as unknown) as Draft);
			if (seq !== generation || destroyed || recovery !== lifecycle) return;
			dirty = false;
			closeDialog.close();
			await leaveEditor(closeTarget);
		} catch (e) {
			if (seq === generation && !destroyed && !presentFieldErrors(e))
				error = message(e);
		} finally {
			if (seq === generation && !destroyed) closing = false;
		}
	}
	async function initialize(
		owner: string,
		route: string,
		query: URLSearchParams
	) {
		preserveRecovery();
		previews?.dispose();
		previews = undefined;
		previewState = {
			index: 0,
			total: 0,
			loading: false,
			dismissed: false,
			suppressed: false
		};
		recovery = undefined;
		closeDialog?.close();
		closing = false;
		const seq = ++generation;
		const ownedApi = api;
		controller.abort();
		controller = new AbortController();
		loading = true;
		initialized = false;
		error = '';
		notice = '';
		context = undefined;
		stored = '';
		busy = false;
		dirty = false;
		scheduleFull = false;
		try {
			let nextContext: Post | undefined;
			let next = newDraft(
				owner,
				route.includes('project') ? 'project' : 'standard'
			);
			const saved = query.get('draft');
			const scheduled = query.get('schedule');
			const mode = route.includes('edit')
				? 'edit'
				: route.includes('reply')
					? 'reply'
					: route.includes('quote')
						? 'quote'
						: query.has('editDid')
							? 'edit'
							: query.has('replyDid')
								? 'reply'
								: query.has('quoteDid')
									? 'quote'
									: '';
			const did = query.get(`${mode}Did`) || query.get('did');
			const rkey = query.get(`${mode}Rkey`) || query.get('rkey');
			if (saved) {
				const value = await drafts.get(owner, saved);
				if (!value)
					throw new Error('This draft was not found for your account.');
				next = value;
			} else if (scheduled) {
				const item = await ownedApi.schedules.get(scheduled);
				const payload = item.payload;
				next = {
					...next,
					id: item.operationId,
					kind: payload.kind === 'project' ? 'project' : 'standard',
					text: str(payload.text),
					langs: Array.isArray(payload.langs)
						? (payload.langs as string[])
						: ['en'],
					sponsored: payload.sponsored === true,
					project: payload.project ? record(payload.project) : undefined,
					scheduleId: item.id,
					scheduledAt: item.scheduledAt,
					scheduleLater: true
				};
				const media = Array.isArray(payload.media)
					? payload.media.map(record)
					: [];
				next.images = await Promise.all(
					media.map(async (value) => {
						const id = str(value.id);
						const bytes = await ownedApi.schedules.media(id, controller.signal);
						return {
							id,
							scheduledId: id,
							name: 'Scheduled image',
							blob: new Blob([bytes], { type: 'image/jpeg' }),
							mime: 'image/jpeg',
							alt: str(value.alt),
							width: typeof value.width === 'number' ? value.width : undefined,
							height:
								typeof value.height === 'number' ? value.height : undefined
						};
					})
				);
				if (payload.external) {
					const external = record(payload.external);
					next.external = {
						uri: str(external.uri),
						title: str(external.title),
						description: str(external.description)
					};
					if (external.thumbMediaId) {
						const id = str(external.thumbMediaId);
						const bytes = await ownedApi.schedules.media(id, controller.signal);
						next.externalImage = {
							id,
							scheduledId: id,
							name: 'Link preview',
							blob: new Blob([bytes], { type: 'image/jpeg' }),
							mime: 'image/jpeg',
							alt: ''
						};
					}
				}
			} else if (mode === 'edit') {
				if (!did || !rkey || did !== owner)
					throw new Error('Choose one of your posts to edit.');
				const seed = await ownedApi.posts.editSeed(did, rkey);
				const value = seed.editable;
				next = {
					...next,
					text: str(value.text),
					sponsored: value.sponsored === true,
					reply: seed.post.reply ?? undefined,
					quote: seed.post.quote ?? undefined,
					langs: Array.isArray(value.langs)
						? (value.langs as string[])
						: ['en'],
					kind: value.project ? 'project' : 'standard',
					project: value.project ? record(value.project) : undefined,
					edit: seed
				};
				next.images = (Array.isArray(value.images) ? value.images : []).map(
					(raw, index) => {
						const image = record(raw);
						const ratio = record(image.aspectRatio);
						return {
							id: crypto.randomUUID(),
							name: 'Published image',
							mime: str(record(image.image).mimeType),
							alt: str(image.alt),
							existingIndex:
								typeof image.existingIndex === 'number'
									? image.existingIndex
									: index,
							remoteUrl:
								seed.post.images?.[index]?.fullsize ||
								seed.post.images?.[index]?.thumb,
							width: typeof ratio.width === 'number' ? ratio.width : undefined,
							height:
								typeof ratio.height === 'number' ? ratio.height : undefined
						};
					}
				);
				const external = record(record(value.embed).external);
				if (external.uri)
					next.external = {
						uri: str(external.uri),
						title: str(external.title),
						description: str(external.description),
						...(external.thumb
							? {
									thumb:
										external.thumb as unknown as import('#lib/api/client.ts').BlobReference
								}
							: {})
					};
			} else if (mode === 'reply' || mode === 'quote') {
				if (!did || !rkey)
					throw new Error('Choose a post to reply to or quote.');
				const [target, preferences] = await Promise.all([
					ownedApi.posts.get(did, rkey),
					ownedApi.languages.get().catch(() => undefined)
				]);
				if (preferences) next.langs = [preferences.primaryLanguage];
				if (!('author' in target)) throw new Error('This post is unavailable.');
				if (mode === 'quote' && target.reply)
					throw new Error(
						'Replies cannot be quoted. Open the original post to quote it.'
					);
				nextContext = target;
				const ref: StrongRef = { uri: target.uri, cid: target.cid };
				if (mode === 'reply')
					next.reply = { root: target.reply?.root || ref, parent: ref };
				else {
					next.quote = ref;
					next.quoteImageCid = selectedQuoteImage(
						target,
						query.get('imageCid')
					);
				}
			} else {
				const [preferences, profile] = await Promise.all([
					ownedApi.languages.get().catch(() => undefined),
					next.kind === 'project'
						? ownedApi.profiles.me().catch(() => undefined)
						: Promise.resolve(undefined)
				]);
				if (preferences) next.langs = [preferences.primaryLanguage];
				if (next.kind === 'project') {
					next.project = defaultProject(profile?.crafts);
				}
			}
			if (seq !== generation || destroyed) return;
			if (saved && (next.reply || next.quote)) {
				const ref = next.reply?.parent || next.quote;
				const parts = ref?.uri.split('/');
				if (parts?.length === 5 && parts[3] === 'social.craftsky.feed.post') {
					const target = await ownedApi.posts.get(parts[2], parts[4]);
					if ('author' in target) nextContext = target;
					if (next.quote && nextContext)
						next.quoteImageCid = selectedQuoteImage(
							nextContext,
							next.quoteImageCid
						);
				}
			}
			if (seq !== generation || destroyed) return;
			if (!next.reply && !next.quote && !next.edit) {
				void ownedApi.schedules
					.list()
					.then((capacity) => {
						if (seq === generation && !destroyed)
							scheduleFull = !next.scheduleId && capacity.items.length >= 3;
					})
					.catch(() => {});
			}
			if (next.kind === 'project' && !next.project)
				next.project = defaultProject();
			context = nextContext;
			if (nextContext && query.get('imageCid') && !next.quoteImageCid)
				notice =
					'The selected image is no longer available. Your quote will use the original post.';
			existingDraft = !!saved;
			closeDestination =
				origin ||
				(saved
					? '/drafts'
					: scheduled
						? '/scheduled'
						: did && rkey
							? `/posts/${encodeURIComponent(did)}/${encodeURIComponent(rkey)}`
							: '/feed');
			closeTarget = closeDestination;
			recovery = new ComposerRecovery(owner, next.id, next, !!saved);
			draft = next;
			externalFrozen =
				!!next.edit &&
				!!next.external &&
				next.external.uri ===
					str(record(record(next.edit.editable.embed).external).uri);
			previews = new ComposerLinkPreviews(
				async (candidate, signal) => {
					const value = await ownedApi.media.linkPreview(
						candidate.identity,
						signal
					);
					if (
						typeof value.title !== 'string' ||
						typeof value.description !== 'string' ||
						!value.title.trim()
					)
						throw new Error('Invalid link preview response');
					const external = {
						uri: previewNavigation(
							{ ...candidate, fragment: '' },
							value.url || value.uri || candidate.identity
						),
						title: value.title,
						description: value.description
					};
					const image = await previewThumbnail(record(value.thumbnail), signal);
					return { external, image };
				},
				(state) => {
					if (seq !== generation || destroyed) return;
					previewState = state;
					if (loading || busy || closing || draft.pending || retainedExternal)
						return;
					draft.external = state.selected?.external;
					draft.externalImage = state.selected?.image;
					draft.externalSource = state.selected?.sourceIdentity;
					draft.externalResolvedURI = state.selected?.resolvedURI;
				}
			);
			if (next.external && !retainedExternal)
				previews.seed(next.text, {
					external: next.external,
					image: next.externalImage,
					sourceIdentity: next.externalSource,
					resolvedURI: next.externalResolvedURI
				});
			initialized = true;
			showSchedule =
				!!next.scheduleId || next.scheduleLater === true || !!next.scheduledAt;
			scheduleAt = next.scheduledAt ? localDate(next.scheduledAt) : '';
		} catch (e) {
			if (seq === generation && !destroyed && !presentFieldErrors(e))
				error = message(e);
		} finally {
			if (seq === generation && !destroyed) loading = false;
		}
	}
	function localDate(value: string) {
		return scheduledLocalDate(value);
	}
	function changeSchedule(later: boolean, date: string) {
		showSchedule = later;
		scheduleAt = date;
		draft.scheduleLater = later;
		draft.scheduledAt = later ? scheduledUTCDate(date) : undefined;
	}
	function message(e: unknown) {
		void reportFailure(e, 'composer');
		if (e instanceof APIError && e.fields)
			return `${e.message} ${Object.entries(e.fields)
				.map(([field, value]) => `${field}: ${value}`)
				.join(' · ')}`;
		return e instanceof Error
			? e.message
			: 'The action could not be completed.';
	}
	async function saveDraft(destination = closeDestination) {
		const lifecycle = recovery,
			seq = generation;
		if (
			!lifecycle?.active ||
			!canSaveDraft ||
			locked ||
			mediaEdits.length ||
			!changed
		)
			return;
		previews?.pause();
		closing = true;
		error = '';
		try {
			await lifecycle.checkpoint($state.snapshot(draft as unknown) as Draft);
			if (seq !== generation || destroyed || recovery !== lifecycle) return;
			dirty = false;
			stored = 'Draft saved on this device';
			closeDialog.close();
			window.dispatchEvent(
				new CustomEvent('craftsky-draft-saved', {
					detail: { accountId: session.id, did: lifecycle.owner }
				})
			);
			await leaveEditor(destination);
		} catch (e) {
			if (seq === generation && !destroyed && !presentFieldErrors(e))
				error = message(e);
		} finally {
			if (seq === generation && !destroyed) closing = false;
		}
	}

	async function addAttachments(files: FileList | null) {
		if (!files || locked) return;
		try {
			const kind = attachmentSelection(Array.from(files));
			if (kind === 'video') {
				if (!canAddVideo)
					throw new Error('Video cannot be added to this post.');
				await addVideo(files);
			} else if (kind === 'images') await addImages(files);
		} catch (e) {
			error = message(e);
		}
	}
	async function addImages(files: FileList | null) {
		if (!files || locked) return;
		const seq = generation;
		busy = true;
		error = '';
		try {
			if (draft.images.length + files.length > 8)
				throw new Error('Choose up to eight images.');
			const prepared: DraftMedia[] = [];
			for (const file of Array.from(files)) {
				progress = `Preparing image ${prepared.length + 1} of ${files.length}…`;
				prepared.push(await prepareImage(file, file.name));
			}
			if (seq === generation && !destroyed) {
				draft.images = [...draft.images, ...prepared];
				if (!retainedExternal) {
					draft.external = undefined;
					draft.externalImage = undefined;
					draft.externalSource = undefined;
					draft.externalResolvedURI = undefined;
				}
				if (draft.edit?.post.video) draft.embedChanges = null;
			}
		} catch (e) {
			if (seq === generation && !destroyed && !presentFieldErrors(e))
				error = message(e);
		} finally {
			if (seq === generation && !destroyed) {
				busy = false;
				publishing = false;
				progress = '';
			}
		}
	}
	async function addVideo(files: FileList | null) {
		if (!files?.[0] || locked) return;
		const seq = generation;
		busy = true;
		error = '';
		try {
			const value = await inspectVideo(files[0]);
			if (seq === generation && !destroyed) {
				draft.video = value;
				videoDetailsRequest = value.id;
				draft.external = undefined;
				draft.externalImage = undefined;
				draft.externalSource = undefined;
				draft.externalResolvedURI = undefined;
				draft.videoRecovery = 0;
			}
		} catch (e) {
			if (seq === generation && !destroyed && !presentFieldErrors(e))
				error = message(e);
		} finally {
			if (seq === generation && !destroyed) busy = false;
		}
	}
	function moveImage(index: number, direction: number) {
		const copy = [...draft.images];
		const other = index + direction;
		if (other < 0 || other >= copy.length) return;
		[copy[index], copy[other]] = [copy[other], copy[index]];
		draft.images = copy;
	}
	async function previewThumbnail(
		thumb: JsonObject,
		signal: AbortSignal
	): Promise<DraftMedia | undefined> {
		if (
			typeof thumb.bytes !== 'string' ||
			!/^(image\/jpeg|image\/png|image\/webp)$/.test(str(thumb.mimeType)) ||
			thumb.bytes.length > 1_333_336 ||
			thumb.bytes.length % 4 !== 0
		)
			return;
		try {
			const decoded = atob(thumb.bytes);
			if (
				decoded.length > 1_000_000 ||
				btoa(decoded) !== thumb.bytes ||
				signal.aborted
			)
				return;
			const image = await prepareImage(
				new Blob([Uint8Array.from(decoded, (char) => char.charCodeAt(0))], {
					type: str(thumb.mimeType)
				}),
				'Link preview'
			);
			return signal.aborted ? undefined : image;
		} catch {
			return;
		}
	}
	function dismissPreview() {
		if (locked) return;
		draft.externalDismissed = true;
		externalFrozen = false;
		draft.external = undefined;
		draft.externalImage = undefined;
		draft.externalSource = undefined;
		draft.externalResolvedURI = undefined;
		if (draft.edit && record(record(draft.edit.editable.embed).external).uri)
			draft.embedChanges = null;
	}
	async function immediateImage(
		image: DraftMedia,
		signal: AbortSignal
	): Promise<void> {
		if (image.uploaded || image.existingIndex !== undefined) return;
		if (!image.blob) throw new Error('An image file is missing.');
		const result = await api.media.image(image.blob, image.mime, signal);
		image.uploaded = result.blob;
	}
	async function privateMedia(
		image: DraftMedia,
		signal: AbortSignal
	): Promise<string> {
		if (image.scheduledId) return image.scheduledId;
		if (!image.blob) throw new Error('A media file is missing.');
		const id = image.id;
		await api.schedules.stageMedia(id, image.blob, image.mime, signal);
		image.scheduledId = id;
		return id;
	}
	async function makeSchedulePayload(
		value: Draft,
		signal: AbortSignal
	): Promise<JsonObject> {
		const payload: JsonObject = {
			kind: value.kind,
			text: value.text,
			langs: value.langs,
			sponsored: value.sponsored
		};
		const facets = await buildFacets(
			value.text,
			async (handle) => (await api.search.resolveMention(handle, signal)).did
		);
		if (facets.length) payload.facets = facets as unknown as JsonObject[];
		if (value.kind === 'project' && value.project)
			payload.project = await projectFacets(
				value.project,
				undefined,
				async (handle) => (await api.search.resolveMention(handle, signal)).did
			);
		const media: JsonObject[] = [];
		for (const image of value.images) {
			progress = 'Saving private media…';
			const id = await privateMedia(image, signal);
			media.push({
				id,
				alt: image.alt,
				...(image.width && image.height
					? { width: image.width, height: image.height }
					: {})
			});
		}
		if (media.length) payload.media = media;
		if (value.external) {
			const external: JsonObject = {
				sourceUri: value.externalSource || value.external.uri,
				uri: value.external.uri,
				title: value.external.title,
				description: value.external.description
			};
			if (value.externalImage)
				external.thumbMediaId = await privateMedia(value.externalImage, signal);
			payload.external = external;
		}
		return payload;
	}
	function checkpoint(owner: string, value: Draft) {
		const lifecycle = recovery;
		return lifecycle?.owner === owner && lifecycle.id === value.id
			? lifecycle.checkpoint(value)
			: drafts.save(owner, value);
	}
	async function submit(
		action: 'publish' | 'schedule' | 'publish-now' = 'publish'
	) {
		if (busy || closing || mediaTasks.length || mediaEdits.length || loading)
			return;
		if (draft.pending) {
			await retry();
			return;
		}
		const owner = session.did,
			seq = generation;
		const signal = controller.signal;
		const working = $state.snapshot(draft as unknown) as Draft;
		previews?.pause();
		busy = true;
		error = '';
		notice = '';
		try {
			const errors = validateDraft(working, action !== 'publish');
			if (errors.length) {
				const text = findField(
					working.reply
						? `Write your ${responseLabel(working).toLowerCase()}`
						: working.kind === 'project'
							? 'Project description'
							: 'What would you like to share?'
				);
				const textErrors = errors.filter(
					(message) =>
						message === 'Add some text to your post.' ||
						message.startsWith('Posts can contain')
				);
				if (textErrors.length) showFieldError(text, textErrors.join(' '));
				const remaining = errors.filter(
					(message) => !textErrors.includes(message)
				);
				const mapped: string[] = [];
				const groups = new Map<HTMLElement, string[]>();
				const labels: [RegExp, string][] = [
					[/^Choose a primary craft/, 'Primary craft'],
					[/^Project title/, 'Project title'],
					[/^Duration/, 'Duration'],
					[/^Pattern name/, 'Pattern tag or name'],
					[/^(Designer|Publisher)/, 'Pattern details'],
					[/languages/, 'Post languages'],
					[/materials/, 'Materials and style'],
					[/^Video descriptions/, 'Video description'],
					[/images|Image descriptions|^The video/, 'Photos'],
					[/^Link previews|link preview URL/, 'Link preview']
				];
				for (const message of remaining) {
					const craft = str(record(working.project?.common).craftType)
						.split('#')
						.at(-1);
					const label = /^(Gauge|Fit notes)/.test(message)
						? `${craft} details`
						: labels.find(([pattern]) => pattern.test(message))?.[1];
					const field =
						(label && findField(label)) || findField('Post options');
					if (field) {
						groups.set(field, [...(groups.get(field) || []), message]);
						mapped.push(message);
					}
				}
				for (const [field, messages] of groups)
					showFieldError(field, messages.join(' '));

				for (const message of mapped)
					remaining.splice(remaining.indexOf(message), 1);
				if (!remaining.length) {
					focusField(
						textErrors.length
							? text
							: document.querySelector<HTMLElement>('[aria-invalid=true]') ||
									undefined
					);
					return;
				}
				throw new Error(remaining.join(' '));
			}
			publishing = true;
			scheduling = action === 'schedule';
			let pending: PendingWrite;
			if (action === 'schedule' || action === 'publish-now') {
				const payload = await makeSchedulePayload(working, signal);
				if (action === 'publish-now') {
					if (!working.scheduleId)
						throw new Error('This post is not scheduled.');
					pending = {
						key: working.id,
						kind: 'schedule-publish',
						body: { payload },
						rkey: working.scheduleId
					};
				} else {
					const date = new Date(scheduleAt);
					if (
						!scheduleAt ||
						!Number.isFinite(date.getTime()) ||
						date.getTime() < Date.now() + 5 * 60 * 1000
					)
						throw new Error(
							'Choose a time at least five minutes in the future.'
						);
					pending = {
						key: working.id,
						kind: working.scheduleId ? 'schedule-update' : 'schedule-create',
						body: { scheduledAt: date.toISOString(), payload },
						rkey: working.scheduleId
					};
				}
			} else {
				for (let index = 0; index < working.images.length; index++) {
					progress = `Uploading image ${index + 1} of ${working.images.length}…`;
					await immediateImage(working.images[index], signal);
				}
				if (working.externalImage) {
					await immediateImage(working.externalImage, signal);
					if (working.external)
						working.external.thumb = working.externalImage.uploaded;
				}
				const facets = await buildFacets(
					working.text,
					async (handle) =>
						(await api.search.resolveMention(handle, signal)).did
				);
				if (working.kind === 'project' && working.project)
					working.project = await projectFacets(
						working.project,
						working.edit ? record(working.edit.editable.project) : undefined,
						async (handle) =>
							(await api.search.resolveMention(handle, signal)).did
					);
				let body: JsonObject;
				if (working.edit) {
					body = editChanges(working, facets) as unknown as JsonObject;
					if (working.video) {
						const completed = await uploadVideo(
							api,
							owner,
							working.video,
							signal,
							(value) => {
								if (seq === generation && !destroyed) progress = value;
							},
							undefined,
							(working.videoRecovery || 0) === 1
						);
						if ('video' in completed) {
							body.embed = {
								$type: 'app.bsky.embed.video',
								video: completed.video.blob as unknown as JsonObject,
								jobId: completed.video.jobId,
								alt: completed.video.alt,
								...(completed.video.aspectRatio
									? { aspectRatio: completed.video.aspectRatio }
									: {})
							};
							body.images = null;
						}
					}
					if (!Object.keys(body).length) {
						notice = 'There are no changes to save.';
						return;
					}
				} else {
					const payload = postPayload(working, facets);
					if (working.video)
						payload.embed = await uploadVideo(
							api,
							owner,
							working.video,
							signal,
							(value) => {
								if (seq === generation && !destroyed) progress = value;
							},
							undefined,
							(working.videoRecovery || 0) === 1
						);
					body = payload as unknown as JsonObject;
				}
				pending = freezeWrite(body, working);
			}
			if (seq !== generation || destroyed) return;
			working.pending = pending;
			draft = working;
			// Keep an exact durable request before starting an externally visible write.
			try {
				await checkpoint(owner, $state.snapshot(draft as unknown) as Draft);
			} catch (storageError) {
				draft.pending = undefined;
				throw storageError;
			}
			if (seq !== generation || destroyed) return;
			await perform(pending, owner, seq, working.id);
		} catch (e) {
			if (seq === generation && !destroyed && !presentFieldErrors(e))
				error = message(e);
		} finally {
			if (seq === generation && !destroyed) {
				busy = false;
				publishing = false;
				progress = '';
			}
		}
	}
	async function retry() {
		if (busy || !draft.pending) return;
		const seq = generation,
			owner = session.did;
		busy = true;
		publishing = true;
		scheduling =
			!!draft.pending.kind?.startsWith('schedule-') &&
			draft.pending.kind !== 'schedule-publish';
		error = '';
		try {
			await perform(
				$state.snapshot(draft.pending as unknown) as PendingWrite,
				owner,
				seq,
				draft.id
			);
		} catch (e) {
			if (seq === generation && !destroyed && !presentFieldErrors(e))
				error = message(e);
		} finally {
			if (seq === generation && !destroyed) {
				busy = false;
				publishing = false;
				progress = '';
			}
		}
	}
	async function perform(
		pending: PendingWrite,
		owner: string,
		seq: number,
		draftId: string
	) {
		progress = 'Confirming your post…';
		const lifecycle =
			recovery?.owner === owner && recovery.id === draftId
				? recovery
				: undefined;
		try {
			let destination = closeDestination;
			let resultNotice = 'Your post has been published.';
			if (pending.kind === 'schedule-create') {
				await api.schedules.create(
					pending.key,
					str(pending.body.scheduledAt),
					record(pending.body.payload)
				);
				destination = origin || '/scheduled';
				resultNotice = 'Your post has been scheduled.';
			} else if (pending.kind === 'schedule-update') {
				await api.schedules.update(
					pending.rkey!,
					str(pending.body.scheduledAt),
					record(pending.body.payload)
				);
				destination = origin || '/scheduled';
				resultNotice = 'Your schedule has been updated.';
			} else if (pending.kind === 'schedule-publish') {
				const result = await api.schedules.publish(
					pending.rkey!,
					record(pending.body.payload)
				);
				destination = origin || '/scheduled';
				resultNotice =
					result.status === 'publishing'
						? 'Your post is being published.'
						: 'Your post has been published.';
			} else if (pending.kind === 'edit') {
				const result = await api.posts.edit(
					pending.did!,
					pending.rkey!,
					pending.body as unknown as PostEditChanges,
					pending.cid!,
					pending.key
				);
				if (seq !== generation || destroyed) return;
				publications?.watchEdit(pending.rkey!, result, () =>
					api.posts.get(pending.did!, pending.rkey!)
				);
				destination = `/posts/${encodeURIComponent(pending.did!)}/${encodeURIComponent(pending.rkey!)}`;
				resultNotice = 'Your changes have been saved.';
			} else {
				const post = await api.posts.create(
					pending.body as unknown as PostPayload,
					pending.key
				);
				if (seq !== generation || destroyed) return;
				publications?.accept(post, context);
				destination = origin || closeDestination;
			}
			// An accepted public write stays accepted even if local cleanup fails.
			// A retained pending draft can only recheck the original idempotent action.
			await (
				lifecycle ? lifecycle.complete() : drafts.remove(owner, draftId)
			).catch(() => {});
			if (seq !== generation || destroyed) return;
			draft.pending = undefined;
			notify(resultNotice);
			dirty = false;
			try {
				await leaveEditor(destination);
			} catch {
				error =
					'Your action completed, but the next page could not be opened. Use navigation to continue.';
			}
		} catch (e) {
			if (seq !== generation || destroyed) return;
			const uncertain =
				e instanceof MutationOutcomeUnknownError ||
				(e instanceof APIError &&
					e.code !== 'video_blob_missing' &&
					(e.status === 0 || e.status >= 500));
			if (uncertain) {
				error =
					'This action is still being confirmed. The exact request has been saved. Retry to check the same action; do not create a second post.';
				await checkpoint(owner, $state.snapshot(draft as unknown) as Draft);
			} else {
				draft.pending = undefined;
				if (
					e instanceof APIError &&
					e.code === 'video_blob_missing' &&
					draft.video
				) {
					draft.videoRecovery = (draft.videoRecovery || 0) + 1;
				}
				await checkpoint(owner, $state.snapshot(draft as unknown) as Draft);
				if (e instanceof APIError && e.status === 409)
					error =
						'This post changed while you were editing. Your draft is saved. Reload the current post before applying your changes.';
				else if (e instanceof APIError && e.code === 'video_blob_missing')
					error =
						(draft.videoRecovery || 0) < 2
							? 'The video could not be retrieved. Try publishing again to upload a fresh copy.'
							: 'The video could not be restored. Remove it and choose the file again.';
				else error = message(e);
			}
		}
	}
	function beforeLeave(event: BeforeUnloadEvent) {
		if ((dirty || mediaEdits.length) && !draft.pending) {
			event.preventDefault();
			event.returnValue = '';
		}
	}
</script>

<PublishingOverlay
	active={publishing}
	video={!!draft.video}
	fraction={progress.match(/(\d+(?:\.\d+)?)%/)
		? Number(progress.match(/(\d+(?:\.\d+)?)%/)![1])
		: undefined}
	label={progress.startsWith('Uploading') || progress.startsWith('Processing')
		? progress
		: draft.edit
			? 'Saving changes…'
			: scheduling
				? 'Scheduling post…'
				: draft.reply
					? `Publishing ${responseLabel(draft).toLowerCase()}…`
					: 'Publishing post…'}
/>
<svelte:window onbeforeunload={beforeLeave} />
<section class="composer">
	<header class="composer-heading">
		<a
			class="close-composer"
			href={closeDestination}
			aria-label="Close editor"
			aria-disabled={closing}
			onclick={(event) => {
				event.preventDefault();
				requestClose();
			}}><Icon name="close" /></a
		>
		<h1>{title}</h1>
		{#if canSaveDraft}<button
				class="save-draft"
				type="button"
				onclick={() => saveDraft()}
				disabled={!initialized ||
					!changed ||
					closing ||
					busy ||
					mediaTasks.length > 0 ||
					mediaEdits.length > 0 ||
					!!draft.pending}
				>{existingDraft ? 'Save changes' : 'Save draft'}</button
			>{/if}
	</header>
	{#if error}<Snackbar
			message={error}
			warning
			onDismiss={() => (error = '')}
		/>{/if}
	{#if notice}<Snackbar message={notice} onDismiss={() => (notice = '')} />{/if}
	{#if loading}<p role="status" class="loading">
			Loading your editor…
		</p>{:else if !initialized}<button
			onclick={() =>
				initialize(
					session.did,
					path.join('/'),
					new URLSearchParams(page.url.search)
				)}>Try loading the editor again</button
		>{:else}
		<form
			class:project-composer={draft.kind === 'project'}
			onsubmit={(e) => {
				e.preventDefault();
			}}
		>
			<div class="composer-scroll">
				{#if draft.pending}<section class="composer-pending">
						<h2>Waiting for confirmation</h2>
						<p>
							Your original action is saved and editing is paused while its
							outcome is checked.
						</p>
						<button class="primary" disabled={busy} onclick={retry}
							>Check the same action</button
						>
					</section>{/if}
				{#if context}<div class="target-context">
						<ComposerTarget post={context} fullText={!!draft.reply} />
					</div>{/if}
				<fieldset disabled={locked} class="editor-fields">
					{#if draft.kind === 'project' && draft.project}<ProjectEditor
							{api}
							owner={session.id + ':' + session.did}
							disabled={locked}
							project={draft.project}
							part="main"
							onchange={(value) => (draft.project = value)}
						/>{/if}
					<div class="post-text-label">
						<FacetTextInput
							{api}
							label={draft.reply
								? `Write your ${responseLabel(draft).toLowerCase()}`
								: draft.kind === 'project'
									? 'Project description'
									: 'What would you like to share?'}
							hint="required"
							value={draft.text}
							onchange={(value) => (draft.text = value)}
							rows={draft.reply ? 5 : draft.kind === 'project' ? 6 : 3}
							disabled={locked}
							className="post-text"
							placeholder={draft.reply
								? `Add your ${responseLabel(draft).toLowerCase()}…`
								: draft.kind === 'project'
									? 'Tell everyone about your project'
									: "Share what you're making, learning, or thinking..."}
						/>
					</div>
					<p class:over-limit={draftSize > 2000} class="character-count">
						{draftSize}/2000
					</p>

					<section
						class="composer-section attachment-section"
						data-field="Photos"
					>
						{#if !draft.video && !retainedVideo && !retainedExternal}<PhotoAttachments
								images={draft.images}
								required={draft.kind === 'project'}
								disabled={locked}
								{canAddVideo}
								onadd={addAttachments}
								onchange={(images) => (draft.images = images)}
								onBusyChange={mediaBusy}
								onEditingChange={mediaEditing}
							/>{/if}
						{#if draft.video}<VideoAttachment
								media={draft.video}
								autoOpen={videoDetailsRequest === draft.video.id}
								disabled={locked}
								onalt={(value) => {
									if (draft.video) draft.video.alt = value;
								}}
								onremove={() => {
									draft.video = undefined;
									draft.videoRecovery = 0;
								}}
							/>{/if}
						{#if retainedVideo && draft.edit?.post.video}
							<div class="video-editor">
								<VideoPlayer
									video={draft.edit.post.video}
									{api}
									did={draft.edit.post.author.did}
									rkey={draft.edit.post.rkey}
								/>
								<label
									>Video description<textarea
										rows="2"
										maxlength="1000"
										value={str(
											draft.embedChanges?.alt ??
												record(draft.edit.editable.embed).alt
										)}
										oninput={(e) =>
											(draft.embedChanges = { alt: e.currentTarget.value })}
									></textarea></label
								>
								<button
									type="button"
									class="danger"
									onclick={() => (draft.embedChanges = null)}
									>Remove published video</button
								>
							</div>
						{/if}
					</section>
					{#if draft.external && !draft.images.length && !draft.quote && !draft.video && !retainedVideo}<div
							class="link-preview-section"
							data-field="Link preview"
						>
							<ComposerLinkPreview
								external={draft.external}
								image={draft.externalImage}
								thumbnailUrl={draft.edit?.post.external?.thumb?.url}
								loading={previewState.loading}
								index={retainedExternal ? 1 : previewState.index || 1}
								total={retainedExternal ? 1 : previewState.total || 1}
								disabled={locked}
								onremove={dismissPreview}
								onprevious={() => previews?.move(-1)}
								onnext={() => previews?.move(1)}
							/>
						</div>
					{:else if previewState.loading && !previewSuppressed && !draft.externalDismissed}<p
							role="status"
							class="link-preview-section"
							data-field="Link preview"
						>
							Loading link preview
						</p>{/if}
					{#if draft.externalDismissed && !previewSuppressed}<p
							class="link-preview-section preview-dismissed"
							role="status"
						>
							Link previews hidden <button
								type="button"
								disabled={locked}
								onclick={() => {
									draft.externalDismissed = false;
								}}>Undo</button
							>
						</p>{/if}
					<div
						class="metadata-controls"
						aria-label="Post options"
						data-field="Post options"
					>
						<LanguagePicker
							owner={session.id + ':' + session.did}
							languages={draft.langs}
							disabled={locked}
							onchange={(value) => (draft.langs = value)}
						/>
						{#if !draft.reply}<SponsoredPicker
								owner={session.id + ':' + session.did}
								value={draft.sponsored}
								disabled={locked}
								onchange={(value) => (draft.sponsored = value)}
							/>{/if}
						{#if canChooseSchedule && !draft.pending}<SchedulePicker
								owner={session.id + ':' + session.did}
								later={showSchedule}
								value={scheduleAt}
								disabled={locked || (!!tier && !tier() && !draft.scheduleId)}
								fixed={!!draft.scheduleId}
								allowed={canSchedule}
								full={scheduleFull}
								onchange={changeSchedule}
							/>{/if}
					</div>
					{#if canChooseSchedule && scheduleFull}<p class="schedule-warning">
							You can't schedule another post because you already have 3
							scheduled posts. <a href="/scheduled">Manage scheduled posts</a>
						</p>{/if}
					{#if draft.kind === 'project' && draft.project}<ProjectEditor
							{api}
							owner={session.id + ':' + session.did}
							disabled={locked}
							project={draft.project}
							part="details"
							onchange={(value) => (draft.project = value)}
						/>{/if}
				</fieldset>
			</div>
			<footer class="composer-actions">
				<span class="muted" role="status">{progress || stored}</span>
				<div class="toolbar">
					{#if draft.scheduleId && !draft.pending}<button
							type="button"
							onclick={() => submit('publish-now')}
							disabled={busy}>Publish now</button
						>{/if}<button
						class="primary"
						type="button"
						onclick={() => submit(showSchedule ? 'schedule' : 'publish')}
						disabled={busy ||
							closing ||
							mediaTasks.length > 0 ||
							mediaEdits.length > 0 ||
							(showSchedule && scheduleFull) ||
							loading}
						>{busy
							? 'Working…'
							: draft.pending
								? 'Check action'
								: showSchedule
									? 'Save schedule'
									: draft.edit
										? 'Save changes'
										: draft.reply
											? responseLabel(draft)
											: 'Post'}</button
					>
				</div>
			</footer>
		</form>
	{/if}
</section>

<dialog
	bind:this={closeDialog}
	class="composer-close-dialog"
	aria-label={canSaveDraft ? 'Save your draft?' : 'Discard draft?'}
	oncancel={(event) => {
		if (closing) event.preventDefault();
	}}
>
	<h2>{canSaveDraft ? 'Save your draft?' : 'Discard draft?'}</h2>
	<p>
		{canSaveDraft
			? 'You can save this work on this device before closing.'
			: "Your changes won't be saved."}
	</p>
	{#if draft.pending}<p>
			Check the original action before discarding this draft.
		</p>{/if}
	{#if mediaEdits.length || mediaTasks.length || busy}<p>
			Finish or cancel your media edit before closing.
		</p>{/if}
	<div class="composer-close-actions">
		<button type="button" disabled={closing} onclick={keepEditing}
			>Keep editing</button
		><button
			type="button"
			class="danger"
			disabled={locked || mediaEdits.length > 0}
			onclick={discardDraft}
			>{existingDraft ? 'Discard changes' : 'Discard'}</button
		>{#if canSaveDraft}<button
				type="button"
				class="primary"
				disabled={locked || mediaEdits.length > 0 || !changed}
				onclick={() => saveDraft(closeTarget)}
				>{existingDraft ? 'Save changes' : 'Save draft'}</button
			>{/if}
	</div>
</dialog>
