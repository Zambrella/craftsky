import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';

List<Post> appendUniquePosts(List<Post> current, List<Post> next) {
  final seen = current.map((post) => post.uri).toSet();
  return [
    ...current,
    for (final post in next)
      if (seen.add(post.uri)) post,
  ];
}

List<ProfileSearchResult> appendUniqueProfiles(
  List<ProfileSearchResult> current,
  List<ProfileSearchResult> next,
) {
  final seen = current.map((profile) => profile.did.toString()).toSet();
  final seenHandles = current
      .map((profile) => profile.handle.toString())
      .toSet();
  return [
    ...current,
    for (final profile in next)
      if (seen.add(profile.did.toString()) &&
          seenHandles.add(profile.handle.toString()))
        profile,
  ];
}
