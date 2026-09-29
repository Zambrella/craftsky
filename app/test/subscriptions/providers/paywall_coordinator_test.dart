import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/services/billing_owner_guard.dart';
import 'package:craftsky_app/subscriptions/services/paywall_coordinator.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('IT-005 presents each named monthly offering directly', () async {
    for (final tier in [SubscriptionTier.plus, SubscriptionTier.business]) {
      final service = _PaywallService(
        offerings: {
          tier.name: _Offering(tier.name, {r'$rc_monthly'}),
        },
        result: DirectPaywallResult.purchased,
      );

      final outcome = await _present(service, tier);

      expect(outcome, PaywallOutcome.reconcile);
      expect(service.requestedOfferings, [tier.name]);
      expect(service.presentedOfferings, [tier.name]);
    }
  });

  test('UT-006 maps direct paywall results deterministically', () async {
    final expected = {
      DirectPaywallResult.purchased: PaywallOutcome.reconcile,
      DirectPaywallResult.restored: PaywallOutcome.reconcile,
      DirectPaywallResult.cancelled: PaywallOutcome.cancelled,
      DirectPaywallResult.error: PaywallOutcome.providerError,
    };
    for (final entry in expected.entries) {
      final service = _PaywallService(
        offerings: {
          'plus': const _Offering('plus', {r'$rc_monthly'}),
        },
        result: entry.key,
      );
      expect(
        await _present(service, SubscriptionTier.plus),
        entry.value,
      );
    }
  });

  test(
    'AT-006 missing offering or package stops before presentation',
    () async {
      for (final offerings in <Map<String, RevenueCatOffering>>[
        {},
        {'plus': const _Offering('plus', {})},
      ]) {
        final service = _PaywallService(
          offerings: offerings,
          result: DirectPaywallResult.purchased,
        );

        expect(
          await _present(service, SubscriptionTier.plus),
          PaywallOutcome.offeringUnavailable,
        );
        expect(service.presentedOfferings, isEmpty);
      }
    },
  );

  test(
    'AT-006 presentation attachment failure becomes provider error',
    () async {
      final service = _PaywallService(
        offerings: {
          'plus': const _Offering('plus', {r'$rc_monthly'}),
        },
        result: DirectPaywallResult.purchased,
        failPresentation: true,
      );

      expect(
        await _present(service, SubscriptionTier.plus),
        PaywallOutcome.providerError,
      );
      expect(service.presentedOfferings, ['plus']);
    },
  );

  test('IT-014 stale owner during offering read cancels paywall', () async {
    var registry = _ownerRegistry();
    final service = _PaywallService(
      offerings: {
        'plus': const _Offering('plus', {r'$rc_monthly'}),
      },
      result: DirectPaywallResult.purchased,
      onOffering: () {
        registry = registry.upsertAndActivate(
          token: 'bob-token',
          did: 'did:plc:bob',
          handle: 'bob.test',
        );
      },
    );
    final guard = BillingOwnerGuard(() => registry);

    final result = await PaywallCoordinator(
      service,
      guard,
      RevenueCatIdentityGuard(service),
    ).present(guard.capture(), SubscriptionTier.plus);

    expect(result, PaywallOutcome.cancelled);
    expect(service.presentedOfferings, isEmpty);
  });

  test(
    'IT-014 confirmed purchase survives owner switch during presentation',
    () async {
      var registry = _ownerRegistry();
      final service = _PaywallService(
        offerings: {
          'plus': const _Offering('plus', {r'$rc_monthly'}),
        },
        result: DirectPaywallResult.purchased,
        onPresentation: () {
          registry = registry.upsertAndActivate(
            token: 'bob-token',
            did: 'did:plc:bob',
            handle: 'bob.test',
          );
        },
      );
      final guard = BillingOwnerGuard(() => registry);

      final result = await PaywallCoordinator(
        service,
        guard,
        RevenueCatIdentityGuard(service),
      ).present(guard.capture(), SubscriptionTier.plus);

      expect(result, PaywallOutcome.reconcile);
      expect(service.presentedOfferings, ['plus']);
    },
  );
}

Future<PaywallOutcome> _present(
  _PaywallService service,
  SubscriptionTier tier,
) {
  final registry = _ownerRegistry();
  final ownerGuard = BillingOwnerGuard(() => registry);
  return PaywallCoordinator(
    service,
    ownerGuard,
    RevenueCatIdentityGuard(service),
  ).present(ownerGuard.capture(), tier);
}

SessionRegistry _ownerRegistry() => SessionRegistry.empty()
    .upsertAndActivate(
      token: 'alice-token',
      did: 'did:plc:alice',
      handle: 'alice.test',
    )
    .reserveBillingOwner('did:plc:alice')
    .completeBillingOwner(
      'did:plc:alice',
      '20000000-0000-4000-8000-000000000001',
    );

final class _Offering implements RevenueCatOffering {
  const _Offering(this.identifier, this.packages);

  @override
  final String identifier;
  final Set<String> packages;

  @override
  bool hasPackage(String identifier) => packages.contains(identifier);

  @override
  String? packagePrice(String identifier) => null;
}

final class _PaywallService implements RevenueCatService {
  _PaywallService({
    required this.offerings,
    required this.result,
    this.failPresentation = false,
    this.onOffering,
    this.onPresentation,
  });

  final Map<String, RevenueCatOffering> offerings;
  final DirectPaywallResult result;
  final bool failPresentation;
  final void Function()? onOffering;
  final void Function()? onPresentation;
  final requestedOfferings = <String>[];
  final presentedOfferings = <String>[];

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async {
    requestedOfferings.add(identifier);
    onOffering?.call();
    return offerings[identifier];
  }

  @override
  Future<DirectPaywallResult> presentPaywall(
    RevenueCatOffering offering,
  ) async {
    presentedOfferings.add(offering.identifier);
    if (failPresentation) throw StateError('missing attachment');
    onPresentation?.call();
    return result;
  }

  @override
  Future<RevenueCatIdentity> currentIdentity() async =>
      const RevenueCatIdentity.identified(
        '20000000-0000-4000-8000-000000000001',
      );

  @override
  Future<void> identify(String appViewUuid) async {}

  @override
  Future<void> restorePurchases() => throw UnimplementedError();

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) => throw UnimplementedError();
}
