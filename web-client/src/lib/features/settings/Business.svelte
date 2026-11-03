<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import MediaEditor from '../composer/MediaEditor.svelte';
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
		https,
		zonedTimestamp,
		wallTime,
		dateTime,
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
		section
	}: { api: CraftskyAPI; session: Session; section: string } = $props();
	const work = new Work();
	let loaded = $state(false);
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
	let productDialog = $state<HTMLDialogElement>();
	let productSnapshot = $state<Product[] | null>(null);
	$effect(() => {
		if (productEditor !== null)
			void tick().then(() => productDialog?.showModal());
	});
	function editProduct(index: number | null) {
		productSnapshot = structuredClone($state.snapshot(products));
		work.error = '';
		work.notice = '';
		productEditor = index === null ? products.length : index;
		if (index === null)
			products = [
				...products,
				{ title: '', uri: '', amount: '', currency: '' }
			];
	}
	function cancelProduct() {
		if (work.busy || work.retry || imageEditor || imageTransformBusy) return;
		if (productSnapshot) products = productSnapshot;
		productSnapshot = null;
		productEditor = null;
	}
	let products = $state<Product[]>([]),
		events = $state<BusinessEvent[]>([]),
		cursor = $state<string | null>(null),
		filter = $state<'upcoming' | 'history'>('upcoming');
	let eventDialog = $state<HTMLDialogElement>();
	$effect(() => {
		if (eventOpen) void tick().then(() => eventDialog?.showModal());
	});
	let eventOpen = $state(false),
		editing = $state<BusinessEvent | null>(null),
		eventName = $state(''),
		summary = $state(''),
		venue = $state(''),
		eventUri = $state(''),
		registrationUri = $state(''),
		mode = $state('in-person'),
		status = $state('scheduled'),
		eventRoles = $state<string[]>(['organizer']);
	let timeZone = $state('UTC'),
		start = $state(''),
		end = $state(''),
		allDay = $state(false),
		eventImage = $state<BusinessImage | undefined>(undefined);
	let zones = $state<string[]>(['UTC']);
	const previews = new Set<string>();
	let imageDialog = $state<HTMLDialogElement>();
	let imageEditor = $state<{
		target: number | null;
		media: DraftMedia;
		original?: BusinessImage;
	} | null>(null);
	let imageTransformBusy = $state(false),
		imageEditing = $state(false);
	async function openImage(
		target: number | null,
		media: DraftMedia,
		original?: BusinessImage
	) {
		imageEditor = { target, media, original };
		imageTransformBusy = false;
		imageEditing = false;
		work.error = '';
		await tick();
		imageDialog?.showModal();
	}
	function cancelImage() {
		if (work.busy || imageTransformBusy) return;
		imageDialog?.close();
		imageEditor = null;
	}
	function editImage(target: number | null, image: BusinessImage) {
		void openImage(
			target,
			{
				id: crypto.randomUUID(),
				name: 'business-image.jpg',
				mime: image.mime,
				alt: image.alt,
				remoteUrl: image.thumb
			},
			image
		);
	}
	function useImage() {
		if (!imageEditor || imageTransformBusy || imageEditing || work.busy) return;
		const selection = imageEditor,
			media = $state.snapshot(selection.media);
		void work.run(
			async () => {
				if (!media.blob) {
					if (!selection.original) throw new Error('Choose an image first.');
					return { ...selection.original, alt: media.alt };
				}
				const prepared = await prepareImage(media.blob, media.name);
				const upload = await api.media.image(prepared.blob!, prepared.mime);
				return {
					cid: upload.blob.ref.$link,
					mime: upload.blob.mimeType,
					size: upload.blob.size,
					alt: media.alt,
					thumb: URL.createObjectURL(prepared.blob!),
					...(prepared.width && prepared.height
						? {
								aspectRatio: { width: prepared.width, height: prepared.height }
							}
						: {})
				};
			},
			(image) => {
				if (image.thumb?.startsWith('blob:')) previews.add(image.thumb);
				if (selection.target === null) eventImage = image;
				else
					products = products.map((product, index) =>
						index === selection.target ? { ...product, image } : product
					);
				imageDialog?.close();
				imageEditor = null;
			},
			'Image ready. Save to update your business profile.'
		);
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
		products = Array.isArray(business.products)
			? business.products.map((value) => {
					const p = record(value),
						price = record(p.price);
					return {
						title: text(p.title),
						uri: text(p.uri),
						image: fromImage(p.image),
						amount: text(price.amount),
						currency: text(price.currency)
					};
				})
			: [];
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
		return { profile, allowed, page };
	}
	function load() {
		return work.load(data, (value) => {
			loaded = true;
			applyProfile(value.profile);
			allowsBusiness = value.allowed;
			events = value.page?.items || [];
			cursor = value.page?.cursor || null;
		});
	}
	function profileBody(preserveProducts: boolean): JsonObject {
		requireRecordText(tagline, 'Tagline', 100, 1000);
		requireRecordText(hours, 'Opening hours', 300, 3000);
		requireRecordText(area, 'Service area', 200, 2000);
		requireRecordText(locality, 'Town or locality', 100, 1000);
		if (types.length > 5 || offers.length > 10)
			throw new Error('Choose at most five business types and ten offerings.');
		if (country && !/^[A-Z]{2}$/.test(country))
			throw new Error('Use a two-letter country code, such as GB or US.');
		if (locality && !country)
			throw new Error('Choose a country for your locality.');
		if (action) {
			if (
				action === 'email'
					? !/^mailto:[^\s@,;?%]+@[^\s@,;?%]+\.[^\s@,;?%]+$/.test(destination)
					: !https(destination)
			)
				throw new Error(
					action === 'email'
						? 'Use a mailto: email address.'
						: 'Use a secure HTTPS action destination.'
				);
		} else if (destination)
			throw new Error('Choose an action for this destination.');
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
			...(action ? { primaryAction: { type: action, destination } } : {})
		};
	}
	function saveProfile(event?: SubmitEvent) {
		event?.preventDefault();
		if (imageEditor || imageTransformBusy || imageEditing) return;
		try {
			const body = profileBody(section !== 'products'),
				expected = cid,
				key = newOperationKey();
			void work.run(
				() => api.business.saveProfile(body, expected, key),
				(result) => {
					cid = result.cid;
					if (section === 'products') {
						productEditor = null;
						productSnapshot = null;
					}
				},
				section === 'products' ? 'Products saved.' : 'Business profile saved.'
			);
		} catch (error) {
			work.error =
				error instanceof Error ? error.message : 'Check your business details.';
		}
	}
	function deleteProfile() {
		if (
			!cid ||
			!confirm(
				'Remove your business declaration and return to a regular account? Your posts remain.'
			)
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
			},
			'Business profile removed.'
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
	async function upload(event: Event, index: number | null) {
		const file = (event.currentTarget as HTMLInputElement).files?.[0];
		if (!file) return;
		if (
			!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) ||
			file.size > 40000000
		) {
			work.error = 'Choose a JPEG, PNG or WebP image smaller than 40 MB.';
			return;
		}
		await openImage(index, {
			id: crypto.randomUUID(),
			name: file.name,
			mime: file.type,
			blob: file,
			alt: (index === null ? eventImage : products[index]?.image)?.alt || ''
		});
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
		eventRoles = Array.isArray(value?.roles)
			? value.roles.map(openValue)
			: ['organizer'];
		timeZone =
			text(value?.timeZone) ||
			Intl.DateTimeFormat().resolvedOptions().timeZone ||
			'UTC';
		start = wallTime(value?.startsAt, timeZone);
		end = wallTime(value?.endsAt, timeZone);
		allDay = value?.isAllDay === true;
		if (allDay) {
			start = start.slice(0, 10);
			end = end.slice(0, 10);
		}
		eventImage = fromImage(value?.image);
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
	function saveEvent(event: SubmitEvent) {
		event.preventDefault();
		if (imageEditor || imageTransformBusy || imageEditing) return;
		try {
			requireRecordText(eventName, 'Event name', 200, 2000);
			requireRecordText(summary, 'Summary', 1000, 10000);
			requireRecordText(venue, 'Venue name', 200, 2000);
			if (!eventName.trim()) throw new Error('Enter an event name.');
			if (!eventRoles.length || eventRoles.length > 4)
				throw new Error('Choose between one and four roles.');
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
				throw new Error('The end must be after the start, within 31 days.');
			if (
				(eventUri && !https(eventUri)) ||
				(registrationUri && !https(registrationUri))
			)
				throw new Error('Use secure HTTPS event and registration links.');
			if (eventUri && eventUri === registrationUri)
				throw new Error('Event and registration links must be different.');
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
				...(eventImage ? { image: imageBody(eventImage) } : {})
			};
			const original = editing,
				key = newOperationKey();
			void work.run(
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
				(result) => {
					const updated = {
						...body,
						...result,
						did: session.did,
						rkey:
							result.rkey ||
							original?.rkey ||
							result.uri?.split('/').at(-1) ||
							'',
						roles: eventRoles.map((value) => ({ value, known: true })),
						mode: { value: mode, known: true },
						status: { value: status, known: true },
						image: eventImage
					} as unknown as BusinessEvent;
					events = original
						? events.map((item) =>
								item.rkey === original.rkey ? updated : item
							)
						: [updated, ...events];
					eventOpen = false;
					editing = null;
				},
				original ? 'Event updated.' : 'Event created.'
			);
		} catch (error) {
			work.error =
				error instanceof Error ? error.message : 'Check your event details.';
		}
	}
	function deleteEvent(event: BusinessEvent) {
		if (!confirm(`Delete “${event.name}”? This cannot be undone.`)) return;
		const key = newOperationKey();
		void work.run(
			() => api.business.deleteEvent(event.did, event.rkey, event.cid, key),
			() => {
				events = events.filter((item) => item.rkey !== event.rkey);
			},
			'Event deleted.'
		);
	}
	function more() {
		const next = cursor;
		if (next)
			void work.run(
				() => api.business.ownerEvents(filter, { cursor: next }),
				(page) => {
					events = [...events, ...page.items];
					cursor = page.cursor || null;
				},
				''
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

<h1 class="workspace-heading">
	{section === 'business' ? 'Business profile' : title(section)}
</h1>
<Feedback {work} reload={load} />
{#if !work.loading && loaded}
	{#if section !== 'business' && !allowsBusiness}<section class="panel">
			<h2>CraftSky Business</h2>
			<p>
				{section === 'products' ? 'Featured products' : 'Business events'} are included
				with CraftSky Business. Purchase or change your subscription in the mobile
				app.
			</p>
			<a class="action" href="/app/settings/subscriptions"
				>View subscription access</a
			>
		</section>
	{:else if section === 'business'}
		<form class="panel stack" onsubmit={saveProfile}>
			<fieldset
				class="form-fields"
				disabled={work.busy ||
					!!work.retry ||
					!!imageEditor ||
					imageTransformBusy}
			>
				<h2>{cid ? 'Your business details' : 'Declare your craft business'}</h2>
				<p class="muted">
					A business declaration makes commercial activity transparent to the
					community. Your business account is clearly labelled.
				</p>
				<div class="stack">
					<MultiSelect
						label="Business types"
						values={types}
						maximum={5}
						options={businessTypes
							.map((value) => ({ id: value, label: title(value) }))
							.sort((a, b) => a.label.localeCompare(b.label))}
						disabled={work.busy ||
							!!work.retry ||
							!!imageEditor ||
							imageTransformBusy}
						onchange={(value) => {
							types = value;
						}}
					/>
					<p class="helper">Choose up to five business types.</p>
					{#if unknownTypes.length}<p class="helper">
							Other business types: {unknownTypes.map(title).join(', ')}.
						</p>{/if}
				</div>
				<div class="stack">
					<MultiSelect
						label="What you offer"
						values={offers}
						maximum={10}
						options={offerings
							.map((value) => ({ id: value, label: title(value) }))
							.sort((a, b) => a.label.localeCompare(b.label))}
						disabled={work.busy ||
							!!work.retry ||
							!!imageEditor ||
							imageTransformBusy}
						onchange={(value) => {
							offers = value;
						}}
					/>
					<p class="helper">Choose up to ten offerings.</p>
					{#if unknownOffers.length}<p class="helper">
							Other offerings: {unknownOffers.map(title).join(', ')}.
						</p>{/if}
				</div>
				<label>Tagline<textarea rows="2" bind:value={tagline}></textarea></label
				><label
					>Opening hours<textarea rows="4" bind:value={hours}></textarea></label
				><label
					>Service area<textarea rows="3" bind:value={area}></textarea></label
				>
				<div class="two-col">
					<label
						>Country code<input
							maxlength="2"
							bind:value={country}
							oninput={() => (country = country.toUpperCase())}
							placeholder="GB"
						/></label
					><label>Town or locality<input bind:value={locality} /></label>
				</div>
				<label
					>Primary action<select bind:value={action}
						><option value="">None</option>{#each actions as value}<option
								{value}>{title(value)}</option
							>{/each}{#if action && !actions.includes(action)}<option
								value={action}>{title(action)}</option
							>{/if}</select
					></label
				><label
					>Action destination<input
						bind:value={destination}
						placeholder={action === 'email'
							? 'mailto:hello@example.com'
							: 'https://example.com'}
					/></label
				>
				<div class="actions">
					<button
						class="primary"
						disabled={work.busy ||
							!!work.retry ||
							!!imageEditor ||
							imageTransformBusy}
						>{cid ? 'Save business profile' : 'Create business profile'}</button
					>{#if cid}<button
							type="button"
							class="danger"
							disabled={work.busy ||
								!!work.retry ||
								!!imageEditor ||
								imageTransformBusy}
							onclick={deleteProfile}>Remove business declaration</button
						>{/if}
				</div>
			</fieldset>
		</form>
	{:else if section === 'products'}
		{#if !cid}<section class="panel">
				<p>Create your business declaration before adding products.</p>
				<a class="action" href="/app/settings/business"
					>Set up business profile</a
				>
			</section>{:else}<form class="stack" onsubmit={saveProfile}>
				<fieldset
					class="form-fields"
					disabled={work.busy ||
						!!work.retry ||
						!!imageEditor ||
						imageTransformBusy}
				>
					<p class="muted" aria-live="polite">
						{products.length} of 4 products
					</p>
					{#if !products.length}<div class="empty">
							<SettingsIcon name="products" /><strong>Products</strong>No
							featured products yet.
						</div>{/if}
					{#each products as product, index}<section class="panel product-item">
							<button
								type="button"
								class="product-summary"
								disabled={work.busy || !!work.retry}
								aria-label={`Edit ${product.title || 'new product'}`}
								onclick={() => editProduct(index)}
								>{#if safeURL(product.image?.thumb)}<img
										src={safeURL(product.image?.thumb)}
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
							{#if productEditor === index}<dialog
									class="settings-dialog product-dialog"
									bind:this={productDialog}
									oncancel={(event) => {
										event.preventDefault();
										cancelProduct();
									}}
								>
									<Feedback {work} />
									<div class="stack product-editor">
										<div class="row">
											<h2>
												{productSnapshot && index < productSnapshot.length
													? 'Edit product'
													: 'Add product'}
											</h2>
											<div class="actions">
												<button
													type="button"
													disabled={work.busy || index === 0}
													aria-label={`Move product ${index + 1} up`}
													onclick={() => move(index, -1)}>↑</button
												><button
													type="button"
													disabled={work.busy || index === products.length - 1}
													aria-label={`Move product ${index + 1} down`}
													onclick={() => move(index, 1)}>↓</button
												><button
													type="button"
													class="danger"
													disabled={work.busy}
													onclick={() => {
														products = products.filter((_, i) => i !== index);
														productEditor = null;
														saveProfile();
													}}>Remove</button
												>
											</div>
										</div>
										<label
											>Title<input required bind:value={product.title} /></label
										><label
											>Destination link<input
												type="url"
												required
												bind:value={product.uri}
												placeholder="https://…"
											/></label
										>{#if product.image?.thumb}<img
												class="preview-image"
												src={product.image.thumb}
												alt={product.image.alt}
											/>{/if}<label
											>{product.image ? 'Replace image' : 'Product image'}<input
												type="file"
												accept="image/jpeg,image/png,image/webp"
												disabled={work.busy}
												onchange={(event) => upload(event, index)}
											/></label
										>{#if product.image}<button
												type="button"
												disabled={work.busy}
												onclick={() => editImage(index, product.image!)}
												>Edit image</button
											><label
												>Image alternative text<textarea
													bind:value={product.image.alt}></textarea></label
											>{/if}
										<div class="two-col">
											<label
												>Price · optional<input
													inputmode="decimal"
													bind:value={product.amount}
													placeholder="24.5"
												/></label
											><label
												>Currency<select bind:value={product.currency}
													><option value="">None</option
													>{#each currencyOptions as currency}<option
															value={currency.code}>{currency.label}</option
														>{/each}</select
												></label
											>
										</div>
										<div class="actions">
											<button
												class="primary"
												disabled={work.busy || !!work.retry}
												>Save product</button
											><button
												type="button"
												disabled={work.busy || !!work.retry}
												onclick={cancelProduct}>Cancel</button
											>
										</div>
									</div>
								</dialog>{/if}
						</section>{/each}
					<div class="actions product-actions">
						{#if products.length < 4}<button
								type="button"
								class="primary"
								disabled={work.busy ||
									!!work.retry ||
									!!imageEditor ||
									imageTransformBusy}
								onclick={() => editProduct(null)}
								><SettingsIcon name="plus" />Add product</button
							>{/if}
					</div>
				</fieldset>
			</form>{/if}
	{:else if section === 'events'}
		<section class="events-manager">
			<div class="actions events-toolbar">
				<button
					class="primary"
					disabled={work.busy ||
						!!work.retry ||
						!!imageEditor ||
						imageTransformBusy}
					onclick={() => editEvent(null)}>Create event</button
				>
			</div>
			<div class="settings-tabs" aria-label="Events">
				{#each [['upcoming', 'Upcoming'], ['history', 'History']] as [value, label]}<button
						aria-pressed={filter === value}
						disabled={work.busy || !!work.retry}
						onclick={() => {
							filter = value as typeof filter;
							load();
						}}>{label}</button
					>{/each}
			</div>
			{#each events as event}<article class="panel event-item">
					{#if safeURL(event.image?.thumb)}<img
							class="event-cover"
							src={safeURL(event.image?.thumb)}
							alt={event.image?.alt || ''}
							loading="lazy"
						/>{/if}
					<h3>
						<a
							href={`/app/events/${encodeURIComponent(event.did)}/${encodeURIComponent(event.rkey)}`}
							>{event.name}</a
						>
					</h3>
					<p>{dateTime(event.startsAt)} – {dateTime(event.endsAt)}</p>
					<p class="muted">
						{title(openValue(event.status))} · {title(
							openValue(event.mode)
						)}{event.timeZone ? ` · ${event.timeZone}` : ''}
					</p>
					{#if event.summary}<p>
							{text(event.summary)}
						</p>{/if}{#if event.venueName}<p>
							{text(event.venueName)}
						</p>{/if}{#if Array.isArray(event.publicSuppressionReasons) && event.publicSuppressionReasons.length}<p
							class="notice"
						>
							Public visibility: {event.publicSuppressionReasons
								.map((value) => title(text(value)))
								.join(', ')}
						</p>{/if}
					<div class="actions">
						{#if safeURL(text(event.eventUri))}<a
								class="action"
								href={safeURL(text(event.eventUri))}
								target="_blank"
								rel="noreferrer">Event website ↗</a
							>{/if}<button
							disabled={work.busy ||
								!!work.retry ||
								!!imageEditor ||
								imageTransformBusy}
							onclick={() => editEvent(event)}>Edit</button
						><button
							class="danger"
							disabled={work.busy ||
								!!work.retry ||
								!!imageEditor ||
								imageTransformBusy}
							onclick={() => deleteEvent(event)}>Delete</button
						>
					</div>
				</article>{:else}<p class="empty">
					<SettingsIcon name="events" /><strong>Events</strong>{filter ===
					'upcoming'
						? 'No upcoming events yet.'
						: 'No event history yet.'}
				</p>{/each}{#if cursor}<button disabled={work.busy} onclick={more}
					>Load more events</button
				>{/if}
		</section>
		{#if eventOpen}<dialog
				class="settings-dialog event-dialog"
				bind:this={eventDialog}
				oncancel={(event) => {
					if (work.busy || !!imageEditor || imageTransformBusy)
						event.preventDefault();
					else eventOpen = false;
				}}
			>
				<form class="stack" onsubmit={saveEvent}>
					<Feedback {work} />
					<fieldset
						class="form-fields"
						disabled={work.busy ||
							!!work.retry ||
							!!imageEditor ||
							imageTransformBusy}
					>
						<h2>{editing ? 'Edit event' : 'Create event'}</h2>
						<label>Event name<input required bind:value={eventName} /></label>
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
								>Starts<input
									type={allDay ? 'date' : 'datetime-local'}
									required
									bind:value={start}
								/></label
							><label
								>Ends<input
									type={allDay ? 'date' : 'datetime-local'}
									required
									bind:value={end}
								/></label
							>
						</div>
						{#if allDay}<p class="helper">
								All-day events end on the following date, in the selected time
								zone.
							</p>{/if}
						<label
							>Time zone<select bind:value={timeZone}
								>{#each zones as value}<option {value}>{value}</option
									>{/each}</select
							></label
						>
						<MultiSelect
							label="Your roles"
							values={eventRoles}
							maximum={4}
							options={[
								...roles,
								...eventRoles.filter((value) => !roles.includes(value))
							]
								.map((value) => ({ id: value, label: title(value) }))
								.sort((a, b) => a.label.localeCompare(b.label))}
							disabled={work.busy ||
								!!work.retry ||
								!!imageEditor ||
								imageTransformBusy}
							onchange={(value) => {
								eventRoles = value;
							}}
						/>
						<p class="helper">Choose up to four roles.</p>
						<div class="two-col">
							<label
								>Mode<select bind:value={mode}
									>{#each ['in-person', 'online', 'hybrid', ...(!['in-person', 'online', 'hybrid'].includes(mode) ? [mode] : [])] as value}<option
											{value}>{title(value)}</option
										>{/each}</select
								></label
							><label
								>Status<select bind:value={status}
									>{#each ['scheduled', 'cancelled', 'postponed', ...(!['scheduled', 'cancelled', 'postponed'].includes(status) ? [status] : [])] as value}<option
											{value}>{title(value)}</option
										>{/each}</select
								></label
							>
						</div>
						<label>Summary<textarea bind:value={summary}></textarea></label>
						<label>Venue name<input bind:value={venue} /></label><label
							>Event website<input
								type="url"
								bind:value={eventUri}
								placeholder="https://…"
							/></label
						><label
							>Registration link<input
								type="url"
								bind:value={registrationUri}
								placeholder="https://…"
							/></label
						>{#if eventImage?.thumb}<img
								class="preview-image"
								src={eventImage.thumb}
								alt={eventImage.alt}
							/>{/if}<label
							>Event image · optional<input
								type="file"
								accept="image/jpeg,image/png,image/webp"
								disabled={work.busy}
								onchange={(event) => upload(event, null)}
							/></label
						>{#if eventImage}<button
								type="button"
								disabled={work.busy}
								onclick={() => editImage(null, eventImage!)}>Edit image</button
							><label
								>Image alternative text<textarea bind:value={eventImage.alt}
								></textarea></label
							><button
								type="button"
								disabled={work.busy}
								onclick={() => (eventImage = undefined)}
								>Remove event image</button
							>{/if}
						<div class="actions">
							<button
								class="primary"
								disabled={work.busy ||
									!!work.retry ||
									!!imageEditor ||
									imageTransformBusy}>Save event</button
							><button
								type="button"
								disabled={work.busy}
								onclick={() => (eventOpen = false)}>Cancel</button
							>
						</div>
					</fieldset>
				</form>
			</dialog>{/if}
	{/if}
{/if}

<dialog
	bind:this={imageDialog}
	class="business-image-dialog"
	oncancel={(event) => {
		if (work.busy || imageTransformBusy) event.preventDefault();
		else imageEditor = null;
	}}
>
	{#if imageEditor}<h2>
			{imageEditor.target === null ? 'Event image' : 'Product image'}
		</h2>
		<MediaEditor
			media={imageEditor.media}
			index={0}
			count={1}
			allowReorder={false}
			disabled={work.busy}
			onBusyChange={(value) => (imageTransformBusy = value)}
			onEditingChange={(value) => (imageEditing = value)}
			onchange={(media) => {
				if (imageEditor) imageEditor = { ...imageEditor, media };
			}}
			onremove={cancelImage}
			onmove={() => {}}
		/>
		{#if work.error}<p class="error" role="alert">{work.error}</p>{/if}
		<div class="actions">
			<button
				class="primary"
				disabled={work.busy || imageTransformBusy || imageEditing}
				onclick={useImage}
				>{work.busy ? 'Preparing image…' : 'Use image'}</button
			><button disabled={work.busy || imageTransformBusy} onclick={cancelImage}
				>Cancel image</button
			>
		</div>{/if}
</dialog>

<style>
	.business-image-dialog {
		max-width: 620px;
		width: calc(100vw - 3rem);
		max-height: 90vh;
		overflow: auto;
		background: var(--surface);
		color: var(--ink);
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		padding: 1.25rem;
		box-shadow: 4px 4px 0 var(--ink);
	}
	.business-image-dialog::backdrop {
		background: rgb(0 0 0 / 0.5);
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
	.product-item {
		padding: 12px;
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
	.product-editor {
		padding: 16px 4px 4px;
	}
	.events-toolbar {
		justify-content: flex-end;
	}
	.event-item h3 a {
		color: inherit;
		text-decoration: none;
	}
	.event-cover {
		display: block;
		width: 100%;
		max-height: 240px;
		object-fit: cover;
		margin-bottom: 12px;
		border-radius: 6px;
	}
	.event-dialog {
		padding: 24px;
	}
</style>
