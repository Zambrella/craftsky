<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { beforeNavigate, goto } from '$app/navigation';
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
	import { validateText } from '../composer/model';
	import { prepareImage } from '../composer/media';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import catalog from './catalog.json';
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
	let avatar = $state<File | null | undefined>(),
		banner = $state<File | null | undefined>();
	let avatarURL = $state(''),
		bannerURL = $state('');
	let dead = false;
	const choices = catalog.filter((item) => item.family === 'craft');
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
				bannerURL = profile.banner || '';
			}
		);
	});
	onDestroy(() => {
		dead = true;
		work.dispose();
		if (avatarURL.startsWith('blob:')) URL.revokeObjectURL(avatarURL);
		if (bannerURL.startsWith('blob:')) URL.revokeObjectURL(bannerURL);
	});
	beforeNavigate((nav) => {
		if (dirty && !confirm('Discard your unsaved profile changes?'))
			nav.cancel();
	});
	function unload(event: BeforeUnloadEvent) {
		if (dirty) {
			event.preventDefault();
			event.returnValue = '';
		}
	}
	function choose(event: Event, kind: 'avatar' | 'banner') {
		const file = (event.target as HTMLInputElement).files?.[0];
		if (!file) return;
		if (
			!['image/jpeg', 'image/png', 'image/webp'].includes(file.type) ||
			file.size > 20 * 1024 * 1024
		) {
			work.error = 'Choose a JPEG, PNG or WebP image under 20 MB.';
			return;
		}
		const old = kind === 'avatar' ? avatarURL : bannerURL;
		if (old.startsWith('blob:')) URL.revokeObjectURL(old);
		if (kind === 'avatar') {
			avatar = file;
			avatarURL = URL.createObjectURL(file);
		} else {
			banner = file;
			bannerURL = URL.createObjectURL(file);
		}
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
		if (kind === 'avatar') avatar = file;
		else banner = file;
		dirty = true;
	}
	function move(index: number, direction: number) {
		const copy = [...crafts];
		[copy[index], copy[index + direction]] = [
			copy[index + direction],
			copy[index]
		];
		crafts = copy;
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
			key = newOperationKey();
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
		const a = avatar,
			b = banner;
		let frozen: ProfileUpdate | undefined;
		void work.run(
			async () => {
				if (!frozen) {
					for (const [kind, file] of [
						['avatar', a],
						['banner', b]
					] as const) {
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
			() => {
				if (dead) return;
				dirty = false;
				if (onSaved) onSaved();
				else void goto('/app/profile/' + encodeURIComponent(session.did));
			}
		);
	}
</script>

<svelte:window onbeforeunload={unload} />
<div class="workspace-header">
	<h1>
		{scope === 'identity'
			? 'Set up your profile'
			: scope === 'crafts'
				? 'Choose your crafts'
				: 'Edit profile'}
	</h1>
	{#if scope === 'all'}<a
			href={'/app/profile/' + encodeURIComponent(session.did)}>Cancel</a
		>{/if}
</div>
<Status {work} />
{#if original}<form
		class="panel stack"
		oninput={() => (dirty = true)}
		onsubmit={(event) => {
			event.preventDefault();
			save();
		}}
	>
		<div inert={work.busy || !!work.retry} class="stack">
			{#if scope !== 'crafts'}<div class="form-grid">
					{#each ['avatar', 'banner'] as kind}<div class="stack">
							<label
								>{kind === 'avatar' ? 'Profile picture' : 'Cover photo'}<input
									type="file"
									accept="image/jpeg,image/png,image/webp"
									onchange={(event) =>
										choose(event, kind as 'avatar' | 'banner')}
								/></label
							>{#if imageDrafts[kind]}<MediaEditor
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
										} else {
											banner = null;
											bannerURL = '';
										}
										dirty = true;
									}}
									onmove={() => {}}
								/>{:else if kind === 'avatar' ? avatarURL : bannerURL}<img
									class="profile-edit-image"
									src={kind === 'avatar' ? avatarURL : bannerURL}
									alt={kind === 'avatar'
										? 'Profile picture preview'
										: 'Cover photo preview'}
								/><button
									type="button"
									onclick={() => {
										if (kind === 'avatar') {
											avatar = null;
											avatarURL = '';
										} else {
											banner = null;
											bannerURL = '';
										}
										dirty = true;
									}}>Remove {kind === 'avatar' ? 'picture' : 'cover'}</button
								>{/if}
						</div>{/each}
				</div>
				<label
					>Display name<input bind:value={displayName} maxlength="640" /></label
				><label>Pronouns<input bind:value={pronouns} maxlength="200" /></label
				><label
					>Bio<textarea bind:value={description} maxlength="2560" rows="5"
					></textarea></label
				>
			{/if}{#if scope !== 'identity'}<fieldset>
					<legend>Your crafts, in feed order</legend>
					<div class="choice-grid">
						{#each choices as craft}<label class="check"
								><input
									type="checkbox"
									checked={crafts.includes(craft.id)}
									onchange={(event) => {
										crafts = event.currentTarget.checked
											? [...crafts, craft.id]
											: crafts.filter((value) => value !== craft.id);
										dirty = true;
									}}
								/>{craft.labels.en}</label
							>{/each}
					</div>
					<ol class="reorder-list">
						{#each crafts as craft, i}<li>
								{choices.find((item) => item.id === craft)?.labels.en ||
									craft.split('#').pop()}<button
									type="button"
									aria-label="Move craft up"
									disabled={i === 0}
									onclick={() => move(i, -1)}>↑</button
								><button
									type="button"
									aria-label="Move craft down"
									disabled={i === crafts.length - 1}
									onclick={() => move(i, 1)}>↓</button
								>
							</li>{/each}
					</ol>
				</fieldset>
			{/if}{#if scope === 'all'}<fieldset class="stack">
					<legend>Profile links</legend>{#each links as link, i (link.rkey)}<div
							class="panel stack"
						>
							<label
								>Title<input bind:value={link.title} maxlength="100" /></label
							><label
								>Web address<input
									type="url"
									required
									bind:value={link.url}
									maxlength="2048"
									placeholder="https://"
								/></label
							><button
								type="button"
								onclick={() => {
									links = links.filter((_, index) => index !== i);
									dirty = true;
								}}>Remove link</button
							>
						</div>{/each}{#if links.length < 10}<button
							type="button"
							onclick={() => {
								links = [
									...links,
									{
										rkey: crypto.randomUUID(),
										url: '',
										title: '',
										createdAt: new Date().toISOString()
									}
								];
								dirty = true;
							}}>Add link</button
						>{/if}
				</fieldset>
			{/if}{#if imageBusy}<p role="status">
					Apply or cancel your image edits before continuing.
				</p>{/if}<button
				class="button"
				disabled={work.busy ||
					!!work.retry ||
					imageBusy ||
					(!dirty && !onSaved)}>{onSaved ? 'Continue' : 'Save profile'}</button
			>
		</div>
	</form>{/if}
