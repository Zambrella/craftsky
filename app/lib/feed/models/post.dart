import 'package:dart_mappable/dart_mappable.dart';

part 'post.mapper.dart';

/// Wire shape for `social.craftsky.feed.post` records as returned by
/// the AppView's post endpoints. Author hydration (`{did, handle,
/// displayName, avatarCid}`) is embedded so the client can render a
/// feed without N+1 lookups.
///
/// `facets` is preserved as raw JSON (`List<Map<String, dynamic>>?`) —
/// no client code renders rich text yet, and the AppView treats facets
/// as a pass-through (lexicon-validated by the receiving PDS). A typed
/// `Facet` model lands when the richtext renderer does.
///
/// `images` is omitted from this model entirely — the v1 AppView
/// response shape does not include it.
@MappableClass(ignoreNull: true)
class Post with PostMappable {
  const Post({
    required this.uri,
    required this.cid,
    required this.rkey,
    required this.text,
    required this.tags,
    required this.createdAt,
    required this.indexedAt,
    required this.author,
    this.facets,
    this.reply,
    this.quote,
  });

  final String uri;
  final String cid;
  final String rkey;
  final String text;
  final List<Map<String, dynamic>>? facets;
  final List<String> tags;
  final PostReply? reply;
  final PostRef? quote;
  final DateTime createdAt;
  final DateTime indexedAt;
  final PostAuthor author;
}

/// Author identity embedded in every [Post] response.
///
/// `avatarCid` is a bare CID, not a URL — image proxying is its own
/// future spec.
@MappableClass(ignoreNull: true)
class PostAuthor with PostAuthorMappable {
  const PostAuthor({
    required this.did,
    required this.handle,
    this.displayName,
    this.avatarCid,
  });

  final String did;
  final String handle;
  final String? displayName;
  final String? avatarCid;
}

/// `(uri, cid)` reference to another atproto record. Used for reply
/// roots/parents and embedded quotes.
@MappableClass()
class PostRef with PostRefMappable {
  const PostRef({required this.uri, required this.cid});
  final String uri;
  final String cid;
}

/// Reply target, lexicon-shaped.
@MappableClass()
class PostReply with PostReplyMappable {
  const PostReply({required this.root, required this.parent});
  final PostRef root;
  final PostRef parent;
}
