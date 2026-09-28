import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/account_switcher_state.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/widgets/account_switcher_content.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_page_model.dart';
import 'package:craftsky_app/subscriptions/models/subscription_presentation.dart';
import 'package:craftsky_app/subscriptions/models/subscription_rules.dart';
import 'package:craftsky_app/subscriptions/providers/customer_center_coordinator.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_assignment_controller.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_page_model_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_pending_operation_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_purchase_controller.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_restore_controller.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_setup_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/paywall_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

export 'package:craftsky_app/subscriptions/models/subscription_page_model.dart';

class SubscriptionPage extends ConsumerStatefulWidget {
  const SubscriptionPage({super.key});

  @override
  ConsumerState<SubscriptionPage> createState() => _SubscriptionPageState();
}

class _SubscriptionPageState extends ConsumerState<SubscriptionPage>
    with WidgetsBindingObserver {
  bool _busy = false;
  bool _foreground = true;
  SubscriptionOperationNotice? _notice;
  Future<SubscriptionOperationNotice?> Function()? _pendingRetry;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    final lifecycleState = WidgetsBinding.instance.lifecycleState;
    _foreground =
        lifecycleState == null || lifecycleState == AppLifecycleState.resumed;
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    if (_foreground && mounted) _refresh();
  }

  bool get _operationIsCurrent => mounted;
  bool get _operationIsForeground => mounted && _foreground;

  @override
  Widget build(BuildContext context) {
    final model = ref.watch(subscriptionPageModelProvider);
    final pendingOperation = ref.watch(subscriptionPendingOperationProvider);
    final operationBlocked = _busy || pendingOperation != null;
    return switch (model) {
      AsyncData(:final value) => SubscriptionPageView(
        pendingOperation: pendingOperation,
        model: SubscriptionPageModel(
          role: value.role,
          access: value.access,
          billingAvailability: value.billingAvailability,
          ownerRetained: value.ownerRetained,
          ownerRecoveryLocked: value.ownerRecoveryLocked,
          ownerSignInRequired: value.ownerSignInRequired,
          billingState: value.billingState,
          operationNotice:
              _notice ??
              (value.role == SubscriptionPageRole.owner &&
                      pendingOperation?.canReconcile == true &&
                      !_busy
                  ? SubscriptionOperationNotice.pending
                  : value.operationNotice),
          assignedAccountLabels: value.assignedAccountLabels,
          tierPrices: value.tierPrices,
          onSwitchToOwner: _busy ? null : _switchToOwner,
          onConfirmOwner: _busy ? null : _setupOwner,
          onRetrySetup: _busy ? null : _setupOwner,
          onPurchase: operationBlocked ? null : _purchase,
          onRestore: operationBlocked ? null : _restore,
          onManage: operationBlocked ? null : _manage,
          onRefresh:
              _busy ||
                  (pendingOperation != null && !pendingOperation.canReconcile)
              ? null
              : _refresh,
          onAssign: operationBlocked ? null : _assign,
          onUnassign: operationBlocked ? null : _unassign,
        ),
      ),
      AsyncLoading() => Scaffold(
        appBar: AppBar(
          title: Text(AppLocalizations.of(context).subscriptionsTitle),
        ),
        body: const Center(child: CircularProgressIndicator()),
      ),
      AsyncError() => Scaffold(
        appBar: AppBar(
          title: Text(AppLocalizations.of(context).subscriptionsTitle),
        ),
        body: Center(
          child: _ActionButton(
            label: AppLocalizations.of(context).subscriptionsRefresh,
            onPressed: _refresh,
          ),
        ),
      ),
    };
  }

  Future<void> _run(
    Future<SubscriptionOperationNotice?> Function() action,
  ) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      SubscriptionOperationNotice? notice;
      try {
        notice = await action();
      } on ApiUnauthorized {
        notice = SubscriptionOperationNotice.unauthorized;
      } on ApiException {
        notice = SubscriptionOperationNotice.actionFailed;
      } on Object {
        notice = SubscriptionOperationNotice.providerFailure;
      }
      if (!mounted) return;
      setState(() => _notice = notice);
      ref.invalidate(subscriptionPageModelProvider);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _refresh() {
    if (_busy) return;
    final pendingOperation = ref.read(subscriptionPendingOperationProvider);
    final registry = ref.read(sessionRegistryProvider).value;
    if (pendingOperation != null &&
        pendingOperation.canReconcile &&
        registry?.activeLease?.session.account.did ==
            pendingOperation.ownerDid) {
      unawaited(_run(() => _reconcilePending(pendingOperation)));
      return;
    }
    if (pendingOperation != null) return;
    final retry = _pendingRetry;
    if (retry != null && !_busy) {
      unawaited(_run(retry));
      return;
    }
    _pendingRetry = null;
    setState(() => _notice = null);
    ref.invalidate(subscriptionPageModelProvider);
    final lease = ref.read(sessionRegistryProvider).value?.activeLease?.session;
    if (lease != null) ref.invalidate(subscriptionAccessProvider(lease));
  }

  void _switchToOwner() => unawaited(
    _run(() async {
      final registry = ref.read(sessionRegistryProvider).requireValue;
      final owner = registry.billingOwner;
      if (owner == null) return null;
      final lease = registry.leaseFor(AccountKey(owner.did.value));
      if (lease == null) return SubscriptionOperationNotice.providerFailure;
      await ref.read(sessionRegistryProvider.notifier).activate(lease);
      return null;
    }),
  );

  void _setupOwner() => unawaited(
    _run(() async {
      final registry = ref.read(sessionRegistryProvider).requireValue;
      final active = registry.activeLease;
      if (active == null) return SubscriptionOperationNotice.providerFailure;
      final api = await ref.read(
        subscriptionRepositoryProvider(active.session.account).future,
      );
      final service = ref.read(revenueCatServiceProvider);
      await BillingOwnerSetupCoordinator(
        readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
        reserveOwner: ref
            .read(sessionRegistryProvider.notifier)
            .reserveBillingOwner,
        completeOwner: ref
            .read(sessionRegistryProvider.notifier)
            .completeBillingOwner,
        api: api,
        revenueCat: service,
      ).setup(active);
      return null;
    }),
  );

  void _purchase(SubscriptionTier tier) {
    final registry = ref.read(sessionRegistryProvider).value;
    final ownerDid = registry?.billingOwner?.did;
    if (ownerDid == null ||
        !ref
            .read(subscriptionPendingOperationProvider.notifier)
            .beginPurchase(ownerDid, tier)) {
      return;
    }
    unawaited(
      _run(() async {
        try {
          final dependencies = await _ownerDependencies();
          final paywall = PaywallCoordinator(
            dependencies.service,
            dependencies.guard,
            RevenueCatIdentityGuard(dependencies.service),
          );
          final controller = SubscriptionPurchaseController(
            ownerGuard: dependencies.guard,
            api: dependencies.api,
            presentPaywall: paywall.present,
            wait: Future<void>.delayed,
            isCurrent: () => _operationIsCurrent,
            isForeground: () => _operationIsForeground,
            onProviderConfirmed: ref
                .read(subscriptionPendingOperationProvider.notifier)
                .recordPurchase,
            readAccess: _readAssignedAccess,
          );
          return _handlePurchaseResult(await controller.purchase(tier));
        } on Object {
          ref.read(subscriptionPendingOperationProvider.notifier).clear();
          rethrow;
        }
      }),
    );
  }

  Future<SubscriptionOperationNotice?> _handlePurchaseResult(
    SubscriptionPurchaseResult result,
  ) async {
    if (result.outcome != SubscriptionPurchaseOutcome.pending) {
      ref.read(subscriptionPendingOperationProvider.notifier).clear();
    }
    final retry = result.retry;
    _pendingRetry = retry == null
        ? null
        : () async => _handlePurchaseResult(await retry());
    if (result.assignmentLicense case final license?) {
      _pendingRetry = null;
      return _assignLicense(license, state: result.state);
    }
    return switch (result.outcome) {
      SubscriptionPurchaseOutcome.completed ||
      SubscriptionPurchaseOutcome.cancelled ||
      SubscriptionPurchaseOutcome.notEligible => null,
      SubscriptionPurchaseOutcome.pending =>
        SubscriptionOperationNotice.pending,
      SubscriptionPurchaseOutcome.offeringUnavailable =>
        SubscriptionOperationNotice.offeringUnavailable,
      SubscriptionPurchaseOutcome.providerError =>
        SubscriptionOperationNotice.providerFailure,
      SubscriptionPurchaseOutcome.anomaly =>
        SubscriptionOperationNotice.supportRequired,
      SubscriptionPurchaseOutcome.failed =>
        SubscriptionOperationNotice.providerFailure,
    };
  }

  void _restore() {
    final registry = ref.read(sessionRegistryProvider).value;
    final ownerDid = registry?.billingOwner?.did;
    if (ownerDid == null ||
        !ref
            .read(subscriptionPendingOperationProvider.notifier)
            .beginRestore(ownerDid)) {
      return;
    }
    unawaited(
      _run(() async {
        try {
          final dependencies = await _ownerDependencies();
          final controller = SubscriptionRestoreController(
            ownerGuard: dependencies.guard,
            api: dependencies.api,
            revenueCat: dependencies.service,
            wait: Future<void>.delayed,
            isCurrent: () => _operationIsCurrent,
            isForeground: () => _operationIsForeground,
            onProviderConfirmed: ref
                .read(subscriptionPendingOperationProvider.notifier)
                .recordRestore,
          );
          return _handleRestoreResult(await controller.restore());
        } on Object {
          ref.read(subscriptionPendingOperationProvider.notifier).clear();
          rethrow;
        }
      }),
    );
  }

  Future<SubscriptionOperationNotice?> _handleRestoreResult(
    SubscriptionRestoreResult result,
  ) async {
    if (result.outcome != SubscriptionRestoreOutcome.pending) {
      ref.read(subscriptionPendingOperationProvider.notifier).clear();
    }
    final retry = result.retry;
    _pendingRetry = retry == null
        ? null
        : () async => _handleRestoreResult(await retry());
    if (result.assignmentLicense case final license?) {
      _pendingRetry = null;
      return _assignLicense(license, state: result.state);
    }
    return switch (result.outcome) {
      SubscriptionRestoreOutcome.completed ||
      SubscriptionRestoreOutcome.cancelled => null,
      SubscriptionRestoreOutcome.pending => SubscriptionOperationNotice.pending,
      SubscriptionRestoreOutcome.providerError =>
        SubscriptionOperationNotice.providerFailure,
      SubscriptionRestoreOutcome.anomaly =>
        SubscriptionOperationNotice.supportRequired,
      SubscriptionRestoreOutcome.failed =>
        SubscriptionOperationNotice.actionFailed,
    };
  }

  Future<SubscriptionOperationNotice?> _reconcilePending(
    PendingSubscriptionOperation pending,
  ) async {
    final dependencies = await _ownerDependencies();
    return switch (pending.kind) {
      PendingSubscriptionOperationKind.purchase => _handlePurchaseResult(
        await SubscriptionPurchaseController(
          ownerGuard: dependencies.guard,
          api: dependencies.api,
          presentPaywall: (_, _) => throw StateError(
            'Pending reconciliation cannot present checkout',
          ),
          wait: Future<void>.delayed,
          readAccess: _readAssignedAccess,
          isCurrent: () => _operationIsCurrent,
          isForeground: () => _operationIsForeground,
        ).reconcilePending(pending.tier!, pending.beforeProvider!),
      ),
      PendingSubscriptionOperationKind.restore => _handleRestoreResult(
        await SubscriptionRestoreController(
          ownerGuard: dependencies.guard,
          api: dependencies.api,
          revenueCat: dependencies.service,
          wait: Future<void>.delayed,
          isCurrent: () => _operationIsCurrent,
          isForeground: () => _operationIsForeground,
        ).reconcilePending(pending.beforeProvider!),
      ),
      PendingSubscriptionOperationKind.customerCenter =>
        _handleCustomerCenterResult(
          await CustomerCenterCoordinator(
            ownerGuard: dependencies.guard,
            api: dependencies.api,
            revenueCat: dependencies.service,
            wait: Future<void>.delayed,
            isCurrent: () => _operationIsCurrent,
            isForeground: () => _operationIsForeground,
          ).reconcilePending(),
        ),
      PendingSubscriptionOperationKind.assignment ||
      PendingSubscriptionOperationKind.unassignment => null,
    };
  }

  Future<SubscriptionAccess> _readAssignedAccess(Did did) async {
    final registry = ref.read(sessionRegistryProvider).requireValue;
    final target = registry.leaseFor(AccountKey(did.value));
    if (target == null) throw StateError('Assigned account unavailable');
    final api = await ref.read(
      subscriptionRepositoryProvider(target.account).future,
    );
    final access = await api.getAccess();
    final current = ref.read(sessionRegistryProvider).requireValue;
    if (!mounted || current.leaseFor(target.account) != target) {
      throw StateError('Assigned account session changed');
    }
    return access;
  }

  void _manage() {
    final registry = ref.read(sessionRegistryProvider).value;
    final ownerDid = registry?.billingOwner?.did;
    if (ownerDid == null ||
        !ref
            .read(subscriptionPendingOperationProvider.notifier)
            .beginCustomerCenter(ownerDid)) {
      return;
    }
    unawaited(
      _run(() async {
        try {
          final dependencies = await _ownerDependencies();
          final coordinator = CustomerCenterCoordinator(
            ownerGuard: dependencies.guard,
            api: dependencies.api,
            revenueCat: dependencies.service,
            wait: Future<void>.delayed,
            isCurrent: () => _operationIsCurrent,
            isForeground: () => _operationIsForeground,
            onProviderMutationConfirmed: ref
                .read(subscriptionPendingOperationProvider.notifier)
                .recordCustomerCenterMutation,
          );
          return _handleCustomerCenterResult(await coordinator.present());
        } on Object {
          ref.read(subscriptionPendingOperationProvider.notifier).clear();
          rethrow;
        }
      }),
    );
  }

  Future<SubscriptionOperationNotice?> _handleCustomerCenterResult(
    CustomerCenterResult result,
  ) async {
    if (result.outcome != CustomerCenterOutcome.pending) {
      ref.read(subscriptionPendingOperationProvider.notifier).clear();
    }
    final retry = result.retry;
    _pendingRetry = retry == null
        ? null
        : () async => _handleCustomerCenterResult(await retry());
    return switch (result.outcome) {
      CustomerCenterOutcome.completed ||
      CustomerCenterOutcome.refreshed ||
      CustomerCenterOutcome.cancelled => null,
      CustomerCenterOutcome.pending => SubscriptionOperationNotice.pending,
      CustomerCenterOutcome.providerError =>
        SubscriptionOperationNotice.providerFailure,
      CustomerCenterOutcome.anomaly =>
        SubscriptionOperationNotice.supportRequired,
      CustomerCenterOutcome.failed => SubscriptionOperationNotice.actionFailed,
    };
  }

  void _assign(BillingLicense license) =>
      unawaited(_run(() => _assignLicense(license)));

  Future<SubscriptionOperationNotice?> _assignLicense(
    BillingLicense license, {
    BillingState? state,
  }) async {
    final registry = ref.read(sessionRegistryProvider).requireValue;
    final pageModel = ref.read(subscriptionPageModelProvider).requireValue;
    final candidates = assignmentCandidates(
      registry: registry,
      retainedLeases: registry.sessions.keys
          .map((did) => registry.leaseFor(AccountKey(did.value)))
          .nonNulls,
      state: state ?? pageModel.billingState!,
    );
    final candidateAccounts = candidates
        .map((candidate) => candidate.account)
        .toSet();
    final target = await _selectTarget(
      AccountSwitcherState.fromRegistry(
        registry,
      ).rows.where((row) => candidateAccounts.contains(row.lease.account)),
    );
    if (target == null || !mounted) {
      return null;
    }
    final targetLabel = pageModel.assignedAccountLabels[target.account.did];
    if (targetLabel == null ||
        !await _confirmAssignment(
          license.tier,
          targetLabel,
          isReassignment: license.assignedDid != null,
        )) {
      return null;
    }
    final pendingController = ref.read(
      subscriptionPendingOperationProvider.notifier,
    );
    if (!pendingController.beginAssignment(
      registry.activeLease!.session.account.did,
      license.id,
    )) {
      return null;
    }
    try {
      final dependencies = await _ownerDependencies();
      final result = await SubscriptionAssignmentController(
        ownerGuard: dependencies.guard,
        readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
        api: dependencies.api,
        readTargetAccess: (lease) async {
          final api = await ref.read(
            subscriptionRepositoryProvider(lease.account).future,
          );
          return api.getAccess();
        },
        isCurrent: () => _operationIsCurrent,
      ).assign(licenseId: license.id, target: target);
      if (result.targetAccess != null) {
        ref.invalidate(subscriptionAccessProvider(target));
      }
      return _assignmentNotice(result.outcome);
    } finally {
      pendingController.completeAssignmentMutation(
        kind: PendingSubscriptionOperationKind.assignment,
        affectedTarget: target,
      );
    }
  }

  void _unassign(BillingLicense license) => unawaited(
    _run(() async {
      if (!await _confirmUnassignment()) return null;
      final registry = ref.read(sessionRegistryProvider).requireValue;
      final affectedTarget = license.assignedDid == null
          ? null
          : registry.leaseFor(AccountKey(license.assignedDid!.value));
      final pendingController = ref.read(
        subscriptionPendingOperationProvider.notifier,
      );
      if (!pendingController.beginUnassignment(
        registry.activeLease!.session.account.did,
        license.id,
      )) {
        return null;
      }
      try {
        final dependencies = await _ownerDependencies();
        final result = await SubscriptionAssignmentController(
          ownerGuard: dependencies.guard,
          readRegistry: () => ref.read(sessionRegistryProvider).requireValue,
          api: dependencies.api,
          readTargetAccess: (_) =>
              throw StateError('No target for unassignment'),
          isCurrent: () => _operationIsCurrent,
        ).unassign(licenseId: license.id);
        return _assignmentNotice(result.outcome);
      } finally {
        pendingController.completeAssignmentMutation(
          kind: PendingSubscriptionOperationKind.unassignment,
          affectedTarget: affectedTarget,
        );
      }
    }),
  );

  Future<
    ({
      BillingOwnerGuard guard,
      RevenueCatService service,
      SubscriptionApi api,
    })
  >
  _ownerDependencies() async {
    final registry = ref.read(sessionRegistryProvider).requireValue;
    final lease = registry.activeLease!.session;
    return (
      guard: BillingOwnerGuard(
        () => ref.read(sessionRegistryProvider).requireValue,
      ),
      service: ref.read(revenueCatServiceProvider),
      api: await ref.read(subscriptionRepositoryProvider(lease.account).future),
    );
  }

  Future<AccountSessionLease?> _selectTarget(
    Iterable<AccountSwitcherRow> candidates,
  ) => showCraftskyModal<AccountSessionLease>(
    context,
    builder: (context) => CraftskyDialog(
      title: AppLocalizations.of(context).subscriptionsAssign,
      body: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          for (final candidate in candidates)
            AccountSwitcherRowTile(
              row: candidate,
              onTap: () => Navigator.pop(context, candidate.lease),
            ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(MaterialLocalizations.of(context).cancelButtonLabel),
        ),
      ],
    ),
  );

  Future<bool> _confirmAssignment(
    SubscriptionTier tier,
    String target, {
    required bool isReassignment,
  }) => _confirm(
    AppLocalizations.of(context).subscriptionsAssignmentConfirmTitle,
    [
      AppLocalizations.of(context).subscriptionsAssignmentConfirmBody(
        _tierLabel(AppLocalizations.of(context), tier),
        target,
      ),
      if (isReassignment)
        AppLocalizations.of(context).subscriptionsReassignmentCooldown,
    ].join('\n\n'),
    AppLocalizations.of(context).subscriptionsAssign,
  );

  Future<bool> _confirmUnassignment() => _confirm(
    AppLocalizations.of(context).subscriptionsUnassignmentConfirmTitle,
    AppLocalizations.of(context).subscriptionsUnassignmentConfirmBody,
    AppLocalizations.of(context).subscriptionsUnassign,
  );

  Future<bool> _confirm(String title, String body, String action) =>
      showCraftskyConfirmDialog(
        context,
        title: title,
        message: body,
        confirmLabel: action,
      );

  SubscriptionOperationNotice? _assignmentNotice(
    AssignmentMutationOutcome outcome,
  ) => switch (outcome) {
    AssignmentMutationOutcome.success => null,
    AssignmentMutationOutcome.conflict =>
      SubscriptionOperationNotice.assignmentConflict,
    AssignmentMutationOutcome.cooldown =>
      SubscriptionOperationNotice.assignmentCooldown,
    AssignmentMutationOutcome.licenseNotFound =>
      SubscriptionOperationNotice.licenseNotFound,
    AssignmentMutationOutcome.targetIneligible =>
      SubscriptionOperationNotice.targetIneligible,
    AssignmentMutationOutcome.unauthorized =>
      SubscriptionOperationNotice.unauthorized,
    AssignmentMutationOutcome.failed =>
      SubscriptionOperationNotice.actionFailed,
    AssignmentMutationOutcome.cancelled ||
    AssignmentMutationOutcome.inProgress => null,
  };
}

