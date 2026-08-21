import 'package:flutter/foundation.dart';

enum InstagramSuggestionState {
  pending,
  accepting,
  followed,
  alreadyFollowing,
  dismissed,
  invalidated,
  unknown;

  static InstagramSuggestionState fromWire(String value) => switch (value) {
    'pending' => pending,
    'accepting' => accepting,
    'followed' => followed,
    'alreadyFollowing' => alreadyFollowing,
    'dismissed' => dismissed,
    'invalidated' => invalidated,
    _ => unknown,
  };
}

@immutable
final class InstagramSuggestionTarget {
  const InstagramSuggestionTarget({
    required this.did,
    required this.handle,
    this.displayName,
    this.avatar,
  });

  factory InstagramSuggestionTarget.fromMap(Map<String, dynamic> map) {
    final did = map['did'];
    final handle = map['handle'];
    final displayName = map['displayName'];
    final avatar = map['avatar'];
    if (did is! String ||
        did.isEmpty ||
        handle is! String ||
        handle.isEmpty ||
        displayName is! String? ||
        avatar is! String?) {
      throw const FormatException('invalid_instagram_suggestion_target');
    }
    return InstagramSuggestionTarget(
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

  String get displayLabel =>
      (displayName?.trim().isNotEmpty ?? false) ? displayName!.trim() : handle;

  @override
  String toString() => 'InstagramSuggestionTarget([REDACTED])';
}

@immutable
final class InstagramSuggestion {
  const InstagramSuggestion({
    required this.suggestionId,
    required this.target,
    required this.createdAt,
  });

  factory InstagramSuggestion.fromMap(Map<String, dynamic> map) {
    final suggestionId = map['suggestionId'];
    final target = map['target'];
    final createdAt = map['createdAt'];
    if (suggestionId is! String ||
        suggestionId.isEmpty ||
        target is! Map<String, dynamic> ||
        createdAt is! String) {
      throw const FormatException('invalid_instagram_suggestion');
    }
    return InstagramSuggestion(
      suggestionId: suggestionId,
      target: InstagramSuggestionTarget.fromMap(target),
      createdAt: DateTime.parse(createdAt).toUtc(),
    );
  }

  final String suggestionId;
  final InstagramSuggestionTarget target;
  final DateTime createdAt;

  @override
  String toString() => 'InstagramSuggestion([REDACTED])';
}

@immutable
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
              throw const FormatException('invalid_instagram_suggestion_item');
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

@immutable
final class InstagramSuggestionActionResult {
  const InstagramSuggestionActionResult({
    required this.suggestionId,
    required this.state,
  });

  factory InstagramSuggestionActionResult.fromMap(Map<String, dynamic> map) {
    final suggestionId = map['suggestionId'];
    final state = map['state'];
    if (suggestionId is! String || suggestionId.isEmpty || state is! String) {
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
