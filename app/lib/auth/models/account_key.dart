import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

/// A provider-family key whose diagnostics never expose account identity.
@immutable
final class AccountKey {
  AccountKey(String did) : did = Did.parse(did);

  final Did did;

  @override
  bool operator ==(Object other) => other is AccountKey && other.did == did;

  @override
  int get hashCode => did.hashCode;

  @override
  String toString() => 'AccountKey(<redacted>)';
}
