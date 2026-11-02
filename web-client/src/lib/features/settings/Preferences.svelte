<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type {
		CraftskyAPI,
		NotificationPreferences,
		LanguagePreferences
	} from '#lib/api/client.ts';
	import {
		backgrounds,
		colours,
		languageOptions,
		languagePreferences,
		record,
		text,
		title,
		type Session
	} from './model.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import PushSettings from '#lib/features/push/PushSettings.svelte';
	import {
		configured as diagnosticsConfigured,
		consent,
		setConsent
	} from '#lib/features/diagnostics/client.ts';
	let {
		api,
		section,
		session
	}: { api: CraftskyAPI; section: string; session: Session } = $props();
	const work = new Work();
	let theme = $state('system'),
		diagnostics = $state(false);
	let preferences = $state<NotificationPreferences>({ preferences: {} });
	let primary = $state('en'),
		content = $state('');
	let languageSearch = $state('');
	let languageMatches = $derived(
		languageOptions.filter((value) =>
			`${value.name} ${value.tag}`
				.toLowerCase()
				.includes(languageSearch.toLowerCase())
		)
	);
	function toggleLanguage(tag: string) {
		const selected = content.split(', ').filter(Boolean);
		content = (
			selected.includes(tag)
				? selected.filter((value) => value !== tag)
				: [...selected, tag]
		).join(', ');
	}
	let colour = $state('cobalt'),
		background = $state('none');
	let pattern = $derived(
		background === 'none'
			? ''
			: background.startsWith('craft-')
				? `/crafts/${background.slice(6)}.svg`
				: `/profile_backgrounds/${background}.png`
	);
	let allowsPlus = $state(false);
	let canSetPolicy = $state(false),
		chatState = $state('disabled');
	let incoming = $state<'everyone' | 'following' | 'nobody'>('everyone'),
		push = $state(false),
		connected = $state(false);
	const notificationLabels: Record<string, string> = {
		like: 'Likes',
		follow: 'New followers',
		reply: 'Replies',
		mention: 'Mentions',
		quote: 'Quotes',
		repost: 'Reposts',
		instagramMatch: 'Instagram matches',
		moderation: 'Moderation',
		everythingElse: 'Everything else'
	};
	function load() {
		if (section === 'notifications')
			return work.load(
				() => api.notifications.preferences(),
				(value) => {
					preferences = value;
				}
			);
		if (section === 'languages')
			return work.load(
				() => api.languages.get(),
				(value) => {
					primary = value.primaryLanguage;
					content = value.contentLanguages.join(', ');
				}
			);
		if (section === 'customisation')
			return work.load(
				() => Promise.all([api.profiles.me(), api.billing.access()]),
				([profile, access]) => {
					const custom = record(profile.customisation);
					colour = text(custom.colour) || 'cobalt';
					background =
						text(custom.profileBackground) || text(custom.background) || 'none';
					allowsPlus = access.givesAccess && access.effectiveTier !== 'free';
				}
			);
		if (section === 'messaging')
			return work.load(
				async () => {
					const status = await api.chat.status();
					const [policy, prefs] = await Promise.all([
						status.capabilities.canSetPolicy
							? api.chat.incoming()
							: Promise.resolve(null),
						api.chat.pushPreference()
					]);
					return [status, policy, prefs] as const;
				},
				([status, policy, prefs]) => {
					connected = status.enabled;
					chatState = status.state;
					canSetPolicy = status.capabilities.canSetPolicy;
					incoming = policy?.allowIncoming || 'everyone';
					push = prefs.pushEnabled;
				}
			);
		work.loading = false;
	}
	onMount(() => {
		try {
			theme = localStorage.getItem('craftsky.theme') || 'system';
			diagnostics = consent();
		} catch {}
		load();
	});
	onDestroy(() => work.dispose());
	function setTheme(value: string) {
		theme = value;
		document.documentElement.dataset.theme = value;
		try {
			localStorage.setItem('craftsky.theme', value);
		} catch {}
		work.notice = 'Appearance updated.';
	}
	function saveLanguages(event: SubmitEvent) {
		event.preventDefault();
		try {
			const value: LanguagePreferences = languagePreferences(primary, content);
			void work.run(
				() => api.languages.replace(value),
				(result) => {
					primary = result.primaryLanguage;
					content = result.contentLanguages.join(', ');
				}
			);
		} catch (error) {
			work.error =
				error instanceof Error ? error.message : 'Check your languages.';
		}
	}
	function saveCustomisation(event: SubmitEvent) {
		event.preventDefault();
		const value = { colour, profileBackground: background };
		void work.run(() =>
			api.transport.request('PUT', '/v1/profiles/me/customisation', {
				body: value
			})
		);
	}
	function notification(
		category: string,
		field: 'scope' | 'pushEnabled',
		value: string | boolean
	) {
		void work.run(
			() =>
				api.notifications.patchPreferences({
					preferences: { [category]: { [field]: value } }
				}),
			(result) => {
				preferences = result;
			}
		);
	}
