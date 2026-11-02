<script lang="ts">
	import { getContext, onDestroy, onMount, untrack } from 'svelte';
	import { page } from '$app/state';
	import { goto, beforeNavigate } from '$app/navigation';
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
		validateDraft
	} from './model.ts';
	import { inspectVideo, prepareImage, uploadVideo } from './media.ts';
	import ProjectEditor from './ProjectEditor.svelte';
	import MediaEditor from './MediaEditor.svelte';
	import FacetTextInput from './FacetTextInput.svelte';
	import VideoPlayer from '../social/VideoPlayer.svelte';
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
	let mediaTasks = $state<string[]>([]);
	let mediaEdits = $state<string[]>([]);
	let error = $state('');
	let notice = $state('');
	let progress = $state('');
	let stored = $state('');
	let mounted = $state(false);
	let dirty = $state(false);
	let context = $state<Post>();
	let language = $state('en');
	let scheduleAt = $state('');
	let showSchedule = $state(false);
	let link = $state('');
	let videoURL = $state('');
	let generation = 0;
	let destroyed = false;
	let controller = new AbortController();
	const tier =
		getContext<
			() => import('#lib/api/client.ts').SubscriptionAccess | undefined
		>('craftsky-access');
	const locked = $derived(busy || mediaTasks.length > 0 || !!draft.pending);
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
	const title = $derived(
		draft.edit
			? 'Edit post'
			: draft.reply
				? 'Reply'
				: draft.quote
					? 'Quote post'
					: draft.scheduleId
						? 'Edit scheduled post'
						: draft.kind === 'project'
							? 'New project'
							: 'New post'
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
	$effect(() => {
		const identity = session.did;
		const route = path.join('/');
		const query = page.url.search;
		if (mounted)
			untrack(() => {
				void initialize(identity, route, new URLSearchParams(query));
			});
	});
	$effect(() => {
		const video = draft.video;
		const next = video?.blob
			? URL.createObjectURL(video.blob)
			: video?.remoteUrl || '';
		videoURL = next;
		return () => {
			if (video?.blob) URL.revokeObjectURL(next);
		};
	});
	$effect(() => {
		if (!mounted || loading || busy) return;
		const snapshot = $state.snapshot(draft as unknown) as Draft;
		JSON.stringify(snapshot);
		if (
			!snapshot.text &&
			!snapshot.images.length &&
			!snapshot.video &&
			!snapshot.external &&
			!snapshot.edit &&
			!snapshot.scheduleId &&
			!snapshot.pending &&
			JSON.stringify(snapshot.project || {}) ===
				JSON.stringify(snapshot.kind === 'project' ? defaultProject() : {})
		)
			return;
		dirty = true;
		const owner = session.did;
		const seq = generation;
		const timer = setTimeout(() => {
			void drafts
				.save(owner, snapshot)
				.then(() => {
					if (seq === generation && !destroyed) {
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
		if (mediaEdits.length && !confirm('Discard unapplied image edits?')) {
			nav.cancel();
			return;
		}
		if (dirty && !loading && !busy && !draft.pending)
			void drafts
				.save(session.did, $state.snapshot(draft as unknown) as Draft)
				.catch(() => {});
	});
	onDestroy(() => {
		destroyed = true;
		generation++;
		controller.abort();
	});
	async function initialize(
		owner: string,
		route: string,
		query: URLSearchParams
	) {
		const seq = ++generation;
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
		try {
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
				const item = await api.schedules.get(scheduled);
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
					scheduledAt: item.scheduledAt
				};
				const media = Array.isArray(payload.media)
					? payload.media.map(record)
					: [];
				next.images = await Promise.all(
					media.map(async (value) => {
						const id = str(value.id);
						const bytes = await api.schedules.media(id, controller.signal);
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
						const bytes = await api.schedules.media(id, controller.signal);
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
				const seed = await api.posts.editSeed(did, rkey);
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
				const target = await api.posts.get(did, rkey);
				if (!('author' in target)) throw new Error('This post is unavailable.');
				if (mode === 'quote' && target.reply)
					throw new Error(
						'Replies cannot be quoted. Open the original post to quote it.'
					);
				context = target;
				const ref: StrongRef = { uri: target.uri, cid: target.cid };
				if (mode === 'reply')
					next.reply = { root: target.reply?.root || ref, parent: ref };
				else next.quote = ref;
			} else {
				const preferences = await api.languages.get().catch(() => undefined);
				if (preferences) next.langs = [preferences.primaryLanguage];
			}
			if (seq !== generation || destroyed) return;
			if (next.kind === 'project' && !next.project)
				next.project = defaultProject();
			draft = next;
			initialized = true;
			language = next.langs.join(', ');
			showSchedule = !!next.scheduleId;
			scheduleAt = next.scheduledAt ? localDate(next.scheduledAt) : '';
		} catch (e) {
			if (seq === generation && !destroyed) error = message(e);
		} finally {
			if (seq === generation && !destroyed) loading = false;
		}
	}
	function localDate(value: string) {
		const date = new Date(value);
		return new Date(date.getTime() - date.getTimezoneOffset() * 60000)
			.toISOString()
			.slice(0, 16);
	}
	function message(e: unknown) {
		if (e instanceof APIError && e.fields)
			return `${e.message} ${Object.entries(e.fields)
				.map(([field, value]) => `${field}: ${value}`)
				.join(' · ')}`;
		return e instanceof Error
			? e.message
			: 'The action could not be completed.';
	}
	async function saveDraft() {
		const owner = session.did,
			seq = generation;
		try {
			await drafts.save(owner, $state.snapshot(draft as unknown) as Draft);
			if (seq === generation && !destroyed) {
				stored = 'Draft saved on this device';
				notice = 'Your draft and media are saved on this device.';
				dirty = false;
			}
		} catch (e) {
			if (seq === generation && !destroyed) error = message(e);
		}
	}
	function setLanguages(value: string) {
		language = value;
		draft.langs = value
			.split(',')
			.map((item) => item.trim())
			.filter(Boolean);
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
				if (draft.edit?.post.video) draft.embedChanges = null;
			}
		} catch (e) {
			if (seq === generation && !destroyed) error = message(e);
		} finally {
			if (seq === generation && !destroyed) {
				busy = false;
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
				draft.videoRecovery = 0;
			}
		} catch (e) {
			if (seq === generation && !destroyed) error = message(e);
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
	async function previewLink() {
		const seq = generation;
		busy = true;
		error = '';
		try {
			const value = await api.media.linkPreview(link, controller.signal);
			if (seq !== generation || destroyed) return;
			draft.external = {
				uri: value.url || link,
				title: value.title,
				description: value.description
			};
			const thumb = record(value.thumbnail);
			if (
				typeof thumb.bytes === 'string' &&
				thumb.bytes.length < 1_400_000 &&
				typeof thumb.mimeType === 'string'
			) {
				const bytes = Uint8Array.from(atob(thumb.bytes), (char) =>
					char.charCodeAt(0)
				);
				const prepared = await prepareImage(
					new Blob([bytes], { type: thumb.mimeType }),
					'Link preview'
				);
				if (seq === generation && !destroyed) draft.externalImage = prepared;
			}
		} catch (e) {
			if (seq === generation && !destroyed) error = message(e);
		} finally {
			if (seq === generation && !destroyed) busy = false;
		}
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
				sourceUri: value.external.uri,
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
	async function submit(
		action: 'publish' | 'schedule' | 'publish-now' = 'publish'
	) {
		if (busy || mediaTasks.length || mediaEdits.length || loading) return;
		if (draft.pending) {
			await retry();
			return;
		}
		const owner = session.did,
			seq = generation;
		const signal = controller.signal;
		const working = $state.snapshot(draft as unknown) as Draft;
		busy = true;
		error = '';
		notice = '';
		try {
			const errors = validateDraft(working, action !== 'publish');
			if (errors.length) throw new Error(errors.join(' '));
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
				await drafts.save(owner, $state.snapshot(draft as unknown) as Draft);
			} catch (storageError) {
				draft.pending = undefined;
				throw storageError;
			}
			if (seq !== generation || destroyed) return;
			await perform(pending, owner, seq, working.id);
		} catch (e) {
			if (seq === generation && !destroyed) error = message(e);
		} finally {
			if (seq === generation && !destroyed) {
				busy = false;
				progress = '';
			}
		}
	}
	async function retry() {
		if (busy || !draft.pending) return;
		const seq = generation,
			owner = session.did;
		busy = true;
		error = '';
		try {
			await perform(
				$state.snapshot(draft.pending as unknown) as PendingWrite,
				owner,
				seq,
				draft.id
			);
		} catch (e) {
			if (seq === generation && !destroyed) error = message(e);
		} finally {
			if (seq === generation && !destroyed) {
				busy = false;
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
		try {
			let destination = '/app/home';
			let resultNotice = 'Your post has been published.';
			if (pending.kind === 'schedule-create') {
				await api.schedules.create(
					pending.key,
					str(pending.body.scheduledAt),
					record(pending.body.payload)
				);
				destination = '/app/scheduled';
				resultNotice = 'Your post has been scheduled.';
			} else if (pending.kind === 'schedule-update') {
				await api.schedules.update(
					pending.rkey!,
					str(pending.body.scheduledAt),
					record(pending.body.payload)
				);
				destination = '/app/scheduled';
				resultNotice = 'Your schedule has been updated.';
			} else if (pending.kind === 'schedule-publish') {
				const result = await api.schedules.publish(
					pending.rkey!,
					record(pending.body.payload)
				);
				destination = '/app/scheduled';
				resultNotice =
					result.status === 'publishing'
						? 'Your post is being published.'
						: 'Your post has been published.';
			} else if (pending.kind === 'edit') {
				await api.posts.edit(
					pending.did!,
					pending.rkey!,
					pending.body as unknown as PostEditChanges,
					pending.cid!,
					pending.key
				);
				destination = `/app/posts/${encodeURIComponent(pending.did!)}/${encodeURIComponent(pending.rkey!)}`;
				resultNotice = 'Your changes have been saved.';
			} else {
				const post = await api.posts.create(
					pending.body as unknown as PostPayload,
					pending.key
				);
				destination = `/app/posts/${encodeURIComponent(post.author.did)}/${encodeURIComponent(post.rkey)}`;
			}
			// An accepted public write stays accepted even if local cleanup fails.
			// A retained pending draft can only recheck the original idempotent action.
			await drafts.remove(owner, draftId).catch(() => {});
			if (seq !== generation || destroyed) return;
			draft.pending = undefined;
			notice = resultNotice;
			dirty = false;
			await goto(destination);
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
				await drafts.save(owner, $state.snapshot(draft as unknown) as Draft);
			} else {
				draft.pending = undefined;
				if (
					e instanceof APIError &&
					e.code === 'video_blob_missing' &&
					draft.video
				) {
					draft.videoRecovery = (draft.videoRecovery || 0) + 1;
				}
				await drafts.save(owner, $state.snapshot(draft as unknown) as Draft);
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

<svelte:window onbeforeunload={beforeLeave} />
<section class="composer">
	<header class="composer-heading">
		<div>
			<p class="eyebrow">Share what you make</p>
			<h1>{title}</h1>
		</div>
		<a href="/app/drafts">Drafts</a>
	</header>
	{#if error}<div class="composer-error" role="alert">{error}</div>{/if}
	{#if notice}<p role="status" class="composer-notice">{notice}</p>{/if}
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
		{#if draft.pending}<section class="composer-pending">
				<h2>Waiting for confirmation</h2>
				<p>
					Your original action is saved and editing is paused while its outcome
					is checked.
				</p>
				<button class="primary" disabled={busy} onclick={retry}
					>Check the same action</button
				>
			</section>{/if}
		{#if context}<blockquote class="context">
				<strong>{context.author.displayName || context.author.handle}</strong>
				<p>{context.text}</p>
				{#if draft.quote && context.images?.length}<label
						>Image from the quoted post<select
							disabled={locked}
							value={draft.quoteImageCid || ''}
							onchange={(e) =>
								(draft.quoteImageCid = e.currentTarget.value || undefined)}
							><option value="">No selected image</option
							>{#each context.images as image, index}{#if image.cid}<option
										value={image.cid}
										>{image.alt || `Image ${index + 1}`}</option
									>{/if}{/each}</select
						></label
					>{/if}
			</blockquote>{/if}
		<form
			onsubmit={(e) => {
				e.preventDefault();
				void submit(showSchedule ? 'schedule' : 'publish');
			}}
		>
			<fieldset disabled={locked} class="editor-fields">
				{#if !draft.edit && !draft.reply && !draft.quote}<div
						class="kind-switch"
						role="group"
						aria-label="Post type"
					>
						<button
							type="button"
							aria-pressed={draft.kind === 'standard'}
							onclick={() => (draft.kind = 'standard')}>Post</button
						><button
							type="button"
							aria-pressed={draft.kind === 'project'}
							onclick={() => {
								draft.kind = 'project';
								draft.project ||= defaultProject();
							}}>Project</button
						>
					</div>{/if}
				<div class="post-text-label">
					<FacetTextInput
						{api}
						label={draft.reply
							? 'Your reply'
							: draft.kind === 'project'
								? 'Tell the story of your project'
								: 'What have you been making?'}
						value={draft.text}
						onchange={(value) => (draft.text = value)}
						rows={6}
						disabled={locked}
						className="post-text"
						placeholder={draft.reply
							? 'Add your reply…'
							: 'Share a little of your craft…'}
					/>
				</div>
				<p class:over-limit={draftSize > 2000} class="character-count">
					{draftSize.toLocaleString()} / 2,000
				</p>
				{#if draft.kind === 'project' && draft.project}<ProjectEditor
						{api}
						disabled={locked}
						project={draft.project}
						onchange={(value) => (draft.project = value)}
					/>{/if}
				<section class="composer-section">
					<h2>Photos and video</h2>
					<div class="toolbar">
						{#if !draft.video && draft.images.length < 8 && !draft.external}<label
								class="file-button"
								>＋ Add images<input
									type="file"
									accept="image/jpeg,image/png,image/webp"
									multiple
									onchange={(e) => {
										void addImages(e.currentTarget.files);
										e.currentTarget.value = '';
									}}
								/></label
							>{/if}{#if !fixedEmbed && !draft.images.length && !draft.quote && !draft.reply && !draft.external && !draft.video}<label
								class="file-button"
								>▷ Add video<input
									type="file"
									accept="video/mp4"
									onchange={(e) => {
										void addVideo(e.currentTarget.files);
										e.currentTarget.value = '';
									}}
								/></label
							>{/if}
					</div>
					{#if draft.images.length}<div class="media-grid">
							{#each draft.images as image, index (image.id)}<MediaEditor
									media={image}
									{index}
									count={draft.images.length}
									disabled={locked}
									onBusyChange={(value) => mediaBusy(image.id, value)}
									onEditingChange={(value) => mediaEditing(image.id, value)}
									onchange={(value) => (draft.images[index] = value)}
									onremove={() =>
										(draft.images = draft.images.filter(
											(item) => item.id !== image.id
										))}
									onmove={(direction) => moveImage(index, direction)}
								/>{/each}
						</div>{/if}
					{#if draft.video}<div class="video-editor">
							<video controls src={videoURL} aria-label="Video preview"
								><track kind="captions" /></video
							><label
								>Video description<textarea
									maxlength="1000"
									rows="2"
									bind:value={draft.video.alt}
									placeholder="Describe what happens in this video"
								></textarea></label
							><button
								type="button"
								class="danger"
								onclick={() => (draft.video = undefined)}>Remove video</button
							>
							<p class="muted">MP4 · up to ten minutes · up to 300 MB</p>
						</div>{/if}
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
				{#if (draft.kind !== 'project' || draft.external) && !fixedEmbed && !retainedVideo && !draft.quote && !draft.video && !draft.images.length}<section
						class="composer-section"
					>
						<h2>Link preview</h2>
						{#if draft.external}<div class="external-preview">
								<label>Title<input bind:value={draft.external.title} /></label
								><label
									>Description<textarea
										rows="2"
										bind:value={draft.external.description}></textarea></label
								><a href={draft.external.uri} target="_blank" rel="noreferrer"
									>{draft.external.uri}</a
								><button
									type="button"
									onclick={() => {
										draft.external = undefined;
										draft.externalImage = undefined;
									}}>Remove preview</button
								>
							</div>{:else}<div class="inline-field">
								<input
									type="url"
									aria-label="Link to preview"
									bind:value={link}
									placeholder="https://"
								/><button type="button" onclick={previewLink} disabled={!link}
									>Add preview</button
								>
							</div>{/if}
					</section>{/if}
				<section class="composer-section">
					<div class="fields">
						<label
							>Post languages<span class="muted"
								>Language tags separated by commas</span
							><input
								value={language}
								oninput={(e) => setLanguages(e.currentTarget.value)}
								placeholder="en, fr"
							/></label
						>{#if !draft.reply}<label class="check"
								><input
									type="checkbox"
									bind:checked={draft.sponsored}
								/>Sponsored content</label
							>{/if}
					</div>
				</section>
			</fieldset>
			{#if canSchedule && !draft.pending}<section class="composer-section">
					<label class="check"
						><input
							type="checkbox"
							bind:checked={showSchedule}
							disabled={busy || !!draft.scheduleId}
						/>Schedule for later</label
					>{#if showSchedule}<label
							>Publish date and time<input
								type="datetime-local"
								bind:value={scheduleAt}
								min={localDate(
									new Date(Date.now() + 5 * 60 * 1000).toISOString()
								)}
								disabled={busy}
								required
							/><span class="muted"
								>Your time zone: {Intl.DateTimeFormat().resolvedOptions()
									.timeZone}. CraftSky Plus or Business is required.</span
							></label
						>{/if}
				</section>{/if}
			<footer class="composer-actions">
				<span class="muted" role="status">{progress || stored}</span>
				<div class="toolbar">
					<button
						type="button"
						onclick={saveDraft}
						disabled={busy || mediaTasks.length > 0 || mediaEdits.length > 0}
						>Save draft</button
					>{#if draft.scheduleId && !draft.pending}<button
							type="button"
							onclick={() => submit('publish-now')}
							disabled={busy}>Publish now</button
						>{/if}<button
						class="primary"
						type="button"
						onclick={() => submit(showSchedule ? 'schedule' : 'publish')}
						disabled={busy ||
							mediaTasks.length > 0 ||
							mediaEdits.length > 0 ||
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
											? 'Reply'
											: 'Publish'}</button
					>
				</div>
			</footer>
		</form>
	{/if}
</section>
