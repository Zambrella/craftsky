import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service_native.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:purchases_flutter/purchases_flutter.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const purchases = MethodChannel('purchases_flutter');
  const ownership = MethodChannel('social.craftsky/store_ownership');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;
  final calls = <MethodCall>[];
  var current = true;
  var grouped = true;
  var changeDuringLookup = false;
  var target = 'business';
  setUp(() {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    calls.clear();
    current = true;
    grouped = true;
    changeDuringLookup = false;
    target = 'business';
    messenger
      ..setMockMethodCallHandler(purchases, (call) async {
        calls.add(call);
        if (call.method == 'getOfferings') {
          final offering = _offering(target);
          return {
            'all': {target: offering},
            'current': offering,
          };
        }
        if (call.method == 'getCustomerInfo') {
          return _customer(
            debugDefaultTargetPlatformOverride == TargetPlatform.iOS
                ? 'APP_STORE'
                : 'PLAY_STORE',
          );
        }
        if (call.method == 'purchasePackage') {
          throw PlatformException(
            code: PurchasesErrorCode.purchaseCancelledError.index.toString(),
          );
        }
        return null;
      })
      ..setMockMethodCallHandler(ownership, (call) async {
        expect(call.method, 'canChangePlan');
        if (changeDuringLookup) current = false;
        return grouped;
      });
  });
  tearDown(() {
    debugDefaultTargetPlatformOverride = null;
    messenger
      ..setMockMethodCallHandler(purchases, null)
      ..setMockMethodCallHandler(ownership, null);
  });

  test(
    'Android upgrade explicitly replaces existing SKU with prorated charge',
    () async {
      const service = NativeRevenueCatService();
      final offering = await service.getOffering(target);
      expect(
        await service.changePlan(offering!, isCurrent: () => current),
        DirectPaywallResult.cancelled,
      );
      final purchase = calls.singleWhere(
        (call) => call.method == 'purchasePackage',
      );
      expect(
        (purchase.arguments
            as Map<Object?, Object?>)['googleOldProductIdentifier'],
        'old.plan',
      );
      expect(
        (purchase.arguments as Map<Object?, Object?>)['storeReplacementMode'],
        'CHARGE_PRORATED_PRICE',
      );
    },
  );
  test(
    'Android downgrade explicitly defers replacement until renewal',
    () async {
      target = 'plus';
      const service = NativeRevenueCatService();
      final offering = await service.getOffering(target);
      await service.changePlan(offering!, isCurrent: () => current);
      final purchase = calls.singleWhere(
        (call) => call.method == 'purchasePackage',
      );
      expect(
        (purchase.arguments as Map<Object?, Object?>)['storeReplacementMode'],
        'DEFERRED',
      );
    },
  );
  test('iOS separate subscription groups prevent any purchase', () async {
    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    grouped = false;
    const service = NativeRevenueCatService();
    final offering = await service.getOffering(target);
    await expectLater(
      service.changePlan(offering!, isCurrent: () => current),
      throwsA(isA<NativePlanChangeUnavailableException>()),
    );
    expect(calls.where((call) => call.method == 'purchasePackage'), isEmpty);
  });
  test(
    'account switch while iOS groups load cancels before purchase',
    () async {
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
      changeDuringLookup = true;
      const service = NativeRevenueCatService();
      final offering = await service.getOffering(target);
      expect(
        await service.changePlan(offering!, isCurrent: () => current),
        DirectPaywallResult.cancelled,
      );
      expect(calls.where((call) => call.method == 'purchasePackage'), isEmpty);
    },
  );
}

Map<String, dynamic> _offering(String tier) => {
  'identifier': tier,
  'serverDescription': tier,
  'metadata': <String, dynamic>{},
  'availablePackages': [
    {
      'identifier': r'$rc_monthly',
      'packageType': 'MONTHLY',
      'presentedOfferingContext': {'offeringIdentifier': tier},
      'product': {
        'identifier': 'new.$tier',
        'description': tier,
        'title': tier,
        'price': 10,
        'priceString': '£10',
        'currencyCode': 'GBP',
        'productCategory': 'SUBSCRIPTION',
      },
    },
  ],
};

Map<String, dynamic> _customer(String store) {
  final entitlement = {
    'identifier': 'plus',
    'isActive': true,
    'willRenew': true,
    'latestPurchaseDate': '2026-11-01T00:00:00Z',
    'originalPurchaseDate': '2026-11-01T00:00:00Z',
    'productIdentifier': 'old.plan',
    'isSandbox': true,
    'store': store,
  };
  return {
    'entitlements': {
      'all': {'plus': entitlement},
      'active': {'plus': entitlement},
    },
    'allPurchaseDates': <String, String>{},
    'activeSubscriptions': ['old.plan'],
    'allPurchasedProductIdentifiers': ['old.plan'],
    'nonSubscriptionTransactions': <dynamic>[],
    'firstSeen': '2026-11-01T00:00:00Z',
    'originalAppUserId': '20000000-0000-4000-8000-000000000001',
    'allExpirationDates': <String, String>{},
    'requestDate': '2026-11-13T00:00:00Z',
  };
}
