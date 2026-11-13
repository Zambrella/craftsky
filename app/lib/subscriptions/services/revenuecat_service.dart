enum RevenueCatIdentityKind { anonymous, identified }

enum BillingAvailability { available, unavailable }

enum DirectPaywallResult {
  purchased,
  restored,
  cancelled,
  error,
  storeAccountConflict,
}

enum CustomerCenterEvent {
  restoreCompleted,
  managementStarted,
  managementOptionSelected,
  promotionalOfferSucceeded,
}

abstract interface class RevenueCatOffering {
  String get identifier;
  bool hasPackage(String identifier);
  String? packagePrice(String identifier);
}

class RevenueCatIdentity {
  const new _(this.kind, this.appUserId);

  const new anonymous() : this._(RevenueCatIdentityKind.anonymous, null);

  const new identified(String appUserId)
    : this._(RevenueCatIdentityKind.identified, appUserId);

  final RevenueCatIdentityKind kind;
  final String? appUserId;
}

abstract interface class RevenueCatIdentityService {
  BillingAvailability get availability;
  Future<RevenueCatIdentity> currentIdentity();
  Future<void> identify(String appViewUuid);
}

abstract interface class NativeStoreOwnershipService {
  String get store;
  bool get supportsPlanChanges;

  /// Reads store-owned transactions without posting or transferring receipts.
  /// Unknown ownership must fail closed before opening a purchase sheet.
  Future<void> verifyStoreOwnership(String appUserId);
}

abstract interface class NativePlanChangeService {
  Future<DirectPaywallResult> changePlan(
    RevenueCatOffering offering, {
    required bool Function() isCurrent,
  });
}

final class NativePlanChangeUnavailableException implements Exception {
  const new();
  @override
  String toString() => 'NativePlanChangeUnavailableException()';
}

abstract interface class RevenueCatService
    implements RevenueCatIdentityService {
  Future<RevenueCatOffering?> getOffering(String identifier);
  Future<DirectPaywallResult> presentPaywall(RevenueCatOffering offering);
  Future<void> restorePurchases();
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  );
}

final class UnavailableRevenueCatService implements RevenueCatService {
  const new();

  @override
  BillingAvailability get availability => BillingAvailability.unavailable;

  @override
  Future<RevenueCatIdentity> currentIdentity() =>
      throw const RevenueCatUnavailableException();

  @override
  Future<void> identify(String appViewUuid) =>
      throw const RevenueCatUnavailableException();

  @override
  Future<RevenueCatOffering?> getOffering(String identifier) =>
      throw const RevenueCatUnavailableException();

  @override
  Future<DirectPaywallResult> presentPaywall(RevenueCatOffering offering) =>
      throw const RevenueCatUnavailableException();

  @override
  Future<void> restorePurchases() =>
      throw const RevenueCatUnavailableException();

  @override
  Future<void> presentCustomerCenter(
    void Function(CustomerCenterEvent) onEvent,
  ) => throw const RevenueCatUnavailableException();
}

final class RevenueCatUnavailableException implements Exception {
  const new();

  @override
  String toString() => 'RevenueCatUnavailableException()';
}

/// A store receipt is linked to another identified CraftSky customer.
/// The exception deliberately contains no unverified account identity.
final class StoreAccountConflictException implements Exception {
  const new({this.ownerAppUserId, this.ownerHash});
  final String? ownerAppUserId;
  final String? ownerHash;
  @override
  String toString() => 'StoreAccountConflictException(<redacted>)';
}

final class StoreOwnershipUnverifiedException implements Exception {
  const new();
  @override
  String toString() => 'StoreOwnershipUnverifiedException()';
}
