import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/active_account_initialization.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart'
    show sessionRegistryProvider;
import 'package:craftsky_app/subscriptions/providers/revenuecat_service_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_repository_provider.dart';
import 'package:craftsky_app/subscriptions/services/billing_account_setup_coordinator.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final subscriptionIdentitySyncEnabledProvider = Provider<bool>(
  (ref) => subscriptionsEnabled,
  name: 'subscriptionIdentitySyncEnabledProvider',
);

/// Starts optional billing identity work after the active account is ready.
/// Its asynchronous state is deliberately outside account-critical readiness.
final Provider<void> activeAccountSubscriptionIdentitySyncProvider =
    Provider.autoDispose<void>(
      (ref) {
        if (!ref.watch(subscriptionIdentitySyncEnabledProvider)) return;
        ActiveAccountLease? target;
        ProviderSubscription<AsyncValue<void>>? operation;
        ref.onDispose(() => operation?.close());
        ref.listen(activeAccountInitializationProvider, (_, next) {
          if (next is! AsyncData<ActiveAccountInitialization?>) return;
          final initialized = next.value;
          final lease =
              initialized != null &&
                  initialized.onboardingComplete &&
                  !initialized.accountEligibility.restricted
              ? initialized.lease
              : null;
          if (target == lease) return;
          target = lease;
          operation?.close();
          operation = lease == null
              ? null
              : ref.listen(subscriptionIdentitySyncProvider(lease), (_, _) {});
        }, fireImmediately: true);
      },
      name: 'activeAccountSubscriptionIdentitySyncProvider',
    );

/// One attempt per active session lease. Billing actions retain their own
/// identity checks and provide a recovery path if this optional attempt fails.
final FutureProvider<void> Function(ActiveAccountLease)
subscriptionIdentitySyncProvider = FutureProvider.autoDispose
    .family<void, ActiveAccountLease>(
      (ref, lease) async {
        bool isCurrent() {
          if (!ref.mounted ||
              ref.read(sessionRegistryProvider).value?.isCurrent(lease) !=
                  true) {
            return false;
          }
          final initialization = ref.read(activeAccountInitializationProvider);
          return switch (initialization) {
            AsyncData(:final value) =>
              value?.lease == lease &&
                  value!.onboardingComplete &&
                  !value.accountEligibility.restricted,
            AsyncError() => false,
            // A refresh can retain an already-authorized active account while
            // its dependencies reload. A new account still fails the lease.
            _ => true,
          };
        }

        SessionRegistry readRegistry() {
          if (!isCurrent()) throw AccountOperationCanceled();
          return ref.read(sessionRegistryProvider).requireValue;
        }

        if (!isCurrent()) return;
        try {
          final api = await ref.watch(
            subscriptionRepositoryProvider(lease.session.account).future,
          );
          if (!isCurrent()) return;
          await BillingAccountSetupCoordinator(
            readRegistry: readRegistry,
            setCustomer: ref
                .read(sessionRegistryProvider.notifier)
                .setBillingCustomer,
            api: api,
            revenueCat: ref.read(revenueCatServiceProvider),
          ).setup(lease);
          if (isCurrent()) {
            ref.invalidate(subscriptionAccessProvider(lease.session));
          }
        } on Object {
          if (!isCurrent()) return;
          rethrow;
        }
      },
      retry: (_, _) => null,
      name: 'subscriptionIdentitySyncProvider',
    )
    .call;
