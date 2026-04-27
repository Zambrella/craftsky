import 'package:dart_mappable/dart_mappable.dart';

part 'placeholder_post.mapper.dart';

/// Stand-in for a real post record while feed/post lexicon wiring is
/// being built. Lets us draft `PostCard` against a stable shape so the
/// real model swap-in is just an import change at the call site.
///
/// `craftLabel` mirrors the inline `Sewing · WIP` editorial pattern from
/// the design system. Once real records land it'll be derived from a
/// post's craft / status fields.
@MappableClass()
class PlaceholderPost with PlaceholderPostMappable {
  const PlaceholderPost({
    required this.id,
    required this.authorHandle,
    required this.authorDisplayName,
    required this.body,
    required this.postedAt,
    required this.replyCount,
    required this.repostCount,
    required this.likeCount,
    this.craftLabel,
  });

  final String id;
  final String authorHandle;
  final String authorDisplayName;
  final String body;
  final DateTime postedAt;
  final int replyCount;
  final int repostCount;
  final int likeCount;
  final String? craftLabel;
}
