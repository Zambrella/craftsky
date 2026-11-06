<script lang="ts">
	import PublishingAnimation from './PublishingAnimation.svelte';
	let {
		active,
		label,
		video = false,
		fraction,
		onstop
	}: {
		active: boolean;
		label: string;
		video?: boolean;
		fraction?: number;
		onstop: () => void;
	} = $props();
	let dialog: HTMLDialogElement;
	$effect(() => {
		if (!dialog) return;
		if (active && !dialog.open) {
			dialog.showModal();
			dialog.focus();
		} else if (!active && dialog.open) dialog.close();
	});
</script>

<dialog
	bind:this={dialog}
	class="publishing-overlay"
	aria-label={label}
	tabindex="-1"
	oncancel={(event) => event.preventDefault()}
>
	<div class="publishing-status" role="status" aria-live="polite">
		<PublishingAnimation {active} />
		<p>{label}</p>
		{#if video}<progress max="100" value={fraction} aria-label="Video progress"
			></progress>{/if}
		<button type="button" onclick={onstop}>Stop waiting</button>
	</div>
</dialog>

<style>
	.publishing-overlay {
		position: fixed;
		inset: 0;
		margin: 0;
		width: 100dvw;
		height: 100dvh;
		max-width: none;
		max-height: none;
		border: 0;
		border-radius: 0;
		padding: 32px;
		background: color-mix(in srgb, var(--paper) 94%, transparent);
		color: var(--ink);
		box-shadow: none;
		outline: none;
	}
	.publishing-overlay::backdrop {
		background: transparent;
	}
	.publishing-status {
		height: 100%;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: 20px;
		text-align: center;
	}
	progress {
		width: 240px;
		max-width: 100%;
		accent-color: var(--primary);
	}
	p {
		margin: 0;
		font-size: 16px;
		font-weight: 600;
	}
</style>
