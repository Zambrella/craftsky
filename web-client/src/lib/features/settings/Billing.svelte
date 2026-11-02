<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
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
	let { api, session }: { api: CraftskyAPI; session: Session } = $props();
	const work = new Work();
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

<h1>Subscriptions and billing</h1>
<Feedback {work} reload={load} />
{#if !work.loading && access}
	<section class="panel">
		<h2>CraftSky {title(access.effectiveTier)}</h2>
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
		<p class="muted">
			Purchase or restore subscriptions in the CraftSky mobile app. You can view
			existing access, open your store's subscription settings and manage
			license assignments here.
		</p>
	</section>
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
	{:else}<section class="panel">
			<div class="row">
				<h2>Store subscriptions</h2>
				<button disabled={work.busy} onclick={refresh}>Refresh access</button>
			</div>
			{#if billing.reconciliationStale}<p class="notice">
					Subscription changes are still being checked with the store.
				</p>{/if}{#if billing.reconciledAt}<p class="muted">
					Last checked {dateTime(billing.reconciledAt)}
				</p>{/if}{#each billing.subscriptions as subscription}{@const managementURL =
					storeManagementURL(subscription.store)}
				<article class="subscription">
					<h3>{subscription.productId}</h3>
					<p>
						{title(subscription.store)} · {title(subscription.status)} · {subscription.givesAccess
							? 'Access active'
							: 'No active access'}
					</p>
					{#if managementURL}<a
							class="action"
							href={managementURL}
							target="_blank"
							rel="noopener noreferrer"
							>Manage in {subscription.store === 'app_store'
								? 'App Store'
								: 'Google Play'}</a
						>{/if}
					{#if subscription.pendingPayment}<p class="notice">
							Payment is pending. Open your mobile store to resolve payment.
						</p>{/if}{#if subscription.anomaly && subscription.anomaly !== 'none'}<p
							class="error"
						>
							This subscription needs attention: {title(subscription.anomaly)}.
							Refresh access or contact support.
						</p>{/if}{#if subscription.currentPeriodEndsAt}<p class="muted">
							Current period ends {dateTime(subscription.currentPeriodEndsAt)}
						</p>{/if}{#if subscription.endsAt}<p class="muted">
							Subscription ends {dateTime(subscription.endsAt)}
						</p>{/if}{#if subscription.autoRenewalStatus}<p class="muted">
							Renewal: {title(subscription.autoRenewalStatus)}
						</p>{/if}
				</article>{:else}<p class="empty">
					No subscriptions are linked to this billing account.
				</p>{/each}
		</section>
		<section class="panel">
			<h2>License assignments</h2>
			<p class="muted">
				Assign an available license to the CraftSky account that should receive
				its features. Use the account’s complete DID.
			</p>
			{#each billing.licenses as license}<form
					class="license stack"
					onsubmit={(event) => assign(license, event)}
				>
					<fieldset class="form-fields" disabled={work.busy || !!work.retry}>
						<h3>CraftSky {title(license.tier)}</h3>
						<p>
							{license.assignedDid ? 'Assigned to:' : 'Unassigned'}
							{#if license.assignedDid}<code>{license.assignedDid}</code>{/if}
						</p>
						{#if license.assignedAt}<p class="muted">
								Assigned {dateTime(license.assignedAt)}
							</p>{/if}{#if license.anomaly && license.anomaly !== 'none'}<p
								class="error"
							>
								This license needs attention: {title(license.anomaly)}
							</p>{:else if license.assignable}<label
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
				</form>{:else}<p class="empty">No licenses are available.</p>{/each}
		</section>{/if}
{/if}

<style>
	.license,
	.subscription {
		padding: 1rem 0;
		border-bottom: 1px solid var(--hairline);
	}
	.license:last-child,
	.subscription:last-child {
		border: 0;
	}
</style>
