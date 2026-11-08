<script lang="ts">
	import { onMount } from 'svelte';
	import type { ReportSubmission } from '#lib/api/types.ts';
	import Icon from '../shell/Icon.svelte';
	import catalogue from './reasons.json';
	let {
		subject,
		busy = false,
		showContactEmail = false,
		onRetry,
		onSubmit,
		onCancel
	}: {
		subject: string;
		busy?: boolean;
		showContactEmail?: boolean;
		onRetry?: () => unknown;
		onSubmit: (value: ReportSubmission) => void;
		onCancel: () => void;
	} = $props();
	let group = $state(''),
		reason = $state(''),
		details = $state('');
	let contactEmail = $state('');
	let dialog = $state<HTMLDialogElement>(null!);
	const choices = $derived(
		catalogue.reasons.filter((value) => value.group === group)
	);
	const copyright =
		'mailto:moderation@craftsky.social?' +
		new URLSearchParams({
			subject: 'CraftSky copyright or trade mark report',
			body: 'Please identify the protected work, your authority, and the CraftSky URL or AT URI. Do not attach media.'
		});
	function close() {
		if (!busy) onCancel();
	}
	onMount(() => dialog.showModal());
</script>

<dialog
	bind:this={dialog}
	class="report-page"
	aria-labelledby="report-title"
	oncancel={(event) => {
		event.preventDefault();
		close();
	}}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			if (
				!busy &&
				!onRetry &&
				choices.some((choice) => choice.value === reason) &&
				reason !== 'intellectualProperty' &&
				details.length <= 1000
			)
				onSubmit({
					reasonType: reason,
					details: details.trim() || undefined,
					...(showContactEmail
						? { contactEmail: contactEmail.trim() || undefined }
						: {})
				});
		}}
	>
		<header>
			<button
				class="icon-button"
				type="button"
				aria-label="Cancel report"
				disabled={busy}
				onclick={close}><Icon name="close" /></button
			>
			<h2 id="report-title">Report {subject}</h2>
			{#if group === 'intellectualProperty'}<a class="submit" href={copyright}
					>Email moderation</a
				>{:else}<button
					class="submit"
					aria-label="Submit report"
					disabled={busy || !reason || !!onRetry}
					>{busy ? 'Submitting…' : 'Submit'}</button
				>{/if}
		</header>
		<div class="report-content">
			{#if !group}<fieldset>
					<legend>What is the concern about?</legend>
					{#each catalogue.groups as option}<label class="radio-row"
							><input
								type="radio"
								name="category"
								value={option.value}
								disabled={busy}
								onchange={() => {
									group = option.value;
									reason = '';
								}}
							/><span>{option.label}</span></label
						>{/each}
				</fieldset>{:else}<fieldset>
					<legend>Reason</legend><button
						class="change-category"
						type="button"
						disabled={busy}
						onclick={() => {
							group = '';
							reason = '';
						}}>Change category</button
					>
					{#if group !== 'intellectualProperty'}{#each choices as option}<label
								class="radio-row"
								><input
									type="radio"
									name="reason"
									value={option.value}
									bind:group={reason}
									disabled={busy}
									required
								/><span>{option.label}</span></label
							>{/each}{/if}
				</fieldset>
				{#if group === 'childSafety'}<p class="guidance">
						Identify where the concern is. Do not download, attach, forward or
						redistribute suspected material.
					</p>{:else if group === 'intellectualProperty'}<p class="guidance">
						Copyright and trade mark concerns use our dedicated email process.
						Identify the work, your authority and the CraftSky URL or AT URI. Do
						not attach media.
					</p>{:else if reason === 'immediateDanger'}<p class="guidance">
						If someone is in immediate danger, contact local emergency services
						first. CraftSky is not an emergency service, but you can still send
						this report.
					</p>{/if}
				{#if group !== 'intellectualProperty'}<label class="details-label"
						>Details<textarea
							name="details"
							bind:value={details}
							maxlength="1000"
							rows="4"
							disabled={busy}></textarea></label
					>
					<p class="details-count">{details.length}/1000</p>
					{#if showContactEmail}<label class="details-label"
							>Contact email (optional)
							<input
								name="contactEmail"
								type="email"
								maxlength="254"
								bind:value={contactEmail}
								disabled={busy}
								autocomplete="email"
							/>
						</label>
						<p class="guidance">
							Only CraftSky moderators can see this. Add it if you would like us
							to contact you about your report.
						</p>{/if}
				{/if}{/if}
			{#if onRetry}<button
					type="button"
					disabled={busy}
					onclick={() => onRetry?.()}>Retry the same action</button
				>{/if}
		</div>
	</form>
</dialog>

<style>
	.report-page {
		width: min(640px, 100vw);
		max-width: 100vw;
		height: min(850px, 100dvh);
		max-height: 100dvh;
		padding: 0;
		border-radius: 14px;
		background: var(--paper);
		color: var(--ink);
	}
	form {
		margin: 0;
		min-height: 100%;
	}
	header {
		position: sticky;
		top: 0;
		z-index: 1;
		display: flex;
		align-items: center;
		gap: 8px;
		min-height: 56px;
		padding: 0 12px;
		background: var(--paper);
		border-bottom: 1.5px solid var(--ink);
	}
	header h2 {
		flex: 1;
		margin: 0;
		font-size: 20px;
		font-weight: 600;
	}
	.icon-button {
		width: 40px;
		height: 44px;
		padding: 8px;
	}
	.submit {
		background: transparent;
		border: 0;
		color: var(--primary);
		font-size: 14px;
		font-weight: 700;
		padding: 12px 8px;
		text-decoration: none;
		box-shadow: none;
	}
	.submit:disabled {
		opacity: 0.5;
	}
	.report-content {
		padding: 16px 16px 32px;
	}
	fieldset {
		min-width: 0;
		border: 0;
		padding: 0;
		margin: 0;
	}
	legend {
		font-size: 16px;
		font-weight: 700;
		margin-bottom: 8px;
	}
	.radio-row {
		display: flex;
		align-items: center;
		gap: 14px;
		min-height: 48px;
		font-size: 14px;
		font-weight: 400;
		cursor: pointer;
		padding: 8px 0;
	}
	.radio-row input {
		flex: 0 0 20px;
		width: 20px;
		height: 20px;
		margin: 0;
		accent-color: var(--primary);
	}
	.change-category {
		display: block;
		margin-bottom: 4px;
		padding: 10px 8px;
		border: 0;
		box-shadow: none;
		background: transparent;
		color: var(--primary);
		font-size: 14px;
	}
	.details-label {
		display: block;
		margin-top: 16px;
		font-size: 13px;
	}
	textarea {
		margin-top: 6px;
		min-height: 112px;
		resize: vertical;
	}
	.details-count {
		text-align: right;
		margin: 4px 0 0;
		font-size: 12px;
		color: var(--ink-2);
	}
	.guidance {
		font-size: 14px;
		line-height: 1.5;
		margin: 8px 0;
	}

	@media (max-width: 600px) {
		.report-page {
			position: fixed;
			inset: 0;
			margin: 0;
			width: 100vw;
			height: 100dvh;
			max-height: 100dvh;
			border: 0;
			border-radius: 0;
		}
	}
</style>
