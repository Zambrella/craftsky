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
	<header class="onboarding-header">
		<p>Set up your CraftSky profile</p>
		<span>Step {step + 1} of 3</span>
	</header>
	<div
		class="step-progress"
		role="progressbar"
		aria-label="Profile setup progress"
		aria-valuemin="1"
		aria-valuemax="3"
		aria-valuenow={step + 1}
	>
		{#each [0, 1, 2] as item}<span class:active={item <= step}></span>{/each}
	</div>
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
		>{:else}<section class="guidelines stack">
			<h1>Our community guidelines</h1>
			<h2>A welcoming place to make and share</h2>
			<p>
				CraftSky exists to be a welcoming, inspiring home for the textile
				crafting community. We want it to be a place where people feel
				comfortable sharing projects, asking questions, celebrating each other's
				successes and learning from the community.
			</p>
			<h2>Be kind and keep CraftSky safe</h2>
			<p>
				Respect other people and help keep CraftSky safe for everyone. We do not
				allow harassment, hate speech, scams, spam, sexually explicit content or
				content materially created by generative AI. Businesses are welcome when
				they are transparent about commercial relationships and follow the same
				rules as everyone else.
			</p>
			<h2>Keep it craft-focused</h2>
			<p>
				Most posts should relate to textile and fibre craft projects, materials,
				techniques, tools or the wider crafting community. Occasional off-topic
				posts are fine, but should remain secondary. Political discussion
				belongs here only when it directly affects textile crafting or the craft
				community.
			</p>
			<h2>A useful check</h2>
			<p>
				If you are unsure whether something belongs on CraftSky, ask: Does this
				make the community a better place? If the answer is yes, you are
				probably on the right track.
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

<style>
	.onboarding {
		max-width: 640px;
		margin: 0 auto;
		padding-bottom: 24px;
	}
	.onboarding-header {
		padding: 20px 16px 12px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
	}
	.onboarding-header p {
		font-size: 16px;
		font-weight: 600;
		margin: 0;
	}
	.onboarding-header span {
		font-size: 13px;
		color: var(--ink-2);
		flex-shrink: 0;
	}
	.step-progress {
		display: flex;
		gap: 6px;
		padding: 0 16px 16px;
	}
	.step-progress span {
		height: 4px;
		border-radius: 2px;
		flex: 1;
		background: var(--hairline);
	}
	.step-progress .active {
		background: var(--primary);
	}
	.guidelines {
		padding: 16px;
		gap: 12px;
	}
	.guidelines h1 {
		font-size: 28px;
		margin-bottom: 8px;
	}
	.guidelines h2 {
		font-size: 18px;
		margin-top: 8px;
	}
	.guidelines p {
		font-size: 15px;
		line-height: 1.55;
	}
	.guidelines > a {
		font-size: 14px;
	}
	.guidelines .check {
		align-items: flex-start;
		padding: 12px 0;
	}
	.guidelines .actions {
		padding-top: 12px;
		justify-content: space-between;
	}
</style>
