import 'dart:io';

import 'package:craftsky_app/subscriptions/services/revenuecat_bootstrap.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-003 selects only the matching native public key', () async {
    final configurator = _RecordingConfigurator();

    final ios = await bootstrapRevenueCat(
      platform: RevenueCatPlatform.ios,
      iosPublicKey: 'appl_public',
      androidPublicKey: 'goog_public',
      testStorePublicKey: 'test_public',
      useTestStore: false,
      isDebug: true,
      configurator: configurator,
    );

    expect(ios.availability, BillingAvailability.available);
    expect(configurator.keys, ['appl_public']);
    expect(configurator.levels, [RevenueCatSdkLogLevel.debug]);

    await bootstrapRevenueCat(
      platform: RevenueCatPlatform.android,
      iosPublicKey: 'appl_public',
      androidPublicKey: 'goog_public',
      testStorePublicKey: 'test_public',
      useTestStore: false,
      isDebug: false,
      configurator: configurator,
    );
    expect(configurator.keys, ['appl_public', 'goog_public']);
    expect(configurator.levels.last, RevenueCatSdkLogLevel.info);
  });

  test(
    'IT-012 unsupported, blank, and failed setup stay unavailable',
    () async {
      final configurator = _RecordingConfigurator();

      for (final platform in [
        RevenueCatPlatform.web,
        RevenueCatPlatform.macos,
        RevenueCatPlatform.windows,
        RevenueCatPlatform.linux,
      ]) {
        final service = await bootstrapRevenueCat(
          platform: platform,
          iosPublicKey: 'appl_public',
          androidPublicKey: 'goog_public',
          testStorePublicKey: 'test_public',
          useTestStore: true,
          isDebug: false,
          configurator: configurator,
        );
        expect(service.availability, BillingAvailability.unavailable);
      }
      final blank = await bootstrapRevenueCat(
        platform: RevenueCatPlatform.ios,
        iosPublicKey: '  ',
        androidPublicKey: 'goog_public',
        testStorePublicKey: '',
        useTestStore: true,
        isDebug: false,
        configurator: configurator,
      );
      expect(blank.availability, BillingAvailability.unavailable);

      configurator.fail = true;
      final failed = await bootstrapRevenueCat(
        platform: RevenueCatPlatform.android,
        iosPublicKey: 'appl_public',
        androidPublicKey: 'goog_public',
        testStorePublicKey: 'test_public',
        useTestStore: false,
        isDebug: false,
        configurator: configurator,
      );
      expect(failed.availability, BillingAvailability.unavailable);
    },
  );

  test('UT-003 Test Store requires explicit debug opt-in', () async {
    final configurator = _RecordingConfigurator();

    await bootstrapRevenueCat(
      platform: RevenueCatPlatform.ios,
      iosPublicKey: 'appl_public',
      androidPublicKey: 'goog_public',
      testStorePublicKey: 'test_public',
      useTestStore: true,
      isDebug: true,
      configurator: configurator,
    );
    await bootstrapRevenueCat(
      platform: RevenueCatPlatform.android,
      iosPublicKey: 'appl_public',
      androidPublicKey: 'goog_public',
      testStorePublicKey: 'test_public',
      useTestStore: true,
      isDebug: false,
      configurator: configurator,
    );

    expect(configurator.keys, ['test_public', 'goog_public']);
  });

  test('IT-012 native dependencies, targets, and key inputs are explicit', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();
    final podfile = File('ios/Podfile').readAsStringSync();
    final config = File('config/staging.env.example').readAsStringSync();
    final nativeService = File(
      'lib/subscriptions/services/revenuecat_service_native.dart',
    ).readAsStringSync();

    expect(pubspec, contains('purchases_flutter: ^10.12.0'));
    expect(pubspec, contains('purchases_ui_flutter: ^10.12.0'));
    expect(podfile, contains("platform :ios, '15.0'"));
    expect(config, contains('REVENUECAT_IOS_PUBLIC_KEY='));
    expect(config, contains('REVENUECAT_ANDROID_PUBLIC_KEY='));
    expect(config, contains('REVENUECAT_TEST_STORE_PUBLIC_KEY='));
    expect(config, contains('REVENUECAT_USE_TEST_STORE=false'));
    expect(nativeService, isNot(contains('setLogHandler')));
    expect(nativeService, isNot(contains('Purchases.logOut')));
  });
}

final class _RecordingConfigurator implements RevenueCatConfigurator {
  final keys = <String>[];
  final levels = <RevenueCatSdkLogLevel>[];
  bool fail = false;

  @override
  Future<RevenueCatService> configure({
    required String publicKey,
    required RevenueCatSdkLogLevel logLevel,
  }) async {
    keys.add(publicKey);
    levels.add(logLevel);
    if (fail) throw StateError('provider details must be discarded');
    return const _AvailableRevenueCat();
  }
}

final class _AvailableRevenueCat implements RevenueCatService {
  const _AvailableRevenueCat();

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async =>
      const RevenueCatIdentity.anonymous();

  @override
  Future<void> identify(String appViewUuid) async {}

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async => null;

  @override
  Future<DirectPaywallResult> presentPaywall(
    RevenueCatOffering offering,
  ) async => DirectPaywallResult.cancelled;

  @override
  Future<void> restorePurchases() => throw UnimplementedError();

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) => throw UnimplementedError();
}
