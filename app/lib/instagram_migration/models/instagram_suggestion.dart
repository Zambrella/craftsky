enum InstagramSuggestionState {
  pending,
  accepting,
  accepted,
  alreadyFollowing,
  dismissed,
  invalidated,
  unknown;

  static InstagramSuggestionState fromWire(String value) => switch (value) {
    'pending' => pending,
    'accepting' => accepting,
    'accepted' => accepted,
    'alreadyFollowing' => alreadyFollowing,
    'dismissed' => dismissed,
    'invalidated' => invalidated,
    _ => unknown,
  };
}

enum InstagramSuggestionReason {
  verifiedInstagramFollow,
  unknown;

  static InstagramSuggestionReason fromWire(String value) => switch (value) {
    'verifiedInstagramFollow' => verifiedInstagramFollow,
    _ => unknown,
  };
}

final class InstagramSuggestionProfile {
  const InstagramSuggestionProfile({
    required this.did,
    required this.handle,
    this.displayName,
    this.avatar,
  });

  factory InstagramSuggestionProfile.fromMap(Map<String, dynamic> map) {
    final did = map['did'];
    final handle = map['handle'];
    final displayName = map['displayName'];
    final avatar = map['avatar'];
    if (did is! String ||
        handle is! String ||
        displayName is! String? ||
        avatar is! String?) {
      throw const FormatException('invalid_instagram_suggestion_profile');
    }
    return InstagramSuggestionProfile(
      did: did,
      handle: handle,
      displayName: displayName,
      avatar: avatar,
    );
  }

  final String did;
  final String handle;
  final String? displayName;
  final String? avatar;

  @override
  String toString() => 'InstagramSuggestionProfile([REDACTED])';
}

final class InstagramSuggestion {
  const InstagramSuggestion({
    required this.suggestionId,
    required this.profile,
    required this.reason,
    required this.state,
  });

  factory InstagramSuggestion.fromMap(Map<String, dynamic> map) {
    final suggestionId = map['suggestionId'];
    final profile = map['profile'];
    final reason = map['reason'];
    final state = map['state'];
    if (suggestionId is! String ||
        profile is! Map<String, dynamic> ||
        reason is! String ||
        state is! String) {
      throw const FormatException('invalid_instagram_suggestion');
    }
    return InstagramSuggestion(
      suggestionId: suggestionId,
      profile: InstagramSuggestionProfile.fromMap(profile),
      reason: InstagramSuggestionReason.fromWire(reason),
      state: InstagramSuggestionState.fromWire(state),
    );
  }

  final String suggestionId;
  final InstagramSuggestionProfile profile;
  final InstagramSuggestionReason reason;
  final InstagramSuggestionState state;

  @override
  String toString() => 'InstagramSuggestion([REDACTED])';
}

final class InstagramSuggestionPage {
  InstagramSuggestionPage({
    required List<InstagramSuggestion> items,
    required this.cursor,
  }) : items = List.unmodifiable(items);

  factory InstagramSuggestionPage.fromMap(Map<String, dynamic> map) {
    final items = map['items'];
    final cursor = map['cursor'];
    if (items is! List<dynamic> || cursor is! String?) {
      throw const FormatException('invalid_instagram_suggestion_page');
    }
    return InstagramSuggestionPage(
      items: items
          .map((item) {
            if (item is! Map<String, dynamic>) {
              throw const FormatException(
                'invalid_instagram_suggestion_page_item',
              );
            }
            return InstagramSuggestion.fromMap(item);
          })
          .toList(growable: false),
      cursor: cursor,
    );
  }

  final List<InstagramSuggestion> items;
  final String? cursor;

  @override
  String toString() => 'InstagramSuggestionPage([REDACTED])';
}

final class InstagramSuggestionActionResult {
  const InstagramSuggestionActionResult({
    required this.suggestionId,
    required this.state,
  });

  factory InstagramSuggestionActionResult.fromMap(Map<String, dynamic> map) {
    final suggestionId = map['suggestionId'];
    final state = map['state'];
    if (suggestionId is! String || state is! String) {
      throw const FormatException('invalid_instagram_suggestion_action');
    }
    return InstagramSuggestionActionResult(
      suggestionId: suggestionId,
      state: InstagramSuggestionState.fromWire(state),
    );
  }

  final String suggestionId;
  final InstagramSuggestionState state;

  @override
  String toString() => 'InstagramSuggestionActionResult([REDACTED])';
}
