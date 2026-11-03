<script lang="ts">
	import { confirmNavigation } from '../social/navigation-prompt';
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		CraftskyAPI,
		newOperationKey,
		type Profile,
		type ProfileLinkInput,
		type ProfileUpdate,
		type BlobReference
	} from '#lib/api/client.ts';
	import MediaEditor from '../composer/MediaEditor.svelte';
	import type { DraftMedia } from '../drafts/store';
	import { validateText, textLength } from '../composer/model';
	import { prepareImage } from '../composer/media';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import catalog from './catalog.json';
	import Icon from '../shell/Icon.svelte';
	import Avatar from '#lib/Avatar.svelte';
	import { profileStyle, safeLink } from './model';
	import { contextMenu } from './context-menu';
	import { label } from '#lib/public.ts';
	let {
		api,
		session,
		scope = 'all',
		onSaved
	}: {
		api: CraftskyAPI;
		session: { id: string; did: string; handle: string };
		scope?: 'all' | 'identity' | 'crafts';
		onSaved?: () => void;
	} = $props();
	const work = new Work();
	let original = $state<Profile>();
	let displayName = $state(''),
		description = $state(''),
		pronouns = $state(''),
		crafts = $state<string[]>([]),
		links = $state<ProfileLinkInput[]>([]),
		dirty = $state(false);
	let imageDrafts = $state<Record<string, DraftMedia | undefined>>({});
	let transforming = $state<Record<string, boolean>>({});
	const imageBusy = $derived(Object.values(transforming).some(Boolean));
	function imageState(key: string, value: boolean) {
		if (transforming[key] !== value)
			transforming = { ...transforming, [key]: value };
	}
	let avatar = $state<File | null | undefined>();
	const editorId = $props.id();
	let linkDialog = $state<HTMLDialogElement>(null!);
	let editingLink = $state<string | null>(null),
		linkURL = $state(''),
		linkTitle = $state(''),
		linkError = $state('');
	let draggedLink = $state<string>();
	let avatarURL = $state('');
	let dead = false;
	const craftOrder = [
		'sewing',
		'knitting',
		'crochet',
		'quilting',
		'embroidery'
	];
	const choices = catalog
		.filter((item) => item.family === 'craft')
		.sort(
			(a, b) =>
				craftOrder.indexOf(craftKey(a.id)) - craftOrder.indexOf(craftKey(b.id))
		);
	let draggedCraft = $state<string>();
	function craftKey(value: string) {
		return value.split('#').at(-1)?.trim().toLowerCase() || '';
	}
	const craftRows = $derived([
		...crafts.map((value, index) => ({
			value,
			key: craftKey(value),
			label:
				choices.find((item) => craftKey(item.id) === craftKey(value))?.labels
					.en || label(craftKey(value)),
			index
		})),
		...choices
			.filter(
				(item) => !crafts.some((value) => craftKey(value) === craftKey(item.id))
			)
			.map((item) => ({
				value: item.id,
				key: craftKey(item.id),
				label: item.labels.en,
				index: -1
			}))
	]);
	function toggleCraft(value: string) {
		if (work.busy || work.retry) return;
		const selected = crafts.some(
			(craft) => craftKey(craft) === craftKey(value)
		);
		crafts = selected
			? crafts.filter((craft) => craftKey(craft) !== craftKey(value))
			: [...crafts, value];
		dirty = true;
	}
	function reorderCraft(from: number, to: number) {
		if (
			work.busy ||
			work.retry ||
			from < 0 ||
			to < 0 ||
			from >= crafts.length ||
			to >= crafts.length ||
			from === to
		)
			return;
		const next = [...crafts];
		next.splice(to, 0, next.splice(from, 1)[0]);
		crafts = next;
		dirty = true;
	}
	function reorderCraftKey(event: KeyboardEvent, index: number) {
		if (
			index >= 0 &&
			event.altKey &&
			['ArrowUp', 'ArrowDown'].includes(event.key)
		) {
			event.preventDefault();
			reorderCraft(index, index + (event.key === 'ArrowUp' ? -1 : 1));
		}
	}
	onMount(() => {
		void work.run(
			() => api.profiles.me(),
			(profile) => {
				original = profile;
				displayName = profile.displayName || '';
				description = profile.description || '';
				pronouns = profile.pronouns || '';
				crafts = profile.crafts;
				links = (profile.editableLinks || profile.links || []).map((link) => ({
					rkey: link.uri.split('/').pop()!,
					cid: link.cid,
					url: link.url,
					title: link.title,
					createdAt: link.createdAt || new Date().toISOString(),
					icon: link.iconBlob
				}));
				avatarURL = profile.avatar || '';
			}
		);
	});
	onDestroy(() => {
		dead = true;
		work.dispose();
		if (avatarURL.startsWith('blob:')) URL.revokeObjectURL(avatarURL);
	});
	confirmNavigation(() => dirty, {
		title: 'Discard changes?',
		detail: "Your edits won't be saved.",
		confirmLabel: 'Discard',
		cancelLabel: 'Keep editing',
		danger: true
	});
	function unload(event: BeforeUnloadEvent) {
		if (dirty) {
			event.preventDefault();
			event.returnValue = '';
		}
	}
	function choose(event: Event, kind: 'avatar') {
		const file = (event.target as HTMLInputElement).files?.[0];
		if (!file) return;
		if (
			!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) ||
			file.size > 20 * 1024 * 1024
		) {
			work.error = 'Choose a JPEG, PNG or WebP image under 20 MB.';
			return;
		}
		const old = avatarURL;
		if (old.startsWith('blob:')) URL.revokeObjectURL(old);
		avatar = file;
		avatarURL = URL.createObjectURL(file);
		imageDrafts = {
			...imageDrafts,
			[kind]: {
				id: crypto.randomUUID(),
				name: file.name,
				blob: file,
				mime: file.type,
				alt: ''
			}
		};
		dirty = true;
	}
	function edited(kind: string, value: DraftMedia) {
		imageDrafts = { ...imageDrafts, [kind]: value };
		if (!value.blob) return;
		const file = new File([value.blob], value.name, { type: value.mime });
		avatar = file;
		dirty = true;
	}

	function normalizedLink(value: string): string | undefined {
		const raw = value.trim();
		if (!raw || /\s/.test(raw)) return;
		try {
			const url = new URL(raw.includes('://') ? raw : 'https://' + raw);
			if (
				url.protocol === 'https:' &&
				url.hostname &&
				!url.username &&
				!url.password
			)
				return url.href;
		} catch {
			/* Invalid drafts remain local to the link dialog. */
		}
	}
	function linkLabel(link: ProfileLinkInput) {
		try {
			return (
				link.title?.trim() || new URL(link.url).hostname.replace(/^www\./, '')
			);
		} catch {
			return link.title?.trim() || link.url;
		}
	}
	function linkIcon(link: ProfileLinkInput) {
		const stored = (original?.editableLinks || original?.links || []).find(
			(value) =>
				value.uri.split('/').pop() === link.rkey && value.url === link.url
		);
		return safeLink(stored?.icon || '');
	}
	function editLink(key: string | null) {
		if (work.busy || work.retry) return;
		const link = links.find((value) => value.rkey === key);
		if (key && !link) return;
		editingLink = key;
		linkURL = link?.url || '';
		linkTitle = link?.title || '';
		linkError = '';
		linkDialog.showModal();
	}
	function acceptLink() {
		if (work.busy || work.retry) return;
		const url = normalizedLink(linkURL),
			title = linkTitle.trim();
		if (!url || linkURL.length > 2048) {
			linkError = 'Enter a valid HTTPS URL without a username or password.';
			return;
		}
		if (!validateText(title, 30, 320)) {
			linkError = 'Use up to 30 characters for the title.';
			return;
		}
		if (
			links.some(
				(value) =>
					value.rkey !== editingLink && normalizedLink(value.url) === url
			)
		) {
			linkError = 'This link is already on your profile.';
			return;
		}
		const current = links.find((value) => value.rkey === editingLink);
		if (editingLink && !current) {
			linkError = 'This link has been removed from the draft.';
			return;
		}
		if (!current && links.length >= 10) return;
		const unchanged = current?.url === url && (current?.title || '') === title;
		const link: ProfileLinkInput = {
			...current,
			rkey: current?.rkey || crypto.randomUUID(),
			url,
			title: title || undefined,
			createdAt: unchanged ? current!.createdAt : new Date().toISOString(),
			icon: current?.url === url ? current?.icon : undefined
		};
		links = current
			? links.map((value) => (value.rkey === current.rkey ? link : value))
			: [...links, link];
		if (!unchanged) dirty = true;
		linkDialog.close();
	}
	function moveLink(from: number, to: number) {
		if (
			work.busy ||
			work.retry ||
			from === to ||
			from < 0 ||
			to < 0 ||
			from >= links.length ||
			to >= links.length
		)
			return;
		const next = [...links];
		next.splice(to, 0, next.splice(from, 1)[0]);
		links = next;
		dirty = true;
	}
	function save() {
		if (!original || work.busy || work.retry || imageBusy) return;
		if (!dirty && onSaved) {
			onSaved();
			return;
		}
		if (
			!validateText(displayName, 64) ||
			!validateText(description, 256) ||
			!validateText(pronouns, 20)
		) {
			work.error =
				'Use up to 64 characters for your name, 20 for pronouns and 256 for your bio.';
			return;
		}
		if (
			links.some((link) => {
				try {
					const url = new URL(link.url);
					return url.protocol !== 'https:' || !!url.username || !!url.password;
				} catch {
					return true;
				}
			})
		) {
			work.error = 'Use a full HTTPS address for each profile link.';
			return;
		}
		const base = original,
			key = newOperationKey(),
			owner = { accountId: session.id, did: session.did };
		const values: ProfileUpdate = {
			displayName: displayName.trim(),
			description,
			pronouns: pronouns.trim(),
			crafts: [...crafts],
			links: structuredClone($state.snapshot(links)),
			removedLinks: (base.editableLinks || base.links || [])
				.filter(
					(link) =>
						!links.some((value) => value.rkey === link.uri.split('/').pop())
				)
				.map(({ uri, cid }) => ({ uri, cid }))
		};
		const a = avatar;
		let frozen: ProfileUpdate | undefined;
		void work.run(
			async () => {
				if (!frozen) {
					for (const [kind, file] of [['avatar', a]] as const) {
						if (file === null) values[kind] = null;
						else if (file instanceof File && values[kind] === undefined) {
							const prepared = await prepareImage(file, file.name);
							values[kind] = (
								await api.media.image(prepared.blob!, prepared.mime)
							).blob;
						}
					}
					frozen = structuredClone(values);
				}
				return api.profiles.update(frozen, key);
			},
			(saved) => {
				if (dead || session.id !== owner.accountId || session.did !== owner.did)
					return;
				if (saved.did !== owner.did) {
					work.error =
						'The saved profile could not be confirmed for this account.';
					return;
				}
				dirty = false;
				window.dispatchEvent(
					new CustomEvent('craftsky-profile-updated', { detail: owner })
				);
				if (onSaved) onSaved();
				else void goto('/app/profile/' + encodeURIComponent(owner.did));
			}
		);
	}
