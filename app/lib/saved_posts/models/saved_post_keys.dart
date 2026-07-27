import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'saved_post_keys.mapper.dart';

/// Account/URI selector key whose diagnostics expose no private values.
@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavedPostKey with SavedPostKeyMappable {
  const SavedPostKey({required this.account, required this.uri});

  final AccountKey account;
  final AtUri uri;

  @override
  String toString() => 'SavedPostKey(<redacted>)';
}

/// Per-opening dialog key with private account, URI, and folder values
/// redacted.
@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavePostDialogKey with SavePostDialogKeyMappable {
  const SavePostDialogKey({
    required this.account,
    required this.uri,
    this.initialFolderId,
  });

  final AccountKey account;
  final AtUri uri;
  final String? initialFolderId;

  @override
  String toString() => 'SavePostDialogKey(<redacted>)';
}

/// Account/scope/sort key with private folder values redacted.
@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavedPostListKey with SavedPostListKeyMappable {
  const SavedPostListKey({
    required this.account,
    required this.scope,
    required this.sort,
  });

  final AccountKey account;
  final SavedPostScope scope;
  final SavedPostSort sort;

  @override
  String toString() => 'SavedPostListKey(<redacted>)';
}
