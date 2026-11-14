import 'package:craftsky_app/subscriptions/services/revenuecat_bootstrap.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter/foundation.dart';
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
    // Configuration starts native receipt synchronization. Wait until AppView
    // supplies the authenticated customer's UUID and ownership is verified.
    return NativeRevenueCatService._deferred(
      publicKey: publicKey,
      isTestStore: kDebugMode && publicKey.startsWith('test_'),
    );
  }
}

final class NativeRevenueCatService
    implements
        RevenueCatService,
        NativeStoreOwnershipService,
        NativePlanChangeService {
  const new({this.isTestStore = false}) : _activation = null;
  new _deferred({
    required String publicKey,
    required this.isTestStore,
  }) : _activation = _NativeRevenueCatActivation(publicKey);

  final bool isTestStore;
  final _NativeRevenueCatActivation? _activation;

  bool get _configured => _activation?.configured ?? true;

  void _requireConfigured() {
    if (!_configured) {
      throw const RevenueCatServiceException('identified_customer_required');
    }
  }

  @override
  String get store => isTestStore
      ? 'test_store'
      : defaultTargetPlatform == TargetPlatform.iOS
      ? 'app_store'
      : 'play_store';
  // Changes use an explicit Android replacement or a verified same-group
  // StoreKit purchase; they never fall back to buying an independent plan.
  @override
  bool get supportsPlanChanges => !isTestStore;

  static const _ownership = MethodChannel('social.craftsky/store_ownership');

  @override
  Future<void> verifyStoreOwnership(
    String appUserId, {
    bool requireNoActiveSubscription = false,
  }) async {
    if (isTestStore) return;
    Map<String, dynamic>? ownership;
    try {
      ownership = await _ownership
          .invokeMapMethod<String, dynamic>('check', {'appUserId': appUserId})
          .timeout(const Duration(seconds: 15));
    } on Object {
      throw const StoreOwnershipUnverifiedException();
    }
    if (ownership?['status'] == 'conflict') {
      throw StoreAccountConflictException(
        ownerAppUserId: ownership?['ownerAppUserId'] as String?,
        ownerHash: ownership?['ownerHash'] as String?,
      );
    }
    if (ownership?['status'] != 'allowed') {
      throw const StoreOwnershipUnverifiedException();
    }
    if (requireNoActiveSubscription) {
      final hasActiveSubscription = ownership?['hasActiveSubscription'];
      if (hasActiveSubscription is! bool) {
        throw const StoreOwnershipUnverifiedException();
      }
      if (hasActiveSubscription) {
        throw const StoreSubscriptionExistsException();
      }
    }
  }

  @override
  BillingAvailability get availability => BillingAvailability.available;

  @override
  Future<RevenueCatIdentity> currentIdentity() async {
    if (!_configured) return const RevenueCatIdentity.anonymous();
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
      final activation = _activation;
      if (activation != null && !activation.configured) {
        await Purchases.configure(
          PurchasesConfiguration(activation.publicKey)..appUserID = appViewUuid,
        );
        activation.configured = true;
      } else {
        await Purchases.logIn(appViewUuid);
      }
    } on Object {
      throw const RevenueCatServiceException('identity_write_failed');
    }
  }

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) async {
    _requireConfigured();
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
    _requireConfigured();
    if (offering is! _NativeRevenueCatOffering) {
      throw const RevenueCatServiceException('invalid_offering');
    }
    try {
      var storeConflict = false;
      void onError(PurchasesError error) {
        storeConflict =
            error.code == PurchasesErrorCode.receiptAlreadyInUseError ||
            error.code == PurchasesErrorCode.productAlreadyPurchasedError;
      }

      final result = await RevenueCatUI.presentPaywall(
        offering: offering.value,
        displayCloseButton: true,
        onPurchaseError: onError,
        onRestoreError: onError,
      );
      if (storeConflict &&
          result != PaywallResult.purchased &&
          result != PaywallResult.restored) {
        return DirectPaywallResult.storeAccountConflict;
      }
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
  Future<DirectPaywallResult> changePlan(
    RevenueCatOffering offering, {
    required bool Function() isCurrent,
  }) async {
    _requireConfigured();
    if (offering is! _NativeRevenueCatOffering || isTestStore) {
      throw const NativePlanChangeUnavailableException();
    }
    final package = offering.value.getPackage(r'$rc_monthly');
    if (package == null) throw const NativePlanChangeUnavailableException();
    await Purchases.invalidateCustomerInfoCache();
    final customer = await Purchases.getCustomerInfo();
    if (!isCurrent()) return DirectPaywallResult.cancelled;
    final expectedStore = defaultTargetPlatform == TargetPlatform.iOS
        ? Store.appStore
        : Store.playStore;
    final oldProduct = _activePaidProduct(customer, expectedStore);
    if (oldProduct == package.storeProduct.identifier) {
      throw const NativePlanChangeUnavailableException();
    }
    if (defaultTargetPlatform == TargetPlatform.iOS) {
      final grouped = await _ownership
          .invokeMethod<bool>('canChangePlan', {
            'oldProductId': oldProduct,
            'newProductId': package.storeProduct.identifier,
          })
          .timeout(const Duration(seconds: 15));
      if (grouped != true) throw const NativePlanChangeUnavailableException();
    }
    if (!isCurrent()) return DirectPaywallResult.cancelled;
    try {
      await Purchases.purchase(
        PurchaseParams.package(
          package,
          productChangeInfo: defaultTargetPlatform == TargetPlatform.android
              ? StoreProductChangeInfo(
                  oldProduct,
                  replacementMode: offering.identifier == 'business'
                      ? StoreReplacementMode.chargeProratedPrice
                      : StoreReplacementMode.deferred,
                )
              : null,
        ),
      );
      return DirectPaywallResult.purchased;
    } on PlatformException catch (error) {
      final code = PurchasesErrorHelper.getErrorCode(error);
      if (code == PurchasesErrorCode.purchaseCancelledError) {
        return DirectPaywallResult.cancelled;
      }
      if (code == PurchasesErrorCode.receiptAlreadyInUseError ||
          code == PurchasesErrorCode.productAlreadyPurchasedError) {
        throw const StoreAccountConflictException();
      }
      rethrow;
    }
  }

  @override
  Future<void> restorePurchases() async {
    _requireConfigured();
    try {
      await Purchases.restorePurchases();
    } on PlatformException catch (error) {
      final code = PurchasesErrorHelper.getErrorCode(error);
      if (code == PurchasesErrorCode.receiptAlreadyInUseError ||
          code == PurchasesErrorCode.productAlreadyPurchasedError) {
        throw const StoreAccountConflictException();
      }
      rethrow;
    } on Object {
      throw const RevenueCatServiceException('restore_failed');
    }
  }

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) async {
    _requireConfigured();
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

String _activePaidProduct(CustomerInfo customer, Store expectedStore) {
  // An entitlement may be supplied by a longer complimentary grant while the
  // real store subscription remains active. Prefer purchase-level evidence.
  final purchases = customer.subscriptionsByProductIdentifier;
  final activeSubscriptions = customer.activeSubscriptions.toSet();
  final active = purchases.isNotEmpty
      ? purchases.entries
            .where(
              (entry) =>
                  entry.value.isActive &&
                  (activeSubscriptions.contains(entry.key) ||
                      activeSubscriptions.contains(
                        entry.value.productIdentifier,
                      )) &&
                  entry.value.store != Store.promotional,
            )
            .map(
              (entry) => (
                product: entry.value.productIdentifier,
                store: entry.value.store,
              ),
            )
      : customer.entitlements.active.values
            .where((value) => value.store != Store.promotional)
            .map(
              (value) => (
                product: value.productIdentifier,
                store: value.store,
              ),
            );
  final products = <String>{};
  for (final value in active) {
    if (value.store != expectedStore || value.product.trim().isEmpty) {
      throw const NativePlanChangeUnavailableException();
    }
    products.add(value.product);
  }
  if (products.length != 1) {
    throw const NativePlanChangeUnavailableException();
  }
  return products.single;
}

final class _NativeRevenueCatActivation {
  new(this.publicKey);

  final String publicKey;
  bool configured = false;
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

final class RevenueCatServiceException implements Exception {
  const new(this.code);

  final String code;

  @override
  String toString() => 'RevenueCatServiceException($code)';
}
