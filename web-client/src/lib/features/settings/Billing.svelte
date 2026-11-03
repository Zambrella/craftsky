<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
	import SettingsIcon from './SettingsIcon.svelte';
	import {
		APIError,
		type CraftskyAPI,
		type SubscriptionAccess,
		type BillingState,
		type BillingLicense
	} from '#lib/api/client.ts';
	import { dateTime, title, validDid, type Session } from './model.ts';
	import { Work } from './work.svelte.ts';
	import Feedback from './Feedback.svelte';
	import { storeManagementURL } from './navigation.ts';
	import {
		canAssignLicense,
		tierState,
		tierStateLabel
	} from './native-settings.ts';
	let { api, session }: { api: CraftskyAPI; session: Session } = $props();
	const work = new Work();
	let accounts = $derived(
		(page.data.accounts || []) as { did: string; handle: string }[]
	);
	let access = $state<SubscriptionAccess | null>(null),
		billing = $state<BillingState | null>(null),
		targets = $state<Record<string, string>>({});
	const noAccount = (error: unknown) => {
		if (error instanceof APIError && error.status === 404) return null;
		throw error;
	};
	function apply(value: BillingState | null) {
		billing = value;
		targets = Object.fromEntries(
			(value?.licenses || []).map((l) => [l.id, l.assignedDid || session.did])
		);
	}
	function load() {
		return work.load(
			() =>
				Promise.all([api.billing.access(), api.billing.get().catch(noAccount)]),
			([a, b]) => {
				access = a;
				apply(b);
			}
		);
	}
	function refresh() {
		void work.run(
			async () => {
				await api.billing.reconcile();
				return Promise.all([api.billing.access(), api.billing.get()]);
			},
			([a, b]) => {
				access = a;
				apply(b);
			},
			'Refresh requested. Store updates can take a little time; refresh again if needed.'
		);
	}
	function assign(license: BillingLicense, event: SubmitEvent) {
		event.preventDefault();
		if (!billing || !canAssignLicense(billing, license)) {
			work.error =
				'Refresh subscription status before changing this assignment.';
			return;
		}
		const did = (targets[license.id] || '').trim();
		if (!validDid(did)) {
			work.error = 'Enter a complete valid DID, such as did:plc:…';
			return;
		}
		void work.run(
			async () => {
				await api.billing.assign(license.id, did);
				return api.billing.get();
			},
			apply,
			'License assignment updated.'
		);
	}
	function unassign(license: BillingLicense) {
		if (
			!confirm(
				'Remove this license assignment? The assigned account will lose its subscription access.'
			)
		)
			return;
		void work.run(
			async () => {
				await api.billing.unassign(license.id);
				return api.billing.get();
			},
			apply,
			'License unassigned.'
		);
	}
	onMount(load);
	onDestroy(() => work.dispose());
</script>

