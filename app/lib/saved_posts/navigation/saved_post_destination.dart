import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/foundation.dart';

@immutable
final class SavedPostDestination {
  const SavedPostDestination({required this.threadUri, this.focusUri});

  factory SavedPostDestination.forItem(SavedPostItem item) {
    final post = item.post;
    final root = post.reply?.root.uri;
    return SavedPostDestination(
      threadUri: root ?? post.uri,
      focusUri: root == null ? null : post.uri,
    );
  }

  final AtUri threadUri;
  final AtUri? focusUri;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is SavedPostDestination &&
          threadUri == other.threadUri &&
          focusUri == other.focusUri;

  @override
  int get hashCode => Object.hash(threadUri, focusUri);

  @override
  String toString() => 'SavedPostDestination(<redacted>)';
}
