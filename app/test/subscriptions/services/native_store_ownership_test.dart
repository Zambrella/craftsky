import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service_native.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const channel = MethodChannel('social.craftsky/store_ownership');
  const customer = '20000000-0000-4000-8000-000000000001';
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  tearDown(() => messenger.setMockMethodCallHandler(channel, null));

  test(
    'checks customer ownership through a read-only native channel',
    () async {
      messenger.setMockMethodCallHandler(channel, (call) async {
        expect(call.method, 'check');
        expect(call.arguments, {'appUserId': customer});
        return {'status': 'allowed'};
      });
      await const NativeRevenueCatService().verifyStoreOwnership(customer);
    },
  );

  test('verified conflict redacts its private owner reference', () async {
    messenger.setMockMethodCallHandler(
      channel,
      (_) async => {
        'status': 'conflict',
        'ownerAppUserId': customer,
      },
    );
    await expectLater(
      const NativeRevenueCatService().verifyStoreOwnership(customer),
      throwsA(
        isA<StoreAccountConflictException>()
            .having((error) => error.ownerAppUserId, 'verified owner', customer)
            .having(
              (error) => error.toString(),
              'diagnostic',
              isNot(contains(customer)),
            ),
      ),
    );
  });

  for (final status in ['unverified', 'unknown', null]) {
    test('unknown ownership $status prevents purchase', () async {
      messenger.setMockMethodCallHandler(
        channel,
        (_) async => {'status': status},
      );
      await expectLater(
        const NativeRevenueCatService().verifyStoreOwnership(customer),
        throwsA(isA<StoreOwnershipUnverifiedException>()),
      );
    });
  }

  test('missing native plugin fails closed', () async {
    await expectLater(
      const NativeRevenueCatService().verifyStoreOwnership(customer),
      throwsA(isA<StoreOwnershipUnverifiedException>()),
    );
  });

  for (final active in [false, true]) {
    test(
      'new purchase requires explicit store vacancy: active=$active',
      () async {
        messenger.setMockMethodCallHandler(
          channel,
          (_) async => {
            'status': 'allowed',
            'hasActiveSubscription': active,
          },
        );
        const service = NativeRevenueCatService();
        // Restore/replacement may inspect the same owner's existing purchase.
        await service.verifyStoreOwnership(customer);
        final newPurchase = service.verifyStoreOwnership(
          customer,
          requireNoActiveSubscription: true,
        );
        if (active) {
          await expectLater(
            newPurchase,
            throwsA(isA<StoreSubscriptionExistsException>()),
          );
        } else {
          await newPurchase;
        }
      },
    );
  }

  for (final active in [null, 'false', 0]) {
    test(
      'missing or malformed vacancy evidence $active blocks checkout',
      () async {
        messenger.setMockMethodCallHandler(
          channel,
          (_) async => {'status': 'allowed', 'hasActiveSubscription': active},
        );
        await expectLater(
          const NativeRevenueCatService().verifyStoreOwnership(
            customer,
            requireNoActiveSubscription: true,
          ),
          throwsA(isA<StoreOwnershipUnverifiedException>()),
        );
      },
    );
  }

  test(
    'foreign ownership takes precedence over an existing purchase',
    () async {
      messenger.setMockMethodCallHandler(
        channel,
        (_) async => {'status': 'conflict', 'hasActiveSubscription': true},
      );
      await expectLater(
        const NativeRevenueCatService().verifyStoreOwnership(
          customer,
          requireNoActiveSubscription: true,
        ),
        throwsA(isA<StoreAccountConflictException>()),
      );
    },
  );
}
