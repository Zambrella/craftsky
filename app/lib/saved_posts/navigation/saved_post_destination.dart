import 'package:craftsky_app/saved_posts/models/saved_post.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

part 'saved_post_destination.mapper.dart';

@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class SavedPostDestination with SavedPostDestinationMappable {
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
  String toString() => 'SavedPostDestination(<redacted>)';
}
