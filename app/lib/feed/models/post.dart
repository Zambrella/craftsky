import 'package:craftsky_app/shared/atproto/identifiers.dart';
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
@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [
    DidMapper(),
    HandleMapper(),
    CidMapper(),
    AtUriMapper(),
    RecordKeyMapper(),
  ],
)
class Post with PostMappable {
  Post({
    required String uri,
    required String cid,
    required String rkey,
    required this.text,
    required this.tags,
    required this.createdAt,
    required this.indexedAt,
    required this.author,
    required this.likeCount,
    required this.repostCount,
    required this.replyCount,
    required this.viewerHasLiked,
    required this.viewerHasReposted,
    this.viewerHasReplied = false,
    this.facets,
    this.reply,
    this.quote,
  }) : uri = AtUri.parse(uri),
       cid = Cid.parse(cid),
       rkey = RecordKey.parse(rkey);

  final AtUri uri;
  final Cid cid;
  final RecordKey rkey;
  final String text;
  final List<Map<String, dynamic>>? facets;
  final List<String> tags;
  final PostReply? reply;
  final PostRef? quote;
  final DateTime createdAt;
  final DateTime indexedAt;
  final PostAuthor author;
  final int likeCount;
  final int repostCount;
  final int replyCount;
  final bool viewerHasLiked;
  final bool viewerHasReposted;
  final bool viewerHasReplied;
}

/// Author identity embedded in every [Post] response.
///
/// `avatarCid` is a bare CID, not a URL — image proxying is its own
/// future spec.
@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [DidMapper(), HandleMapper(), CidMapper()],
)
class PostAuthor with PostAuthorMappable {
  PostAuthor({
    required String did,
    required String handle,
    this.displayName,
    String? avatarCid,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle),
       avatarCid = avatarCid == null ? null : Cid.parse(avatarCid);

  final Did did;
  final Handle handle;
  final String? displayName;
  final Cid? avatarCid;
}

/// `(uri, cid)` reference to another atproto record. Used for reply
/// roots/parents and embedded quotes.
@MappableClass(includeCustomMappers: [AtUriMapper(), CidMapper()])
class PostRef with PostRefMappable {
  PostRef({required String uri, required String cid})
    : uri = AtUri.parse(uri),
      cid = Cid.parse(cid);
  final AtUri uri;
  final Cid cid;
}

/// Reply target, lexicon-shaped.
@MappableClass()
class PostReply with PostReplyMappable {
  const PostReply({required this.root, required this.parent});
  final PostRef root;
  final PostRef parent;
}
