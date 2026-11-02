<script lang="ts">
	import { page } from '$app/state';
	let handle = $state(''),
		busy = $state(false),
		message = $state('');
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
<section class="intro auth-card">
	<p class="eyebrow">Your craft. Your community.</p>
	<h1>A place for what you make.</h1>
	<p>Sign in with your AT Protocol account to join your crafting community.</p>
	{#if page.url.searchParams.get('error')}<p role="status" class="notice">
			{page.url.searchParams.get('error') === 'confirmation_pending'
				? 'Your sign-in is saved. Retry confirmation to finish.'
				: 'Sign-in could not be completed. You can try again.'}
		</p>{/if}
	{#if page.url.searchParams.get('error') === 'confirmation_pending'}<p>
			<a class="button" href="/auth/browser/confirm">Retry confirmation</a>
		</p>{/if}
	<form
		onsubmit={(e) => {
			e.preventDefault();
			start('login');
		}}
	>
		<label for="handle">Your handle</label><input
			id="handle"
			bind:value={handle}
			placeholder="you.bsky.social"
			autocomplete="username"
			required
			disabled={busy}
		/><button class="button" disabled={busy}
			>{busy ? 'Connecting…' : 'Sign in'}</button
		>
	</form>
	<p class="muted">
		Your account provider handles your password. CraftSky never asks you to
		enter it here.
	</p>
	<button
		class="secondary-button"
		onclick={() => start('register')}
		disabled={busy}>Create an account</button
	>
	{#if message}<p role="alert" class="notice error">{message}</p>{/if}
</section>