<h1 class="workspace-heading">Subscriptions</h1>
<Feedback {work} reload={load} />
{#if !work.loading && access}
	<section class="panel">
		<h2 class="section-heading access-title">
			<SettingsIcon name="plus" />Current access: {title(access.effectiveTier)}
		</h2>
		<p>
			{access.givesAccess
				? 'Your subscription features are available on the web.'
				: 'Your account currently has free access.'}
		</p>
		{#if access.assignedTier && !access.givesAccess}<p>
				Assigned plan: {title(access.assignedTier)}. This plan is not currently
				providing access.
			</p>{/if}{#if access.accessEndsAt}<p class="muted">
				Access ends {dateTime(access.accessEndsAt)}
			</p>{/if}
	</section>
	<p class="muted mobile-purchases">
		Purchase or restore subscriptions in the CraftSky mobile app. Existing
		access and license assignments are available here.
	</p>
	<p class="muted">
		Plus and Business are separate account licenses. After purchase, assign each
		license to a CraftSky account.
	</p>
	{#if !billing}<section class="panel stack">
			<h2>Billing account</h2>
			<p>
				A billing account links your store subscriptions and license
				assignments.
			</p>
			<button
				disabled={work.busy}
				class="primary"
				onclick={() =>
					work.run(() => api.billing.ensure(), apply, 'Billing account ready.')}
				>Set up billing account</button
			>
		</section>
	{:else}<div class="billing-refresh">
			<button disabled={work.busy} onclick={refresh}>Refresh access</button>
		</div>
		{#if billing.reconciliationStale}<p class="notice">
				Subscription changes are still being checked with the store.
			</p>{/if}
		{#each ['plus', 'business'] as tierValue}{@const tier = tierValue as
				'plus' | 'business'}{@const status = tierState(billing, tier)}
			<section
				class="panel tier-card"
				class:business-tier={tier === 'business'}
				aria-label={`CraftSky ${title(tier)}`}
			>
				<div class="tier-heading">
					<span class="tier-icon"
						><SettingsIcon
							name={tier === 'business' ? 'business' : 'subscription'}
						/></span
					>
					<div>
						<h2>{title(tier)}</h2>
						<p class="tier-status">{tierStateLabel[status]}</p>
					</div>
				</div>
				<p class="muted">
					{tier === 'business'
						? 'Everything in Plus, plus featured products, a profile action button, business details and events.'
						: 'Schedule posts, organise saves into folders, personalise your profile, pin posts and track follower growth.'}
				</p>
				{#each billing.subscriptions.filter( (subscription) => billing!.licenses.some((license) => license.tier === tier && license.subscriptionId === subscription.id) ) as subscription}{@const managementURL =
						storeManagementURL(subscription.store)}
					{#if subscription.currentPeriodEndsAt}<p class="muted">
							Current period ends {dateTime(subscription.currentPeriodEndsAt)}
						</p>{/if}
					{#if subscription.endsAt}<p class="muted">
							Access ends {dateTime(subscription.endsAt)}
						</p>{/if}
					{#if subscription.autoRenewalStatus}<p class="muted">
							Renewal: {title(subscription.autoRenewalStatus)}
						</p>{/if}
					{#if subscription.pendingPayment}<p class="notice">
							Payment is pending. Open your mobile store to resolve payment.
						</p>{/if}
					{#if managementURL}<a
							class="action"
							href={managementURL}
							target="_blank"
							rel="noopener noreferrer"
							>Manage in {subscription.store === 'app_store'
								? 'App Store'
								: 'Google Play'}</a
						>{/if}
				{/each}

				{#each billing.licenses.filter((value) => value.tier === tier) as license}<form
						class="license stack"
						onsubmit={(event) => assign(license, event)}
					>
						<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
							<h3>CraftSky {title(license.tier)}</h3>
							<p>
								{license.assignedDid ? 'Assigned to:' : 'Unassigned'}
								{#if license.assignedDid}<strong
										>{accounts.find(
											(account) => account.did === license.assignedDid
										)?.handle
											? '@' +
												accounts.find(
													(account) => account.did === license.assignedDid
												)!.handle
											: license.assignedDid}</strong
									>{/if}
							</p>
							{#if license.assignedAt}<p class="muted">
									Assigned {dateTime(license.assignedAt)}
								</p>{/if}{#if license.anomaly && license.anomaly !== 'none'}<p
									class="error"
								>
									This license needs attention: {title(license.anomaly)}
								</p>{:else if canAssignLicense(billing, license)}<label
									>Account<select
										disabled={work.busy}
										value={targets[license.id]}
										onchange={(event) =>
											(targets = {
												...targets,
												[license.id]: event.currentTarget.value
											})}
										>{#each accounts as account}<option value={account.did}
												>@{account.handle}</option
											>{/each}{#if !accounts.some((account) => account.did === targets[license.id])}<option
												value={targets[license.id]}
												>{targets[license.id] || 'Choose an account'}</option
											>{/if}</select
									></label
								>
								<details>
									<summary>Assign another account</summary><label
										>Account DID<input
											required
											value={targets[license.id] || ''}
											oninput={(e) =>
												(targets = {
													...targets,
													[license.id]: e.currentTarget.value
												})}
											placeholder="did:plc:…"
											autocomplete="off"
											spellcheck="false"
										/></label
									>
								</details>
								<div class="actions">
									<button class="primary" disabled={work.busy}
										>Assign license</button
									><button
										type="button"
										disabled={work.busy}
										onclick={() =>
											(targets = { ...targets, [license.id]: session.did })}
										>Use this account</button
									>
								</div>{:else}<p class="muted">
									This license cannot currently be reassigned.
								</p>
							{/if}
							{#if license.assignedDid}<button
									type="button"
									class="danger"
									disabled={work.busy}
									onclick={() => unassign(license)}>Unassign</button
								>{/if}
						</fieldset>
					</form>{/each}
			</section>{/each}
		{#if billing.reconciledAt}<p class="muted">
				Last checked {dateTime(billing.reconciledAt)}
			</p>{/if}
	{/if}
{/if}

<style>
	.billing-refresh {
		display: flex;
		justify-content: flex-end;
		margin-bottom: 16px;
	}
	.tier-card.panel {
		background: color-mix(in srgb, var(--primary) 8%, var(--surface));
	}
	.business-tier.panel {
		background: color-mix(in srgb, var(--moss) 8%, var(--surface));
	}
	.tier-heading {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		margin-bottom: 10px;
	}
	.tier-heading h2 {
		margin: 0 0 2px;
	}
	.tier-icon {
		width: 44px;
		height: 44px;
		border: 1.5px solid var(--primary);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		background: color-mix(in srgb, var(--primary) 16%, var(--surface));
		color: var(--primary);
		flex-shrink: 0;
	}
	.business-tier .tier-icon {
		border-color: var(--moss);
		background: color-mix(in srgb, var(--moss) 16%, var(--surface));
		color: var(--ink);
	}
	.tier-status {
		color: var(--primary);
		font-size: 14px;
		font-weight: 600;
	}
	.tier-card .license h3 {
		font-size: 16px;
	}

	.access-title {
		font-size: 16px;
	}
	.mobile-purchases {
		margin: 0 0 16px;
	}
	summary {
		cursor: pointer;
		padding: 8px 0;
		color: var(--primary);
		font-size: 14px;
	}
	.license {
		padding: 1rem 0;
		border-bottom: 1px solid var(--hairline);
	}
	.license:last-child {
		border: 0;
	}
</style>
