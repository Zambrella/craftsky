import 'package:flutter/foundation.dart';

@immutable
final class AccountSubscriptionId {
  factory AccountSubscriptionId.parse(String value) {
    if (!_identifierPattern.hasMatch(value)) {
      throw const FormatException('Invalid account subscription ID');
    }
    return AccountSubscriptionId._(value);
  }

  const AccountSubscriptionId._(this.wireValue);

  static final _identifierPattern = RegExp(r'^[A-Za-z0-9_-]{1,128}$');

  /// Wire access is reserved for secure routing storage and comparison.
  final String wireValue;

  @override
  bool operator ==(Object other) =>
      other is AccountSubscriptionId && other.wireValue == wireValue;

  @override
  int get hashCode => wireValue.hashCode;

  @override
  String toString() => '<redacted-account-subscription-id>';
}
