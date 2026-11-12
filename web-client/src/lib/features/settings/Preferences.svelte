<script lang="ts">
	import ProfileShaderBackground from '../social/ProfileShaderBackground.svelte';
	import { profileShaderStyle } from '../social/profile-background-shader.ts';
	import { UserFacingError, userErrorMessage } from '#lib/user-facing-error.ts';
	import LoggedOutVisibility from './LoggedOutVisibility.svelte';

	import { fieldError } from '#lib/field-validation.ts';
	import SingleSelect from '#lib/SingleSelect.svelte';
	import { confirmNavigation } from '../social/navigation-prompt';
	import { onMount, onDestroy } from 'svelte';

	import Avatar from '#lib/Avatar.svelte';
	import MultiSelect from '#lib/MultiSelect.svelte';
	import SettingsIcon from './SettingsIcon.svelte';
	import { effectiveCustomisation } from './native-settings.ts';
	import {
		backgroundLabels,
		notificationCategories,
		notificationLabels
	} from './presentation.ts';
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
		title,
		type Session
	} from './model.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import PushSettings from '#lib/features/push/PushSettings.svelte';
	import { consent, setConsent } from '#lib/features/diagnostics/client.ts';
	let {
		api,
		section,
		session
	}: { api: CraftskyAPI; section: string; session: Session } = $props();
	const work = new Work('settings.Preferences.action');
	let loaded = $state(false);
	let theme = $state('system'),
		diagnostics = $state(false);
	let preferences = $state<NotificationPreferences>({ preferences: {} });
	let primary = $state('en'),
		content = $state('');
	let colour = $state('cobalt'),
		background = $state('none');
	let savedColour = $state('cobalt'),
		savedBackground = $state('none'),
		profileAvatar = $state(''),
		profileName = $state('');
	let customDirty = $derived(
		colour !== savedColour || background !== savedBackground
	);
	confirmNavigation(
		() => section === 'customisation' && customDirty && !work.busy,
		{
			title: 'Discard customisation changes?',
			detail: "Your customisation changes won't be saved.",
			confirmLabel: 'Discard',
			cancelLabel: 'Keep editing',
			danger: true
		}
	);
	let pattern = $derived(
		background === 'none' || profileShaderStyle(background) !== undefined
			? ''
			: background.startsWith('craft-')
				? `/profile_backgrounds/${background}.svg`
				: `/profile_backgrounds/${background}.png`
	);
	let allowsPlus = $state(false);
	let canSetPolicy = $state(false),
		chatState = $state('disabled');
	let incoming = $state<'everyone' | 'following' | 'nobody'>('everyone'),
		push = $state(false),
		connected = $state(false);

	function load() {
		if (section === 'notifications')
			return work.load(
				() => api.notifications.preferences(),
				(value) => {
					preferences = value;
					loaded = true;
				},
				'settings.Preferences.load'
			);
		if (section === 'languages')
			return work.load(
				() => api.languages.get(),
				(value) => {
					loaded = true;
					primary = value.primaryLanguage;
					content = value.contentLanguages.join(', ');
				},
				'settings.Preferences.load'
			);
		if (section === 'customisation')
			return work.load(
				() => Promise.all([api.profiles.me(), api.billing.access()]),
				([profile, access]) => {
					loaded = true;
					const custom = effectiveCustomisation(profile.customisation);
					colour = custom.colour;
					background = custom.background;
					savedColour = colour;
					savedBackground = background;
					profileAvatar = profile.avatar || '';
					profileName = profile.displayName || '';
					allowsPlus = access.givesAccess && access.effectiveTier !== 'free';
				},
				'settings.Preferences.load'
			);
		if (section === 'messaging')
			return work.load(
				async () => {
					const status = await api.chat.status();
					const [policy, prefs] = await Promise.all([
						status.enabled && status.capabilities.canSetPolicy
							? api.chat.incoming()
							: Promise.resolve(null),
						status.enabled
							? api.chat.pushPreference()
							: Promise.resolve({ pushEnabled: status.pushEnabled })
					]);
					return [status, policy, prefs] as const;
				},
				([status, policy, prefs]) => {
					loaded = true;
					connected = status.enabled;
					chatState = status.state;
					canSetPolicy = status.capabilities.canSetPolicy;
					incoming = policy?.allowIncoming || 'everyone';
					push = prefs.pushEnabled;
				},
				'settings.Preferences.load'
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
	let disposed = false;
	onDestroy(() => {
		disposed = true;
		work.dispose();
	});
	function setTheme(value: string) {
		theme = value;
		document.documentElement.dataset.theme = value;
		try {
			localStorage.setItem('craftsky.theme', value);
		} catch {}
	}
	async function replaceLanguages(nextPrimary: string, nextContent: string) {
		if (work.busy || work.retry) return;
		const previous = { primary, content };
		try {
			const value: LanguagePreferences = languagePreferences(
				nextPrimary,
				nextContent
			);
			primary = value.primaryLanguage;
			content = value.contentLanguages.join(', ');
			await work.run(
				() => api.languages.replace(value),
				(result) => {
					primary = result.primaryLanguage;
					content = result.contentLanguages.join(', ');
				},
				undefined,
				'settings.Preferences.replaceLanguages'
			);
			if (!disposed && work.error) {
				primary = previous.primary;
				content = previous.content;
			}
		} catch (error) {
			fieldError('Primary language', userErrorMessage(error));
		}
	}
	function saveCustomisation(event: SubmitEvent) {
		event.preventDefault();
		const value = { colour, profileBackground: background };
		void work.run(
			() =>
				api.transport.request('PUT', '/v1/profiles/me/customisation', {
					body: value
				}),
			() => {
				savedColour = value.colour;
				savedBackground = value.profileBackground;
			},
			'Profile customisation saved.',
			'settings.Preferences.saveCustomisation'
		);
	}
	async function setMessaging(enabled: boolean) {
		if (work.busy || work.retry) return;
		const previous = connected;
		let confirmed = false;
		connected = enabled;
		await work.run(
			async () => {
				const status = await api.chat.setConnection(enabled);
				if (status.enabled !== enabled)
					throw new UserFacingError(
						'Messaging connection change was not confirmed. Try again.'
					);
				const [policy, preference] = await Promise.allSettled([
					status.enabled && status.capabilities.canSetPolicy
						? api.chat.incoming()
						: Promise.resolve(null),
					status.enabled
						? api.chat.pushPreference()
						: Promise.resolve({ pushEnabled: status.pushEnabled })
				]);
				return { status, policy, preference };
			},
			({ status, policy, preference }) => {
				confirmed = true;
				connected = status.enabled;
				chatState = status.state;
				canSetPolicy = status.capabilities.canSetPolicy;
				push =
					preference.status === 'fulfilled'
						? preference.value.pushEnabled
						: status.pushEnabled;
				incoming =
					policy.status === 'fulfilled'
						? policy.value?.allowIncoming || 'everyone'
						: 'everyone';
				if (policy.status === 'rejected' || preference.status === 'rejected')
					work.error = 'Some messaging settings could not load. Try again.';
			},
			'',
			'settings.Preferences.setMessaging'
		);
		if (!disposed && !confirmed) connected = previous;
	}
	async function changePush(next: boolean) {
		if (work.busy || work.retry) return;
		const previous = push;
		push = next;
		await work.run(
			() => api.chat.setPushPreference(next),
			(saved) => {
				push = saved.pushEnabled;
			},
			'',
			'settings.Preferences.changePush'
		);
		if (!disposed && work.error) push = previous;
	}
	async function changeIncoming(next: 'everyone' | 'following' | 'nobody') {
		if (work.busy || work.retry) return;
		const previous = incoming;
		incoming = next;
		await work.run(
			() => api.chat.setIncoming(next),
			(saved) => {
				incoming = saved.allowIncoming;
			},
			'',
			'settings.Preferences.changeIncoming'
		);
		if (!disposed && work.error) incoming = previous;
	}
	async function notification(
		category: string,
		field: 'scope' | 'pushEnabled',
		value: string | boolean
	) {
		if (work.busy || work.retry) return;
		const previous = preferences;
		const pref = preferences.preferences[category];
		if (!pref) return;
		const next =
			field === 'scope'
				? { ...pref, scope: value as typeof pref.scope }
				: { ...pref, pushEnabled: value === true };
		preferences = {
			...preferences,
			preferences: { ...preferences.preferences, [category]: next }
		};
		await work.run(
			() =>
				api.notifications.patchPreferences({
					preferences: { [category]: { [field]: value } }
				}),
			(result) => {
				preferences = result;
			},
			undefined,
			'settings.Preferences.notification'
		);
		if (!disposed && work.error) preferences = previous;
	}
</script>

<h1 class="workspace-heading">
	{section === 'customisation'
		? 'Customisation'
		: section === 'notifications'
			? 'Notification settings'
			: section === 'messaging'
				? 'Direct messages'
				: title(section)}
</h1>
<Feedback {work} reload={load} />
{#if section === 'appearance'}
	<div class="native-list" role="radiogroup" aria-label="Colour theme">
		{#each [['system', 'Use device setting'], ['light', 'Light'], ['dark', 'Dark']] as [value, label]}<label
				class="check native-option"
				><input
					type="radio"
					name="theme"
					checked={theme === value}
					onchange={() => setTheme(value)}
				/><span>{label}</span><SettingsIcon name={value} /></label
			>{/each}
	</div>
{:else if section === 'privacy'}
	<div class="native-list">
		<label class="switch-row native-option"
			><SettingsIcon name="errorReporting" /><span class="tile-copy"
				><span>Error reporting</span><small
					>Send crash reports and diagnostic logs to help improve CraftSky.</small
				></span
			><input
				type="checkbox"
				role="switch"
				aria-label="Error reporting"
				checked={diagnostics}
				onchange={(event) => {
					try {
						setConsent(event.currentTarget.checked);
						diagnostics = event.currentTarget.checked;
					} catch {
						work.error = 'This browser could not save your preference.';
					}
				}}
			/></label
		>
	</div>
	<LoggedOutVisibility {api} />
{:else if !work.loading && loaded && section === 'languages'}
	<section class="panel stack">
		<SingleSelect
			label="App language"
			helperText="Select which language to use for the app's user interface."
			value="en"
			required
			showRequired={false}
			options={[{ id: 'en', label: 'English' }]}
			onchange={() => {}}
		/>
		<p class="helper section-heading">
			<SettingsIcon name="translate" size={18} />More app languages are coming.
		</p>
	</section>
	<section class="panel stack">
		<SingleSelect
			label="Primary language"
			helperText="Select the default language used when you create a post."
			value={primary}
			required
			showRequired={false}
			options={languageOptions.map((language) => ({
				id: language.tag,
				label: language.name,
				description: language.tag
			}))}
			disabled={work.busy || !!work.retry}
			onchange={(value) => replaceLanguages(value, content)}
		/>
	</section>
	<section class="panel stack">
		<MultiSelect
			label="Content languages"
			helperText="Select which languages you want posts in your feeds and discovery results to include. If none are selected, all languages will be shown."
			placeholder="Add more languages…"
			values={content.split(', ').filter(Boolean)}
			options={languageOptions.map((value) => ({
				id: value.tag,
				label: value.name,
				description: value.tag
			}))}
			maximum={50}
			disabled={work.busy || !!work.retry}
			onchange={(values) => {
				replaceLanguages(primary, values.join(', '));
			}}
		/>
	</section>
{:else if !work.loading && loaded && section === 'notifications'}
	<p class="muted notification-intro">
		Category preferences apply to all devices signed in to this account.
	</p>
	<PushSettings {api} {session} />
	{#each notificationCategories as category}{@const pref =
			preferences.preferences[category]}{#if pref}<section
				class="panel notification-card"
			>
				<h2 class="section-heading">
					<SettingsIcon
						name={category === 'instagramMatch'
							? 'instagram'
							: category === 'everythingElse'
								? 'notifications'
								: category}
					/>{notificationLabels[category]}
				</h2>
				{#if category === 'instagramMatch'}<p class="helper">
						Push alerts are based on your private Instagram matches and never
						name the matched account.
					</p>{:else if category === 'moderation'}<p class="helper">
						Push alerts notify you about changes to your account standing.
						Turning them off does not change your standing, moderation history,
						or enforcement.
					</p>{:else}<SingleSelect
						label="From"
						value={pref.scope}
						required
						showRequired={false}
						options={[
							{ id: 'everyone', label: 'Everyone' },
							{ id: 'peopleIFollow', label: 'People I follow' }
						]}
						disabled={work.busy}
						onchange={(value) => notification(category, 'scope', value)}
					/>{/if}
				<label class="switch-row notification-push"
					><span class="tile-copy">Push notifications</span><input
						type="checkbox"
						role="switch"
						aria-label={`${notificationLabels[category]} push notifications`}
						checked={pref.pushEnabled}
						disabled={work.busy}
						onchange={(event) =>
							notification(
								category,
								'pushEnabled',
								event.currentTarget.checked
							)}
					/></label
				>
			</section>{/if}{/each}
{:else if !work.loading && loaded && section === 'customisation'}
	{#if !allowsPlus}<section class="panel">
			<h2>Make your profile your own</h2>
			<p>
				Profile colours and backgrounds are included with CraftSky Plus and
				Business. Manage purchases in the mobile app.
			</p>
			<a class="action" href="/settings/subscriptions"
				>View subscription access</a
			>
		</section>
	{:else}<form class="stack customisation-form" onsubmit={saveCustomisation}>
			<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
				<div
					class="profile-preview"
					role="img"
					aria-label="Profile customisation preview"
					style={`--chosen-colour:${colours[colour]}`}
				>
					{#if profileShaderStyle(background) !== undefined}<ProfileShaderBackground
							style={profileShaderStyle(background)!}
						/>{/if}
					{#if pattern}<div
							class="preview-pattern"
							aria-hidden="true"
							style={`mask-image:url('${pattern}')`}
						></div>{/if}
					<Avatar
						author={{
							did: session.did,
							handle: session.handle,
							displayName: profileName,
							avatar: profileAvatar
						}}
						large
					/>
				</div>
				<fieldset>
					<legend>Colour</legend>
					<div class="pills colour-choices">
						{#each Object.entries(colours) as [value, hex]}<label
								class="check"
								style={colour === value ? `background:${hex};color:white` : ''}
								><input
									type="radio"
									name="colour"
									bind:group={colour}
									{value}
								/>{value === 'lime'
									? 'Green'
									: value === 'dusty-pink'
										? 'Dusty pink'
										: title(value)}</label
							>{/each}
					</div>
				</fieldset>
				<fieldset>
					<legend>Profile background</legend>
					<div class="pills">
						{#each backgrounds as value}<label class="check"
								><input
									type="radio"
									name="background"
									bind:group={background}
									{value}
								/>{backgroundLabels[value] || title(value)}</label
							>{/each}
					</div>
				</fieldset>
				<button
					class="primary customisation-save"
					disabled={work.busy || !customDirty}>Save</button
				>
			</fieldset>
		</form>{/if}
{:else if !work.loading && loaded && section === 'messaging'}
	<div class="stack messaging-preferences">
		{#if !connected}<div class="stack">
				<p>
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
				</p>
			</div>{/if}
		<label class="switch-row"
			><span class="tile-copy"
				><span>Enable messaging</span><small
					>Use messaging in CraftSky. Turning it off stops CraftSky message
					access and push notifications.</small
				></span
			><input
				type="checkbox"
				role="switch"
				checked={connected}
				disabled={work.busy || !!work.retry}
				onchange={(event) => setMessaging(event.currentTarget.checked)}
			/></label
		>
		{#if chatState === 'authorizationNeeded'}<p class="notice">
				Verify your identity to finish connecting messaging.
			</p>
			<a class="action" href="/messages">Review permissions</a
			>{:else if chatState === 'unavailable'}<p class="notice">
				Messaging is temporarily unavailable.
			</p>{:else if chatState === 'restricted'}<p class="notice">
				Messaging is restricted for this account.
			</p>{/if}
		{#if connected}<label class="switch-row"
				><span class="tile-copy"
					><span>Push notifications</span><small
						>Receive push notifications about new messages.</small
					></span
				><input
					type="checkbox"
					role="switch"
					checked={push}
					disabled={work.busy || !!work.retry}
					onchange={(event) => changePush(event.currentTarget.checked)}
				/></label
			>
			<p class="muted">
				This setting is shared with Bluesky and other apps. Changes may take
				time to appear there and do not remove existing conversations.
			</p>
			{#if canSetPolicy}<SingleSelect
					label="Who can message me"
					value={incoming}
					required
					showRequired={false}
					options={[
						{ id: 'everyone', label: 'Everyone' },
						{ id: 'following', label: 'People I follow' },
						{ id: 'nobody', label: 'Nobody' }
					]}
					disabled={work.busy || !!work.retry}
					onchange={(value) => changeIncoming(value as typeof incoming)}
				/>{:else}<p class="helper">
					Reconnect with incoming-message settings permission to change this
					shared setting.
				</p>
				<a class="action" href="/messages">Review messaging permissions</a>{/if}
			<a class="action" href="/messages">Open messages</a>
		{/if}
	</div>
{/if}

<style>
	.profile-preview {
		height: 160px;
		position: relative;
		overflow: hidden;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--chosen-colour);
		border-radius: 0;
		margin-bottom: 8px;
	}
	.preview-pattern {
		position: absolute;
		inset: 0;
		opacity: 0.18;
		background: #fff;
		mask-repeat: repeat;
		mask-mode: alpha;
		image-rendering: pixelated;
		pointer-events: none;
	}
	.profile-preview :global(.avatar) {
		position: relative;
		width: 106px;
		height: 106px;
		border: 5px solid var(--chosen-colour);
		border-radius: 50%;
		box-shadow: none;
	}
	.profile-preview :global(.avatar > span) {
		font-size: 38px;
	}

	:global(.settings-workspace) .customisation-form .pills label {
		font-size: 13px;
		line-height: 1.3;
		font-weight: 600;
		min-height: 34px;
		padding: 6px 12px;
		gap: 0;
		position: relative;
	}
	:global(.settings-workspace) .customisation-form .pills input {
		position: absolute;
		width: 1px;
		height: 1px;
		opacity: 0;
	}
	:global(.settings-workspace)
		.customisation-form
		.pills
		label:has(input:focus-visible) {
		outline: 3px solid var(--primary);
		outline-offset: 3px;
	}
	:global(.settings-workspace[data-settings-page='languages']) label {
		font-size: 16px;
	}
	.profile-preview :global(.avatar[data-swatch='paper']) {
		background: var(--paper-2);
	}
	:global(.settings-workspace) .customisation-save {
		width: 100%;
		margin-top: 12px;
	}
	:global(.settings-workspace) .notification-intro {
		font-size: 16px;
	}
	:global(.settings-workspace) .notification-card > label {
		font-size: 16px;
	}
	.notification-push .tile-copy {
		font-weight: 600;
	}
	:global(.settings-workspace) .customisation-save:disabled {
		opacity: 1;
		background: color-mix(in srgb, var(--ink) 30%, var(--paper));
		color: var(--surface);
		border-color: color-mix(in srgb, var(--ink) 40%, var(--paper));
		box-shadow: 3px 3px 0 color-mix(in srgb, var(--ink) 20%, var(--paper));
	}
	.notification-intro {
		margin: 16px 0 24px;
	}
	:global(.settings-workspace) .notification-card h2 {
		margin-bottom: 16px;
	}
	:global(.settings-workspace) .notification-push {
		border-top: 1px solid var(--ink-4);
		padding-top: 8px;
		margin-top: 20px;
		min-height: 57px;
	}
	.notification-card > label:not(.notification-push) {
		margin-top: 24px;
	}
	.messaging-preferences {
		padding: 8px;
		gap: 24px;
	}
</style>
