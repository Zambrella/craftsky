<script lang="ts">
	import { fieldError } from '#lib/field-validation.ts';
	import { onDestroy, untrack } from 'svelte';
	import Icon from '../shell/Icon.svelte';
	import { subscriptionsEnabled, comingSoon } from '../shell/release-config';
	import { scheduledLocalDate, scheduleWindow } from './model';
	let {
		owner,
		later,
		value,
		disabled = false,
		fixed = false,
		allowed = true,
		full = false,
		onchange
	}: {
		owner: string;
		later: boolean;
		value: string;
		disabled?: boolean;
		fixed?: boolean;
		allowed?: boolean;
		full?: boolean;
		onchange: (later: boolean, value: string) => void;
	} = $props();
	const uid = $props.id();
	let menu: HTMLElement;
	let dialog: HTMLDialogElement;
	let upgrade: HTMLDialogElement;
	let trigger: HTMLButtonElement;
	let selected = $state('');
	let error = $state('');
	let disposed = false;
	$effect(() => {
		owner;
		untrack(() => {
			menu?.hidePopover();
			dialog?.close();
			upgrade?.close();
		});
	});
	function positionMenu() {
		if (!subscriptionsEnabled()) {
			comingSoon('Scheduled posts');
			return;
		}
		if (!allowed) {
			upgrade.showModal();
			return;
		}
		const bounds = trigger.getBoundingClientRect(),
			width = 260;
		menu.style.left = `${Math.max(16, Math.min(bounds.left, innerWidth - width - 16))}px`;
		menu.style.top = `${Math.max(16, Math.min(bounds.bottom + 8, innerHeight - 136 - 16))}px`;
	}
	const label = $derived(
		later && value
			? new Date(value).toLocaleString(undefined, {
					month: 'short',
					day: 'numeric',
					hour: 'numeric',
					minute: '2-digit'
				})
			: 'Now'
	);
	function close() {
		dialog.close();
		if (!disposed) trigger.focus({ preventScroll: true });
	}
	function chooseLater() {
		if (disabled || !allowed || full) return;
		menu.hidePopover();
		selected = value || '';
		error = '';
		dialog.showModal();
	}
	function apply() {
		if (disabled || !allowed || full) return;
		const date = new Date(selected),
			window = scheduleWindow();
		if (
			!selected ||
			!Number.isFinite(date.getTime()) ||
			date.getTime() < window.earliest ||
			date.getTime() > window.latest
		) {
			fieldError(
				'Publish date and time',
				'Choose a whole-minute time from 5 minutes through 28 days from now',
				dialog
			);
			return;
		}
		onchange(true, selected);
		close();
	}
	onDestroy(() => {
		disposed = true;
		menu?.hidePopover();
		dialog?.close();
		upgrade?.close();
	});
</script>

