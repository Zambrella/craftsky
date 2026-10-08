import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:uuid/uuid.dart';

/// Permanent, install-scoped billing ownership reserved before provider setup.
class BillingOwnerBinding {
  BillingOwnerBinding({required String did, this.revenueCatAppUserId})
    : did = Did.parse(did) {
    final appUserId = revenueCatAppUserId;
    if (appUserId != null && !Uuid.isValidUUID(fromString: appUserId)) {
      throw const FormatException('Invalid RevenueCat app user ID');
    }
  }

  factory BillingOwnerBinding.fromMap(Map<String, Object?> map) {
    if (map.keys.any(
      (key) => key != 'did' && key != 'revenueCatAppUserId',
    )) {
      throw const FormatException('Unsupported billing owner field');
    }
    final did = map['did'];
    final appUserId = map['revenueCatAppUserId'];
    if (did is! String || (appUserId != null && appUserId is! String)) {
      throw const FormatException('Invalid billing owner');
    }
    return BillingOwnerBinding(
      did: did,
      revenueCatAppUserId: appUserId as String?,
    );
  }

  final Did did;
  final String? revenueCatAppUserId;

  Map<String, Object?> toMap() => {
    'did': did.value,
    'revenueCatAppUserId': revenueCatAppUserId,
  };

  @override
  String toString() => 'BillingOwnerBinding(<redacted>)';
}