</script>

<h1>
	{section === 'customisation' ? 'Profile customisation' : title(section)}
</h1>
<Feedback {work} reload={load} />
{#if section === 'appearance'}
	<section class="panel">
		<h2>Colour theme</h2>
		<p class="muted">Choose how CraftSky looks on this browser.</p>
		<div class="stack">
			{#each [['system', 'Use device setting'], ['light', 'Light'], ['dark', 'Dark']] as [value, label]}<label
					class="check"
					><input
						type="radio"
						name="theme"
						checked={theme === value}
						onchange={() => setTheme(value)}
					/>{label}</label
				>{/each}
		</div>
	</section>
{:else if section === 'privacy'}
	<section class="panel">
		<h2>Error reporting</h2>
		{#if diagnosticsConfigured()}<label class="check"
				><input
					type="checkbox"
					bind:checked={diagnostics}
					onchange={() => {
						try {
							setConsent(diagnostics);
							work.notice = 'Privacy preference saved.';
						} catch {
							work.error = 'This browser could not save your preference.';
						}
					}}
				/>Allow optional diagnostics on this browser</label
			>
			<p class="muted">
				Help improve CraftSky by allowing technical error reports. Your posts,
				messages and credentials are never included.
			</p>
		{:else}<p class="muted">
				Optional error reporting is not configured for this web client.
			</p>{/if}
		<a href="https://craftsky.social/privacy" target="_blank" rel="noreferrer"
			>Privacy policy ↗</a
		>
	</section>
{:else if !work.loading && section === 'languages'}
	<form class="panel stack" onsubmit={saveLanguages}>
		<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
			<h2>Your languages</h2>
			<p class="muted">
				App language: English. CraftSky currently offers its interface in
				English.
			</p>
			<label
				>Primary language<select required bind:value={primary}
					>{#each languageOptions as language}<option value={language.tag}
							>{language.name}</option
						>{/each}</select
				></label
			>
			<fieldset>
				<legend>Content languages</legend><label
					>Find a language<input
						type="search"
						bind:value={languageSearch}
						placeholder="Search languages"
					/></label
				>
				<p class="muted">
					{content
						? content
								.split(', ')
								.map(
									(tag) =>
										languageOptions.find((value) => value.tag === tag)?.name ||
										tag
								)
								.join(', ')
						: 'All languages'}
				</p>
				<div class="language-list">
					{#each languageMatches as language}<label class="check"
							><input
								type="checkbox"
								checked={content.split(', ').includes(language.tag)}
								onchange={() => toggleLanguage(language.tag)}
							/>{language.name}</label
						>{/each}
				</div>
			</fieldset>
			<p class="muted">
				These preferences help label your posts and find content you can read.
				Leave content languages empty to include all languages.
			</p>
			<button class="primary" disabled={work.busy}>Save languages</button>
		</fieldset>
	</form>
{:else if !work.loading && section === 'notifications'}
	<PushSettings {api} {session} />
	<section class="panel">
		<h2>Activity notifications</h2>
		<p class="muted">
			Choose whose activity appears and which categories can send push
			notifications to your registered devices.
		</p>
		{#each Object.entries(preferences.preferences) as [category, pref]}<div
				class="row"
			>
				<div>
					<h3>{notificationLabels[category] || title(category)}</h3>
					{#if !['moderation', 'instagramMatch'].includes(category)}<label
							><span class="muted">Notify me about</span><select
								value={pref.scope}
								disabled={work.busy}
								onchange={(e) =>
									notification(category, 'scope', e.currentTarget.value)}
								><option value="everyone">Everyone</option><option
									value="peopleIFollow">People I follow</option
								></select
							></label
						>{/if}
				</div>
				<label class="check"
					><input
						type="checkbox"
						checked={pref.pushEnabled}
						disabled={work.busy}
						onchange={(e) =>
							notification(category, 'pushEnabled', e.currentTarget.checked)}
					/>Push</label
				>
			</div>{/each}
	</section>
{:else if !work.loading && section === 'customisation'}
	{#if !allowsPlus}<section class="panel">
			<h2>Make your profile your own</h2>
			<p>
				Profile colours and backgrounds are included with CraftSky Plus and
				Business. Manage purchases in the mobile app.
			</p>
			<a class="action" href="/app/settings/subscriptions"
				>View subscription access</a
			>
		</section>
	{:else}<form class="panel stack" onsubmit={saveCustomisation}>
			<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
				<fieldset>
					<legend>Profile colour</legend>
					<div class="pills">
						{#each Object.entries(colours) as [value, hex]}<label class="check"
								><input
									type="radio"
									name="colour"
									bind:group={colour}
									{value}
								/><span class="swatch" style={`background:${hex}`}
								></span>{title(value)}</label
							>{/each}
					</div>
				</fieldset>
				<label
					>Profile background<select bind:value={background}
						>{#each backgrounds as value}<option {value}>{title(value)}</option
							>{/each}</select
					></label
				>
				<div class="preview" style={`--chosen-colour:${colours[colour]}`}>
					{#if pattern}<div
							class="preview-pattern"
							aria-hidden="true"
							style={`background-image:url('${pattern}');background-size:${background.startsWith('craft-') ? '100px' : '160px'}`}
						></div>{/if}
					<strong>{session.handle}</strong><span
						>{background === 'none'
							? 'Your craft, your space.'
							: title(background)}</span
					>
				</div>
				<button class="primary" disabled={work.busy}>Save customisation</button>
			</fieldset>
		</form>{/if}
{:else if !work.loading && section === 'messaging'}
	<section class="panel stack">
		<h2>Direct messages</h2>
		<p>
			{chatState === 'ready'
				? 'Messaging is connected to your account.'
				: chatState === 'authorizationNeeded'
					? 'Verify your identity to finish connecting messaging.'
					: chatState === 'unavailable'
						? 'Messaging is temporarily unavailable.'
						: chatState === 'restricted'
							? 'Messaging is restricted for this account.'
							: 'Messaging is not connected.'}
		</p>
		<a class="action" href="/app/messages"
			>{connected ? 'Open messages' : 'Connect messaging'}</a
		>{#if connected}{#if canSetPolicy}<label
					>Who can send message requests?<select
						bind:value={incoming}
						disabled={work.busy}
						onchange={() => work.run(() => api.chat.setIncoming(incoming))}
						><option value="everyone">Everyone</option><option value="following"
							>People I follow</option
						><option value="nobody">Nobody</option></select
					></label
				>{/if}<label class="check"
				><input
					type="checkbox"
					bind:checked={push}
					disabled={work.busy}
					onchange={() => work.run(() => api.chat.setPushPreference(push))}
				/>Message notifications on registered mobile devices</label
			><button
				class="danger"
				disabled={work.busy}
				onclick={() => {
					if (
						confirm(
							'Disconnect direct messaging? Your messages remain on the messaging service.'
						)
					)
						void work.run(
							() => api.chat.setConnection(false),
							() => {
								connected = false;
								chatState = 'disabled';
							},
							'Messaging disconnected.'
						);
				}}>Disconnect messaging</button
			>{/if}
	</section>
{/if}

<style>
	.swatch {
		height: 18px;
		width: 18px;
		border: 1px solid var(--ink);
		border-radius: 50%;
	}
	.preview {
		position: relative;
		overflow: hidden;
		border-radius: 12px;
		background: var(--chosen-colour);
		color: white;
		padding: 2rem;
		display: grid;
		gap: 0.5rem;
	}
	.preview strong,
	.preview span {
		position: relative;
		z-index: 1;
	}
	.preview-pattern {
		position: absolute;
		inset: 0;
		opacity: 0.18;
		pointer-events: none;
	}
	.preview strong {
		font-size: 1.5rem;
	}
	.language-list {
		max-height: 320px;
		overflow: auto;
		display: grid;
		gap: 0.7rem;
		padding: 0.5rem 0;
	}
</style>