</script>

<svelte:window onbeforeunload={unload} />
<div class="workspace-header">
	{#if scope === 'all'}<a
			class="icon-button"
			href={'/app/profile/' + encodeURIComponent(session.did)}
			aria-label="Close editor"
			onclick={(event) => {
				if (work.busy) event.preventDefault();
			}}
			aria-disabled={work.busy || undefined}><Icon name="close" /></a
		>{/if}
	<h1>
		{scope === 'identity'
			? 'Make it yours'
			: scope === 'crafts'
				? 'What do you make?'
				: 'Edit profile'}
	</h1>
	{#if scope === 'all'}<button
			type="submit"
			class="appbar-save"
			form={editorId + '-profile-form'}
			aria-label="Save profile"
			disabled={!original ||
				work.busy ||
				!!work.retry ||
				imageBusy ||
				(!dirty && !onSaved)}>{work.busy ? 'Saving…' : 'Save'}</button
		>{/if}
</div>
{#if scope !== 'all'}<p class="onboarding-description">
		{scope === 'identity'
			? 'Add a name, pronouns, bio, and photo so other crafters can recognize you.'
			: 'Choose as many crafts as you like. You can change these later.'}
	</p>{/if}
<Status {work} />
{#if original}<form
		id={editorId + '-profile-form'}
		class="profile-form stack"
		oninput={() => (dirty = true)}
		onsubmit={(event) => {
			event.preventDefault();
			save();
		}}
	>
		<div inert={work.busy || !!work.retry} class="stack">
			{#if scope !== 'crafts'}<div
					class="form-grid profile-images"
					class:onboarding-images={scope === 'identity'}
				>
					{#each ['avatar'] as kind}<div
							class="stack"
							class:avatar-field={kind === 'avatar'}
						>
							{#if kind === 'avatar'}<div
									class="avatar-picker"
									style={profileStyle(original.customisation)}
								>
									<Avatar
										author={{
											did: original.did,
											handle: original.handle,
											displayName: displayName || undefined,
											avatar: avatarURL || undefined
										}}
										large
									/><label class="camera-picker" title="Change profile picture"
										><Icon name="camera" size={18} /><span
											class="visually-hidden">Profile picture</span
										><input
											type="file"
											accept="image/jpeg,image/png,image/webp"
											onchange={(event) => choose(event, 'avatar')}
										/></label
									>
								</div>
								{#if avatarURL && !imageDrafts.avatar}<button
										class="remove-avatar"
										type="button"
										onclick={() => {
											avatar = null;
											avatarURL = '';
											dirty = true;
										}}>Remove picture</button
									>{/if}{/if}
							{#if imageDrafts[kind]}<MediaEditor
									describe={false}
									allowReorder={false}
									onBusyChange={(busy) => imageState(kind, busy)}
									onEditingChange={(editing) =>
										imageState(kind + '-editing', editing)}
									media={imageDrafts[kind]!}
									index={0}
									count={1}
									onchange={(value) => edited(kind, value)}
									onremove={() => {
										imageDrafts = { ...imageDrafts, [kind]: undefined };
										if (kind === 'avatar') {
											avatar = null;
											avatarURL = '';
										}
										dirty = true;
									}}
									onmove={() => {}}
								/>{/if}
						</div>{/each}
				</div>
				<label
					>Display name<input bind:value={displayName} maxlength="640" /><span
						class="field-count"
						aria-hidden="true">{textLength(displayName)}/64</span
					></label
				><label
					>Pronouns<input bind:value={pronouns} maxlength="200" /><span
						class="field-count"
						aria-hidden="true">{textLength(pronouns)}/20</span
					></label
				><label
					>Bio<textarea bind:value={description} maxlength="2560" rows="5"
					></textarea><span class="field-count" aria-hidden="true"
						>{textLength(description)}/256</span
					></label
				>
			{/if}{#if scope !== 'crafts'}<fieldset class="stack profile-links-editor">
					<legend>Links</legend>
					<p class="links-helper">Add up to 10 links</p>
					{#each links as link, i (link.rkey)}<div
							class="profile-link"
							ondragover={(event) => {
								if (draggedLink) event.preventDefault();
							}}
							ondrop={(event) => {
								event.preventDefault();
								const from = links.findIndex(
									(value) => value.rkey === draggedLink
								);
								draggedLink = undefined;
								moveLink(from, i);
							}}
							role="group"
							aria-label={linkLabel(link)}
						>
							<button
								class="link-card"
								type="button"
								onclick={() => editLink(link.rkey)}
								aria-label={'Edit link ' + linkLabel(link)}
								>{#if linkIcon(link)}<img
										src={linkIcon(link)}
										alt=""
									/>{:else}<Icon name="globe" size={18} />{/if}<span
									>{linkLabel(link)}</span
								></button
							>
							<details class="editor-actions link-actions" use:contextMenu>
								<summary
									class="icon-button"
									aria-label={'Actions for ' + linkLabel(link)}
									><Icon name="more" /></summary
								><button
									class="link-menu-backdrop"
									type="button"
									tabindex="-1"
									aria-label="Close link actions"
									onclick={(event) => {
										event.currentTarget.closest('details')!.open = false;
									}}
								></button>
								<div class="menu-items">
									<button type="button" onclick={() => editLink(link.rkey)}
										><Icon name="edit" />Edit link</button
									>
									{#if i > 0}<button
											type="button"
											onclick={() => moveLink(i, i - 1)}
											><span class="move-up"><Icon name="back" /></span>Move up</button
										>{/if}
									{#if i < links.length - 1}<button
											type="button"
											onclick={() => moveLink(i, i + 1)}
											><span class="move-down"><Icon name="back" /></span>Move
											down</button
										>{/if}
									<button
										class="remove-link"
										type="button"
										onclick={() => {
											links = links.filter((value) => value.rkey !== link.rkey);
											dirty = true;
										}}><Icon name="trash" />Remove link</button
									>
								</div>
							</details>
							{#if links.length > 1}<button
									class="drag-handle"
									onkeydown={(event) => {
										if (
											event.altKey &&
											['ArrowUp', 'ArrowDown'].includes(event.key)
										) {
											event.preventDefault();
											moveLink(i, i + (event.key === 'ArrowUp' ? -1 : 1));
										}
									}}
									type="button"
									draggable="true"
									aria-label={'Reorder link ' + linkLabel(link)}
									title="Drag to reorder, or use Alt and the up or down arrow keys."
									ondragstart={(event) => {
										draggedLink = link.rkey;
										event.dataTransfer?.setData('text/plain', link.rkey);
									}}
									ondragend={() => (draggedLink = undefined)}
									><span aria-hidden="true"></span></button
								>{/if}
						</div>{/each}
					{#if links.length < 10}<button
							class="add-link"
							type="button"
							onclick={() => editLink(null)}
							><Icon name="plus" size={18} />Add link</button
						>{/if}
				</fieldset>
			{/if}{#if scope !== 'identity'}<fieldset class="crafts-editor">
					<legend>Crafts</legend>
					<p class="crafts-helper">Pick the crafts you make</p>
					<div class="craft-rows">
						{#each craftRows as craft (craft.value)}<div
								class="craft-row"
								role="group"
								aria-label={craft.label}
								ondragover={(event) => {
									if (draggedCraft && craft.index >= 0) event.preventDefault();
								}}
								ondrop={(event) => {
									event.preventDefault();
									const from = crafts.indexOf(draggedCraft || '');
									draggedCraft = undefined;
									reorderCraft(from, craft.index);
								}}
							>
								<label class="craft-choice"
									><input
										type="checkbox"
										aria-label={craft.label}
										checked={craft.index >= 0}
										onchange={() => toggleCraft(craft.value)}
										onkeydown={(event) => reorderCraftKey(event, craft.index)}
									/><span class="craft-copy"
										><span class="craft-name"
											>{#if craftOrder.includes(craft.key)}<span
													class="craft-glyph"
													style:--craft-icon={`url('/craft_icons/${craft.key}.svg')`}
													aria-hidden="true"
												></span>{/if}{craft.label}</span
										>{#if craft.index === 0}<small class="default-craft"
												><Icon name="star" filled size={16} />Default</small
											>{/if}</span
									></label
								>
								{#if craft.index >= 0 && crafts.length > 1}<details
										class="editor-actions craft-actions"
										use:contextMenu
									>
										<summary
											class="icon-button"
											aria-label={'Reorder ' + craft.label}
											><Icon name="more" /></summary
										><button
											class="link-menu-backdrop"
											type="button"
											tabindex="-1"
											aria-label="Close craft actions"
											onclick={(event) => {
												event.currentTarget.closest('details')!.open = false;
											}}
										></button>
										<div class="menu-items">
											{#if craft.index > 0}<button
													type="button"
													onclick={() =>
														reorderCraft(craft.index, craft.index - 1)}
													><span class="move-up"><Icon name="back" /></span>Move
													up</button
												>{/if}{#if craft.index < crafts.length - 1}<button
													type="button"
													onclick={() =>
														reorderCraft(craft.index, craft.index + 1)}
													><span class="move-down"><Icon name="back" /></span
													>Move down</button
												>{/if}
										</div>
									</details>
									<button
										type="button"
										class="drag-handle"
										draggable="true"
										aria-label={'Drag to reorder ' + craft.label}
										title="Drag to reorder, or use Alt and the up or down arrow keys."
										onkeydown={(event) => reorderCraftKey(event, craft.index)}
										ondragstart={(event) => {
											draggedCraft = craft.value;
											event.dataTransfer?.setData('text/plain', craft.value);
										}}
										ondragend={() => (draggedCraft = undefined)}
										><span aria-hidden="true"></span></button
									>{/if}
							</div>{/each}
					</div>
					<a
						class="request-crafts"
						href="https://userinput.app/s/did:plc:lmmx63zcns6gewgxqfdt4kof/3mpr5izppvt2k?lang=en"
						target="_blank"
						rel="noopener noreferrer">Request more</a
					>
				</fieldset>{/if}
			{#if imageBusy}<p role="status">
					Apply or cancel your image edits before continuing.
				</p>{/if}{#if scope !== 'all'}<button
					class="button"
					disabled={work.busy ||
						!!work.retry ||
						imageBusy ||
						(!dirty && !onSaved)}
					>{onSaved ? 'Continue' : 'Save profile'}</button
				>{/if}
		</div>
	</form>{/if}

<dialog
	bind:this={linkDialog}
	class="link-editor-dialog"
	aria-labelledby={editorId + '-link-title'}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			acceptLink();
		}}
	>
		<h2 id={editorId + '-link-title'}>
			{editingLink ? 'Edit link' : 'Add link'}
		</h2>
		<label
			>URL<input
				type="text"
				inputmode="url"
				autocomplete="url"
				spellcheck="false"
				bind:value={linkURL}
				oninput={() => (linkError = '')}
				placeholder="https://"
				maxlength="2048"
				required
			/></label
		>
		<label
			>Title (optional)<input
				bind:value={linkTitle}
				oninput={() => (linkError = '')}
				maxlength="320"
			/></label
		>
		<p class="link-title-count">{textLength(linkTitle)}/30</p>
		{#if linkError}<p class="link-error" role="alert">{linkError}</p>{/if}
		<div class="link-dialog-actions">
			<button type="button" onclick={() => linkDialog.close()}>Cancel</button
			><button>Save</button>
		</div>
	</form>
</dialog>

<style>
	.profile-images {
		grid-template-columns: 1fr;
	}
	.avatar-field {
		align-items: center;
	}
	.avatar-picker {
		position: relative;
		width: 106px;
		height: 106px;
		margin: 0 auto 8px;
	}
	.avatar-picker :global(.profile-avatar) {
		outline: 0;
		box-shadow: none;
	}
	.camera-picker {
		position: absolute;
		right: -4px;
		bottom: -4px;
		display: grid;
		place-items: center;
		width: 36px;
		height: 36px;
		border-radius: 50%;
		background: var(--primary-soft);
		color: var(--ink);
		border: 1.5px solid var(--paper);
		cursor: pointer;
		overflow: hidden;
	}
	.camera-picker input {
		position: absolute;
		inset: 0;
		width: 100%;
		height: 100%;
		opacity: 0;
		cursor: pointer;
	}
	.camera-picker:focus-within {
		outline: 3px solid var(--primary);
		outline-offset: 2px;
	}
	.remove-avatar {
		align-self: center;
		background: transparent;
		border: 0;
		box-shadow: none;
		color: var(--primary);
	}
	.workspace-header {
		justify-content: flex-start;
		gap: 8px;
	}
	.workspace-header h1 {
		flex: 1;
	}
	.profile-form {
		padding: 16px;
	}
	.profile-form > .stack {
		gap: 20px;
	}
	.profile-form fieldset {
		border: 0;
		padding: 0;
		margin: 4px 0 0;
		min-width: 0;
	}
	.profile-form legend {
		padding: 0;
		margin-bottom: 12px;
		font-size: 16px;
		font-weight: 700;
	}
	.profile-form label {
		gap: 6px;
		font-size: 16px;
	}
	.profile-form input,
	.profile-form textarea {
		margin: 0;
		font-size: 16px;
	}
	.profile-images {
		padding-bottom: 4px;
		align-items: start;
	}
	.onboarding-images {
		grid-template-columns: 1fr;
		max-width: 320px;
		margin: 0 auto 8px;
	}
	.profile-images input[type='file'] {
		font-size: 12px;
		max-width: 100%;
	}
	.profile-images button {
		font-size: 13px;
	}
	.appbar-save {
		border: 0;
		background: transparent;
		box-shadow: none;
		color: var(--primary);
		font-size: 14px;
		font-weight: 600;
		padding: 12px 8px;
	}
	.appbar-save:disabled {
		opacity: 0.5;
	}
	.field-count {
		display: block;
		font-size: 12px;
		color: var(--ink-2);
		text-align: right;
		font-weight: 400;
	}
	.crafts-helper {
		margin: -8px 0 8px;
		font-size: 14px;
		color: var(--ink-2);
	}
	.craft-rows {
		display: grid;
		gap: 0;
	}
	.craft-row {
		display: flex;
		align-items: center;
		min-width: 0;
		min-height: 56px;
	}
	.craft-row .craft-choice {
		flex-direction: row;
		flex: 1;
		display: flex;
		align-items: center;
		gap: 12px;
		min-width: 0;
		min-height: 56px;
		padding: 8px 0;
		font-size: 16px;
		font-weight: 400;
		cursor: pointer;
	}
	.craft-choice input {
		width: 18px;
		height: 18px;
		flex: 0 0 18px;
		margin: 0 8px;
		accent-color: var(--primary);
	}
	.craft-copy {
		display: grid;
		min-width: 0;
		gap: 2px;
	}
	.craft-name {
		display: flex;
		align-items: center;
		gap: 6px;
	}
	.craft-glyph {
		display: block;
		width: 18px;
		height: 18px;
		flex-shrink: 0;
		background: currentColor;
		mask: var(--craft-icon) center / contain no-repeat;
	}
	.default-craft {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--ink-2);
	}
	.request-crafts {
		display: inline-block;
		color: var(--primary);
		text-decoration: none;
		font-size: 14px;
		padding: 12px 8px;
	}

	.links-helper {
		margin: -8px 0 0;
		color: var(--ink-2);
		font-size: 14px;
	}
	.profile-link {
		display: flex;
		align-items: center;
		min-width: 0;
		min-height: 48px;
		gap: 0;
	}
	.link-card {
		justify-content: flex-start;
		display: flex;
		align-items: center;
		gap: 16px;
		flex: 1;
		min-width: 0;
		padding: 10px 8px;
		border: 0;
		box-shadow: none;
		background: transparent;
		text-align: left;
		font-size: 14px;
	}
	.link-card span {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.link-card img {
		width: 18px;
		height: 18px;
		object-fit: contain;
	}
	.editor-actions {
		position: relative;
		flex-shrink: 0;
	}
	.editor-actions summary {
		list-style: none;
		cursor: pointer;
		width: 44px;
		height: 44px;
		padding: 10px;
	}
	.editor-actions summary::-webkit-details-marker {
		display: none;
	}
	.editor-actions .remove-link {
		color: var(--red-ink);
	}
	.move-up {
		display: flex;
		transform: rotate(90deg);
	}
	.move-down {
		display: flex;
		transform: rotate(-90deg);
	}
	.drag-handle {
		width: 40px;
		height: 44px;
		border: 0;
		box-shadow: none;
		background: transparent;
		padding: 12px;
		cursor: grab;
	}
	.drag-handle span {
		width: 16px;
		flex: 0 0 16px;
		display: block;
		height: 7px;
		border-block: 2px solid var(--ink-2);
	}
	.add-link {
		display: inline-flex;
		gap: 6px;
		align-items: center;
		align-self: flex-start;
		border: 1px solid var(--hairline);
		border-radius: 999px;
		box-shadow: none;
		background: var(--surface);
		padding: 6px 12px;
		font-size: 14px;
	}
	.link-editor-dialog {
		width: min(440px, calc(100vw - 48px));
		padding: 24px;
		background: var(--surface);
	}
	.link-editor-dialog form {
		margin: 0;
	}
	.link-editor-dialog h2 {
		margin: 0 0 16px;
		font-size: 20px;
	}
	.link-editor-dialog label {
		display: block;
		font-size: 14px;
		font-weight: 600;
		margin-top: 16px;
	}
	.link-editor-dialog input {
		margin: 8px 0 0;
		font-size: 16px;
		min-height: 52px;
	}
	.link-title-count {
		font-size: 12px;
		color: var(--ink-2);
		margin: 4px 0 0;
		text-align: right;
	}
	.link-error {
		color: var(--red-ink);
		font-size: 14px;
	}
	.link-dialog-actions {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
		margin-top: 24px;
	}
	.link-dialog-actions button {
		color: var(--primary);
		border: 0;
		box-shadow: none;
		background: transparent;
		font-size: 14px;
		font-weight: 600;
		padding: 8px;
	}
	.onboarding-description {
		margin: 0;
		padding: 0 16px 8px;
		color: var(--ink-2);
		font-size: 15px;
		line-height: 1.5;
	}
	@media (max-width: 600px) {
		.profile-images:not(.onboarding-images) {
			grid-template-columns: 1fr;
		}
	}
</style>
