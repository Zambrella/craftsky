import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:craftsky_app/subscriptions/services/subscription_diagnostics.dart';

enum RevenueCatPlatform { ios, android, macos, windows, linux }

enum RevenueCatSdkLogLevel { debug, info }

// Kept as an interface so native setup can be replaced in tests and bootstrap.
abstract interface class RevenueCatConfigurator {
  Future<RevenueCatService> configure({
    required String publicKey,
    required RevenueCatSdkLogLevel logLevel,
  });
}

Future<RevenueCatService> bootstrapRevenueCat({
  required RevenueCatPlatform platform,
  required String iosPublicKey,
  required String androidPublicKey,
  required String testStorePublicKey,
  required bool useTestStore,
  required bool isDebug,
  required RevenueCatConfigurator configurator,
  SubscriptionDiagnostics diagnostics = const SubscriptionDiagnostics(),
}) async {
  final publicKey = switch (platform) {
    RevenueCatPlatform.ios || RevenueCatPlatform.android
        when isDebug && useTestStore =>
      testStorePublicKey.trim(),
    RevenueCatPlatform.ios => iosPublicKey.trim(),
    RevenueCatPlatform.android => androidPublicKey.trim(),
    _ => '',
  };
  if (publicKey.isEmpty) return const UnavailableRevenueCatService();

  try {
    return await configurator.configure(
      publicKey: publicKey,
      logLevel: isDebug
          ? RevenueCatSdkLogLevel.debug
          : RevenueCatSdkLogLevel.info,
    );
  } on Object catch (error, stack) {
    diagnostics.report(error, stack, operation: 'billing.configure');
    return const UnavailableRevenueCatService();
  }
}
