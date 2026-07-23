import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

/// Account/URI selector key whose diagnostics expose no private values.
@immutable
final class SavedPostKey {
  const SavedPostKey({required this.account, required this.uri});

  final AccountKey account;
  final AtUri uri;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SavedPostKey && account == other.account && uri == other.uri;

  @override
  int get hashCode => Object.hash(account, uri);

  @override
  String toString() => 'SavedPostKey(<redacted>)';
}

/// Per-opening dialog key with private account, URI, and folder values
/// redacted.
@immutable
final class SavePostDialogKey {
  const SavePostDialogKey({
    required this.account,
    required this.uri,
    this.initialFolderId,
  });

  final AccountKey account;
  final AtUri uri;
  final String? initialFolderId;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SavePostDialogKey &&
          account == other.account &&
          uri == other.uri &&
          initialFolderId == other.initialFolderId;

  @override
  int get hashCode => Object.hash(account, uri, initialFolderId);

  @override
  String toString() => 'SavePostDialogKey(<redacted>)';
}

/// Account/scope/sort key with private folder values redacted.
@immutable
final class SavedPostListKey {
  const SavedPostListKey({
    required this.account,
    required this.scope,
    required this.sort,
  });

  final AccountKey account;
  final SavedPostScope scope;
  final SavedPostSort sort;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SavedPostListKey &&
          account == other.account &&
          scope == other.scope &&
          sort == other.sort;

  @override
  int get hashCode => Object.hash(account, scope, sort);

  @override
  String toString() => 'SavedPostListKey(<redacted>)';
}
