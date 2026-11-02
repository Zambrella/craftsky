<script lang="ts">
	import { onDestroy } from 'svelte';
	import { CraftskyAPI, type OnboardingStatus } from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import {
		configured as diagnosticsConfigured,
		setConsent
	} from '../diagnostics/client.ts';
	let {
		api,
		session,
		status,
		onComplete
	}: {
		api: CraftskyAPI;
		session: { id: string; did: string; handle: string };
		status: OnboardingStatus;
		onComplete: () => void;
	} = $props();
	const work = new Work();
	let age = $state(false),
		diagnostics = $state(false),
		step = $state(0);
	onDestroy(() => work.dispose());
	function finish() {
		void work.run(
			() => api.account.completeOnboarding(age, status.requiredPolicyVersion),
			() => {
				try {
					if (diagnosticsConfigured()) setConsent(diagnostics);
				} catch {}
				onComplete();
			}
		);
	}
</script>

<div class="onboarding">
	<p class="eyebrow">Welcome to CraftSky · Step {step + 1} of 3</p>
	{#if step === 0}{#await import('./ProfileEditor.svelte')}<p
				class="empty"
				role="status"
			>
				Loading your profile…
			</p>{:then Module}<Module.default
				{api}
				{session}
				scope="identity"
				onSaved={() => (step = 1)}
			/>{:catch}<p role="alert">
				Your profile could not load. Reload this page to try again.
			</p>{/await}{:else if step === 1}{#await import('./ProfileEditor.svelte')}<p
				class="empty"
				role="status"
			>
				Loading your profile…
			</p>{:then Module}<Module.default
				{api}
				{session}
				scope="crafts"
				onSaved={() => (step = 2)}
			/>{:catch}<p role="alert">
				Your profile could not load. Reload this page to try again.
			</p>{/await}<button class="next" onclick={() => (step = 0)}>Back</button
		>{:else}<section class="panel stack">
			<h1>Our community guidelines</h1>
			<p>
				CraftSky is a welcoming place to share what you make, learn from each
				other, and celebrate textile crafts.
			</p>
			<h2>Be kind and respectful</h2>
			<p>
				Support other makers. Harassment, hate speech and discrimination have no
				place here.
			</p>
			<h2>Keep it about the craft</h2>
			<p>
				Share your own work and credit the designers and makers who inspire you.
				Business accounts and commercial relationships should be transparent.
			</p>
			<h2>Share thoughtfully</h2>
			<p>
				Check your posts before sharing. Respect other people’s privacy and
				report concerns so we can keep the community safe.
			</p>
			<a
				href="https://docs.google.com/document/d/1BXqycv4IvnsVGWAhK1FZ94XF6hrNiW4FVFFtfjGyT94/edit?usp=sharing"
				target="_blank"
				rel="noreferrer">Read the full community guidelines ↗</a
			><a href="https://craftsky.social/terms" target="_blank" rel="noreferrer"
				>Terms of service ↗</a
			><label class="check"
				><input type="checkbox" bind:checked={age} />I confirm that I am at
				least 16 years old and accept the current CraftSky policies.</label
			>{#if diagnosticsConfigured()}<label class="check"
					><input type="checkbox" bind:checked={diagnostics} />Allow optional
					error reporting to help improve CraftSky.</label
				>
				<p class="muted">
					You can change error reporting in Settings at any time.
				</p>
			{/if}
			<Status {work} />
			<div class="actions">
				<button disabled={work.busy} onclick={() => (step = 1)}>Back</button
				><button class="button" disabled={!age || work.busy} onclick={finish}
					>Join the community</button
				>
			</div>
		</section>{/if}
</div>
