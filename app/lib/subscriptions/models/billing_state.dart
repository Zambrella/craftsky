import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:uuid/uuid.dart';

part 'billing_state.mapper.dart';

bool isBillingAnomaly(String value) => value.isNotEmpty && value != 'none';

bool billingRelationshipsResolved(BillingState state) {
  final subscriptionIds = state.subscriptions.map((item) => item.id).toList();
  final licenseIds = state.licenses.map((item) => item.id).toList();
  final subscriptionIdSet = subscriptionIds.toSet();
  final licenseCountsBySubscription = <String, int>{};
  for (final license in state.licenses) {
    licenseCountsBySubscription.update(
      license.subscriptionId,
      (count) => count + 1,
      ifAbsent: () => 1,
    );
  }
  return subscriptionIdSet.length == subscriptionIds.length &&
      licenseIds.toSet().length == licenseIds.length &&
      state.licenses.every(
        (license) => subscriptionIdSet.contains(license.subscriptionId),
      ) &&
      state.subscriptions.every(
        (subscription) => licenseCountsBySubscription[subscription.id] == 1,
      );
}

const int _billingDecodeMethods =
    GenerateMethods.decode | GenerateMethods.copy | GenerateMethods.equals;

@MappableClass(ignoreNull: true, generateMethods: _billingDecodeMethods)
final class BillingSubscription with BillingSubscriptionMappable {
  const BillingSubscription({
    required this.id,
    required this.productId,
    required this.store,
    required this.status,
    required this.givesAccess,
    required this.pendingPayment,
    required this.anomaly,
    this.autoRenewalStatus,
    this.currentPeriodStartsAt,
    this.currentPeriodEndsAt,
    this.endsAt,
  });

  final String id;
  final String productId;
  final String store;
  final String status;
  final bool givesAccess;
  final bool pendingPayment;
  final String? autoRenewalStatus;
  final DateTime? currentPeriodStartsAt;
  final DateTime? currentPeriodEndsAt;
  final DateTime? endsAt;
  final String anomaly;

  @override
  String toString() => 'BillingSubscription([REDACTED])';
}

@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [DidMapper()],
  generateMethods: _billingDecodeMethods,
)
final class BillingLicense with BillingLicenseMappable {
  const BillingLicense({
    required this.id,
    required this.subscriptionId,
    required this.tier,
    required this.assignable,
    required this.anomaly,
    this.assignedDid,
    this.assignedAt,
  });

  final String id;
  final String subscriptionId;
  final SubscriptionTier tier;
  final Did? assignedDid;
  final DateTime? assignedAt;
  final bool assignable;
  final String anomaly;

  @override
  String toString() => 'BillingLicense([REDACTED])';
}

@MappableClass(ignoreNull: true, generateMethods: _billingDecodeMethods)
final class BillingState with BillingStateMappable {
  const BillingState({
    required this.billingAccountId,
    required this.revenueCatAppUserId,
    required this.requestedGeneration,
    required this.reconciledGeneration,
    required this.reconciliationStale,
    required this.subscriptions,
    required this.licenses,
    this.reconciliationRequestedAt,
    this.reconciledAt,
  });

  factory BillingState.fromMap(Map<String, dynamic> map) {
    _validateBillingStateMap(map);
    final state = BillingStateMapper.fromMap(map);
    _requireUuid(state.billingAccountId, 'billingAccountId');
    _requireUuid(state.revenueCatAppUserId, 'revenueCatAppUserId');

    final subscriptionIds = <String>{};
    for (final subscription in state.subscriptions) {
      _requireUuid(subscription.id, 'subscriptions.id');
      if (!subscriptionIds.add(subscription.id)) {
        throw const FormatException('Duplicate billing subscription');
      }
    }

    final licensedSubscriptionIds = <String>{};
    for (final license in state.licenses) {
      _requireUuid(license.id, 'licenses.id');
      _requireUuid(license.subscriptionId, 'licenses.subscriptionId');
      if (!subscriptionIds.contains(license.subscriptionId)) {
        throw const FormatException('Billing license has no subscription');
      }
      if (!licensedSubscriptionIds.add(license.subscriptionId)) {
        throw const FormatException('Duplicate billing license relationship');
      }
    }
    return state;
  }