class SubscriptionPageView extends StatelessWidget {
  const SubscriptionPageView({
    required this.model,
    this.pendingOperation,
    super.key,
  });

  final SubscriptionPageModel model;
  final PendingSubscriptionOperation? pendingOperation;

  PendingSubscriptionOperation? get _activeOperation =>
      model.operationNotice == SubscriptionOperationNotice.pending
      ? null
      : pendingOperation;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.subscriptionsTitle)),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 720),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _AccessCard(access: model.access),
                const SizedBox(height: 16),
                ..._roleContent(context),
              ],
            ),
          ),
        ),
      ),
    );
  }

  List<Widget> _roleContent(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return switch (model.role) {
      SubscriptionPageRole.neverReserved => [
        _MessageCard(
          title: l10n.subscriptionsChooseOwnerTitle,
          body: l10n.subscriptionsChooseOwnerBody,
          action: l10n.subscriptionsChooseOwnerAction,
          onAction: model.onConfirmOwner,
        ),
      ],
      SubscriptionPageRole.incompleteOwner => [
        _MessageCard(
          body: l10n.subscriptionsSetupIncomplete,
          action: l10n.subscriptionsRetrySetup,
          onAction: model.onRetrySetup,
        ),
      ],
      SubscriptionPageRole.inactiveOwner => [
        _MessageCard(
          body: model.ownerRetained
              ? l10n.subscriptionsOwnerInactive
              : l10n.subscriptionsOwnerReauthenticate,
          action: model.ownerRetained ? l10n.subscriptionsSwitchToOwner : null,
          onAction: model.onSwitchToOwner,
        ),
      ],
      SubscriptionPageRole.beneficiary => [
        _MessageCard(
          body: l10n.subscriptionsBeneficiaryExplanation,
          action: model.ownerRetained ? l10n.subscriptionsSwitchToOwner : null,
          onAction: model.onSwitchToOwner,
        ),
      ],
      SubscriptionPageRole.owner => _ownerContent(context),
    };
  }

  List<Widget> _ownerContent(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = model.billingState;
    if (state == null) {
      return [
        _MessageCard(
          body: model.ownerRecoveryLocked
              ? l10n.subscriptionsOwnerRecoveryLocked
              : model.ownerSignInRequired
              ? l10n.subscriptionsOwnerSignInRequired
              : l10n.subscriptionsProviderFailure,
          isError: model.ownerRecoveryLocked || model.ownerSignInRequired,
          action: model.ownerRecoveryLocked || model.ownerSignInRequired
              ? null
              : l10n.subscriptionsRefresh,
          onAction: model.ownerRecoveryLocked || model.ownerSignInRequired
              ? null
              : model.onRefresh,
        ),
      ];
    }
    return [
      Text(l10n.subscriptionsIndependentLicenses),
      const SizedBox(height: 16),
      if (model.operationNotice case final notice?) ...[
        _OperationNotice(notice: notice, onRefresh: model.onRefresh),
        const SizedBox(height: 16),
      ],
      for (final tier in const [
        SubscriptionTier.plus,
        SubscriptionTier.business,
      ]) ...[
        _TierCard(
          model: model,
          state: state,
          tier: tier,
          pendingOperation: _activeOperation,
        ),
        const SizedBox(height: 16),
      ],
      if (model.billingAvailability == BillingAvailability.unavailable)
        Text(l10n.subscriptionsBillingUnavailable)
      else
        Wrap(
          spacing: 12,
          runSpacing: 8,
          children: [
            _TextActionButton(
              label: l10n.subscriptionsRestore,
              onPressed: model.onRestore,
              isLoading:
                  _activeOperation?.kind ==
                  PendingSubscriptionOperationKind.restore,
            ),
            _PrimaryActionButton(
              label: l10n.subscriptionsManage,
              onPressed: model.onManage,
              isLoading:
                  _activeOperation?.kind ==
                  PendingSubscriptionOperationKind.customerCenter,
            ),
          ],
        ),
    ];
  }
}

