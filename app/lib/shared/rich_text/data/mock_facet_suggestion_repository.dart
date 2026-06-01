// Public constructor names match test data vocabulary while storing private
// fields.
// ignore_for_file: cascade_invocations, prefer_initializing_formals

import 'package:craftsky_app/shared/rich_text/data/facet_suggestion_repository.dart';

/// Mock account suggestions for the Flutter-only facets slice.
class MockAccountSuggestionRepository implements AccountSuggestionRepository {
  /// Creates a repository with injectable [accounts].
  const MockAccountSuggestionRepository({
    required List<AccountSuggestion> accounts,
  }) : _accounts = accounts;

  final List<AccountSuggestion> _accounts;

  @override
  Future<List<AccountSuggestion>> searchAccounts(String query) async {
    final normalizedQuery = query.toLowerCase();
    final matches = _accounts.where((account) {
      if (!account.isCraftskyProfile) {
        return false;
      }
      return account.handle.toLowerCase().contains(normalizedQuery) ||
          (account.displayName?.toLowerCase().contains(normalizedQuery) ??
              false);
    }).toList();

    matches.sort((a, b) {
      if (a.viewerIsFollowing != b.viewerIsFollowing) {
        return a.viewerIsFollowing ? -1 : 1;
      }
      return a.handle.compareTo(b.handle);
    });
    return matches;
  }

  @override
  Future<String?> didForHandle(String handle) async {
    for (final account in _accounts) {
      if (account.isCraftskyProfile && account.handle == handle) {
        return account.did;
      }
    }
    return null;
  }
}

/// Mock hashtag suggestions for the Flutter-only facets slice.
class MockHashtagSuggestionRepository implements HashtagSuggestionRepository {
  /// Creates a repository with injectable [hashtags].
  const MockHashtagSuggestionRepository({
    required List<HashtagSuggestion> hashtags,
  }) : _hashtags = hashtags;

  final List<HashtagSuggestion> _hashtags;

  @override
  Future<List<HashtagSuggestion>> searchHashtags(String query) async {
    final normalizedQuery = query.toLowerCase();
    return _hashtags
        .where((hashtag) => hashtag.tag.toLowerCase().contains(normalizedQuery))
        .toList();
  }
}
