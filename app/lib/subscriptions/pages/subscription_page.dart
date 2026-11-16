import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/errors/app_error.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/subscriptions/data/subscription_api_client.dart';
import 'package:craftsky_app/subscriptions/models/billing_state.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/models/subscription_page_model.dart';
import 'package:craftsky_app/subscriptions/models/subscription_presentation.dart';
import 'package:craftsky_app/subscriptions/models/subscription_rules.dart';
import 'package:craftsky_app/subscriptions/providers/customer_center_coordinator.dart';
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_page_model_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_pending_operation_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_purchase_controller.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_restore_controller.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_guard.dart';
import 'package:craftsky_app/subscriptions/services/paywall_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/subscriptions/services/subscription_diagnostics.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/plus_tier_icon.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:crypto/crypto.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:material_ui/material_ui.dart';

export 'package:craftsky_app/subscriptions/models/subscription_page_model.dart';

final subscriptionManagementLauncherProvider =
    Provider<Future<bool> Function(Uri)>((ref) => launchExternalLink);

class SubscriptionPage extends ConsumerStatefulWidget {
  const new({this.featureTier, super.key});

  /// Feature entry reuses owner checks, purchase and reconciliation.
  final SubscriptionTier? featureTier;

  @override
  ConsumerState<SubscriptionPage> createState() => _SubscriptionPageState();
}