<button
	bind:this={trigger}
	type="button"
	class="schedule-control"
	class:scheduled={later}
	aria-label={subscriptionsEnabled()
		? 'When'
		: 'Scheduled posts is coming soon'}
	title="When"
	popovertarget={allowed && subscriptionsEnabled() ? uid + '-menu' : undefined}
	onclick={positionMenu}
	{disabled}
	><Icon name="clock" />{#if !allowed}<span
			class="plus-badge"
			aria-label="CraftSky Plus">+</span
		>{/if}{#if later && value}<span>{label}</span>{/if}</button
>
<div
	bind:this={menu}
	popover
	id={uid + '-menu'}
	class="schedule-menu"
	role="menu"
	aria-label="When"
>
	<button
		type="button"
		role="menuitemradio"
		aria-checked={!later}
		disabled={disabled || fixed}
		onclick={() => {
			onchange(false, '');
			menu.hidePopover();
		}}
		><Icon name="send" /><span>Now</span>{#if !later}<Icon
				name="check"
				size={20}
			/>{/if}</button
	>
	<button
		type="button"
		role="menuitemradio"
		aria-checked={later}
		disabled={disabled || full}
		onclick={chooseLater}
		><Icon name="clock" /><span>Schedule for later</span>{#if later}<Icon
				name="check"
				size={20}
			/>{/if}</button
	>
</div>
<dialog
	bind:this={dialog}
	class="schedule-dialog"
	aria-labelledby={uid + '-title'}
	oncancel={(event) => {
		event.preventDefault();
		close();
	}}
>
	<h2 id={uid + '-title'}>Schedule for later</h2>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			apply();
		}}
	>
		<label
			>Publish date and time<input
				name="selected"
				type="datetime-local"
				bind:value={selected}
				min={scheduledLocalDate(
					new Date(scheduleWindow().earliest).toISOString()
				)}
				max={scheduledLocalDate(
					new Date(scheduleWindow().latest).toISOString()
				)}
				required
				{disabled}
			/></label
		>
		<p>Your time zone: {Intl.DateTimeFormat().resolvedOptions().timeZone}</p>
		{#if error}<p role="alert" class="error">{error}</p>{/if}
		<div class="schedule-actions">
			<button type="button" onclick={close}>Cancel</button><button
				type="submit"
				class="primary"
				{disabled}>Set time</button
			>
		</div>
	</form>
</dialog>
<dialog
	bind:this={upgrade}
	class="schedule-dialog"
	aria-label="Scheduled posts with CraftSky Plus"
>
	<h2>Scheduled posts</h2>
	<p>
		Schedule posts ahead of time with CraftSky Plus. Manage your subscription in
		the mobile app.
	</p>
	<div class="schedule-actions">
		<button type="button" onclick={() => upgrade.close()}>Close</button><a
			href="/app/settings/subscriptions">Learn more</a
		>
	</div>
</dialog>

<style>
	.schedule-control {
		display: inline-flex;
		align-items: center;
		gap: 8px;
		min-width: 44px;
		min-height: 44px;
		padding: 8px;
		border: 0;
		background: transparent;
		box-shadow: none;
	}
	.schedule-control.scheduled {
		color: var(--primary);
	}
	.schedule-control {
		position: relative;
	}
	.plus-badge {
		position: absolute;
		right: 0;
		top: 0;
		color: var(--ink);
		background: var(--butter);
		border-radius: 50%;
		min-width: 16px;
		height: 16px;
		display: grid;
		place-items: center;
		font-size: 12px;
		font-weight: 700;
	}
	.schedule-menu {
		position: fixed;
		inset: auto;
		margin: auto;
		width: 260px;
		max-width: calc(100vw - 32px);
		padding: 8px;
		background: var(--paper);
		color: var(--ink);
		border: 1.5px solid var(--ink);
		border-radius: 12px;
		box-shadow: 3px 3px 0 var(--ink);
	}
	.schedule-menu:popover-open {
		margin: 0;
	}
	.schedule-menu button {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		min-height: 48px;
		padding: 12px;
		text-align: left;
		border: 0;
		background: transparent;
		box-shadow: none;
		font-size: 16px;
	}
	.schedule-menu button span {
		flex: 1;
	}
	.schedule-menu button:hover {
		background: color-mix(in srgb, var(--primary) 8%, transparent);
	}
	.schedule-dialog {
		margin: auto;
		width: min(420px, calc(100vw - 40px));
		max-height: calc(100dvh - 48px);
		padding: 24px;
		border: 1.5px solid var(--ink);
		border-radius: 20px;
		background: var(--paper);
		color: var(--ink);
		box-shadow: 4px 4px 0 var(--ink);
		overflow: auto;
	}
	.schedule-dialog::backdrop {
		background: #0006;
	}
	.schedule-dialog h2 {
		margin: 0 0 24px;
		font-size: 22px;
	}
	.schedule-dialog form {
		display: grid;
		gap: 16px;
	}
	.schedule-dialog p {
		margin: 0;
		color: var(--ink-3);
		font-size: 13px;
	}
	.schedule-actions {
		display: flex;
		justify-content: flex-end;
		gap: 12px;
	}
</style>
