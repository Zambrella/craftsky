<script lang="ts">
	import type { ReportSubmission } from '#lib/api/types.ts';
	import catalogue from './reasons.json';
	let {
		subject,
		busy = false,
		onSubmit,
		onCancel
	}: {
		subject: string;
		busy?: boolean;
		onSubmit: (value: ReportSubmission) => void;
		onCancel: () => void;
	} = $props();
	let group = $state(''),
		reason = $state(''),
		details = $state('');
	const choices = $derived(
		catalogue.reasons.filter((value) => value.group === group)
	);
	const copyright =
		'mailto:moderation@craftsky.social?' +
		new URLSearchParams({
			subject: 'CraftSky copyright or trade mark report',
			body: 'Please identify the protected work, your authority, and the CraftSky URL or AT URI. Do not attach media.'
		});
</script>

<form
	class="panel stack"
	onsubmit={(event) => {
		event.preventDefault();
		if (reason && reason !== 'intellectualProperty' && details.length <= 1000)
			onSubmit({ reasonType: reason, details: details.trim() || undefined });
	}}
>
	<h2>Report {subject}</h2>
	<label
		>Category<select
			bind:value={group}
			onchange={() => (reason = '')}
			required
			disabled={busy}
			><option value="">Choose a category</option
			>{#each catalogue.groups as option}<option value={option.value}
					>{option.label}</option
				>{/each}</select
		></label
	>{#if group}<label
			>Reason<select bind:value={reason} required disabled={busy}
				><option value="">Choose a reason</option
				>{#each choices as option}<option value={option.value}
						>{option.label}</option
					>{/each}</select
			></label
		>{/if}{#if reason === 'intellectualProperty'}<p>
			Copyright and trade mark reports need information about the protected work
			and your authority to report it. Email our moderation team and include the
			CraftSky URL or AT URI. Do not attach media.
		</p>
		<a class="button" href={copyright}>Email moderation</a>{:else}<label
			>Additional details (optional)<textarea
				bind:value={details}
				maxlength="1000"
				rows="4"
				disabled={busy}></textarea></label
		>
		<p class="muted">{details.length} / 1,000</p>
		<button class="button" disabled={busy || !reason}>Submit report</button
		>{/if}<button type="button" disabled={busy} onclick={onCancel}
		>Cancel</button
	>
</form>