class _AccessCard extends StatelessWidget {
  const _AccessCard({required this.access});

  final SubscriptionAccess access;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return CraftskyCard(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.subscriptionsEffectiveAccess(
              _tierLabel(l10n, access.effectiveTier),
            ),
            style: Theme.of(context).textTheme.titleMedium,
          ),
          if (access.assignedTier case final tier?
              when !access.givesAccess) ...[
            const SizedBox(height: 6),
            Text(l10n.subscriptionsDormantAssignment(_tierLabel(l10n, tier))),
          ],
        ],
      ),
    );
  }
}

class _TierCard extends StatelessWidget {
  const _TierCard({
    required this.model,
    required this.state,
    required this.tier,
    required this.pendingOperation,
  });

  final SubscriptionPageModel model;
  final BillingState state;
  final SubscriptionTier tier;
  final PendingSubscriptionOperation? pendingOperation;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final eligibility = purchaseEligibility(state, tier);
    final presentation = eligibility.reason == PurchaseEligibilityReason.anomaly
        ? SubscriptionTierPresentation.anomaly
        : projectTierPresentation(state, tier: tier);
    final status = _statusLabel(l10n, presentation);
    final subscriptionsById = {
      for (final value in state.subscriptions) value.id: value,
    };
    final tierLicenses = state.licenses
        .where((value) => value.tier == tier)
        .toList(growable: false);
    final license =
        tierLicenses
            .where(
              (value) =>
                  subscriptionsById[value.subscriptionId]?.givesAccess ?? false,
            )
            .firstOrNull ??
        tierLicenses.firstOrNull;
    final secondaryLicenses = tierLicenses
        .where((value) => value != license)
        .toList(growable: false);
    final subscription = license == null
        ? null
        : state.subscriptions
              .where((value) => value.id == license.subscriptionId)
              .firstOrNull;
    final periodEnd = subscription?.currentPeriodEndsAt;
    final accessEnd = subscription?.endsAt ?? periodEnd;
    final summary = _subscriptionSummary(
      context,
      l10n,
      presentation,
      subscription,
      accessEnd,
    );
    final action = _primaryAction(l10n, presentation);
    final purchaseIsLoading =
        pendingOperation?.kind == PendingSubscriptionOperationKind.purchase &&
        pendingOperation?.tier == tier;
    final assignmentIsLoading =
        pendingOperation?.kind == PendingSubscriptionOperationKind.assignment &&
        pendingOperation?.licenseId == license?.id;
    final unassignmentIsLoading =
        pendingOperation?.kind ==
            PendingSubscriptionOperationKind.unassignment &&
        pendingOperation?.licenseId == license?.id;
    final canAssign =
        license != null &&
        license.assignedDid == null &&
        license.assignable &&
        (presentation == SubscriptionTierPresentation.active ||
            presentation == SubscriptionTierPresentation.canceledAccessible) &&
        (model.onAssign != null || assignmentIsLoading);
    final canReassign =
        license != null &&
        license.assignedDid != null &&
        license.assignable &&
        (presentation == SubscriptionTierPresentation.active ||
            presentation == SubscriptionTierPresentation.canceledAccessible) &&
        (model.onAssign != null || assignmentIsLoading);
    final canUnassign =
        license != null &&
        license.assignedDid != null &&
        (model.onUnassign != null || unassignmentIsLoading);
    final tierLabel = _tierLabel(l10n, tier);
    final theme = Theme.of(context);
    final accent = tier == SubscriptionTier.plus
        ? theme.colorScheme.primary
        : theme.brightness == Brightness.dark
        ? theme.colorScheme.tertiary
        : BrandColors.butterDeep;
    final tierIcon = tier == SubscriptionTier.plus
        ? Icons.auto_awesome_outlined
        : Icons.storefront_outlined;
    final price = model.tierPrices[tier];
    final showPitch =
        presentation == SubscriptionTierPresentation.empty ||
        (presentation == SubscriptionTierPresentation.dormant &&
            eligibility.eligible);
    return Semantics(
      container: true,
      label: l10n.subscriptionsActionSemantic(
        tierLabel,
        status,
        action?.$1 ?? '',
      ),
      child: CraftskyCard(
        padding: EdgeInsets.zero,
        child: DecoratedBox(
          decoration: BoxDecoration(
            color: Color.alphaBlend(
              accent.withValues(alpha: 0.08),
              theme.colorScheme.surface,
            ),
          ),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Container(
                      width: 44,
                      height: 44,
                      decoration: BoxDecoration(
                        color: accent.withValues(alpha: 0.16),
                        shape: BoxShape.circle,
                        border: Border.all(color: accent, width: 1.5),
                      ),
                      child: Icon(tierIcon, color: accent),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(tierLabel, style: theme.textTheme.titleLarge),
                          const SizedBox(height: 2),
                          Text(
                            price == null
                                ? l10n.subscriptionsPriceUnavailable
                                : l10n.subscriptionsTierPrice(price),
                            style: theme.textTheme.titleSmall?.copyWith(
                              color: accent,
                              fontWeight: FontWeight.w800,
                            ),
                          ),
                        ],
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                if (summary != null) Text(summary),
                if (showPitch) ...[
                  if (summary != null) const SizedBox(height: 8),
                  Text(
                    tier == SubscriptionTier.plus
                        ? l10n.subscriptionsPlusPitch
                        : l10n.subscriptionsBusinessPitch,
                    style: theme.textTheme.bodyLarge,
                  ),
                ],
                if (license?.assignedDid != null) ...[
                  const SizedBox(height: 6),
                  Text(_assignmentLabel(l10n, license!)),
                ],
                if (action != null) ...[
                  const SizedBox(height: 12),
                  Align(
                    alignment: AlignmentDirectional.centerStart,
                    child: _ActionButton(
                      key: ValueKey('subscription-view-details-${tier.name}'),
                      label: action.$1,
                      onPressed: action.$2,
                      isLoading:
                          purchaseIsLoading ||
                          (pendingOperation?.kind ==
                              PendingSubscriptionOperationKind.customerCenter),
                    ),
                  ),
                ],
                if (canAssign) ...[
                  const SizedBox(height: 8),
                  Align(
                    alignment: AlignmentDirectional.centerStart,
                    child: _ActionButton(
                      label: l10n.subscriptionsAssign,
                      onPressed: model.onAssign == null
                          ? null
                          : () => model.onAssign!(license),
                      isLoading: assignmentIsLoading,
                    ),
                  ),
                ],
                if (canReassign) ...[
                  const SizedBox(height: 8),
                  Align(
                    alignment: AlignmentDirectional.centerStart,
                    child: _ActionButton(
                      label: l10n.subscriptionsReassign,
                      onPressed: model.onAssign == null
                          ? null
                          : () => model.onAssign!(license),
                      isLoading: assignmentIsLoading,
                    ),
                  ),
                ],
                if (canUnassign) ...[
                  const SizedBox(height: 8),
                  Align(
                    alignment: AlignmentDirectional.centerStart,
                    child: _TextActionButton(
                      label: l10n.subscriptionsUnassign,
                      onPressed: model.onUnassign == null
                          ? null
                          : () => model.onUnassign!(license),
                      isLoading: unassignmentIsLoading,
                    ),
                  ),
                ],
                for (final secondary in secondaryLicenses)
                  ..._secondaryLicenseDetails(
                    context,
                    l10n,
                    secondary,
                    subscriptionsById[secondary.subscriptionId],
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  (String, VoidCallback?)? _primaryAction(
    AppLocalizations l10n,
    SubscriptionTierPresentation presentation,
  ) {
    if (model.operationNotice == SubscriptionOperationNotice.pending) {
      return null;
    }
    if (model.billingAvailability == BillingAvailability.unavailable) {
      return null;
    }
    return switch (presentation) {
      SubscriptionTierPresentation.empty => (
        l10n.subscriptionsViewDetails,
        model.onPurchase == null ? null : () => model.onPurchase!(tier),
      ),
      SubscriptionTierPresentation.dormant
          when purchaseEligibility(state, tier).eligible =>
        (
          l10n.subscriptionsViewDetails,
          model.onPurchase == null ? null : () => model.onPurchase!(tier),
        ),
      SubscriptionTierPresentation.dormant ||
      SubscriptionTierPresentation.stale ||
      SubscriptionTierPresentation.pending ||
      SubscriptionTierPresentation.statusUnavailable ||
      SubscriptionTierPresentation.active ||
      SubscriptionTierPresentation.canceledAccessible ||
      SubscriptionTierPresentation.anomaly => (
        l10n.subscriptionsViewDetails,
        model.onManage,
      ),
    };
  }

  String _assignmentLabel(AppLocalizations l10n, BillingLicense license) {
    final assignedDid = license.assignedDid;
    if (assignedDid == null) return l10n.subscriptionsUnassigned;
    return l10n.subscriptionsAssignedTo(
      model.assignedAccountLabels[assignedDid] ??
          l10n.subscriptionsStatusUnavailable,
    );
  }

  String _formatDate(BuildContext context, DateTime value) =>
      MaterialLocalizations.of(context).formatMediumDate(value.toLocal());

  String? _subscriptionSummary(
    BuildContext context,
    AppLocalizations l10n,
    SubscriptionTierPresentation presentation,
    BillingSubscription? subscription,
    DateTime? accessEnd,
  ) => switch (presentation) {
    SubscriptionTierPresentation.empty ||
    SubscriptionTierPresentation.dormant => null,
    SubscriptionTierPresentation.active
        when subscription?.autoRenewalStatus == 'will_renew' &&
            accessEnd != null =>
      l10n.subscriptionsRenewsOn(_formatDate(context, accessEnd)),
    SubscriptionTierPresentation.active ||
    SubscriptionTierPresentation.canceledAccessible when accessEnd != null =>
      l10n.subscriptionsAccessUntil(_formatDate(context, accessEnd)),
    _ => _statusLabel(l10n, presentation),
  };

  List<Widget> _secondaryLicenseDetails(
    BuildContext context,
    AppLocalizations l10n,
    BillingLicense license,
    BillingSubscription? subscription,
  ) {
    final presentation = projectLicensePresentation(state, license);
    final periodEnd = subscription?.currentPeriodEndsAt;
    final accessEnd = subscription?.endsAt ?? periodEnd;
    final summary = _subscriptionSummary(
      context,
      l10n,
      presentation,
      subscription,
      accessEnd,
    );
    final canAssign =
        projectTierPresentation(state, tier: tier) !=
            SubscriptionTierPresentation.anomaly &&
        license.assignable &&
        (presentation == SubscriptionTierPresentation.active ||
            presentation == SubscriptionTierPresentation.canceledAccessible) &&
        model.onAssign != null;
    final canUnassign = license.assignedDid != null && model.onUnassign != null;
    return [
      const SizedBox(height: 12),
      const Divider(),
      if (summary != null) Text(summary),
      if (license.assignedDid != null) Text(_assignmentLabel(l10n, license)),
      if (canAssign) ...[
        const SizedBox(height: 8),
        Align(
          alignment: AlignmentDirectional.centerStart,
          child: _ActionButton(
            label: license.assignedDid == null
                ? l10n.subscriptionsAssign
                : l10n.subscriptionsReassign,
            onPressed: () => model.onAssign!(license),
          ),
        ),
      ],
      if (canUnassign) ...[
        const SizedBox(height: 8),
        Align(
          alignment: AlignmentDirectional.centerStart,
          child: _TextActionButton(
            label: l10n.subscriptionsUnassign,
            onPressed: () => model.onUnassign!(license),
          ),
        ),
      ],
    ];
  }
}

class _MessageCard extends StatelessWidget {
  const _MessageCard({
    required this.body,
    this.title,
    this.action,
    this.onAction,
    this.isError = false,
  });

  final String? title;
  final String body;
  final String? action;
  final VoidCallback? onAction;
  final bool isError;

  @override
  Widget build(BuildContext context) => CraftskyCard(
    clipBehavior: Clip.none,
    padding: const EdgeInsets.all(16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (title case final title?) ...[
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
        ],
        Text(
          body,
          style: isError
              ? TextStyle(color: Theme.of(context).colorScheme.error)
              : null,
        ),
        if (action case final action?) ...[
          const SizedBox(height: 12),
          Align(
            alignment: AlignmentDirectional.centerStart,
            child: _ActionButton(label: action, onPressed: onAction),
          ),
        ],
      ],
    ),
  );
}

class _OperationNotice extends StatelessWidget {
  const _OperationNotice({required this.notice, required this.onRefresh});

  final SubscriptionOperationNotice notice;
  final VoidCallback? onRefresh;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final message = switch (notice) {
      SubscriptionOperationNotice.offeringUnavailable =>
        l10n.subscriptionsOfferingUnavailable,
      SubscriptionOperationNotice.providerFailure =>
        l10n.subscriptionsProviderFailure,
      SubscriptionOperationNotice.assignmentConflict =>
        l10n.subscriptionsAssignmentConflict,
      SubscriptionOperationNotice.assignmentCooldown =>
        l10n.subscriptionsAssignmentCooldown,
      SubscriptionOperationNotice.licenseNotFound =>
        l10n.subscriptionsLicenseNotFound,
      SubscriptionOperationNotice.targetIneligible =>
        l10n.subscriptionsTargetIneligible,
      SubscriptionOperationNotice.unauthorized =>
        l10n.subscriptionsUnauthorized,
      SubscriptionOperationNotice.actionFailed =>
        l10n.subscriptionsActionFailed,
      SubscriptionOperationNotice.pending => l10n.subscriptionsPendingRefresh,
      SubscriptionOperationNotice.supportRequired =>
        l10n.subscriptionsStatusAnomaly,
    };
    return Semantics(
      liveRegion: true,
      child: _MessageCard(
        body: message,
        action: l10n.subscriptionsRefresh,
        onAction: onRefresh,
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.label,
    required this.onPressed,
    this.isLoading = false,
    super.key,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool isLoading;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 6),
    child: ChunkyButton(
      onPressed: onPressed,
      child: _ActionButtonChild(label: label, isLoading: isLoading),
    ),
  );
}

class _PrimaryActionButton extends StatelessWidget {
  const _PrimaryActionButton({
    required this.label,
    required this.onPressed,
    this.isLoading = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool isLoading;

  @override
  Widget build(BuildContext context) => FilledButton(
    onPressed: onPressed,
    child: _ActionButtonChild(label: label, isLoading: isLoading),
  );
}

class _TextActionButton extends StatelessWidget {
  const _TextActionButton({
    required this.label,
    required this.onPressed,
    this.isLoading = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool isLoading;

  @override
  Widget build(BuildContext context) => TextButton(
    onPressed: onPressed,
    style: TextButton.styleFrom(minimumSize: const Size(48, 48)),
    child: _ActionButtonChild(label: label, isLoading: isLoading),
  );
}

class _ActionButtonChild extends StatelessWidget {
  const _ActionButtonChild({required this.label, required this.isLoading});

  final String label;
  final bool isLoading;

  @override
  Widget build(BuildContext context) {
    if (!isLoading) return Text(label);
    return Semantics(
      label: label,
      child: Stack(
        alignment: Alignment.center,
        children: [
          Opacity(opacity: 0, child: Text(label)),
          const ExcludeSemantics(
            child: SizedBox.square(
              dimension: 18,
              child: CircularProgressIndicator(
                key: ValueKey('subscription-action-progress'),
                strokeWidth: 2,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

String _tierLabel(AppLocalizations l10n, SubscriptionTier tier) =>
    switch (tier) {
      SubscriptionTier.free => l10n.subscriptionsTierFree,
      SubscriptionTier.plus => l10n.subscriptionsTierPlus,
      SubscriptionTier.business => l10n.subscriptionsTierBusiness,
    };

String _statusLabel(
  AppLocalizations l10n,
  SubscriptionTierPresentation presentation,
) => switch (presentation) {
  SubscriptionTierPresentation.empty => l10n.subscriptionsStatusAvailable,
  SubscriptionTierPresentation.active => l10n.subscriptionsStatusActive,
  SubscriptionTierPresentation.canceledAccessible =>
    l10n.subscriptionsStatusCanceledAccessible,
  SubscriptionTierPresentation.dormant => l10n.subscriptionsStatusDormant,
  SubscriptionTierPresentation.stale => l10n.subscriptionsStatusStale,
  SubscriptionTierPresentation.pending => l10n.subscriptionsStatusPending,
  SubscriptionTierPresentation.anomaly => l10n.subscriptionsStatusAnomaly,
  SubscriptionTierPresentation.statusUnavailable =>
    l10n.subscriptionsStatusUnavailable,
};
