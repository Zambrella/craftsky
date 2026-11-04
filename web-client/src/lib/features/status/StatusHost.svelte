<script lang="ts" module>
	function focus(node: HTMLElement) {
		const previous = document.activeElement as HTMLElement | null;
		node.focus();
		const trap = (event: KeyboardEvent) => {
			if (event.key !== 'Tab') return;
			const items = [
				...node.querySelectorAll<HTMLElement>(
					'a[href],button:not([disabled]),[tabindex="0"]'
				)
			];
			if (!items.length) {
				event.preventDefault();
				return;
			}
			const index = items.indexOf(document.activeElement as HTMLElement);
			if (event.shiftKey && index <= 0) {
				event.preventDefault();
				items.at(-1)?.focus();
			} else if (
				!event.shiftKey &&
				(index === -1 || index === items.length - 1)
			) {
				event.preventDefault();
				items[0].focus();
			}
		};
		node.addEventListener('keydown', trap);
		return {
			destroy() {
				node.removeEventListener('keydown', trap);
				previous?.focus();
			}
		};
	}
</script>

<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import {
		readStatus,
		compare,
		destinations,
		type ServiceStatus
	} from './model';
	let { onBlocking = () => {} }: { onBlocking?: (value: boolean) => void } =
		$props();
	let status = $state<ServiceStatus>(),
		received = $state(0),
		elapsed = 0;
	let now = $state(Date.now()),
		dismissed = $state<string[]>([]);
	let dead = false,
		busy = false;
	let request: AbortController | undefined;
	const current = __CRAFTSKY_VERSION__;
	const maintenance = $derived(
		!!status?.maintenance &&
			now >= received &&
			now - received < 300000 &&
			performance.now() - elapsed >= 0 &&
			performance.now() - elapsed < 300000
	);
	const update = $derived(
		!!status?.minimumVersions?.web &&
			compare(current, status.minimumVersions.web) < 0
	);
	const message = $derived(
		status?.message &&
			compare(current, status.message.minVersion) >= 0 &&
			(!status.message.maxVersion ||
				compare(current, status.message.maxVersion) <= 0) &&
			!dismissed.includes(status.message.revision)
			? status.message
			: undefined
	);
	$effect(() => onBlocking(maintenance || update || !!message));
	async function refresh() {
		if (dead || busy || document.hidden) return;
		busy = true;
		request = new AbortController();
		try {
			const response = await fetch('https://status.craftsky.social/app.json', {
				credentials: 'omit',
				redirect: 'error',
				cache: 'no-store',
				referrerPolicy: 'no-referrer',
				headers: { Accept: 'application/json' },
				signal: AbortSignal.any([request.signal, AbortSignal.timeout(3000)])
			});
			const value = await readStatus(response);
			if (!dead) {
				status = value;
				received = Date.now();
				elapsed = performance.now();
				now = received;
			}
		} catch {
		} finally {
			busy = false;
		}
	}
	function dismiss() {
		if (!message) return;
		dismissed = [...dismissed, message.revision].slice(-64);
		try {
			localStorage.setItem(
				'craftsky.status-dismissed',
				JSON.stringify(dismissed)
			);
		} catch {}
	}
	onMount(() => {
		try {
			const value = JSON.parse(
				localStorage.getItem('craftsky.status-dismissed') || '[]'
			);
			if (Array.isArray(value))
				dismissed = value
					.filter(
						(item) =>
							typeof item === 'string' && /^[A-Za-z0-9_-]{1,64}$/.test(item)
					)
					.slice(-64);
		} catch {}
		void refresh();
		const poll = setInterval(refresh, 60000),
			expiry = setInterval(() => (now = Date.now()), 1000);
		return () => {
			clearInterval(poll);
			clearInterval(expiry);
		};
	});
	onDestroy(() => {
		dead = true;
		request?.abort();
	});
	async function activate() {
		const cta = message?.cta;
		if (!cta) return;
		if (cta.url) {
			const tab = window.open('', '_blank');
			if (!tab) return;
			tab.opener = null;
			tab.location.replace(cta.url);
			dismiss();
		} else if (cta.destination) {
			await goto(destinations[cta.destination]);
			dismiss();
		}
	}
</script>

<svelte:document
	onvisibilitychange={() => {
		if (!document.hidden) {
			now = Date.now();
			void refresh();
		}
	}}
/>
{#if maintenance || update || message}<div class="status-scrim">
		<div
			class="panel stack status-dialog"
			role="alertdialog"
			aria-modal="true"
			aria-labelledby="service-title"
			tabindex="-1"
			use:focus
		>
			{#if maintenance && status?.maintenance}<p class="eyebrow">
					We’ll be back soon
				</p>
				<h1 id="service-title">{status.maintenance.title}</h1>
				<p class="prose">{status.maintenance.description}</p>
				{#if status.maintenance.estimatedRecoveryAt}<p>
						Estimated return: {new Date(
							status.maintenance.estimatedRecoveryAt
						).toLocaleString()}
					</p>{/if}<button onclick={refresh}>Check again</button
				>{:else if update}<h1 id="service-title">
					A new version of CraftSky is ready
				</h1>
				<p>Reload to continue with the latest version.</p>
				<button class="button" onclick={() => location.reload()}
					>Reload CraftSky</button
				>{:else if message}<h1 id="service-title">{message.title}</h1>
				{#if message.imageUrl}<img src={message.imageUrl} alt="" />{/if}
				<p class="prose">{message.description}</p>
				{#if message.cta}<button class="button" onclick={activate}
						>{message.cta.text}</button
					>{/if}<button onclick={dismiss}>Dismiss</button>{/if}
		</div>
	</div>{/if}

<style>
	.status-scrim {
		position: fixed;
		inset: 0;
		z-index: 100;
		display: grid;
		place-items: center;
		background: #0009;
		padding: 16px;
	}
	.status-dialog {
		width: min(600px, 100%);
		margin: 0;
		max-height: 90svh;
		overflow: auto;
	}
</style>
