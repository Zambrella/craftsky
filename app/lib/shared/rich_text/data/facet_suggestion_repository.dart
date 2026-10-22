// Repository interfaces are intentionally one-method seams for future AppView
// data sources.

import 'package:craftsky_app/shared/rich_text/facet_generator.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'facet_suggestion_repository.mapper.dart';

/// Repository for local/mock CraftSky account suggestions.
abstract interface class AccountSuggestionRepository
    implements MentionResolver {
  /// Searches account suggestions for [query].
  Future<List<AccountSuggestion>> searchAccounts(String query);
}

/// Repository for local/mock hashtag suggestions.
abstract interface class HashtagSuggestionRepository {
  /// Searches hashtag suggestions for [query].
  Future<List<HashtagSuggestion>> searchHashtags(String query);
}

/// Account data needed by mention autocomplete and local mention resolution.
@MappableClass()
class AccountSuggestion with AccountSuggestionMappable {
  /// Creates an account suggestion.
  const new({
    required this.did,
    required this.handle,
    required this.displayName,
    required this.avatar,
    this.crafts = const [],
    this.isCraftskyProfile = false,
    this.viewerIsFollowing = false,
  });

  /// Account DID.
  final String did;

  /// Account handle without leading `@`.
  final String handle;

  /// Display name shown in suggestions.
  final String? displayName;

  /// Avatar URL shown in suggestions.
  final String? avatar;

  final List<String> crafts;

  /// Whether the account belongs to CraftSky for this Flutter-only slice.
  final bool isCraftskyProfile;

  /// Whether the current viewer follows this account.
  final bool viewerIsFollowing;
}

/// Hashtag data needed by hashtag autocomplete.
@MappableClass()
class HashtagSuggestion with HashtagSuggestionMappable {
  /// Creates a hashtag suggestion.
  const new({required this.tag, this.postCount = 0});

  /// Repository display/canonical tag casing, without leading `#`.
  final String tag;

  /// All-time distinct-post count on the suggestion surface.
  final int postCount;
}
