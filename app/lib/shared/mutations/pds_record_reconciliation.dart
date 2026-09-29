import 'package:flutter/foundation.dart';

@immutable
final class PdsRecordProjection {
  const PdsRecordProjection({
    required this.uri,
    required this.cid,
    required this.content,
  });

  final String uri;
  final String cid;
  final Map<String, Object?> content;
}

@immutable
final class PdsCompoundProfileProjection {
  const PdsCompoundProfileProjection({
    required this.bluesky,
    required this.craftsky,
  });

  final PdsRecordProjection? bluesky;
  final PdsRecordProjection? craftsky;
}

// A common generic contract keeps branch-selected reconciliation predicates
// strongly typed at their call sites.
// ignore: one_member_abstracts
abstract interface class PdsReconciliation<T> {
  bool agrees(T authoritativeValue);
}

@immutable
final class PdsAppendReconciliation
    implements PdsReconciliation<PdsRecordProjection?> {
  const PdsAppendReconciliation({
    required this.selectedUri,
    required this.acceptedCid,
    required this.controlledContent,
  });

  final String selectedUri;
  final String acceptedCid;
  final Map<String, Object?> controlledContent;

  @override
  bool agrees(PdsRecordProjection? authoritativeValue) =>
      authoritativeValue != null &&
      authoritativeValue.uri == selectedUri &&
      (authoritativeValue.cid == acceptedCid ||
          _containsControlledContent(
            authoritativeValue.content,
            controlledContent,
          ));
}

@immutable
final class PdsAddressedUpdateReconciliation
    implements PdsReconciliation<PdsRecordProjection?> {
  const PdsAddressedUpdateReconciliation({
    required this.uri,
    required this.acceptedCid,
    required this.controlledContent,
  });

  final String uri;
  final String acceptedCid;
  final Map<String, Object?> controlledContent;

  @override
  bool agrees(PdsRecordProjection? authoritativeValue) =>
      authoritativeValue != null &&
      authoritativeValue.uri == uri &&
      (authoritativeValue.cid == acceptedCid ||
          _containsControlledContent(
            authoritativeValue.content,
            controlledContent,
          ));
}

@immutable
final class PdsAddressedDeleteReconciliation
    implements PdsReconciliation<PdsRecordProjection?> {
  const PdsAddressedDeleteReconciliation({required this.uri});

  final String uri;

  @override
  bool agrees(PdsRecordProjection? authoritativeValue) =>
      authoritativeValue == null;
}

@immutable
final class PdsSetReconciliation implements PdsReconciliation<bool> {
  const PdsSetReconciliation({required this.active});

  final bool active;

  @override
  bool agrees(bool authoritativeValue) => authoritativeValue == active;
}

@immutable
final class PdsFixedKeyReconciliation
    implements PdsReconciliation<PdsRecordProjection?> {
  const PdsFixedKeyReconciliation({
    required this.uri,
    required this.controlledContent,
  });

  final String uri;
  final Map<String, Object?> controlledContent;

  @override
  bool agrees(PdsRecordProjection? authoritativeValue) =>
      authoritativeValue != null &&
      authoritativeValue.uri == uri &&
      _containsControlledContent(
        authoritativeValue.content,
        controlledContent,
      );
}

@immutable
final class PdsCompoundProfileReconciliation
    implements PdsReconciliation<PdsCompoundProfileProjection?> {
  const PdsCompoundProfileReconciliation({
    required this.bluesky,
    required this.craftsky,
  });

  final PdsFixedKeyReconciliation bluesky;
  final PdsFixedKeyReconciliation craftsky;

  @override
  bool agrees(PdsCompoundProfileProjection? authoritativeValue) =>
      authoritativeValue != null &&
      bluesky.agrees(authoritativeValue.bluesky) &&
      craftsky.agrees(authoritativeValue.craftsky);
}

bool _containsControlledContent(
  Map<String, Object?> authoritative,
  Map<String, Object?> controlled,
) {
  for (final entry in controlled.entries) {
    if (!authoritative.containsKey(entry.key) ||
        !_canonicalEquals(authoritative[entry.key], entry.value)) {
      return false;
    }
  }
  return true;
}

bool _canonicalEquals(Object? left, Object? right) {
  if (left is Map && right is Map) {
    if (left.length != right.length) return false;
    for (final entry in left.entries) {
      if (!right.containsKey(entry.key) ||
          !_canonicalEquals(right[entry.key], entry.value)) {
        return false;
      }
    }
    return true;
  }
  if (left is List && right is List) {
    if (left.length != right.length) return false;
    for (var index = 0; index < left.length; index++) {
      if (!_canonicalEquals(left[index], right[index])) return false;
    }
    return true;
  }
  return left == right;
}
