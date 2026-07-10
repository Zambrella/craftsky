import 'package:craftsky_app/moderation/models/moderation_metadata.dart';
import 'package:craftsky_app/projects/models/project.dart';
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
/// `images` carries optional post-image metadata returned by AppView for
/// rendering feed cards and full-screen galleries.
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
    this.quoteCount = 0,
    this.viewerHasReplied = false,
    this.images,
    this.facets,
    this.reply,
    this.quote,
    this.quoteView,
    this.moderation,
    this.project,
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
  final QuoteView? quoteView;
  final DateTime createdAt;
  final DateTime indexedAt;
  final PostAuthor author;
  final int likeCount;
  final int repostCount;
  final int quoteCount;
  final int replyCount;
  final bool viewerHasLiked;
  final bool viewerHasReposted;
  final bool viewerHasReplied;
  final List<PostImage>? images;
  final ModerationMetadata? moderation;
  final Project? project;
}

/// Compact one-level preview for quoted content. The AppView owns visibility
/// policy and sends stable states so quote posts can remain renderable when the
/// target is unavailable.
@MappableClass(ignoreNull: true)
class QuoteView with QuoteViewMappable {
  const QuoteView({required this.state, this.post});

  final String state;
  final QuotePreviewPost? post;
}

@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [AtUriMapper(), CidMapper()],
)
class QuotePreviewPost with QuotePreviewPostMappable {
  QuotePreviewPost({
    required String uri,
    required String cid,
    required this.text,
    required this.author,
    required this.createdAt,
    this.images,
    this.project,
  }) : uri = AtUri.parse(uri),
       cid = Cid.parse(cid);

  final AtUri uri;
  final Cid cid;
  final String text;
  final PostAuthor author;
  final DateTime createdAt;
  final List<PostImage>? images;
  final Project? project;
}

@MappableClass(ignoreNull: true, includeCustomMappers: [CidMapper()])
class PostImage with PostImageMappable {
  PostImage({
    required String cid,
    required this.mime,
    required this.size,
    required this.alt,
    this.aspectRatio,
    this.thumb,
    this.fullsize,
  }) : cid = Cid.parse(cid);

  final Cid cid;
  final String mime;
  final int size;
  final String alt;
  final PostImageAspectRatio? aspectRatio;
  final String? thumb;
  final String? fullsize;
}

@MappableClass()
class PostImageAspectRatio with PostImageAspectRatioMappable {
  const PostImageAspectRatio({required this.width, required this.height});

  final int width;
  final int height;
}

/// Author identity embedded in every [Post] response.
///
/// `avatar` is the display-ready URL. `avatarCid` is retained because older
/// AppView responses and notification-style payloads may still include only
/// the raw CID.
@MappableClass(
  ignoreNull: true,
  includeCustomMappers: [DidMapper(), HandleMapper(), CidMapper()],
)
class PostAuthor with PostAuthorMappable {
  PostAuthor({
    required String did,
    required String handle,
    this.displayName,
    this.avatar,
    String? avatarCid,
  }) : did = Did.parse(did),
       handle = Handle.parse(handle),
       avatarCid = avatarCid == null ? null : Cid.parse(avatarCid);

  final Did did;
  final Handle handle;
  final String? displayName;
  final String? avatar;
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