class _SubscriptionPageState extends ConsumerState<SubscriptionPage>
    with WidgetsBindingObserver {
  bool _busy = false;
  bool _featureStarted = false;
  ActiveAccountLease? _featureOwner;
  bool _foreground = true;
  SubscriptionOperationNotice? _notice;
  Did? _visibleDid;
  String? _conflictingAccountLabel;
  Future<SubscriptionOperationNotice?> Function()? _pendingRetry;
  late final ProviderContainer _operationContainer;
  late final SubscriptionPendingOperationController _pendingOperations;

  @override
  void initState() {
    super.initState();
    // Provider operations can outlive this route. Keep their state and cleanup
    // independent of ConsumerState.ref, which is unusable after disposal.
    _operationContainer = ProviderScope.containerOf(context, listen: false);
    _pendingOperations = _operationContainer.read(
      subscriptionPendingOperationProvider.notifier,
    );
    _featureOwner = _operationContainer
        .read(sessionRegistryProvider)
        .value
        ?.activeLease;
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
    if (_foreground && mounted) unawaited(_refresh());
  }

  bool get _operationIsCurrent => mounted;
  bool get _operationIsForeground => mounted && _foreground;

  @override
  Widget build(BuildContext context) {
    final activeDid = ref.watch(
      sessionRegistryProvider.select(
        (value) => value.value?.activeLease?.session.account.did,
      ),
    );
    if (_visibleDid != activeDid) {
      _visibleDid = activeDid;
      _notice = null;
      _pendingRetry = null;
      _conflictingAccountLabel = null;
    }
    if (widget.featureTier != null &&
        (_featureOwner == null ||
            !ref
                .read(sessionRegistryProvider)
                .requireValue
                .isCurrent(_featureOwner))) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && ModalRoute.of(context)?.isCurrent == true) {
          Navigator.of(context).pop();
        }
      });
      return const SizedBox.shrink();
    }
    final model = ref.watch(subscriptionPageModelProvider);
    final pendingOperation = ref.watch(subscriptionPendingOperationProvider);
    final operationBlocked = _busy || pendingOperation != null;
    final featureTier = widget.featureTier;
    if (featureTier != null &&
        !_featureStarted &&
        model is AsyncData<SubscriptionPageModel>) {
      _featureStarted = true;
      final value = model.value;
      final alreadyAllowed = featureTier == SubscriptionTier.plus
          ? value.access.allowsPlus
          : value.access.allowsBusiness;
      if (alreadyAllowed) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && ModalRoute.of(context)?.isCurrent == true) {
            Navigator.of(context).pop();
          }
        });
        return _featureLoading();
      }
      final state = value.billingState;
      if (!operationBlocked &&
          value.billingAvailability == BillingAvailability.available &&
          state != null &&
          purchaseEligibility(
            state,
            featureTier,
            store: value.nativeStore,
            allowTierChanges: value.allowNativeTierChanges,
          ).eligible) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted &&
              ref
                  .read(sessionRegistryProvider)
                  .requireValue
                  .isCurrent(_featureOwner)) {
            _purchase(featureTier);
          }
        });
        return _featureLoading();
      }
    }
    if (featureTier != null &&
        (_busy || model is AsyncLoading<SubscriptionPageModel>)) {
      return _featureLoading();
    }
    return switch (model) {
      AsyncData(:final value) => SubscriptionPageView(
        pendingOperation: pendingOperation?.ownerDid == value.access.did
            ? pendingOperation
            : null,
        model: SubscriptionPageModel(
          access: value.access,
          billingAvailability: value.billingAvailability,
          billingState: value.billingState,
          nativeStore: value.nativeStore,
          allowNativeTierChanges: value.allowNativeTierChanges,
          conflictingAccountLabel: _conflictingAccountLabel,
          operationNotice:
              _notice ??
              (pendingOperation?.ownerDid == value.access.did &&
                      pendingOperation?.canReconcile == true &&
                      !_busy
                  ? SubscriptionOperationNotice.pending
                  : value.operationNotice),
          tierPrices: value.tierPrices,
          onPurchase: operationBlocked ? null : _purchase,
          onCancelCheckout: operationBlocked ? null : _cancelPendingCheckout,
          onRestore: operationBlocked ? null : _restore,
          onManage: operationBlocked ? null : _manage,
          onRefresh:
              _busy ||
                  (pendingOperation != null && !pendingOperation.canReconcile)
              ? null
              : _refresh,
        ),
      ),
      AsyncLoading() => Scaffold(
        appBar: AppBar(
          title: Text(AppLocalizations.of(context).subscriptionsTitle),
        ),
        body: const Center(child: CircularProgressIndicator()),
      ),
      AsyncError(:final error) => Scaffold(
        appBar: AppBar(
          title: Text(AppLocalizations.of(context).subscriptionsTitle),
        ),
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                AppErrorMapper.map(error).message(AppLocalizations.of(context)),
              ),
              _ActionButton(
                label: AppLocalizations.of(context).subscriptionsRefresh,
                onPressed: _refresh,
              ),
            ],
          ),
        ),
      ),
    };
  }

  Widget _featureLoading() => Center(
    child: Material(
      color: Colors.transparent,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const CircularProgressIndicator(),
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: Text(AppLocalizations.of(context).actionCancel),
          ),
        ],
      ),
    ),
  );

  Future<void> _run(
    Future<SubscriptionOperationNotice?> Function() action,
  ) async {
    if (_busy) return;
    final started = _operationContainer
        .read(sessionRegistryProvider)
        .value
        ?.activeLease;
    final diagnostics = SubscriptionDiagnostics(
      reporter: _operationContainer.read(errorReporterProvider),
      accountDid: started?.session.account.did.value,
    );
    setState(() {
      _busy = true;
      _conflictingAccountLabel = null;
    });
    try {
      SubscriptionOperationNotice? notice;
      try {
        notice = await action();
      } on ApiUnauthorized {
        notice = SubscriptionOperationNotice.unauthorized;
      } on ApiException catch (error, stack) {
        diagnostics.report(error, stack, operation: 'billing.action');
        notice = SubscriptionOperationNotice.actionFailed;
      } on Object catch (error, stack) {
        diagnostics.report(error, stack, operation: 'billing.action');
        notice = SubscriptionOperationNotice.providerFailure;
      }
      if (!mounted ||
          started == null ||
          !_operationContainer
              .read(sessionRegistryProvider)
              .requireValue
              .isCurrent(started)) {
        return;
      }
      setState(() => _notice = notice);
      ref.invalidate(subscriptionPageModelProvider);
    } finally {
      final session = started?.session;
      if (session != null &&
          _operationContainer
                  .read(sessionRegistryProvider)
                  .value
                  ?.leaseFor(session.account) ==
              session) {
        _operationContainer.invalidate(subscriptionAccessProvider(session));
      }
      if (mounted) {
        setState(() => _busy = false);
        if (widget.featureTier != null &&
            _notice == null &&
            ModalRoute.of(context)?.isCurrent == true &&
            _operationContainer
                    .read(sessionRegistryProvider)
                    .value
                    ?.isCurrent(_featureOwner) ==
                true) {
          Navigator.of(context).pop();
        }
      }
    }
  }

  void _cancelPendingCheckout() {
    unawaited(
      _run(() async {
        final dependencies = await _ownerDependencies();
        final api = dependencies.api;
        if (api is! PendingCheckoutCancellationApi) {
          return SubscriptionOperationNotice.actionFailed;
        }
        final owner = dependencies.guard.capture();
        await dependencies.guard.dispatch(
          owner,
          (api as PendingCheckoutCancellationApi).cancelPendingCheckout,
        );
        return null;
      }),
    );
  }

  Future<void> _refresh() async {
    if (_busy) return;
    final pendingOperation = ref.read(subscriptionPendingOperationProvider);
    final registry = ref.read(sessionRegistryProvider).value;
    if (pendingOperation != null &&
        pendingOperation.canReconcile &&
        registry?.activeLease?.session.account.did ==
            pendingOperation.ownerDid) {
      await _run(() => _reconcilePending(pendingOperation));
      return;
    }
    if (pendingOperation != null) return;
    final retry = _pendingRetry;
    if (retry != null && !_busy) {
      await _run(retry);
      return;
    }
    _pendingRetry = null;
    setState(() => _notice = null);
    ref.invalidate(subscriptionPageModelProvider);
    final lease = ref.read(sessionRegistryProvider).value?.activeLease?.session;
    if (lease != null) ref.invalidate(subscriptionAccessProvider(lease));
    try {
      await ref.read(subscriptionPageModelProvider.future);
    } on Object {
      // The observed provider owns the refresh failure and its recovery UI.
    }
  }

  void _recordStoreConflict(StoreAccountConflictException conflict) {
    if (!mounted) return;
    final registry = _operationContainer
        .read(sessionRegistryProvider)
        .requireValue;
    _conflictingAccountLabel = null;
    for (final entry in registry.billingCustomers.entries) {
      final matches =
          entry.value == conflict.ownerAppUserId ||
          sha256.convert(utf8.encode(entry.value)).toString() ==
              conflict.ownerHash;
      final session = registry.sessions[entry.key];
      if (matches && session != null) {
        _conflictingAccountLabel =
            '@${session.handle.value} (${entry.key.value})';
        break;
      }
    }
  }

  void _purchase(SubscriptionTier tier) {
    final registry = ref.read(sessionRegistryProvider).value;
    final ownerDid = registry?.activeLease?.session.account.did;
    if (ownerDid == null || !_pendingOperations.beginPurchase(ownerDid, tier)) {
      return;
    }
    final operationId = _operationContainer
        .read(subscriptionPendingOperationProvider)!
        .operationId;
    unawaited(
      _run(() async {
        try {
          final dependencies = await _ownerDependencies();
          final paywall = PaywallCoordinator(
            dependencies.service,
            dependencies.guard,
            RevenueCatIdentityGuard(dependencies.service),
            diagnostics: dependencies.diagnostics,
            onStoreConflict: _recordStoreConflict,
            isCurrent: () => _operationIsCurrent,
          );
          final controller = SubscriptionPurchaseController(
            ownerGuard: dependencies.guard,
            diagnostics: dependencies.diagnostics,
            api: dependencies.api,
            nativeStore: dependencies.nativeStore,
            allowNativeTierChanges: dependencies.allowNativeTierChanges,
            presentPaywall: paywall.present,
            presentPlanChange: paywall.changePlan,
            wait: Future<void>.delayed,
            isCurrent: () => _operationIsCurrent,
            isForeground: () => _operationIsForeground,
            onProviderConfirmed: (tier, before) =>
                _pendingOperations.recordPurchase(operationId, tier, before),
            readAccess: _readAssignedAccess,
          );
          return await _handlePurchaseResult(
            await controller.purchase(tier),
            operationId,
          );
        } on Object {
          _pendingOperations.clearUnconfirmed(operationId);
          rethrow;
        }
      }),
    );
  }

  Future<SubscriptionOperationNotice?> _handlePurchaseResult(
    SubscriptionPurchaseResult result,
    Object operationId,
  ) async {
    if (result.outcome != SubscriptionPurchaseOutcome.pending) {
      _pendingOperations.clear(operationId);
    }
    final retry = result.retry;
    _pendingRetry = retry == null
        ? null
        : () async => await _handlePurchaseResult(await retry(), operationId);
    return switch (result.outcome) {
      SubscriptionPurchaseOutcome.completed =>
        result.state?.subscriptions.any(
                  (value) =>
                      value.pendingTier != null &&
                      value.autoRenewalStatus == 'will_change_product',
                ) ==
                true
            ? SubscriptionOperationNotice.planChangeScheduled
            : null,
      SubscriptionPurchaseOutcome.cancelled ||
      SubscriptionPurchaseOutcome.notEligible => null,
      SubscriptionPurchaseOutcome.pending =>
        SubscriptionOperationNotice.pending,
      SubscriptionPurchaseOutcome.offeringUnavailable =>
        SubscriptionOperationNotice.offeringUnavailable,
      SubscriptionPurchaseOutcome.planChangeUnavailable =>
        SubscriptionOperationNotice.planChangeUnavailable,
      SubscriptionPurchaseOutcome.storeAccountConflict =>
        SubscriptionOperationNotice.storeAccountConflict,
      SubscriptionPurchaseOutcome.existingStoreSubscription =>
        SubscriptionOperationNotice.existingStoreSubscription,
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
    final ownerDid = registry?.activeLease?.session.account.did;
    if (ownerDid == null || !_pendingOperations.beginRestore(ownerDid)) {
      return;
    }
    final operationId = _operationContainer
        .read(subscriptionPendingOperationProvider)!
        .operationId;
    unawaited(
      _run(() async {
        try {
          final dependencies = await _ownerDependencies();
          final controller = SubscriptionRestoreController(
            ownerGuard: dependencies.guard,
            diagnostics: dependencies.diagnostics,
            api: dependencies.api,
            revenueCat: dependencies.service,
            onStoreConflict: _recordStoreConflict,
            wait: Future<void>.delayed,
            isCurrent: () => _operationIsCurrent,
            isForeground: () => _operationIsForeground,
            onProviderConfirmed: (before) =>
                _pendingOperations.recordRestore(operationId, before),
          );
          return await _handleRestoreResult(
            await controller.restore(),
            operationId,
          );
        } on Object {
          _pendingOperations.clearUnconfirmed(operationId);
          rethrow;
        }
      }),
    );
  }

  Future<SubscriptionOperationNotice?> _handleRestoreResult(
    SubscriptionRestoreResult result,
    Object operationId,
  ) async {
    if (result.outcome != SubscriptionRestoreOutcome.pending) {
      _pendingOperations.clear(operationId);
    }
    final retry = result.retry;
    _pendingRetry = retry == null
        ? null
        : () async => await _handleRestoreResult(await retry(), operationId);
    return switch (result.outcome) {
      SubscriptionRestoreOutcome.completed ||
      SubscriptionRestoreOutcome.cancelled => null,
      SubscriptionRestoreOutcome.pending => SubscriptionOperationNotice.pending,
      SubscriptionRestoreOutcome.storeAccountConflict =>
        SubscriptionOperationNotice.storeAccountConflict,
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
    return await switch (pending.kind) {
      PendingSubscriptionOperationKind.purchase => _handlePurchaseResult(
        await SubscriptionPurchaseController(
          ownerGuard: dependencies.guard,
          diagnostics: dependencies.diagnostics,
          api: dependencies.api,
          nativeStore: dependencies.nativeStore,
          allowNativeTierChanges: dependencies.allowNativeTierChanges,
          presentPaywall: (_, _) => throw DiagnosticStateError(
            'Pending reconciliation cannot present checkout',
          ),
          wait: Future<void>.delayed,
          readAccess: _readAssignedAccess,
          isCurrent: () => _operationIsCurrent,
          isForeground: () => _operationIsForeground,
        ).reconcilePending(pending.tier!, pending.beforeProvider!),
        pending.operationId,
      ),
      PendingSubscriptionOperationKind.restore => _handleRestoreResult(
        await SubscriptionRestoreController(
          ownerGuard: dependencies.guard,
          diagnostics: dependencies.diagnostics,
          api: dependencies.api,
          revenueCat: dependencies.service,
          wait: Future<void>.delayed,
          isCurrent: () => _operationIsCurrent,
          isForeground: () => _operationIsForeground,
        ).reconcilePending(pending.beforeProvider!),
        pending.operationId,
      ),
      PendingSubscriptionOperationKind.customerCenter =>
        _handleCustomerCenterResult(
          await CustomerCenterCoordinator(
            ownerGuard: dependencies.guard,
            diagnostics: dependencies.diagnostics,
            api: dependencies.api,
            revenueCat: dependencies.service,
            wait: Future<void>.delayed,
            isCurrent: () => _operationIsCurrent,
            isForeground: () => _operationIsForeground,
          ).reconcilePending(),
          pending.operationId,
        ),
    };
  }

  Future<SubscriptionAccess> _readAssignedAccess(Did did) async {
    final registry = _operationContainer
        .read(sessionRegistryProvider)
        .requireValue;
    final target = registry.leaseFor(AccountKey(did.value));
    if (target == null) {
      throw DiagnosticStateError('Assigned account unavailable');
    }
    final api = await _operationContainer.read(
      subscriptionRepositoryProvider(target.account).future,
    );
    final access = await api.getAccess();
    final current = _operationContainer
        .read(sessionRegistryProvider)
        .requireValue;
    if (!mounted || current.leaseFor(target.account) != target) {
      throw DiagnosticStateError('Assigned account session changed');
    }
    return access;
  }

  void _manage() {
    final registry = ref.read(sessionRegistryProvider).value;
    final ownerDid = registry?.activeLease?.session.account.did;
    if (ownerDid == null || !_pendingOperations.beginCustomerCenter(ownerDid)) {
      return;
    }
    final operationId = _operationContainer
        .read(subscriptionPendingOperationProvider)!
        .operationId;
    unawaited(
      _run(() async {
        try {
          final dependencies = await _ownerDependencies();
          final owner = dependencies.guard.capture();
          final state = await dependencies.guard.dispatchBillingState(
            owner,
            dependencies.api.getBillingAccount,
          );
          final current = state.subscriptions
              .where((sub) => sub.givesAccess)
              .toList();
          if (current.length == 1 && current.single.store == 'stripe') {
            final api = dependencies.api;
            if (api is! SubscriptionManagementApi) {
              throw DiagnosticStateError('Subscription management unavailable');
            }
            final uri = await dependencies.guard.dispatch(
              owner,
              (api as SubscriptionManagementApi).managementURL,
            );
            if (!mounted || !dependencies.guard.isCurrent(owner)) {
              _pendingOperations.clearUnconfirmed(operationId);
              return null;
            }
            final opened = await dependencies.guard.dispatch(
              owner,
              () => _operationContainer.read(
                subscriptionManagementLauncherProvider,
              )(uri),
            );
            _pendingOperations.clearUnconfirmed(operationId);
            if (!opened) {
              throw DiagnosticStateError('Subscription management unavailable');
            }
            return null;
          }
          final coordinator = CustomerCenterCoordinator(
            ownerGuard: dependencies.guard,
            diagnostics: dependencies.diagnostics,
            api: dependencies.api,
            revenueCat: dependencies.service,
            wait: Future<void>.delayed,
            isCurrent: () => _operationIsCurrent,
            isForeground: () => _operationIsForeground,
            onStoreConflict: _recordStoreConflict,
            onProviderMutationConfirmed: () =>
                _pendingOperations.recordCustomerCenterMutation(operationId),
          );
          return await _handleCustomerCenterResult(
            await coordinator.present(),
            operationId,
          );
        } on Object {
          _pendingOperations.clearUnconfirmed(operationId);
          rethrow;
        }
      }),
    );
  }

  Future<SubscriptionOperationNotice?> _handleCustomerCenterResult(
    CustomerCenterResult result,
    Object operationId,
  ) async {
    if (result.outcome != CustomerCenterOutcome.pending) {
      _pendingOperations.clear(operationId);
    }
    final retry = result.retry;
    _pendingRetry = retry == null
        ? null
        : () async =>
              await _handleCustomerCenterResult(await retry(), operationId);
    return switch (result.outcome) {
      CustomerCenterOutcome.completed ||
      CustomerCenterOutcome.refreshed ||
      CustomerCenterOutcome.cancelled => null,
      CustomerCenterOutcome.pending => SubscriptionOperationNotice.pending,
      CustomerCenterOutcome.storeAccountConflict =>
        SubscriptionOperationNotice.storeAccountConflict,
      CustomerCenterOutcome.providerError =>
        SubscriptionOperationNotice.providerFailure,
      CustomerCenterOutcome.anomaly =>
        SubscriptionOperationNotice.supportRequired,
      CustomerCenterOutcome.failed => SubscriptionOperationNotice.actionFailed,
    };
  }

  Future<
    ({
      BillingAccountGuard guard,
      SubscriptionDiagnostics diagnostics,
      RevenueCatService service,
      SubscriptionApi api,
      String? nativeStore,
      bool allowNativeTierChanges,
    })
  >
  _ownerDependencies() async {
    final registry = _operationContainer
        .read(sessionRegistryProvider)
        .requireValue;
    final activeLease = registry.activeLease!;
    final lease = activeLease.session;
    final diagnostics = SubscriptionDiagnostics(
      reporter: _operationContainer.read(errorReporterProvider),
      accountDid: lease.account.did.value,
    );
    final service = _operationContainer.read(revenueCatServiceProvider);
    final api = await _operationContainer.read(
      subscriptionRepositoryProvider(lease.account).future,
    );
    if (!mounted ||
        !_operationContainer
            .read(sessionRegistryProvider)
            .requireValue
            .isCurrent(activeLease)) {
      throw const BillingAccountGuardException('owner_session_changed');
    }
    return (
      guard: BillingAccountGuard(
        () {
          final current = _operationContainer
              .read(sessionRegistryProvider)
              .requireValue;
          if (!current.isCurrent(activeLease)) {
            throw const BillingAccountGuardException('owner_session_changed');
          }
          return current;
        },
      ),
      diagnostics: diagnostics,
      service: service,
      allowNativeTierChanges:
          service is NativeStoreOwnershipService &&
          (service as NativeStoreOwnershipService).supportsPlanChanges,
      nativeStore: service is NativeStoreOwnershipService
          ? (service as NativeStoreOwnershipService).store
          : null,
      api: api,
    );
  }
}

class SubscriptionPageView extends StatelessWidget {
  const new({
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
      body: YarnRefreshScrollView.list(
        onRefresh: () async => await model.onRefresh?.call(),
        notificationPredicate: (notification) =>
            model.onRefresh != null &&
            defaultScrollNotificationPredicate(notification),
        showSuccess: false,
        padding: const EdgeInsets.all(16),
        children: [
          Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 720),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _AccessCard(access: model.access, state: model.billingState),
                  const SizedBox(height: 16),
                  ..._roleContent(context),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  List<Widget> _roleContent(BuildContext context) => _ownerContent(context);

  List<Widget> _ownerContent(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final state = model.billingState;
    if (state == null) {
      return [
        _MessageCard(
          body: l10n.subscriptionsProviderFailure,
          action: l10n.subscriptionsRefresh,
          onAction: model.onRefresh,
        ),
      ];
    }
    return [
      if (state.webCheckoutPending) ...[
        _MessageCard(
          body: l10n.subscriptionsCheckoutPendingBody,
          action: l10n.subscriptionsCancelCheckout,
          onAction: model.onCancelCheckout,
        ),
        const SizedBox(height: 16),
      ],
      if (state.promotionalAccessEndsByTier.isNotEmpty) ...[
        _ComplimentaryAccessCard(
          state: state,
          showPurchaseTiming:
              model.billingAvailability == BillingAvailability.available &&
              !state.subscriptions.any((value) => value.givesAccess) &&
              const [SubscriptionTier.plus, SubscriptionTier.business].any(
                (tier) => purchaseEligibility(
                  state,
                  tier,
                  store: model.nativeStore,
                  allowTierChanges: model.allowNativeTierChanges,
                ).eligible,
              ),
        ),
        const SizedBox(height: 16),
      ],
      if (model.operationNotice case final notice?) ...[
        _OperationNotice(
          notice: notice,
          onRefresh: model.onRefresh,
          accountLabel: model.conflictingAccountLabel,
        ),
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
          ],
        ),
    ];
  }
}

class _AccessCard extends StatelessWidget {
  const new({required this.access, required this.state});

  final SubscriptionAccess access;
  final BillingState? state;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final locale = Localizations.localeOf(context).toString();
    String expiry(DateTime value) {
      final local = value.toLocal();
      return '${DateFormat.yMMMd(locale).format(local)}, '
          '${DateFormat.jm(locale).format(local)}';
    }

    final billing = state;
    final candidates = billing?.subscriptions
        .where(
          (subscription) =>
              subscription.givesAccess &&
              !subscription.pendingPayment &&
              !isBillingAnomaly(subscription.anomaly) &&
              billing.licenses.any(
                (license) =>
                    license.subscriptionId == subscription.id &&
                    license.tier == access.effectiveTier &&
                    !isBillingAnomaly(license.anomaly),
              ),
        )
        .toList();
    final renewing =
        billing != null &&
        !billing.reconciliationStale &&
        billing.requestedGeneration <= billing.reconciledGeneration &&
        candidates?.length == 1 &&
        const [
          'will_renew',
          'will_change_product',
        ].contains(candidates!.single.autoRenewalStatus);
    final date = renewing
        ? candidates.single.currentPeriodEndsAt ?? access.accessEndsAt
        : access.accessEndsAt;
    return CraftskyCard(
      key: const ValueKey('subscription-access-card'),
      clipBehavior: Clip.none,
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
          if (date case final endsAt?) ...[
            const SizedBox(height: 6),
            Text(
              renewing
                  ? l10n.subscriptionsRenewsOn(expiry(endsAt))
                  : l10n.subscriptionsAccessUntil(expiry(endsAt)),
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _ComplimentaryAccessCard extends StatelessWidget {
  const new({required this.state, required this.showPurchaseTiming});

  final BillingState state;
  final bool showPurchaseTiming;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final grants = state.promotionalAccessEndsByTier;
    return CraftskyCard(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l10n.subscriptionsComplimentaryAccess,
            style: Theme.of(context).textTheme.titleMedium,
          ),
          for (final tier in const [
            SubscriptionTier.plus,
            SubscriptionTier.business,
          ])
            if (grants.containsKey(tier)) ...[
              const SizedBox(height: 6),
              Text(
                grants[tier] != null
                    ? l10n.subscriptionsComplimentaryUntil(
                        _tierLabel(l10n, tier),
                        MaterialLocalizations.of(context)
                            .formatMediumDate(grants[tier]!.toLocal()),
                      )
                    : l10n.subscriptionsComplimentaryNoExpiry(
                        _tierLabel(l10n, tier),
                      ),
              ),
            ],
          if (showPurchaseTiming) ...[
            const SizedBox(height: 10),
            Text(l10n.subscriptionsComplimentaryPurchaseTiming),
          ],
        ],
      ),
    );
  }
}

class _TierCard extends StatelessWidget {
  const new({
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
    final eligibility = purchaseEligibility(
      state,
      tier,
      store: model.nativeStore,
      allowTierChanges: model.allowNativeTierChanges,
    );
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
    final action = _primaryAction(context, l10n, presentation);
    final purchaseIsLoading =
        pendingOperation?.kind == PendingSubscriptionOperationKind.purchase &&
        pendingOperation?.tier == tier;
    final actionIsLoading =
        purchaseIsLoading ||
        pendingOperation?.kind ==
            PendingSubscriptionOperationKind.customerCenter;
    final onActivate = actionIsLoading ? null : action?.$2;
    final tierLabel = _tierLabel(l10n, tier);
    final theme = Theme.of(context);
    final accent = tier == SubscriptionTier.plus
        ? theme.colorScheme.primary
        : theme.brightness == Brightness.dark
        ? theme.colorScheme.tertiary
        : BrandColors.butterDeep;
    final tierIcon = tier == SubscriptionTier.plus
        ? PlusTierIcon(color: accent)
        : Icon(Icons.storefront_outlined, color: accent);
    final price = model.tierPrices[tier];
    final isActive =
        presentation == SubscriptionTierPresentation.active ||
        presentation == SubscriptionTierPresentation.canceledAccessible;
    return Semantics(
      container: true,
      button: action != null,
      enabled: onActivate != null,
      onTap: onActivate,
      label: l10n.subscriptionsActionSemantic(
        tierLabel,
        status,
        action?.$1 ?? '',
      ),
      child: CraftskyCard(
        padding: EdgeInsets.zero,
        child: Ink(
          decoration: BoxDecoration(
            color: Color.alphaBlend(
              accent.withValues(alpha: 0.08),
              theme.colorScheme.surface,
            ),
          ),
          child: InkWell(
            key: ValueKey('subscription-tier-${tier.name}'),
            onTap: onActivate,
            excludeFromSemantics: true,
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
                        child: tierIcon,
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            SizedBox(
                              width: double.infinity,
                              child: Wrap(
                                alignment: WrapAlignment.spaceBetween,
                                crossAxisAlignment: WrapCrossAlignment.center,
                                spacing: 8,
                                runSpacing: 8,
                                children: [
                                  Text(
                                    tierLabel,
                                    style: theme.textTheme.titleLarge,
                                  ),
                                  if (isActive)
                                    Container(
                                      key: ValueKey(
                                        'subscription-status-${tier.name}',
                                      ),
                                      padding: const EdgeInsets.symmetric(
                                        horizontal: 12,
                                        vertical: 4,
                                      ),
                                      decoration: BoxDecoration(
                                        color: theme.colorScheme.primary,
                                        borderRadius: BorderRadius.circular(
                                          999,
                                        ),
                                      ),
                                      child: Text(
                                        l10n.subscriptionsStatusActive,
                                        style: theme.textTheme.labelLarge
                                            ?.copyWith(
                                              color:
                                                  theme.colorScheme.onPrimary,
                                              fontWeight: FontWeight.w700,
                                            ),
                                      ),
                                    ),
                                ],
                              ),
                            ),
                            const SizedBox(height: 2),
                            Text(
                              price == null
                                  ? l10n.subscriptionsPriceUnavailable
                                  : l10n.subscriptionsTierPrice(price),
                              style: theme.textTheme.titleSmall?.copyWith(
                                color: accent,
                                fontWeight: FontWeight.w700,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  if (subscription?.pendingTier case final pendingTier?
                      when subscription?.autoRenewalStatus ==
                          'will_change_product') ...[
                    const SizedBox(height: 8),
                    Text(
                      l10n.subscriptionsScheduledPlan(
                        _tierLabel(l10n, pendingTier),
                      ),
                    ),
                  ],
                  ...[
                    const SizedBox(height: 10),
                    if (!isActive && summary != null) Text(summary),
                    if (!isActive && summary != null) const SizedBox(height: 8),
                    Text(
                      tier == SubscriptionTier.plus
                          ? l10n.subscriptionsPlusBenefits
                          : l10n.subscriptionsBusinessBenefits,
                      style: theme.textTheme.bodyMedium,
                    ),
                  ],
                  if (action != null) ...[
                    const SizedBox(height: 8),
                    Align(
                      alignment: AlignmentDirectional.centerStart,
                      child: ExcludeSemantics(
                        child: ExcludeFocus(
                          child: _ActionButton(
                            key: ValueKey(
                              'subscription-view-details-${tier.name}',
                            ),
                            label: action.$1,
                            flat: true,
                            onPressed: onActivate,
                            isLoading: actionIsLoading,
                          ),
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  (String, VoidCallback?)? _primaryAction(
    BuildContext context,
    AppLocalizations l10n,
    SubscriptionTierPresentation presentation,
  ) {
    if (state.webCheckoutPending) {
      return (
        l10n.subscriptionsViewDetails,
        () => showDialog<void>(
          context: context,
          builder: (dialogContext) => AlertDialog(
            title: Text(l10n.subscriptionsCheckoutPendingTitle),
            content: Text(l10n.subscriptionsCheckoutPendingBody),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(dialogContext).pop(),
                child: Text(l10n.actionCancel),
              ),
              TextButton(
                onPressed: model.onCancelCheckout == null
                    ? null
                    : () {
                        Navigator.of(dialogContext).pop();
                        model.onCancelCheckout!();
                      },
                child: Text(l10n.subscriptionsCancelCheckout),
              ),
            ],
          ),
        ),
      );
    }
    if (model.operationNotice == SubscriptionOperationNotice.pending) {
      return null;
    }
    final current = state.subscriptions
        .where((subscription) => subscription.givesAccess)
        .toList();
    if (current.length == 1) {
      final currentLicense = state.licenses
          .where((license) => license.subscriptionId == current.single.id)
          .firstOrNull;
      if (currentLicense != null) {
        return (
          currentLicense.tier == tier
              ? l10n.subscriptionsManage
              : tier == SubscriptionTier.business
              ? l10n.subscriptionsUpgrade
              : l10n.subscriptionsDowngrade,
          model.onManage,
        );
      }
    }
    if (model.billingAvailability == BillingAvailability.unavailable) {
      return null;
    }
    final nativeSubscription = state.subscriptions.any(
      (subscription) =>
          subscription.givesAccess && subscription.store == model.nativeStore,
    );
    if (purchaseEligibility(
      state,
      tier,
      store: model.nativeStore,
      allowTierChanges: model.allowNativeTierChanges,
    ).eligible) {
      return (
        state.promotionalAccessEndsByTier.isNotEmpty &&
                !state.subscriptions.any((value) => value.givesAccess)
            ? l10n.subscriptionsSubscribeTier(_tierLabel(l10n, tier))
            : l10n.subscriptionsViewDetails,
        model.onPurchase == null ? null : () => model.onPurchase!(tier),
      );
    }
    if (nativeSubscription) {
      return (l10n.subscriptionsViewDetails, model.onManage);
    }
    return null;
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
}

class _MessageCard extends StatelessWidget {
  const new({
    required this.body,
    this.action,
    this.onAction,
  });

  final String body;
  final String? action;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) => CraftskyCard(
    clipBehavior: Clip.none,
    padding: const EdgeInsets.all(16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(body),
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
  const new({required this.notice, required this.onRefresh, this.accountLabel});

  final SubscriptionOperationNotice notice;
  final String? accountLabel;
  final FutureOr<void> Function()? onRefresh;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final message = switch (notice) {
      SubscriptionOperationNotice.offeringUnavailable =>
        l10n.subscriptionsOfferingUnavailable,
      SubscriptionOperationNotice.planChangeScheduled =>
        l10n.subscriptionsPlanChangeScheduled,
      SubscriptionOperationNotice.planChangeUnavailable =>
        l10n.subscriptionsPlanChangeUnavailable,
      SubscriptionOperationNotice.storeAccountConflict =>
        accountLabel == null
            ? l10n.subscriptionsStoreAccountConflict
            : l10n.subscriptionsStoreAccountConflictKnown(accountLabel!),
      SubscriptionOperationNotice.existingStoreSubscription =>
        l10n.subscriptionsExistingStoreSubscription,
      SubscriptionOperationNotice.providerFailure =>
        l10n.subscriptionsProviderFailure,
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
  const new({
    required this.label,
    required this.onPressed,
    this.isLoading = false,
    this.flat = false,
    super.key,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool isLoading;
  final bool flat;

  @override
  Widget build(BuildContext context) => flat
      ? _TextActionButton(
          label: label,
          onPressed: onPressed,
          isLoading: isLoading,
        )
      : Padding(
          padding: const EdgeInsets.only(bottom: 6),
          child: ChunkyButton(
            onPressed: isLoading ? null : onPressed,
            child: _ActionButtonChild(label: label, isLoading: isLoading),
          ),
        );
}

class _TextActionButton extends StatelessWidget {
  const new({
    required this.label,
    required this.onPressed,
    this.isLoading = false,
  });

  final String label;
  final VoidCallback? onPressed;
  final bool isLoading;

  @override
  Widget build(BuildContext context) => TextButton(
    onPressed: isLoading ? null : onPressed,
    style: TextButton.styleFrom(
      minimumSize: const Size(48, 48),
    ),
    child: _ActionButtonChild(label: label, isLoading: isLoading),
  );
}

class _ActionButtonChild extends StatelessWidget {
  const new({required this.label, required this.isLoading});

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
