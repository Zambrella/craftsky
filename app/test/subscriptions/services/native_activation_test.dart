import 'dart:async';

import 'package:craftsky_app/subscriptions/services/revenuecat_bootstrap.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_identity_guard.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service_native.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const purchases = MethodChannel('purchases_flutter');
  const ownership = MethodChannel('social.craftsky/store_ownership');
  const alice = '20000000-0000-4000-8000-000000000001';
  const bob = '20000000-0000-4000-8000-000000000002';
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  final events = <String>[];
  final configuredIds = <String?>[];
  String? sdkIdentity;
  var ownershipStatus = 'allowed';
  Future<void> Function()? ownershipWait;

  setUp(() {
    events.clear();
    configuredIds.clear();
    sdkIdentity = null;
    ownershipStatus = 'allowed';
    ownershipWait = null;
    messenger
      ..setMockMethodCallHandler(purchases, (call) async {
        events.add(call.method);
        switch (call.method) {
          case 'setupPurchases':
            sdkIdentity =
                (call.arguments as Map<Object?, Object?>)['appUserId']
                    as String?;
            configuredIds.add(sdkIdentity);
          case 'isAnonymous':
            return sdkIdentity == null;
          case 'getAppUserID':
            return sdkIdentity ?? r'$RCAnonymousID:unexpected';
          case 'getOfferings':
            return {'all': <String, dynamic>{}, 'current': null};
          case 'logIn':
            fail('The conflict or stale lease must prevent SDK login');
        }
        return null;
      })
      ..setMockMethodCallHandler(ownership, (call) async {
        events.add('ownership:${(call.arguments as Map)['appUserId']}');
        await ownershipWait?.call();
        return {'status': ownershipStatus};
      });
  });
  tearDown(() {
    messenger
      ..setMockMethodCallHandler(purchases, null)
      ..setMockMethodCallHandler(ownership, null);
  });

  Future<RevenueCatService> service() =>
      const NativeRevenueCatConfigurator().configure(
        publicKey: 'appl_public',
        logLevel: RevenueCatSdkLogLevel.info,
      );

  test('bootstrap does not start an anonymous SDK or read receipts', () async {
    final revenueCat = await service();
    expect(configuredIds, isEmpty);
    expect(
      (await revenueCat.currentIdentity()).kind,
      RevenueCatIdentityKind.anonymous,
    );
    expect(events, isNot(contains('getAppUserID')));
    expect(events.where((value) => value.startsWith('ownership:')), isEmpty);
  });

  test(
    'first activation verifies ownership and configures the exact UUID',
    () async {
      final revenueCat = await service();
      await RevenueCatIdentityGuard(revenueCat).dispatch(
        alice,
        () => revenueCat.getOffering('plus'),
      );
      expect(configuredIds, [alice]);
      expect(
        events.indexOf('ownership:$alice'),
        lessThan(events.indexOf('setupPurchases')),
      );
      expect(
        events.indexOf('setupPurchases'),
        lessThan(events.indexOf('getOfferings')),
      );
    },
  );

  test('conflicting first identity never configures the SDK', () async {
    ownershipStatus = 'conflict';
    final revenueCat = await service();
    await expectLater(
      RevenueCatIdentityGuard(revenueCat).dispatch(alice, () async {}),
      throwsA(isA<StoreAccountConflictException>()),
    );
    expect(configuredIds, isEmpty);
  });

  test(
    'switching to a conflicting owner never logs into the next UUID',
    () async {
      final revenueCat = await service();
      await RevenueCatIdentityGuard(revenueCat).dispatch(alice, () async {});
      ownershipStatus = 'conflict';
      await expectLater(
        RevenueCatIdentityGuard(revenueCat).dispatch(bob, () async {}),
        throwsA(isA<StoreAccountConflictException>()),
      );
      expect(sdkIdentity, alice);
      expect(events, isNot(contains('logIn')));
    },
  );

  test(
    'account switch while ownership loads cancels before configuration',
    () async {
      final revenueCat = await service();
      final entered = Completer<void>();
      final release = Completer<void>();
      var current = true;
      ownershipWait = () async {
        entered.complete();
        await release.future;
      };
      final operation = RevenueCatIdentityGuard(revenueCat).dispatch(
        alice,
        () async {},
        isCurrent: () => current,
      );
      await entered.future;
      current = false;
      release.complete();
      await expectLater(operation, throwsA(isA<RevenueCatIdentityException>()));
      expect(configuredIds, isEmpty);
    },
  );
}
