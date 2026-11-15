<script lang="ts">
	import { onMount, onDestroy, untrack, tick } from 'svelte';
	import {
		RevenueCatPaywall,
		type WebPaywallDetails
	} from './revenuecat-paywall';
	let {
		details,
		beforePurchase,
		onComplete,
		onClose
	}: {
		details: WebPaywallDetails;
		beforePurchase: () => Promise<void>;
		onComplete: () => void;
		onClose: () => void;
	} = $props();
	let dialog: HTMLDialogElement,
		target: HTMLDivElement,
		purchaseTarget: HTMLDivElement;
	let checkoutOpen = $state(false);
	let loading = $state(true),
		error = $state('');
	let active = true;
	const paywall = untrack(
		() => new RevenueCatPaywall(details, beforePurchase, () => active)
	);
	function close() {
		paywall.close();
		onClose();
	}
	onMount(() => {
		dialog.showModal();
		let startedOutside = false;
		const outside = (event: MouseEvent) => {
			const bounds = dialog.getBoundingClientRect();
			return (
				event.target === dialog &&
				(event.clientX < bounds.left ||
					event.clientX > bounds.right ||
					event.clientY < bounds.top ||
					event.clientY > bounds.bottom)
			);
		};
		const pointerDown = (event: PointerEvent) => {
			startedOutside = event.isPrimary && event.button === 0 && outside(event);
		};
		const backdropClick = (event: MouseEvent) => {
			const dismiss = startedOutside && !checkoutOpen && outside(event);
			startedOutside = false;
			if (dismiss) close();
		};
		dialog.addEventListener('pointerdown', pointerDown);
		dialog.addEventListener('click', backdropClick);
		void paywall
			.present(
				target,
				() => {
					loading = false;
				},
				onComplete,
				close,
				(message) => {
					if (active) error = message;
				},
				{
					purchaseTarget,
					onCheckoutChange: async (open) => {
						if (!active) return;
						checkoutOpen = open;
						error = '';
						// Stripe measures its frame at mount; expand from the sheet first.
						await tick();
						if (!active) return;
						dialog.scrollTop = 0;
						if (!open)
							requestAnimationFrame(() => {
								if (active)
									target
										.querySelector<HTMLButtonElement>('button:not([disabled])')
										?.focus({ preventScroll: true });
							});
					}
				}
			)
			.catch(() => {
				if (active) {
					loading = false;
					error =
						'This subscription option is unavailable right now. Please close this window and try again.';
				}
			});
		return () => {
			dialog.removeEventListener('pointerdown', pointerDown);
			dialog.removeEventListener('click', backdropClick);
		};
	});
	onDestroy(() => {
		active = false;
		paywall.close();
		dialog?.close();
	});
</script>

<dialog
	bind:this={dialog}
	class="rc-paywall-overlay"
	class:checkout-open={checkoutOpen}
	aria-label={`CraftSky ${details.offering === 'plus' ? 'Plus' : 'Business'}${checkoutOpen ? ' checkout' : ''}`}
	oncancel={(event) => {
		event.preventDefault();
		close();
	}}
>
	{#if checkoutOpen}
		<div class="checkout-controls">
			<button type="button" onclick={close}>Close checkout</button>
		</div>
	{/if}
	{#if loading}<p role="status">Loading subscription details…</p>{/if}
	{#if error}<p role="alert">{error}</p>{/if}
	<div class="paywall-host" hidden={checkoutOpen} bind:this={target}></div>
	<div
		class="checkout-host"
		hidden={!checkoutOpen}
		bind:this={purchaseTarget}
	></div>
</dialog>

<style>
	.rc-paywall-overlay {
		box-sizing: border-box;
		width: min(720px, 100vw);
		max-width: none;
		height: min(900px, 90dvh);
		max-height: 90dvh;
		margin: auto auto 0;
		padding: 0;
		overflow: auto;
		border: 1.5px solid var(--ink);
		border-radius: 24px 24px 0 0;
		background: var(--surface);
		color: var(--ink);
	}
	.rc-paywall-overlay::backdrop {
		background: #161210a6;
	}
	p {
		margin: 16px;
	}
	.paywall-host {
		height: 100%;
		width: 100%;
	}
	.checkout-controls {
		position: sticky;
		top: 0;
		z-index: 10;
		display: flex;
		justify-content: flex-end;
		padding: 8px 12px;
		background: var(--surface);
		border-bottom: 1px solid var(--hairline);
	}
	.checkout-host {
		min-height: calc(100% - 56px);
		width: 100%;
	}
	.rc-paywall-overlay.checkout-open {
		/* Leave room for Stripe's 991px desktop breakpoint plus borders/scrollbar. */
		width: min(1200px, calc(100vw - 32px));
		height: calc(100dvh - 48px);
		max-height: calc(100dvh - 48px);
		margin: auto;
		border-radius: 16px;
	}
	.rc-paywall-overlay:not(.checkout-open) {
		animation: sheet-in 180ms ease-out;
	}
	@keyframes sheet-in {
		from {
			transform: translateY(48px);
			opacity: 0;
		}
		to {
			transform: translateY(0);
			opacity: 1;
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.rc-paywall-overlay:not(.checkout-open) {
			animation: none;
		}
	}
	@media (max-width: 600px) {
		.rc-paywall-overlay.checkout-open {
			width: 100vw;
			height: 100dvh;
			max-height: 100dvh;
			margin: 0;
			border-radius: 0;
		}
	}
</style>
