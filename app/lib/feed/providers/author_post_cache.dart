import 'package:craftsky_app/feed/models/post.dart';

Iterable<String> authorPostCacheIds(Post post) {
  return <String>{post.author.did, post.author.handle};
}

List<Post> prependPostIfAbsent(List<Post> items, Post post) {
  if (items.any((item) => item.uri == post.uri)) return items;
  return [post, ...items];
}

List<Post> removePostByRkey(List<Post> items, String rkey) {
  return items.where((post) => post.rkey != rkey).toList(growable: false);
}

List<Post> replacePostByIdentity(List<Post> items, Post post) {
  return [
    for (final item in items)
      if (item.uri == post.uri || item.rkey == post.rkey) post else item,
  ];
}
