<script lang="ts">
	import Icon from '../shell/Icon.svelte';
	import { onDestroy, onMount, tick } from 'svelte';
	import { CraftskyAPI, type OnboardingStatus } from '#lib/api/client.ts';
	import { Work } from './work.svelte';
	import Status from './Status.svelte';
	import { setConsent, consent } from '../diagnostics/client.ts';
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
	const work = new Work('social.Onboarding.action');
	let editorBusy = $state(false);
	let age = $state(false),
		diagnostics = $state(false),
		step = $state(0);
	let scroller: HTMLDivElement;
	onMount(() => {
		diagnostics = consent();
	});
	$effect(() => {
		void step;
		void tick().then(() => {
			if (scroller) scroller.scrollTop = 0;
		});
	});
	onDestroy(() => work.dispose());
	function finish() {
		void work.run(
			() => api.account.completeOnboarding(age, status.requiredPolicyVersion),
			() => {
				try {
					setConsent(diagnostics);
				} catch {}
				onComplete();
			},
			undefined,
			'social.Onboarding.finish'
		);
	}
</script>

<div class="onboarding">
	<header class="onboarding-header">
		{#if step > 0}<button
				class="icon-button"
				aria-label="Previous onboarding step"
				disabled={editorBusy || work.busy}
				onclick={() => step--}><Icon name="back" /></button
			>{/if}
		<h1>Set up your CraftSky profile</h1>
	</header>
	<div class="progress-area">
		<p>Step {step + 1} of 3</p>
		<div
			class="step-progress"
			role="progressbar"
			aria-label="Profile setup progress"
			aria-valuemin="1"
			aria-valuemax="3"
			aria-valuenow={step + 1}
		>
			<span style:width={((step + 1) / 3) * 100 + '%'}></span>
		</div>
	</div>
	<div class="onboarding-scroll" bind:this={scroller}>
		<div hidden={step === 2}>
			{#await import('./ProfileEditor.svelte')}<p role="status">
					Loading your profile…
				</p>{:then Module}
				<Module.default
					{api}
					{session}
					scope={step === 0 ? 'identity' : 'crafts'}
					onBusyChange={(value) => (editorBusy = value)}
					onSaved={() => step++}
				/>
			{:catch}<p role="alert">
					Your profile could not load. Reload this page to try again.
				</p>{/await}
		</div>
		{#if step === 2}<section class="guidelines stack">
				<h1>Our community guidelines</h1>
				<h2>A welcoming place to make and share</h2>
				<p>
					CraftSky exists to be a welcoming, inspiring home for the textile
					crafting community. We want it to be a place where people feel
					comfortable sharing projects, asking questions, celebrating each
					other's successes and learning from the community.
				</p>
				<h2>Be kind and keep CraftSky safe</h2>
				<p>
					Respect other people and help keep CraftSky safe for everyone. We do
					not allow harassment, hate speech, scams, spam, sexually explicit
					content or content materially created by generative AI. Businesses are
					welcome when they are transparent about commercial relationships and
					follow the same rules as everyone else.
				</p>
				<h2>Keep it craft-focused</h2>
				<p>
					Most posts should relate to textile and fibre craft projects,
					materials, techniques, tools or the wider crafting community.
					Occasional off-topic posts are fine, but should remain secondary.
					Political discussion belongs here only when it directly affects
					textile crafting or the craft community.
				</p>
				<h2>A useful check</h2>
				<p>
					If you are unsure whether something belongs on CraftSky, ask: Does
					this make the community a better place? If the answer is yes, you are
					probably on the right track.
				</p>
				<a
					href="https://docs.google.com/document/d/1BXqycv4IvnsVGWAhK1FZ94XF6hrNiW4FVFFtfjGyT94/edit?usp=sharing"
					target="_blank"
					rel="noreferrer"
					class="secondary-button">View full guidelines ↗</a
				><label class="check"
					><input type="checkbox" bind:checked={age} />I confirm that I am at
					least 16 years old and accept the current CraftSky policies.</label
				><label class="check"
					><input type="checkbox" bind:checked={diagnostics} />Enable error
					reporting (optional)</label
				>
				<p class="muted">
					Send crash reports and diagnostic logs to help improve CraftSky.
				</p>
				<Status {work} />
				<div class="onboarding-action">
					<button class="button" disabled={!age || work.busy} onclick={finish}
						>{work.busy ? 'Finishing…' : 'Finish'}</button
					>
				</div>
			</section>{/if}
	</div>
</div>

<style>
	.onboarding {
		height: 100dvh;
		display: flex;
		flex-direction: column;
		width: 100%;
		max-width: 768px;
		margin: 0 auto;
	}
	.onboarding-header {
		display: flex;
		flex: none;
		align-items: center;
		gap: 12px;
		height: 64px;
		padding: 0 16px;
		border-bottom: 1.5px solid var(--ink);
	}
	.onboarding-header h1 {
		font:
			600 20px/1.25 Outfit,
			sans-serif;
		margin: 0;
	}
	.progress-area {
		padding: 12px 24px 0;
	}
	.progress-area p {
		font-size: 14px;
		font-weight: 600;
		margin: 0 0 8px;
	}
	.step-progress {
		height: 4px;
		background: var(--red-soft);
	}
	.step-progress span {
		display: block;
		height: 100%;
		background: var(--primary);
	}
	.onboarding-scroll {
		flex: 1;
		min-height: 0;
		overflow: auto;
		padding: 24px 24px 110px;
	}
	.guidelines {
		gap: 0;
	}
	.guidelines h1 {
		font:
			700 28px/1.25 Outfit,
			sans-serif;
		margin: 0 0 24px;
	}
	.guidelines h2 {
		font-size: 16px;
		font-weight: 600;
		margin: 0 0 6px;
	}
	.guidelines p {
		font-size: 16px;
		line-height: 1.5;
		margin: 0 0 20px;
	}
	.guidelines > a {
		align-self: start;
		margin: 4px 0 24px;
	}
	.guidelines .check {
		align-items: flex-start;
		padding: 12px 0;
		font-weight: 400;
	}
	.onboarding-action {
		position: fixed;
		bottom: 0;
		left: 50%;
		transform: translateX(-50%);
		width: min(100%, 768px);
		padding: 12px 24px calc(24px + env(safe-area-inset-bottom));
		background: var(--paper);
		z-index: 5;
	}
	.onboarding-action button {
		width: 100%;
	}
	.onboarding-action button:disabled {
		opacity: 1;
		background: color-mix(in srgb, var(--ink) 32%, var(--paper));
		border-color: var(--ink-4);
		box-shadow: 3px 3px var(--ink-4);
	}
</style>
