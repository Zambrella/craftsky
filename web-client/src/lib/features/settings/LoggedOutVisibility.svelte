<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { CraftskyAPI } from '#lib/api/client.ts';
	import { LoggedOutVisibility } from './logged-out-visibility.svelte';
	let { api }: { api: CraftskyAPI } = $props();
	const model = new LoggedOutVisibility({
		read: () => api.loggedOutVisibility.read(),
		save: (body, key) => api.loggedOutVisibility.save(body, key)
	});
	onMount(() => {
		void model.load();
	});
	onDestroy(() => model.dispose());
</script>

<section
	class="visibility-setting"
	aria-label="Logged-out visibility"
	aria-busy={model.busy}
>
	<label class="switch-row native-option"
		><span class="tile-copy"
			><span>Ask apps and sites not to show my account to logged-out users</span
			><small
				>CraftSky and Bluesky will not show your account to logged-out users and
				will ask other apps to do the same. Other apps may not honour this
				request. It doesn't make your account private.</small
			></span
		>
		{#if model.value}<input
				type="checkbox"
				role="switch"
				aria-label="Ask apps and sites not to show my account to logged-out users"
				checked={model.value.hideFromLoggedOut}
				disabled={model.busy || !!model.pending}
				onchange={(event) => {
					const desired = event.currentTarget.checked;
					event.currentTarget.checked = model.value!.hideFromLoggedOut;
					void model.setHidden(desired);
				}}
			/>{:else}<span
				>{model.busy ? 'Loading preference…' : 'Preference unavailable'}</span
			>{/if}</label
	>
	<p>
		This is the shared Bluesky profile preference. Changing it here also changes
		it on Bluesky.
	</p>
	<p>
		CraftSky is part of the Atmosphere, an open public network. Other apps and
		websites may choose not to honour this request. Your public content may
		still appear elsewhere. Previously cached link previews may remain visible.
	</p>
	{#if model.error}<p role="alert">{model.error}</p>{/if}
	{#if model.notice}<p role="status">{model.notice}</p>{/if}
	{#if model.pending}<p role="status">Save pending confirmation.</p>
		<button class="button" disabled={model.busy} onclick={() => model.retry()}
			>Retry save</button
		>{:else if model.error}<button
			class="button"
			disabled={model.busy}
			onclick={() => model.load()}>Reload preference</button
		>{/if}
</section>

<style>
	.visibility-setting {
		margin-block: 20px;
	}
	.visibility-setting p {
		font-size: 14px;
		margin: 12px 20px;
		line-height: 1.5;
	}
</style>
