import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_bootstrap.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter/services.dart';
import 'package:purchases_flutter/purchases_flutter.dart';
import 'package:purchases_ui_flutter/purchases_ui_flutter.dart';

final class NativeRevenueCatConfigurator implements RevenueCatConfigurator {
  const new();

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
  const new();

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
    } on Object catch (error, stackTrace) {
      Error.throwWithStackTrace(
        RevenueCatServiceException(
          'identity_read_failed',
          diagnosticCause: error,
          diagnosticStack: stackTrace,
        ),
        stackTrace,
      );
    }
  }

  @override
  Future<void> identify(String appViewUuid) async {
    try {
      await Purchases.logIn(appViewUuid);
    } on Object catch (error, stackTrace) {
      Error.throwWithStackTrace(
        RevenueCatServiceException(
          'identity_write_failed',
          diagnosticCause: error,
          diagnosticStack: stackTrace,
        ),
        stackTrace,
      );
    }
  }

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async {
    try {
      final offering = (await Purchases.getOfferings()).all[identifier];
      return offering == null ? null : _NativeRevenueCatOffering(offering);
    } on Object catch (error, stackTrace) {
      Error.throwWithStackTrace(
        RevenueCatServiceException(
          'offering_read_failed',
          diagnosticCause: error,
          diagnosticStack: stackTrace,
        ),
        stackTrace,
      );
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
    } on Object catch (error, stackTrace) {
      Error.throwWithStackTrace(
        RevenueCatServiceException(
          'paywall_failed',
          diagnosticCause: error,
          diagnosticStack: stackTrace,
        ),
        stackTrace,
      );
    }
  }

  @override
  Future<void> restorePurchases() async {
    try {
      await Purchases.restorePurchases();
    } on Object catch (error, stackTrace) {
      Error.throwWithStackTrace(
        RevenueCatServiceException(
          'restore_failed',
          diagnosticCause: error,
          diagnosticStack: stackTrace,
        ),
        stackTrace,
      );
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
    } on Object catch (error, stackTrace) {
      Error.throwWithStackTrace(
        RevenueCatServiceException(
          'customer_center_failed',
          diagnosticCause: error,
          diagnosticStack: stackTrace,
        ),
        stackTrace,
      );
    }
  }
}

final class _NativeRevenueCatOffering implements RevenueCatOffering {
  const new(this.value);

  final Offering value;

  @override
  String get identifier => value.identifier;

  @override
  bool hasPackage(String identifier) => value.getPackage(identifier) != null;

  @override
  String? packagePrice(String identifier) =>
      value.getPackage(identifier)?.storeProduct.priceString;
}

final class RevenueCatServiceException
    implements Exception, DiagnosticFailureCause {
  const new(this.code, {this.diagnosticCause, this.diagnosticStack});

  final String code;
  @override
  final Object? diagnosticCause;
  @override
  final StackTrace? diagnosticStack;

  /// Select only SDK-defined recoverable causes; SDK descriptions stay private.
  bool get expectedFailure {
    final cause = diagnosticCause;
    if (cause is! PlatformException) return false;
    try {
      return switch (PurchasesErrorHelper.getErrorCode(cause)) {
        PurchasesErrorCode.purchaseCancelledError ||
        PurchasesErrorCode.networkError => true,
        _ => false,
      };
    } on Object {
      // Unknown or malformed native codes remain actionable.
      return false;
    }
  }

  @override
  String toString() => 'RevenueCatServiceException($code)';
}