  final String billingAccountId;
  final String revenueCatAppUserId;
  final int requestedGeneration;
  final int reconciledGeneration;
  final DateTime? reconciliationRequestedAt;
  final DateTime? reconciledAt;
  final bool reconciliationStale;
  final List<BillingSubscription> subscriptions;
  final List<BillingLicense> licenses;

  Map<String, BillingLicense> get licensesBySubscriptionId => {
    for (final license in licenses) license.subscriptionId: license,
  };

  @override
  String toString() => 'BillingState([REDACTED])';
}

@MappableClass(
  includeCustomMappers: [DidMapper()],
  generateMethods: _billingDecodeMethods,
)
final class BillingAssignment with BillingAssignmentMappable {
  const BillingAssignment({
    required this.licenseId,
    required this.targetDid,
    required this.assignedAt,
  });

  factory BillingAssignment.fromMap(Map<String, dynamic> map) {
    _requireType<String>(map, 'licenseId');
    _requireType<String>(map, 'targetDid');
    _requireType<String>(map, 'assignedAt');
    final assignment = BillingAssignmentMapper.fromMap(map);
    _requireUuid(assignment.licenseId, 'licenseId');
    return assignment;
  }

  final String licenseId;
  final Did targetDid;
  final DateTime assignedAt;

  @override
  String toString() => 'BillingAssignment([REDACTED])';
}

void _requireUuid(String value, String field) {
  if (!Uuid.isValidUUID(fromString: value)) {
    throw FormatException('Invalid $field');
  }
}

void _validateBillingStateMap(Map<String, dynamic> map) {
  _requireType<String>(map, 'billingAccountId');
  _requireType<String>(map, 'revenueCatAppUserId');
  _requireType<int>(map, 'requestedGeneration');
  _requireType<int>(map, 'reconciledGeneration');
  _requireOptionalType<String>(map, 'reconciliationRequestedAt');
  _requireOptionalType<String>(map, 'reconciledAt');
  _requireType<bool>(map, 'reconciliationStale');

  final subscriptions = _requireType<List<dynamic>>(map, 'subscriptions');
  for (final value in subscriptions) {
    if (value is! Map<String, dynamic>) {
      throw const FormatException('Invalid subscriptions item');
    }
    _requireType<String>(value, 'id');
    _requireType<String>(value, 'productId');
    _requireType<String>(value, 'store');
    _requireType<String>(value, 'status');
    _requireType<bool>(value, 'givesAccess');
    _requireType<bool>(value, 'pendingPayment');
    _requireOptionalType<String>(value, 'autoRenewalStatus');
    _requireOptionalType<String>(value, 'currentPeriodStartsAt');
    _requireOptionalType<String>(value, 'currentPeriodEndsAt');
    _requireOptionalType<String>(value, 'endsAt');
    _requireType<String>(value, 'anomaly');
  }

  final licenses = _requireType<List<dynamic>>(map, 'licenses');
  for (final value in licenses) {
    if (value is! Map<String, dynamic>) {
      throw const FormatException('Invalid licenses item');
    }
    _requireType<String>(value, 'id');
    _requireType<String>(value, 'subscriptionId');
    _requireType<String>(value, 'tier');
    _requireOptionalType<String>(value, 'assignedDid');
    _requireOptionalType<String>(value, 'assignedAt');
    _requireType<bool>(value, 'assignable');
    _requireType<String>(value, 'anomaly');
  }
}

T _requireType<T>(Map<String, dynamic> map, String key) {
  final value = map[key];
  if (!map.containsKey(key) || value is! T) {
    throw FormatException('Invalid $key');
  }
  return value;
}

void _requireOptionalType<T>(Map<String, dynamic> map, String key) {
  final value = map[key];
  if (value != null && value is! T) {
    throw FormatException('Invalid $key');
  }
}
