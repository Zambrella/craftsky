import 'dart:async';

import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  const alice = '20000000-0000-4000-8000-000000000001';
  const bob = '20000000-0000-4000-8000-000000000002';
  test(
    'switches identified customer without anonymous logout or aliasing',
    () async {
      final service = _IdentityService(alice);
      await RevenueCatIdentityGuard(service).dispatch(bob, () async {
        expect(service.identity.appUserId, bob);
      });
      expect(service.identifications, [bob]);
    },
  );
  test(
    'a native operation retains its SDK customer until it finishes',
    () async {
      final service = _IdentityService(alice);
      final entered = Completer<void>();
      final release = Completer<void>();
      final first = RevenueCatIdentityGuard(service).dispatch(alice, () async {
        entered.complete();
        await release.future;
        expect(service.identity.appUserId, alice);
      });
      await entered.future;
      final second = RevenueCatIdentityGuard(service).dispatch(bob, () async {
        expect(service.identity.appUserId, bob);
      });
      await Future<void>.delayed(Duration.zero);
      expect(service.identifications, isEmpty);
      release.complete();
      await Future.wait([first, second]);
      expect(service.identifications, [bob]);
    },
  );
  test(
    'stale queued work cannot reidentify SDK for an inactive account',
    () async {
      final service = _IdentityService(alice);
      final entered = Completer<void>();
      final release = Completer<void>();
      final first = RevenueCatIdentityGuard(service).dispatch(alice, () async {
        entered.complete();
        await release.future;
      });
      await entered.future;
      var current = true;
      var calls = 0;
      final second = RevenueCatIdentityGuard(service)
          .dispatch(bob, () async => calls++, isCurrent: () => current);
      current = false;
      release.complete();
      await first;
      await expectLater(second, throwsA(isA<RevenueCatIdentityException>()));
      expect(calls, 0);
      expect(service.identifications, isEmpty);
    },
  );
}

final class _IdentityService implements RevenueCatIdentityService {
  new(String id) : identity = RevenueCatIdentity.identified(id);
  RevenueCatIdentity identity;
  final identifications = <String>[];
  @override
  BillingAvailability get availability => BillingAvailability.available;
  @override
  Future<RevenueCatIdentity> currentIdentity() async => identity;
  @override
  Future<void> identify(String id) async {
    identifications.add(id);
    identity = RevenueCatIdentity.identified(id);
  }
}
