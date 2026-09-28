import 'package:craftsky_app/subscriptions/services/revenuecat_bootstrap.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:purchases_flutter/purchases_flutter.dart';
import 'package:purchases_ui_flutter/purchases_ui_flutter.dart';

final class NativeRevenueCatConfigurator implements RevenueCatConfigurator {
  const NativeRevenueCatConfigurator();

  @override
  Future<RevenueCatService> configure({
    required String publicKey,
    required RevenueCatSdkLogLevel logLevel,
  }) async {
    await Purchases.setLogLevel(
      logLevel == RevenueCatSdkLogLevel.debug ? LogLevel.debug : LogLevel.info,
    );
    await Purchases.configure(PurchasesConfiguration(publicKey));
    return const NativeRevenueCatService();
  }
}

final class NativeRevenueCatService implements RevenueCatService {
  const NativeRevenueCatService();

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async {
    try {
      final anonymous = await Purchases.isAnonymous;
      final appUserId = await Purchases.appUserID;
      return anonymous
          ? const RevenueCatIdentity.anonymous()
          : RevenueCatIdentity.identified(appUserId);
    } on Object {
      throw const RevenueCatServiceException('identity_read_failed');
    }
  }

  @override
  Future<void> identify(String appViewUuid) async {
    try {
      await Purchases.logIn(appViewUuid);
    } on Object {
      throw const RevenueCatServiceException('identity_write_failed');
    }
  }

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async {
    try {
      final offering = (await Purchases.getOfferings()).all[identifier];
      return offering == null ? null : _NativeRevenueCatOffering(offering);
    } on Object {
      throw const RevenueCatServiceException('offering_read_failed');
    }
  }

  @override
  Future<DirectPaywallResult> presentPaywall(
    RevenueCatOffering offering,
  ) async {
    if (offering is! _NativeRevenueCatOffering) {
      throw const RevenueCatServiceException('invalid_offering');
    }
    try {
      final result = await RevenueCatUI.presentPaywall(
        offering: offering.value,
        displayCloseButton: true,
      );
      return switch (result) {
        PaywallResult.purchased => DirectPaywallResult.purchased,
        PaywallResult.restored => DirectPaywallResult.restored,
        PaywallResult.cancelled => DirectPaywallResult.cancelled,
        PaywallResult.error ||
        PaywallResult.notPresented => DirectPaywallResult.error,
      };
    } on Object {
      throw const RevenueCatServiceException('paywall_failed');
    }
  }

  @override
  Future<void> restorePurchases() async {
    try {
      await Purchases.restorePurchases();
    } on Object {
      throw const RevenueCatServiceException('restore_failed');
    }
  }

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) async {
    try {
      await RevenueCatUI.presentCustomerCenter(
        onRestoreCompleted: (_) =>
            onEvent(CustomerCenterEvent.restoreCompleted),
        onShowingManageSubscriptions: () =>
            onEvent(CustomerCenterEvent.managementStarted),
        onManagementOptionSelected: (_, _) =>
            onEvent(CustomerCenterEvent.managementOptionSelected),
        onPromotionalOfferSucceeded: (_, _, _) =>
            onEvent(CustomerCenterEvent.promotionalOfferSucceeded),
      );
    } on Object {
      throw const RevenueCatServiceException('customer_center_failed');
    }
  }
}

final class _NativeRevenueCatOffering implements RevenueCatOffering {
  const _NativeRevenueCatOffering(this.value);

  final Offering value;

  @override
  String get identifier => value.identifier;

  @override
  bool hasPackage(String identifier) => value.getPackage(identifier) != null;

  @override
  String? packagePrice(String identifier) =>
      value.getPackage(identifier)?.storeProduct.priceString;
}

final class RevenueCatServiceException implements Exception {
  const RevenueCatServiceException(this.code);

  final String code;

  @override
  String toString() => 'RevenueCatServiceException($code)';
}
