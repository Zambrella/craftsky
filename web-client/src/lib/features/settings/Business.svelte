<script lang="ts">
	import LoadMore from '#lib/LoadMore.svelte';
	import { InputError, userErrorMessage } from '#lib/user-facing-error.ts';
	import { reportFailure } from '../diagnostics/client.ts';
	import {
		FieldValidationError,
		presentFieldErrors
	} from '#lib/field-validation.ts';
	import { confirmNavigation } from '../social/navigation-prompt';
	import { contextMenu } from '../social/context-menu';
	import Icon from '../shell/Icon.svelte';
	import BusinessEventCard from '../social/BusinessEventCard.svelte';
	import { businessEventLink } from '../social/business-model';
	import { dragScroll } from '#lib/drag-scroll.ts';
	import SingleSelect from '#lib/SingleSelect.svelte';
	import RecordTextField from '#lib/RecordTextField.svelte';
	import { usePrompt } from '../social/prompt';
	const ask = usePrompt();
	import { getContext, onMount, onDestroy, untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import PhotoAttachments from '../composer/PhotoAttachments.svelte';
	import MultiSelect from '#lib/MultiSelect.svelte';
	import { prepareImage } from '../composer/media.ts';
	import type { DraftMedia } from '../drafts/store.ts';
	import {
		newOperationKey,
		type CraftskyAPI,
		type JsonObject,
		type BusinessEvent
	} from '#lib/api/client.ts';
	import {
		actions,
		businessTypes,
		offerings,
		roles,
		title,
		record,
		text,
		openValue,
		productBody,
		imageBody,
		eventStatusBody,
		https,
		zonedTimestamp,
		wallTime,
		type Session,
		type Product,
		type BusinessImage
	} from './model.ts';
	import { safeURL } from '#lib/public.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import SettingsIcon from './SettingsIcon.svelte';
	import { currencyOptions, requireRecordText } from './native-settings.ts';
	let {
		api,
		session,
		section,
		eventRkey,
		productIndex
	}: {
		api: CraftskyAPI;
		session: Session;
		section: string;
		eventRkey?: string;
		productIndex?: string;
	} = $props();
	const work = new Work('settings.Business.action');
	const returnTo =
		getContext<(destination: string) => Promise<void>>('craftsky-return') ||
		((destination: string) => goto(destination, { replaceState: true }));
	const businessFormId = $props.id();
	let loaded = $state(false);
	let loadingMore = $state(false);
	let cid = $state<string | null>(null),
		allowsBusiness = $state(false),
		types = $state<string[]>([]),
		offers = $state<string[]>([]),
		unknownTypes = $state<string[]>([]),
		unknownOffers = $state<string[]>([]);
	let tagline = $state(''),
		hours = $state(''),
		area = $state(''),
		country = $state(''),
		locality = $state(''),
		action = $state(''),
		destination = $state('');
	let productEditor = $state<number | null>(null);
	let productSnapshot = $state<Product[] | null>(null);
	let productPhotos = $state<DraftMedia[]>([]);
	let productPhotoOpen = $state(false);
	let productPhotoBaseline = $state('');
	let productBaseline = $state('');
	let productSaved = $state(false);
	$effect(() => {
		if (productSaved && !work.busy && !work.retry) {
			productSaved = false;
			untrack(() => void returnTo('/settings/products'));
		}
	});
	function editProduct(index: number | null) {
		productSnapshot = structuredClone($state.snapshot(products));
		work.error = '';
		work.notice = '';
		productEditor = index === null ? products.length : index;
		if (index === null)
			products = [
				...products,
				{
					title: '',
					uri: '',
					images: [],
					amount: '',
					currency:
						new Intl.Locale(navigator.language).region === 'GB' ? 'GBP' : 'USD'
				}
			];
		productBaseline = JSON.stringify($state.snapshot(products));
		setProductPhotos(productEditor);
		productPhotoBaseline = JSON.stringify($state.snapshot(productPhotos));
	}
	async function cancelProduct() {
		if (work.busy || work.retry || imageTransformBusy || imageEditing) return;
		if (productPhotoOpen) {
			history.back();
			return;
		}
		if (productDirty && !(await discardChanges())) return;
		if (productSnapshot) products = productSnapshot;
		productSnapshot = null;
		productEditor = null;
		await returnTo('/settings/products');
	}
	let products = $state<Product[]>([]),
		events = $state<BusinessEvent[]>([]),
		cursor = $state<string | null>(null),
		filter = $state<'upcoming' | 'history'>('upcoming');
	let eventOpen = $state(false),
		editing = $state<BusinessEvent | null>(null),
		eventName = $state(''),
		summary = $state(''),
		venue = $state(''),
		eventUri = $state(''),
		registrationUri = $state(''),
		mode = $state('in-person'),
		status = $state('scheduled'),
		eventRoles = $state<string[]>([]);
	let timeZone = $state('UTC'),
		start = $state(''),
		end = $state(''),
		allDay = $state(false),
		eventImage = $state<BusinessImage | undefined>(undefined);
	let zones = $state<string[]>(['UTC']);
	let eventPhotos = $state<DraftMedia[]>([]);
	let eventPhotoRevision = $state(0);
	let eventPhotoBusy = $state(false),
		eventPhotoEditing = $state(false);
	const eventUploads = new WeakMap<Blob, BusinessImage>();
	let eventBaseline = $state('');
	let eventSaved = $state(false);
	$effect(() => {
		if (eventSaved && !work.busy && !work.retry) {
			eventSaved = false;
			untrack(() => void returnTo('/settings/events'));
		}
	});
	let detailsBaseline = $state('');
	let savedProducts: Product[] = [];
	function eventDraft() {
		return JSON.stringify([
			eventName,
			summary,
			venue,
			eventUri,
			registrationUri,
			mode,
			status,
			eventRoles,
			timeZone,
			start,
			end,
			allDay,
			eventImage,
			eventPhotoRevision
		]);
	}
	function detailsDraft() {
		return JSON.stringify([
			types,
			offers,
			tagline,
			hours,
			area,
			country,
			locality,
			action,
			destination
		]);
	}
	const productDirty = $derived(
		productSnapshot !== null &&
			(JSON.stringify(products) !== productBaseline ||
				JSON.stringify(productPhotos) !== productPhotoBaseline)
	);
	const eventDirty = $derived(eventOpen && eventDraft() !== eventBaseline);
	const dirty = $derived(
		productDirty ||
			eventDirty ||
			(loaded && section === 'business' && detailsDraft() !== detailsBaseline)
	);
	const discardOptions = {
		title: 'Discard changes?',
		detail: "Your edits won't be saved.",
		confirmLabel: 'Discard',
		cancelLabel: 'Keep editing',
		danger: true
	};
	function discardChanges() {
		return ask(discardOptions);
	}
	confirmNavigation(
		() =>
			dirty ||
			!!work.retry ||
			work.busy ||
			imageTransformBusy ||
			imageEditing ||
			eventPhotoBusy ||
			eventPhotoEditing,
		discardOptions
	);
	function unload(event: BeforeUnloadEvent) {
		if (
			dirty ||
			work.busy ||
			work.retry ||
			imageTransformBusy ||
			imageEditing ||
			eventPhotoBusy ||
			eventPhotoEditing
		) {
			event.preventDefault();
			event.returnValue = '';
		}
	}
	async function closeEvent() {
		if (
			work.busy ||
			work.retry ||
			imageTransformBusy ||
			eventPhotoBusy ||
			eventPhotoEditing
		)
			return;
		if (eventDirty && !(await discardChanges())) return;
		eventOpen = false;
		await returnTo('/settings/events');
	}
	const previews = new Set<string>();
	let imageTransformBusy = $state(false),
		imageEditing = $state(false);
	const productPhotoSources = new Map<string, BusinessImage>();
	const productUploads = new WeakMap<Blob, BusinessImage>();
	function setProductPhotos(index: number) {
		productPhotoSources.clear();
		productPhotos = products[index].images.map((image) => {
			const id = crypto.randomUUID();
			productPhotoSources.set(id, image);
			return {
				id,
				name: 'product-photo',
				mime: image.mime,
				alt: image.alt,
				remoteUrl: image.thumb,
				width: image.aspectRatio?.width,
				height: image.aspectRatio?.height
			};
		});
	}
	async function addProductPhotos(files: FileList | null) {
		if (
			!files?.length ||
			work.busy ||
			work.retry ||
			imageTransformBusy ||
			imageEditing
		)
			return;
		await work.run(
			async () => {
				if (productPhotos.length + files.length > 4)
					throw new InputError('Choose up to four photos for a product.');
				return Promise.all(
					Array.from(files).map((file) => prepareImage(file, file.name))
				);
			},
			(images) => {
				productPhotos = [...productPhotos, ...images];
			},
			'',
			'settings.Business.useImage'
		);
	}
	async function productPhotosForSave(): Promise<BusinessImage[] | null> {
		if (!productPhotos.length || productPhotos.length > 4)
			throw new FieldValidationError(
				'Photos',
				'Add one to four photos to each product.'
			);
		const images: BusinessImage[] = [];
		for (const media of productPhotos) {
			requireRecordText(media.alt, 'Image description', 1000, 10000);
			if (!media.blob) {
				const original = productPhotoSources.get(media.id);
				if (!original) throw new InputError('Choose the product photo again.');
				images.push({ ...original, alt: media.alt });
				continue;
			}
			const blob = media.blob;
			let uploaded = productUploads.get(blob);
			if (!uploaded) {
				await work.run(
					() => api.media.image(blob, media.mime),
					(upload) => {
						const thumb = URL.createObjectURL(blob);
						previews.add(thumb);
						uploaded = {
							cid: upload.blob.ref.$link,
							mime: upload.blob.mimeType,
							size: upload.blob.size,
							alt: media.alt,
							thumb,
							...(media.width && media.height
								? { aspectRatio: { width: media.width, height: media.height } }
								: {})
						};
						productUploads.set(blob, uploaded);
					},
					'',
					'settings.Business.useImage'
				);
				if (!uploaded) return null;
			}
			images.push({ ...uploaded, alt: media.alt });
		}
		return images;
	}

	function fromImage(value: unknown): BusinessImage | undefined {
		const image = record(value);
		if (typeof image.cid !== 'string') return;
		return {
			cid: image.cid,
			mime: text(image.mime),
			size: Number(image.size),
			alt: text(image.alt),
			thumb: text(image.thumb),
			...(image.aspectRatio
				? {
						aspectRatio: image.aspectRatio as { width: number; height: number }
					}
				: {})
		};
	}
	function applyProfile(value: unknown) {
		const profile = record(value),
			business = record(profile.business);
		cid = text(business.cid) || null;
		const bt = Array.isArray(business.businessTypes)
				? business.businessTypes
				: [],
			of = Array.isArray(business.offerings) ? business.offerings : [];
		types = bt.map(openValue).filter((v) => businessTypes.includes(v));
		unknownTypes = bt.map(openValue).filter((v) => !businessTypes.includes(v));
		offers = of.map(openValue).filter((v) => offerings.includes(v));
		unknownOffers = of.map(openValue).filter((v) => !offerings.includes(v));
		tagline = text(business.tagline);
		hours = text(business.hoursNote);
		area = text(business.serviceArea);
		country = text(record(business.location).country);
		locality = text(record(business.location).locality);
		action = text(record(business.primaryAction).type);
		destination = text(record(business.primaryAction).destination);
		if (action === 'email') destination = destination.replace(/^mailto:/, '');
		products = Array.isArray(business.products)
			? business.products.map((value) => {
					const p = record(value),
						price = record(p.price);
					return {
						title: text(p.title),
						uri: text(p.uri),
						images: Array.isArray(p.images)
							? p.images
									.map(fromImage)
									.filter((image): image is BusinessImage => !!image)
							: [],
						amount: text(price.amount),
						currency: text(price.currency)
					};
				})
			: [];
		detailsBaseline = detailsDraft();
		savedProducts = structuredClone($state.snapshot(products));
	}
	async function data() {
		const [profile, access] = await Promise.all([
			api.profiles.me(),
			api.billing.access()
		]);
		const allowed = access.givesAccess && access.effectiveTier === 'business';
		const page =
			section === 'events' && allowed
				? await api.business.ownerEvents(filter)
				: null;
		if (section === 'product' && allowed) {
			const current = record(record(profile).business).products;
			const count = Array.isArray(current) ? current.length : 0;
			if (
				productIndex !== undefined &&
				(!/^[0-3]$/.test(productIndex) || Number(productIndex) >= count)
			)
				throw new InputError(
					'Choose a product from your Products page to edit.'
				);
			if (productIndex === undefined && count >= 4)
				throw new InputError('You can feature up to four products.');
		}
		if (section === 'event' && eventRkey === '') {
			throw new InputError('Choose an event from your Events page to edit.');
		}
		const event =
			section === 'event' && allowed && eventRkey
				? await api.business.event(session.did, eventRkey)
				: null;
		return { profile, allowed, page, event };
	}
	function load() {
		return work.load(
			data,
			(value) => {
				loaded = true;
				applyProfile(value.profile);
				allowsBusiness = value.allowed;
				events = value.page?.items || [];
				cursor = value.page?.cursor || null;
				if (section === 'event' && value.allowed) editEvent(value.event);
				if (section === 'product' && value.allowed)
					editProduct(productIndex === undefined ? null : Number(productIndex));
			},
			'settings.Business.load'
		);
	}
	const actionField = $derived.by(() => {
		switch (action) {
			case 'shop':
				return { label: 'Shop URL', placeholder: 'https://example.com/shop' };
			case 'browse-patterns':
				return {
					label: 'Patterns URL',
					placeholder: 'https://example.com/patterns'
				};
			case 'request-custom-order':
				return {
					label: 'Custom order URL',
					placeholder: 'https://example.com/custom-orders'
				};
			case 'book-class':
			case 'book-appointment':
				return {
					label: 'Booking URL',
					placeholder: 'https://example.com/book'
				};
			case 'view-event-calendar':
				return {
					label: 'Event calendar URL',
					placeholder: 'https://example.com/events'
				};
			case 'email':
				return { label: 'Email address', placeholder: 'hello@example.com' };
			case 'wholesale-enquiries':
				return {
					label: 'Wholesale enquiries URL',
					placeholder: 'https://example.com/wholesale'
				};
			case 'visit-website':
				return { label: 'Website URL', placeholder: 'https://example.com' };
			default:
				return { label: 'Action URL', placeholder: 'https://example.com' };
		}
	});
	function profileBody(preserveProducts: boolean): JsonObject {
		requireRecordText(tagline, 'Tagline', 100, 1000);
		requireRecordText(hours, 'Hours', 300, 3000);
		requireRecordText(area, 'Service area', 200, 2000);
		requireRecordText(locality, 'Town or locality', 100, 1000);
		if (types.length > 5 || offers.length > 10)
			throw new FieldValidationError(
				types.length > 5 ? 'Business types' : 'Offerings',
				'Choose at most five business types and ten offerings.'
			);
		if (country && !/^[A-Z]{2}$/.test(country))
			throw new FieldValidationError(
				'Country code',
				'Use a two-letter country code, such as GB or US.'
			);
		if (locality && !country)
			throw new FieldValidationError(
				'Country code',
				'Choose a country for your locality.'
			);
		const actionDestination =
			action === 'email'
				? `mailto:${destination.trim().replace(/^mailto:/, '')}`
				: destination.trim();
		if (action) {
			if (
				action === 'email'
					? !/^mailto:[^\s@,;?%]+@[^\s@,;?%]+\.[^\s@,;?%]+$/.test(
							actionDestination
						)
					: !https(actionDestination)
			)
				throw new FieldValidationError(
					actionField.label,
					action === 'email'
						? 'Enter a valid email address.'
						: 'Use a secure HTTPS action destination.'
				);
		} else if (destination)
			throw new FieldValidationError(
				'Primary action',
				'Choose an action for this destination.'
			);
		return {
			businessTypes: [...types],
			offerings: [...offers],
			preserveUnknownCatalogValues: true,
			...(preserveProducts
				? { preserveProducts: true }
				: { products: productBody(products) }),
			...(tagline.trim() ? { tagline: tagline.trim() } : {}),
			...(hours.trim() ? { hoursNote: hours.trim() } : {}),
			...(area.trim() ? { serviceArea: area.trim() } : {}),
			...(country
				? {
						location: {
							country,
							...(locality.trim() ? { locality: locality.trim() } : {})
						}
					}
				: {}),
			...(action
				? { primaryAction: { type: action, destination: actionDestination } }
				: {})
		};
	}
	async function saveProfile(event?: SubmitEvent) {
		event?.preventDefault();
		if (work.busy || work.retry || imageTransformBusy || imageEditing) return;
		try {
			if (section === 'product' && productEditor !== null) {
				const images = await productPhotosForSave();
				if (!images) return;
				products = products.map((product, index) =>
					index === productEditor ? { ...product, images } : product
				);
			}
			const body = profileBody(section !== 'products' && section !== 'product'),
				expected = cid,
				key = newOperationKey();
			const acceptedProducts = structuredClone($state.snapshot(products));
			const acceptedDetails = detailsDraft();
			await work.run(
				() => api.business.saveProfile(body, expected, key),
				(result) => {
					cid = result.cid;
					detailsBaseline = acceptedDetails;
					savedProducts = acceptedProducts;
					if (section === 'product') {
						productSaved = true;
						productEditor = null;
						productSnapshot = null;
					}
				},
				section === 'products' || section === 'product'
					? 'Products saved.'
					: 'Business profile saved.',
				'settings.Business.saveProfile'
			);
			if (
				section === 'products' &&
				productEditor === null &&
				work.error &&
				!work.retry
			)
				products = structuredClone(savedProducts);
		} catch (error) {
			if (presentFieldErrors(error)) return;
			void reportFailure(error, 'settings.Business.saveProfile');
			work.error = userErrorMessage(error);
		}
	}
	async function deleteProfile() {
		if (
			!cid ||
			!(await ask({
				title: 'Remove business declaration?',
				detail:
					'Remove your public business details and featured products? Your subscription and events remain.',
				confirmLabel: 'Remove',
				danger: true
			}))
		)
			return;
		const expected = cid,
			key = newOperationKey();
		void work.run(
			() => api.business.deleteProfile(expected, key),
			() => {
				cid = null;
				types = [];
				offers = [];
				products = [];
				tagline = '';
				hours = '';
				area = '';
				country = '';
				locality = '';
				action = '';
				destination = '';
				detailsBaseline = detailsDraft();
				savedProducts = [];
			},
			'Business profile removed.',
			'settings.Business.deleteProfile'
		);
	}
	function move(index: number, direction: number) {
		const next = [...products];
		[next[index], next[index + direction]] = [
			next[index + direction],
			next[index]
		];
		products = next;
		if (productEditor === index) productEditor = index + direction;
		else if (productEditor === index + direction) productEditor = index;
		saveProfile();
	}
	async function removeProduct(index: number) {
		if (work.busy || work.retry) return;
		const product = products[index];
		if (
			!(await ask({
				title: 'Remove product?',
				detail: `Remove “${product.title}” from your featured products?`,
				confirmLabel: 'Remove',
				danger: true
			}))
		)
			return;
		products = products.filter((_, i) => i !== index);
		saveProfile();
	}

	function editEvent(value: BusinessEvent | null) {
		editing = value;
		work.error = '';
		work.notice = '';
		eventOpen = true;
		eventName = value?.name || '';
		summary = text(value?.summary);
		venue = text(value?.venueName);
		eventUri = text(value?.eventUri);
		registrationUri = text(value?.registrationUri);
		mode = openValue(value?.mode) || 'in-person';
		status = openValue(value?.status) || 'scheduled';
		eventRoles = Array.isArray(value?.roles) ? value.roles.map(openValue) : [];
		timeZone = text(value?.timeZone) || 'UTC';
		start = wallTime(value?.startsAt, timeZone);
		end = wallTime(value?.endsAt, timeZone);
		allDay = value?.isAllDay === true;
		if (allDay) {
			start = start.slice(0, 10);
			end = end.slice(0, 10);
		}
		eventImage = fromImage(value?.image);
		eventPhotos = eventImage
			? [
					{
						id: crypto.randomUUID(),
						name: 'event-image.jpg',
						mime: eventImage.mime,
						alt: eventImage.alt,
						remoteUrl: eventImage.thumb
					}
				]
			: [];
		eventPhotoRevision = 0;
		eventBaseline = eventDraft();
	}
	function changeEventPhotos(images: DraftMedia[]) {
		eventPhotos = images;
		eventPhotoRevision++;
	}
	async function addEventPhotos(files: FileList | null) {
		if (
			!files?.length ||
			work.busy ||
			work.retry ||
			eventPhotoBusy ||
			eventPhotoEditing
		)
			return;
		await work.run(
			async () => {
				if (eventPhotos.length + files.length > 1)
					throw new InputError('Choose one photo for an event.');
				const file = files[0];
				if (!['image/jpeg', 'image/png', 'image/webp'].includes(file.type))
					throw new InputError('Choose a JPEG, PNG or WebP image.');
				return prepareImage(file, file.name);
			},
			(media) => changeEventPhotos([media]),
			'',
			'settings.Business.useImage'
		);
	}
	async function eventPhotoForSave(): Promise<
		BusinessImage | undefined | null
	> {
		const media = eventPhotos[0];
		if (!media) return undefined;
		requireRecordText(media.alt, 'Image description', 1000, 10000);
		if (!media.blob)
			return eventImage ? { ...eventImage, alt: media.alt } : null;
		const blob = media.blob;
		const cached = eventUploads.get(blob);
		if (cached) return { ...cached, alt: media.alt };
		let result: BusinessImage | null = null;
		await work.run(
			() => api.media.image(blob, media.mime),
			(upload) => {
				result = {
					cid: upload.blob.ref.$link,
					mime: upload.blob.mimeType,
					size: upload.blob.size,
					alt: media.alt,
					...(media.width && media.height
						? { aspectRatio: { width: media.width, height: media.height } }
						: {})
				};
				eventUploads.set(blob, result);
			},
			'',
			'settings.Business.useImage'
		);
		return result;
	}
	function setAllDay(value: boolean) {
		allDay = value;
		start = value
			? start.slice(0, 10)
			: start.length === 10
				? start + 'T00:00'
				: start;
		end = value ? end.slice(0, 10) : end.length === 10 ? end + 'T00:00' : end;
		if (value && start && (!end || end <= start)) {
			const next = new Date(start + 'T00:00:00Z');
			next.setUTCDate(next.getUTCDate() + 1);
			end = next.toISOString().slice(0, 10);
		}
	}
	async function saveEvent(event: SubmitEvent) {
		event.preventDefault();
		if (
			work.busy ||
			work.retry ||
			imageTransformBusy ||
			imageEditing ||
			eventPhotoBusy ||
			eventPhotoEditing
		)
			return;
		try {
			requireRecordText(eventName, 'Event name', 200, 2000);
			requireRecordText(summary, 'Summary', 1000, 10000);
			requireRecordText(venue, 'Venue name', 200, 2000);
			if (!eventName.trim())
				throw new FieldValidationError('Event name', 'Enter an event name.');
			if (!eventRoles.length || eventRoles.length > 4)
				throw new FieldValidationError(
					'Your role',
					'Choose between one and four roles.'
				);
			const startsAt = zonedTimestamp(
					allDay ? `${start.slice(0, 10)}T00:00` : start,
					timeZone
				),
				endsAt = zonedTimestamp(
					allDay ? `${end.slice(0, 10)}T00:00` : end,
					timeZone
				);
			const duration = Date.parse(endsAt) - Date.parse(startsAt);
			if (duration <= 0 || duration > 31 * 86400000)
				throw new FieldValidationError(
					'Ends',
					'The end must be after the start, within 31 days.'
				);
			if (
				(eventUri && !https(eventUri)) ||
				(registrationUri && !https(registrationUri))
			)
				throw new FieldValidationError(
					eventUri && !https(eventUri) ? 'Event website' : 'Registration link',
					'Use secure HTTPS event and registration links.'
				);
			if (eventUri && eventUri === registrationUri)
				throw new FieldValidationError(
					'Registration link',
					'Event and registration links must be different.'
				);
			const photo = await eventPhotoForSave();
			if (photo === null) return;
			const body: JsonObject = {
				name: eventName.trim(),
				startsAt,
				endsAt,
				roles: [...eventRoles],
				mode,
				status,
				timeZone,
				isAllDay: allDay,
				...(summary.trim() ? { summary: summary.trim() } : {}),
				...(venue.trim() ? { venueName: venue.trim() } : {}),
				...(eventUri ? { eventUri } : {}),
				...(registrationUri ? { registrationUri } : {}),
				...(photo ? { image: imageBody(photo) } : {})
			};
			const original = editing,
				key = newOperationKey();
			await work.run(
				() =>
					original
						? api.business.updateEvent(
								original.did,
								original.rkey,
								body,
								original.cid,
								key
							)
						: api.business.createEvent(body, key),
				() => {
					eventOpen = false;
					eventSaved = true;
				},
				original ? 'Event updated.' : 'Event created.',
				'settings.Business.saveEvent'
			);
		} catch (error) {
			if (presentFieldErrors(error)) return;
			void reportFailure(error, 'settings.Business.saveEvent');
			work.error = userErrorMessage(error);
		}
	}
	function changeEventStatus(
		event: BusinessEvent,
		status: 'cancelled' | 'postponed'
	) {
		if (event.did !== session.did || work.busy || work.retry) return;
		const key = newOperationKey();
		void work.run(
			() =>
				api.business.updateEvent(
					event.did,
					event.rkey,
					eventStatusBody(event, status),
					event.cid,
					key
				),
			(result) => {
				events = events.flatMap((item) =>
					item.rkey !== event.rkey
						? [item]
						: filter === 'upcoming'
							? []
							: [
									{
										...event,
										cid: result.cid,
										status: { value: status, known: true }
									}
								]
				);
			},
			status === 'cancelled' ? 'Event cancelled.' : 'Event postponed.',
			'settings.Business.saveEvent'
		);
	}
	async function deleteEvent(event: BusinessEvent) {
		if (
			!(await ask({
				title: 'Delete event?',
				detail: `Delete “${event.name}”? This cannot be undone.`,
				confirmLabel: 'Delete',
				danger: true
			}))
		)
			return;
		const key = newOperationKey();
		void work.run(
			() => api.business.deleteEvent(event.did, event.rkey, event.cid, key),
			() => {
				events = events.filter((item) => item.rkey !== event.rkey);
			},
			'Event deleted.',
			'settings.Business.deleteEvent'
		);
	}
	function more() {
		const next = cursor;
		if (next && !work.busy && !work.retry)
			void work.run(
				async () => {
					loadingMore = true;
					try {
						return await api.business.ownerEvents(filter, { cursor: next });
					} finally {
						loadingMore = false;
					}
				},
				(page) => {
					events = [...events, ...page.items];
					cursor = page.cursor || null;
				},
				'',
				'settings.Business.more'
			);
	}
	onMount(() => {
		try {
			zones = ['UTC', ...Intl.supportedValuesOf('timeZone')];
		} catch {}
		load();
	});
	onDestroy(() => {
		work.dispose();
		previews.forEach((url) => URL.revokeObjectURL(url));
	});
</script>

<svelte:window onbeforeunload={unload} />

{#if section === 'product'}
	<header class="business-composer-heading page-appbar">
		<button
			type="button"
			class="close-business-composer"
			aria-label="Close product editor"
			disabled={work.busy || !!work.retry || imageTransformBusy || imageEditing}
			onclick={cancelProduct}><Icon name="close" /></button
		>
		<h1>{productIndex === undefined ? 'Add product' : 'Edit product'}</h1>
	</header>
{:else if section === 'event'}
	<header class="business-composer-heading page-appbar">
		<button
			type="button"
			class="close-business-composer"
			aria-label="Close event editor"
			disabled={work.busy ||
				!!work.retry ||
				imageTransformBusy ||
				eventPhotoBusy ||
				eventPhotoEditing}
			onclick={closeEvent}><Icon name="close" /></button
		>
		<h1>{eventRkey !== undefined ? 'Edit event' : 'Create event'}</h1>
	</header>
{:else if section === 'business'}
	<header class="business-details-header page-appbar">
		<h1>Business details</h1>
		{#if loaded && allowsBusiness}<button
				class="business-save"
				type="submit"
				form={businessFormId}
				disabled={work.loading ||
					work.busy ||
					!!work.retry ||
					detailsDraft() === detailsBaseline}
				>{work.busy ? 'Saving…' : 'Save'}</button
			>{/if}
	</header>
{:else}<h1 class="workspace-heading">{title(section)}</h1>{/if}
<Feedback {work} reload={load} />
{#if !work.loading && loaded}
	{#if !allowsBusiness}<section class="panel">
			<h2>CraftSky Business</h2>
			<p>
				Business details, featured products and events are included with
				CraftSky Business.
			</p>
			<a class="action" href="/settings/subscriptions"
				>View subscription access</a
			>
		</section>
	{:else if section === 'business'}
		<form
			id={businessFormId}
			class="business-details-form stack"
			onsubmit={saveProfile}
		>
			<fieldset
				class="form-fields"
				disabled={work.busy || !!work.retry || imageTransformBusy}
			>
				<p class="business-helper">
					These details appear on your public business profile.
				</p>
				<div class="stack">
					<MultiSelect
						label="Business types"
						placeholder="Select Business types"
						values={types}
						maximum={5}
						options={businessTypes
							.map((value) => ({ id: value, label: title(value) }))
							.sort((a, b) => a.label.localeCompare(b.label))}
						disabled={work.busy || !!work.retry || imageTransformBusy}
						onchange={(value) => {
							types = value;
						}}
					/>
					<p class="helper">Choose up to 5.</p>
					{#if unknownTypes.length}<p class="helper">
							Other business types: {unknownTypes.map(title).join(', ')}.
						</p>{/if}
				</div>
				<div class="stack">
					<MultiSelect
						label="Offerings"
						placeholder="Select Offerings"
						values={offers}
						maximum={10}
						options={offerings
							.map((value) => ({ id: value, label: title(value) }))
							.sort((a, b) => a.label.localeCompare(b.label))}
						disabled={work.busy || !!work.retry || imageTransformBusy}
						onchange={(value) => {
							offers = value;
						}}
					/>
					<p class="helper">Choose up to 10.</p>
					{#if unknownOffers.length}<p class="helper">
							Other offerings: {unknownOffers.map(title).join(', ')}.
						</p>{/if}
				</div>
				<RecordTextField
					label="Tagline"
					placeholder="Hand-dyed yarn inspired by nature"
					name="tagline"
					rows={2}
					limit={{ graphemes: 100, bytes: 1000 }}
					bind:value={tagline}
				/>
				<RecordTextField
					label="Hours"
					placeholder="Mon–Fri, 9am–5pm; Sat, 10am–2pm"
					name="hours"
					rows={4}
					limit={{ graphemes: 300, bytes: 3000 }}
					bind:value={hours}
				/>
				<RecordTextField
					label="Service area"
					placeholder="UK-wide shipping and local workshops"
					name="area"
					rows={3}
					limit={{ graphemes: 200, bytes: 2000 }}
					bind:value={area}
				/>
				<div class="business-location">
					<label
						>Country code<input
							name="country"
							maxlength="2"
							bind:value={country}
							oninput={() => (country = country.toUpperCase())}
							placeholder="GB or US"
						/></label
					>
					<RecordTextField
						label="Town or locality"
						placeholder="Bristol"
						name="locality"
						limit={{ graphemes: 100, bytes: 1000 }}
						bind:value={locality}
					/>
				</div>
				<SingleSelect
					label="Primary action"
					helperText="Choose what visitors can do on your profile."
					placeholder="Select Primary action"
					value={action}
					options={[
						{ id: '', label: 'No primary action' },
						...[
							...actions,
							...(action && !actions.includes(action) ? [action] : [])
						].map((id) => ({ id, label: title(id) }))
					]}
					onchange={(value) => {
						action = value;
						if (!value) destination = '';
					}}
				/>
				{#if action}<label
						>{actionField.label}<input
							name="destination"
							type={action === 'email' ? 'email' : 'url'}
							bind:value={destination}
							placeholder={actionField.placeholder}
						/></label
					>{/if}
				<div class="actions">
					{#if cid}<button
							type="button"
							class="danger"
							disabled={work.busy || !!work.retry || imageTransformBusy}
							onclick={deleteProfile}>Remove business declaration</button
						>{/if}
				</div>
			</fieldset>
		</form>
	{:else if section === 'products'}
		<div class="stack products-manager">
			<div class="stack">
				<p class="muted" aria-live="polite">
					{products.length} of 4 products
				</p>
				{#if !products.length}<div class="empty">
						<SettingsIcon name="products" /><strong>Products</strong>No featured
						products yet.
					</div>{/if}
				{#each products as product, index}<section class="panel product-item">
						<button
							type="button"
							class="product-summary"
							disabled={work.busy || !!work.retry}
							aria-label={`Edit ${product.title || 'new product'}`}
							onclick={() => goto(`/compose/product/edit?product=${index}`)}
							>{#if safeURL(product.images[0]?.thumb)}<img
									src={safeURL(product.images[0]?.thumb)}
									alt=""
								/>{:else}<span class="product-placeholder"
									><SettingsIcon name="products" size={32} /></span
								>{/if}<span class="tile-copy"
								><strong>{product.title || 'New product'}</strong><small
									class="product-destination">{product.uri}</small
								>{#if product.amount}<small
										>{product.currency} {product.amount}</small
									>{/if}</span
							><SettingsIcon name="chevron" size={20} /></button
						>

						<details class="product-menu" use:contextMenu>
							<summary
								class="icon-button"
								aria-label={`Product actions for ${product.title}`}
								><Icon name="more" /></summary
							>
							<div class="menu-items">
								<button
									type="button"
									disabled={work.busy || !!work.retry}
									onclick={() => goto(`/compose/product/edit?product=${index}`)}
									><Icon name="edit" />Edit product</button
								>
								{#if index > 0}<button
										type="button"
										disabled={work.busy || !!work.retry}
										onclick={() => move(index, -1)}>Move up</button
									>{/if}
								{#if index < products.length - 1}<button
										type="button"
										disabled={work.busy || !!work.retry}
										onclick={() => move(index, 1)}>Move down</button
									>{/if}
								<button
									type="button"
									class="danger"
									disabled={work.busy || !!work.retry}
									onclick={() => removeProduct(index)}
									><Icon name="delete" />Remove product</button
								>
							</div>
						</details>
					</section>{/each}
				<div class="actions product-actions">
					{#if products.length < 4}<button
							type="button"
							class="primary"
							disabled={work.busy || !!work.retry || imageTransformBusy}
							onclick={() => goto('/compose/product')}
							><SettingsIcon name="plus" />Add product</button
						>{/if}
				</div>
			</div>
		</div>
	{:else if section === 'product' && productEditor !== null}
		{@const product = products[productEditor]}
		<form
			class="product-composer-form business-composer-form"
			aria-label={productIndex === undefined ? 'Add product' : 'Edit product'}
			onsubmit={saveProfile}
		>
			<fieldset
				class="business-composer-fields"
				disabled={work.busy || !!work.retry || imageTransformBusy}
			>
				<div class="business-composer-scroll">
					<label
						>Title<input
							name="product.title"
							required
							bind:value={product.title}
						/></label
					><label
						>Destination<input
							name="product.uri"
							type="url"
							required
							bind:value={product.uri}
							placeholder="https://example.com/product"
						/></label
					>
					<section data-field="Photos">
						<PhotoAttachments
							images={productPhotos}
							maxImages={4}
							required
							backLabel="Back to product"
							disabled={work.busy || !!work.retry || imageTransformBusy}
							onadd={addProductPhotos}
							onchange={(images) => {
								productPhotos = images;
							}}
							onBusyChange={(_, busy) => {
								imageTransformBusy = busy;
							}}
							onEditingChange={(_, editing) => {
								imageEditing = editing;
							}}
							onOpenChange={(open) => {
								productPhotoOpen = open;
							}}
						/>
					</section>
					<div class="two-col">
						<label
							>Amount<input
								name="product.amount"
								inputmode="decimal"
								bind:value={product.amount}
								placeholder="24.5"
							/></label
						><SingleSelect
							label="Currency"
							value={product.currency}
							options={[
								{ id: '', label: 'None' },
								...currencyOptions.map((currency) => ({
									id: currency.code,
									label: currency.label
								}))
							]}
							onchange={(value) => (product.currency = value)}
						/>
					</div>
				</div>
				<footer class="actions business-composer-actions">
					<button
						class="primary"
						disabled={work.busy ||
							!!work.retry ||
							imageTransformBusy ||
							imageEditing}>Save product</button
					>
				</footer>
			</fieldset>
		</form>
	{:else if section === 'events'}
		<section class="events-manager">
			<div class="settings-tabs" use:dragScroll aria-label="Events">
				{#each [['upcoming', 'Upcoming'], ['history', 'History']] as [value, label]}<button
						aria-pressed={filter === value}
						disabled={work.busy || !!work.retry}
						onclick={() => {
							filter = value as typeof filter;
							load();
						}}><span>{label}</span></button
					>{/each}
			</div>
			{#each events as event}<div class="managed-event">
					<BusinessEventCard {event}>
						{#snippet actions()}
							<details class="event-menu" use:contextMenu>
								<summary
									class="icon-button"
									aria-label={`Actions for ${event.name}`}
									><Icon name="more" /></summary
								>
								<div class="menu-items">
									<button
										disabled={work.busy || !!work.retry}
										onclick={() =>
											void work.run(
												() =>
													navigator.clipboard.writeText(
														businessEventLink(event.did, event.rkey)
													),
												undefined,
												'Link copied.'
											)}><Icon name="link" />Copy link</button
									>
									<a
										inert={work.busy || !!work.retry}
										href={'/messages/new?link=' +
											encodeURIComponent(
												businessEventLink(event.did, event.rkey)
											)}><Icon name="send" />Send via chat</a
									>
									<div class="menu-divider"></div>
									<button
										disabled={work.busy || !!work.retry}
										onclick={() =>
											goto(
												'/compose/event/edit?event=' +
													encodeURIComponent(event.rkey)
											)}><Icon name="edit" />Edit</button
									>
									{#if event.status.value !== 'postponed'}<button
											disabled={work.busy || !!work.retry}
											onclick={() => changeEventStatus(event, 'postponed')}
											><Icon name="clock" />Postpone event</button
										>{/if}
									{#if event.status.value !== 'cancelled'}<button
											disabled={work.busy || !!work.retry}
											onclick={() => changeEventStatus(event, 'cancelled')}
											><Icon name="close" />Cancel event</button
										>{/if}
									<button
										class="danger"
										disabled={work.busy || !!work.retry}
										onclick={() => deleteEvent(event)}
										><Icon name="delete" />Delete</button
									>
								</div>
							</details>
						{/snippet}
					</BusinessEventCard>
					{#if event.publicSuppressionReasons?.length}<p class="notice">
							Public visibility: {event.publicSuppressionReasons
								.map((value) => title(text(value)))
								.join(', ')}
						</p>{/if}
				</div>{:else}<p class="empty">
					<SettingsIcon name="events" /><strong
						>{filter === 'upcoming' ? 'Upcoming' : 'History'}</strong
					>{filter === 'upcoming'
						? 'No upcoming events yet.'
						: 'No event history yet.'}
				</p>{/each}{#if cursor}<LoadMore
					loading={loadingMore}
					disabled={work.busy || !!work.retry}
					label="Load more events"
					loadingLabel="Loading more events"
					onclick={more}
				/>{/if}
			<div class="actions events-toolbar">
				<button
					class="primary"
					disabled={work.busy || !!work.retry || imageTransformBusy}
					onclick={() => goto('/compose/event')}
					><SettingsIcon name="plus" />Create event</button
				>
			</div>
		</section>
	{:else if section === 'event' && eventOpen}
		<form
			class="event-composer-form business-composer-form"
			aria-label={eventRkey !== undefined ? 'Edit event' : 'Create event'}
			onsubmit={saveEvent}
		>
			<fieldset
				class="business-composer-fields"
				disabled={work.busy || !!work.retry || imageTransformBusy}
			>
				<div class="business-composer-scroll">
					<RecordTextField
						label="Event name"
						name="eventName"
						required
						bind:value={eventName}
						limit={{ graphemes: 200, bytes: 2000 }}
						placeholder="Autumn yarn fair"
					/>
					<label class="switch-row"
						><span class="tile-copy">All-day event</span><input
							type="checkbox"
							role="switch"
							checked={allDay}
							onchange={(event) => setAllDay(event.currentTarget.checked)}
						/></label
					>
					<div class="two-col">
						<label
							><span>Start <small aria-hidden="true">required</small></span
							><span class="event-boundary" class:boundary-empty={!start}
								><input
									name="startsAt"
									aria-label="Start"
									type={allDay ? 'date' : 'datetime-local'}
									required
									placeholder={allDay ? 'Select date' : 'Select date and time'}
									bind:value={start}
								/>{#if !start}<span class="boundary-hint" aria-hidden="true"
										>{allDay ? 'Select date' : 'Select date and time'}</span
									>{/if}</span
							></label
						><label
							><span>End <small aria-hidden="true">required</small></span><span
								class="event-boundary"
								class:boundary-empty={!end}
								><input
									name="endsAt"
									aria-label="End"
									type={allDay ? 'date' : 'datetime-local'}
									required
									placeholder={allDay ? 'Select date' : 'Select date and time'}
									bind:value={end}
								/>{#if !end}<span class="boundary-hint" aria-hidden="true"
										>{allDay ? 'Select date' : 'Select date and time'}</span
									>{/if}</span
							></label
						>
					</div>
					{#if allDay}<p class="helper">
							End is the first date after the event, in the selected timezone.
						</p>{/if}
					<SingleSelect
						label="Timezone"
						placeholder="Select Timezone"
						value={timeZone}
						required
						showRequired={false}
						options={zones.map((id) => ({ id, label: id }))}
						onchange={(value) => (timeZone = value)}
					/>
					<MultiSelect
						label="Your role"
						required
						placeholder="Select roles"
						helperText="Choose up to 4 roles."
						values={eventRoles}
						maximum={4}
						options={[...new Set([...roles, ...eventRoles])].map((id) => ({
							id,
							label: title(id)
						}))}
						onchange={(values) => {
							eventRoles = values;
						}}
					/>
					<div class="two-col">
						<SingleSelect
							label="Attendance mode"
							placeholder="Select Attendance mode"
							value={mode}
							required
							showRequired={false}
							options={[
								...new Set(['in-person', 'online', 'hybrid', mode])
							].map((id) => ({
								id,
								label: id === 'in-person' ? 'In person' : title(id)
							}))}
							onchange={(value) => (mode = value)}
						/><SingleSelect
							label="Status"
							placeholder="Select Status"
							value={status}
							required
							showRequired={false}
							options={[
								...new Set(['scheduled', 'cancelled', 'postponed', status])
							].map((id) => ({ id, label: title(id) }))}
							onchange={(value) => (status = value)}
						/>
					</div>
					<RecordTextField
						label="Summary"
						name="summary"
						bind:value={summary}
						rows={4}
						limit={{ graphemes: 1000, bytes: 10000 }}
						placeholder="Meet local makers, browse handmade goods and join a workshop."
					/>
					<RecordTextField
						label="Venue"
						name="venue"
						bind:value={venue}
						limit={{ graphemes: 200, bytes: 2000 }}
						placeholder="Bristol Craft Hall"
					/>
					<label
						>Event link<input
							name="eventUri"
							type="url"
							bind:value={eventUri}
							placeholder="https://your-business.com/events/autumn-fair"
						/></label
					><label
						>Registration link<input
							name="registrationUri"
							type="url"
							bind:value={registrationUri}
							placeholder="https://your-business.com/events/autumn-fair/book"
						/></label
					>
					<section class="event-photos" data-field="Photos">
						<PhotoAttachments
							images={eventPhotos}
							maxImages={1}
							backLabel="Back to event"
							disabled={work.busy || !!work.retry || eventPhotoBusy}
							onadd={addEventPhotos}
							onchange={changeEventPhotos}
							onBusyChange={(_, busy) => (eventPhotoBusy = busy)}
							onEditingChange={(_, editing) => (eventPhotoEditing = editing)}
						/>
					</section>
				</div>
				<footer class="actions business-composer-actions">
					<button
						class="primary"
						disabled={work.busy ||
							!!work.retry ||
							imageTransformBusy ||
							eventPhotoBusy ||
							eventPhotoEditing}
						>{editing ? 'Save event' : 'Create event'}</button
					>
				</footer>
			</fieldset>
		</form>
	{/if}
{/if}

<style>
	.business-details-header {
		margin: 0 -16px 16px;
		padding-left: 72px;
	}
	:global(.settings-workspace) .business-details-header h1 {
		position: static;
		flex: 1;
		margin: 0;
		padding: 0;
		border: 0;
		min-width: 0;
	}
	:global(.settings-workspace) .business-save {
		border: 0;
		border-radius: 0;
		background: transparent;
		box-shadow: none;
		padding: 8px 12px;
		font-size: 16px;
		font-weight: 700;
		color: var(--primary);
	}
	.business-details-form :global(.form-fields) {
		gap: 24px;
	}
	.business-details-form :global(.stack) {
		gap: 8px;
	}
	:global(.settings-workspace) .business-helper {
		margin: 0;
		font-size: 16px;
	}
	.business-location {
		display: grid;
		gap: 24px;
	}
	.business-composer-form,
	.business-composer-fields {
		display: flex;
		flex-direction: column;
		flex: 1;
		min-height: 0;
		min-width: 0;
		border: 0;
		margin: 0;
		padding: 0;
	}
	.business-composer-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
		overscroll-behavior: contain;
		display: flex;
		flex-direction: column;
		gap: 24px;
		padding: 16px;
	}
	.business-composer-heading {
		justify-content: flex-start;
	}
	.business-composer-heading h1 {
		flex: 1;
		min-width: 0;
	}
	:global(.settings-workspace) .close-business-composer {
		border: 0;
		background: transparent;
		box-shadow: none;
		padding: 0;
		width: 40px;
		min-height: 40px;
		margin-left: -8px;
	}
	.event-composer-form :global(label > span > small) {
		color: var(--ink-2);
		font-size: 12px;
		letter-spacing: 1.68px;
		margin-left: 8px;
	}
	.business-composer-actions {
		flex: none;
		border: 0;
		padding: 12px 16px calc(16px + env(safe-area-inset-bottom));
		margin: 0;
		background: var(--paper);
	}
	.business-composer-actions .primary {
		flex: 1;
	}

	.event-boundary {
		position: relative;
	}
	.event-boundary input {
		width: 100%;
	}
	.event-boundary.boundary-empty:not(:focus-within) input {
		color: transparent;
	}
	.boundary-hint {
		position: absolute;
		left: 16px;
		top: 50%;
		transform: translateY(-50%);
		pointer-events: none;
		color: var(--ink-2);
		font-weight: 400;
	}
	.event-boundary:focus-within .boundary-hint {
		display: none;
	}
	@media (max-width: 719px) {
		.business-composer-form .two-col {
			grid-template-columns: 1fr;
			gap: 24px;
		}
	}
	.product-item,
	.managed-event {
		position: relative;
	}

	.product-menu {
		position: absolute;
		right: 12px;
		top: 12px;
	}
	.event-menu {
		position: relative;
		background: var(--paper);
		border-radius: 50%;
	}
	.managed-event {
		margin-bottom: 16px;
	}
	.product-summary :global(.tile-copy) {
		min-width: 0;
	}

	.product-actions {
		position: sticky;
		bottom: 80px;
		justify-content: flex-end;
		padding-top: 16px;
	}
	.product-destination {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	:global(.settings-workspace) .product-item {
		padding: 12px 56px 12px 12px;
	}
	:global(.settings-workspace) .product-summary {
		width: 100%;
		border: 0;
		border-radius: 0;
		background: transparent;
		text-align: left;
		font-weight: 400;
		justify-content: flex-start;
		padding: 0;
		display: flex;
		align-items: center;
		gap: 12px;
		cursor: pointer;
		list-style: none;
		min-height: 88px;
	}
	.product-summary::-webkit-details-marker {
		display: none;
	}
	.product-summary img,
	.product-placeholder {
		width: 88px;
		height: 88px;
		object-fit: cover;
		border-radius: 6px;
		flex-shrink: 0;
	}
	.product-placeholder {
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--paper-2);
		color: var(--ink-3);
	}
	.product-summary strong {
		font-size: 16px;
		font-weight: 600;
	}
	.product-summary small {
		color: var(--ink-2);
		font-size: 14px;
	}
	.events-manager {
		display: flex;
		flex-direction: column;
		min-height: calc(100dvh - 180px);
		gap: 16px;
	}
	.events-manager > .empty {
		flex: 1;
		min-height: 240px;
	}
	.events-toolbar {
		position: sticky;
		bottom: 80px;
		padding-top: 16px;
		justify-content: flex-end;
	}
</style>
