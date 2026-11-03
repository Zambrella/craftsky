<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	let handle = $state(''),
		busy = $state(false),
		message = $state(''),
		ready = $state(false);
	onMount(() => {
		ready = true;
	});
	async function start(kind: 'login' | 'register') {
		busy = true;
		message = '';
		try {
			const response = await fetch('/api/auth/start', {
				method: 'POST',
				headers: { 'content-type': 'application/json' },
				body: JSON.stringify({ kind, handle })
			});
			const result = await response.json();
			if (!response.ok) {
				message = result.message || 'Sign-in is unavailable. Please try again.';
				return;
			}
			window.location.assign(result.authUrl);
		} catch {
			message = 'Could not connect. Please try again.';
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head
	><title>Sign in · CraftSky</title><meta
		name="robots"
		content="noindex"
	/></svelte:head
>
<div class="welcome-scaffold">
	<div class="welcome-crafts" aria-hidden="true">
		{#each ['knitting', 'sewing', 'crochet', 'quilting', 'embroidery'] as craft}<span
				style={'--craft-mask:url(/craft_icons/' + craft + '.svg)'}
			></span>{/each}
	</div>
	<section class="auth-card">
		<h1>Join CraftSky</h1>
		<p class="welcome-subtitle">
			Share what you make. Find people who make what you love.
		</p>
		{#if page.url.searchParams.get('error')}<p role="status" class="notice">
				{page.url.searchParams.get('error') === 'confirmation_pending'
					? 'Your sign-in is saved. Retry confirmation to finish.'
					: 'Sign-in could not be completed. You can try again.'}
			</p>{/if}
		{#if page.url.searchParams.get('error') === 'confirmation_pending'}<p>
				<a class="button" href="/auth/browser/confirm">Retry confirmation</a>
			</p>{/if}
		<button
			class="button register"
			onclick={() => start('register')}
			disabled={busy || !ready}>{busy ? 'Redirecting…' : 'Register'}</button
		>
		<p class="registration-handoff">
			You'll create your account with Bluesky, then return to CraftSky.
		</p>
		<div class="welcome-divider"><span>Or</span></div>
		<form
			onsubmit={(e) => {
				e.preventDefault();
				start('login');
			}}
		>
			<label for="handle">Your Atmosphere Handle</label><input
				id="handle"
				bind:value={handle}
				placeholder="your-handle.bsky.social"
				autocomplete="username"
				required
				disabled={busy || !ready}
			/><button class="button secondary-sign-in" disabled={busy || !ready}
				>{busy ? 'Redirecting…' : 'Sign in'}</button
			>
		</form>
		<details class="atmosphere-explainer">
			<summary>What is an Atmosphere account?</summary>
			<p>
				CraftSky is built on the AT Protocol, so your account, posts and social
				graph are portable. You can use an existing Bluesky or compatible
				Atmosphere account, or register a new one.
			</p>
		</details>
		<p class="welcome-legal">
			By continuing, you agree to our <a
				href="https://craftsky.social/terms"
				target="_blank"
				rel="noreferrer">Terms</a
			>
			and
			<a href="https://craftsky.social/privacy" target="_blank" rel="noreferrer"
				>Privacy Policy</a
			>.
		</p>
		{#if message}<p role="alert" class="notice error">{message}</p>{/if}
	</section>
</div>

<style>
	.welcome-scaffold {
		min-height: 100svh;
		display: grid;
		place-items: center;
		padding: 32px 16px;
		position: relative;
		overflow: hidden;
	}
	.welcome-crafts {
		position: absolute;
		inset: 0;
		pointer-events: none;
	}
	.welcome-crafts span {
		position: absolute;
		width: 112px;
		height: 112px;
		background: var(--primary);
		opacity: 0.07;
		mask: var(--craft-mask) center/contain no-repeat;
		transform: rotate(-14deg);
	}
	.welcome-crafts span:nth-child(1) {
		top: 3%;
		left: 3%;
	}
	.welcome-crafts span:nth-child(2) {
		top: 24%;
		right: -20px;
		transform: rotate(18deg);
	}
	.welcome-crafts span:nth-child(3) {
		top: 53%;
		left: -30px;
		transform: rotate(30deg);
	}
	.welcome-crafts span:nth-child(4) {
		bottom: 3%;
		left: 12%;
		transform: rotate(10deg);
	}
	.welcome-crafts span:nth-child(5) {
		bottom: 16%;
		right: -16px;
	}
	.auth-card {
		position: relative;
		width: 100%;
		max-width: 560px;
		padding: 32px;
		border: 1.5px solid var(--ink);
		border-radius: 14px;
		background: var(--paper-2);
		box-shadow: 6px 6px 0 var(--shadow);
		display: flex;
		flex-direction: column;
	}
	h1 {
		font-family: 'DM Serif Display', Georgia, serif;
		font-size: 42px;
		font-weight: 400;
		line-height: 1.05;
		letter-spacing: -0.02em;
		text-align: center;
	}
	.welcome-subtitle {
		font-size: 16px;
		line-height: 1.5;
		text-align: center;
		color: var(--ink-2);
		margin-top: 8px;
	}
	.register {
		width: 100%;
		margin-top: 24px;
	}
	.registration-handoff {
		font-size: 13px;
		line-height: 1.45;
		color: var(--ink-2);
		text-align: center;
		margin-top: 8px;
	}
	.welcome-divider {
		display: flex;
		align-items: center;
		gap: 16px;
		margin: 24px 0;
		font-size: 13px;
		color: var(--ink-2);
		font-weight: 600;
		letter-spacing: 0.14em;
	}
	.welcome-divider::before,
	.welcome-divider::after {
		content: '';
		height: 1px;
		flex: 1;
		background: var(--hairline);
	}
	form {
		display: flex;
		flex-direction: column;
		gap: 16px;
	}
	label {
		font-size: 16px;
		font-weight: 700;
	}
	input {
		min-height: 56px;
	}
	.secondary-sign-in {
		background: var(--surface);
		color: var(--ink);
		width: 100%;
	}
	.secondary-sign-in:hover {
		background: var(--paper);
		color: var(--ink);
	}
	.atmosphere-explainer {
		margin-top: 24px;
		font-size: 14px;
	}
	.atmosphere-explainer summary {
		justify-content: space-between;
		font-size: 14px;
	}
	.atmosphere-explainer summary::after {
		content: '⌄';
	}
	.atmosphere-explainer[open] summary::after {
		transform: rotate(180deg);
	}
	.atmosphere-explainer p {
		background: var(--surface);
		padding: 16px;
		font-size: 14px;
		line-height: 1.5;
	}
	.welcome-legal {
		margin-top: 24px;
		text-align: center;
		font-size: 13px;
		line-height: 1.5;
		color: var(--ink-2);
	}
	@media (max-width: 420px) {
		.auth-card {
			padding: 32px 16px;
		}
	}
</style>
