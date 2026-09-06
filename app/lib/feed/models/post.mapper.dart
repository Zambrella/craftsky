// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'post.dart';

class PostMapper extends ClassMapperBase<Post> {
  PostMapper._();

  static PostMapper? _instance;
  static PostMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostMapper._());
      MapperContainer.globals.useAll([
        DidMapper(),
        HandleMapper(),
        CidMapper(),
        AtUriMapper(),
        RecordKeyMapper(),
      ]);
      PostAuthorMapper.ensureInitialized();
      PostImageMapper.ensureInitialized();
      PostVideoMapper.ensureInitialized();
      PostReplyMapper.ensureInitialized();
      PostRefMapper.ensureInitialized();
      QuoteViewMapper.ensureInitialized();
      ExternalImportMapper.ensureInitialized();
      PostExternalMapper.ensureInitialized();
      ModerationMetadataMapper.ensureInitialized();
      ProjectMapper.ensureInitialized();
      ContentRelationshipMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'Post';

  static AtUri _$uri(Post v) => v.uri;
  static dynamic _arg$uri(f) => f<AtUri>();
  static const Field<Post, String> _f$uri = Field('uri', _$uri, arg: _arg$uri);
  static Cid _$cid(Post v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<Post, String> _f$cid = Field('cid', _$cid, arg: _arg$cid);
  static RecordKey _$rkey(Post v) => v.rkey;
  static dynamic _arg$rkey(f) => f<RecordKey>();
  static const Field<Post, String> _f$rkey = Field(
    'rkey',
    _$rkey,
    arg: _arg$rkey,
  );
  static String _$text(Post v) => v.text;
  static const Field<Post, String> _f$text = Field('text', _$text);
  static List<String> _$tags(Post v) => v.tags;
  static const Field<Post, List<String>> _f$tags = Field('tags', _$tags);
  static DateTime _$createdAt(Post v) => v.createdAt;
  static const Field<Post, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
  );
  static DateTime _$indexedAt(Post v) => v.indexedAt;
  static const Field<Post, DateTime> _f$indexedAt = Field(
    'indexedAt',
    _$indexedAt,
  );
  static PostAuthor _$author(Post v) => v.author;
  static const Field<Post, PostAuthor> _f$author = Field('author', _$author);
  static int _$likeCount(Post v) => v.likeCount;
  static const Field<Post, int> _f$likeCount = Field('likeCount', _$likeCount);
  static int _$repostCount(Post v) => v.repostCount;
  static const Field<Post, int> _f$repostCount = Field(
    'repostCount',
    _$repostCount,
  );
  static int _$replyCount(Post v) => v.replyCount;
  static const Field<Post, int> _f$replyCount = Field(
    'replyCount',
    _$replyCount,
  );
  static bool _$viewerHasLiked(Post v) => v.viewerHasLiked;
  static const Field<Post, bool> _f$viewerHasLiked = Field(
    'viewerHasLiked',
    _$viewerHasLiked,
  );
  static bool _$viewerHasReposted(Post v) => v.viewerHasReposted;
  static const Field<Post, bool> _f$viewerHasReposted = Field(
    'viewerHasReposted',
    _$viewerHasReposted,
  );
  static bool _$viewerHasSaved(Post v) => v.viewerHasSaved;
  static const Field<Post, bool> _f$viewerHasSaved = Field(
    'viewerHasSaved',
    _$viewerHasSaved,
  );
  static List<String> _$langs(Post v) => v.langs;
  static const Field<Post, List<String>> _f$langs = Field(
    'langs',
    _$langs,
    opt: true,
    def: const [],
  );
  static int _$quoteCount(Post v) => v.quoteCount;
  static const Field<Post, int> _f$quoteCount = Field(
    'quoteCount',
    _$quoteCount,
    opt: true,
    def: 0,
  );
  static bool _$viewerHasReplied(Post v) => v.viewerHasReplied;
  static const Field<Post, bool> _f$viewerHasReplied = Field(
    'viewerHasReplied',
    _$viewerHasReplied,
    opt: true,
    def: false,
  );
  static String? _$viewerSavedFolderId(Post v) => v.viewerSavedFolderId;
  static const Field<Post, String> _f$viewerSavedFolderId = Field(
    'viewerSavedFolderId',
    _$viewerSavedFolderId,
    opt: true,
  );
  static List<PostImage>? _$images(Post v) => v.images;
  static const Field<Post, List<PostImage>> _f$images = Field(
    'images',
    _$images,
    opt: true,
  );
  static PostVideo? _$video(Post v) => v.video;
  static const Field<Post, PostVideo> _f$video = Field(
    'video',
    _$video,
    opt: true,
  );
  static List<Map<String, dynamic>>? _$facets(Post v) => v.facets;
  static const Field<Post, List<Map<String, dynamic>>> _f$facets = Field(
    'facets',
    _$facets,
    opt: true,
  );
  static PostReply? _$reply(Post v) => v.reply;
  static const Field<Post, PostReply> _f$reply = Field(
    'reply',
    _$reply,
    opt: true,
  );
  static PostRef? _$quote(Post v) => v.quote;
  static const Field<Post, PostRef> _f$quote = Field(
    'quote',
    _$quote,
    opt: true,
  );
  static QuoteView? _$quoteView(Post v) => v.quoteView;
  static const Field<Post, QuoteView> _f$quoteView = Field(
    'quoteView',
    _$quoteView,
    opt: true,
  );
  static ExternalImport? _$externalImport(Post v) => v.externalImport;
  static const Field<Post, ExternalImport> _f$externalImport = Field(
    'externalImport',
    _$externalImport,
    opt: true,
  );
  static PostExternal? _$external(Post v) => v.external;
  static const Field<Post, PostExternal> _f$external = Field(
    'external',
    _$external,
    opt: true,
  );
  static ModerationMetadata? _$moderation(Post v) => v.moderation;
  static const Field<Post, ModerationMetadata> _f$moderation = Field(
    'moderation',
    _$moderation,
    opt: true,
  );
  static Project? _$project(Post v) => v.project;
  static const Field<Post, Project> _f$project = Field(
    'project',
    _$project,
    opt: true,
  );
  static String? _$availability(Post v) => v.availability;
  static const Field<Post, String> _f$availability = Field(
    'availability',
    _$availability,
    opt: true,
  );
  static ContentRelationship? _$relationship(Post v) => v.relationship;
  static const Field<Post, ContentRelationship> _f$relationship = Field(
    'relationship',
    _$relationship,
    opt: true,
  );

  @override
  final MappableFields<Post> fields = const {
    #uri: _f$uri,
    #cid: _f$cid,
    #rkey: _f$rkey,
    #text: _f$text,
    #tags: _f$tags,
    #createdAt: _f$createdAt,
    #indexedAt: _f$indexedAt,
    #author: _f$author,
    #likeCount: _f$likeCount,
    #repostCount: _f$repostCount,
    #replyCount: _f$replyCount,
    #viewerHasLiked: _f$viewerHasLiked,
    #viewerHasReposted: _f$viewerHasReposted,
    #viewerHasSaved: _f$viewerHasSaved,
    #langs: _f$langs,
    #quoteCount: _f$quoteCount,
    #viewerHasReplied: _f$viewerHasReplied,
    #viewerSavedFolderId: _f$viewerSavedFolderId,
    #images: _f$images,
    #video: _f$video,
    #facets: _f$facets,
    #reply: _f$reply,
    #quote: _f$quote,
    #quoteView: _f$quoteView,
    #externalImport: _f$externalImport,
    #external: _f$external,
    #moderation: _f$moderation,
    #project: _f$project,
    #availability: _f$availability,
    #relationship: _f$relationship,
  };
  @override
  final bool ignoreNull = true;

  @override
  final MappingHook hook = const PostWireHook();
  static Post _instantiate(DecodingData data) {
    return Post(
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      rkey: data.dec(_f$rkey),
      text: data.dec(_f$text),
      tags: data.dec(_f$tags),
      createdAt: data.dec(_f$createdAt),
      indexedAt: data.dec(_f$indexedAt),
      author: data.dec(_f$author),
      likeCount: data.dec(_f$likeCount),
      repostCount: data.dec(_f$repostCount),
      replyCount: data.dec(_f$replyCount),
      viewerHasLiked: data.dec(_f$viewerHasLiked),
      viewerHasReposted: data.dec(_f$viewerHasReposted),
      viewerHasSaved: data.dec(_f$viewerHasSaved),
      langs: data.dec(_f$langs),
      quoteCount: data.dec(_f$quoteCount),
      viewerHasReplied: data.dec(_f$viewerHasReplied),
      viewerSavedFolderId: data.dec(_f$viewerSavedFolderId),
      images: data.dec(_f$images),
      video: data.dec(_f$video),
      facets: data.dec(_f$facets),
      reply: data.dec(_f$reply),
      quote: data.dec(_f$quote),
      quoteView: data.dec(_f$quoteView),
      externalImport: data.dec(_f$externalImport),
      external: data.dec(_f$external),
      moderation: data.dec(_f$moderation),
      project: data.dec(_f$project),
      availability: data.dec(_f$availability),
      relationship: data.dec(_f$relationship),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static Post fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<Post>(map);
  }

  static Post fromJson(String json) {
    return ensureInitialized().decodeJson<Post>(json);
  }
}

mixin PostMappable {
  String toJson() {
    return PostMapper.ensureInitialized().encodeJson<Post>(this as Post);
  }

  Map<String, dynamic> toMap() {
    return PostMapper.ensureInitialized().encodeMap<Post>(this as Post);
  }

  PostCopyWith<Post, Post, Post> get copyWith =>
      _PostCopyWithImpl<Post, Post>(this as Post, $identity, $identity);
  @override
  String toString() {
    return PostMapper.ensureInitialized().stringifyValue(this as Post);
  }

  @override
  bool operator ==(Object other) {
    return PostMapper.ensureInitialized().equalsValue(this as Post, other);
  }

  @override
  int get hashCode {
    return PostMapper.ensureInitialized().hashValue(this as Post);
  }
}

extension PostValueCopy<$R, $Out> on ObjectCopyWith<$R, Post, $Out> {
  PostCopyWith<$R, Post, $Out> get $asPost =>
      $base.as((v, t, t2) => _PostCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostCopyWith<$R, $In extends Post, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get tags;
  PostAuthorCopyWith<$R, PostAuthor, PostAuthor> get author;
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get langs;
  ListCopyWith<$R, PostImage, PostImageCopyWith<$R, PostImage, PostImage>>?
  get images;
  PostVideoCopyWith<$R, PostVideo, PostVideo>? get video;
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get facets;
  PostReplyCopyWith<$R, PostReply, PostReply>? get reply;
  PostRefCopyWith<$R, PostRef, PostRef>? get quote;
  QuoteViewCopyWith<$R, QuoteView, QuoteView>? get quoteView;
  ExternalImportCopyWith<$R, ExternalImport, ExternalImport>?
  get externalImport;
  PostExternalCopyWith<$R, PostExternal, PostExternal>? get external;
  ModerationMetadataCopyWith<$R, ModerationMetadata, ModerationMetadata>?
  get moderation;
  ProjectCopyWith<$R, Project, Project>? get project;
  ContentRelationshipCopyWith<$R, ContentRelationship, ContentRelationship>?
  get relationship;
  $R call({
    String? uri,
    String? cid,
    String? rkey,
    String? text,
    List<String>? tags,
    DateTime? createdAt,
    DateTime? indexedAt,
    PostAuthor? author,
    int? likeCount,
    int? repostCount,
    int? replyCount,
    bool? viewerHasLiked,
    bool? viewerHasReposted,
    bool? viewerHasSaved,
    List<String>? langs,
    int? quoteCount,
    bool? viewerHasReplied,
    String? viewerSavedFolderId,
    List<PostImage>? images,
    PostVideo? video,
    List<Map<String, dynamic>>? facets,
    PostReply? reply,
    PostRef? quote,
    QuoteView? quoteView,
    ExternalImport? externalImport,
    PostExternal? external,
    ModerationMetadata? moderation,
    Project? project,
    String? availability,
    ContentRelationship? relationship,
  });
  PostCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostCopyWithImpl<$R, $Out> extends ClassCopyWithBase<$R, Post, $Out>
    implements PostCopyWith<$R, Post, $Out> {
  _PostCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<Post> $mapper = PostMapper.ensureInitialized();
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get tags =>
      ListCopyWith(
        $value.tags,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(tags: v),
      );
  @override
  PostAuthorCopyWith<$R, PostAuthor, PostAuthor> get author =>
      $value.author.copyWith.$chain((v) => call(author: v));
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get langs =>
      ListCopyWith(
        $value.langs,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(langs: v),
      );
  @override
  ListCopyWith<$R, PostImage, PostImageCopyWith<$R, PostImage, PostImage>>?
  get images => $value.images != null
      ? ListCopyWith(
          $value.images!,
          (v, t) => v.copyWith.$chain(t),
          (v) => call(images: v),
        )
      : null;
  @override
  PostVideoCopyWith<$R, PostVideo, PostVideo>? get video =>
      $value.video?.copyWith.$chain((v) => call(video: v));
  @override
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get facets => $value.facets != null
      ? ListCopyWith(
          $value.facets!,
          (v, t) => ObjectCopyWith(v, $identity, t),
          (v) => call(facets: v),
        )
      : null;
  @override
  PostReplyCopyWith<$R, PostReply, PostReply>? get reply =>
      $value.reply?.copyWith.$chain((v) => call(reply: v));
  @override
  PostRefCopyWith<$R, PostRef, PostRef>? get quote =>
      $value.quote?.copyWith.$chain((v) => call(quote: v));
  @override
  QuoteViewCopyWith<$R, QuoteView, QuoteView>? get quoteView =>
      $value.quoteView?.copyWith.$chain((v) => call(quoteView: v));
  @override
  ExternalImportCopyWith<$R, ExternalImport, ExternalImport>?
  get externalImport =>
      $value.externalImport?.copyWith.$chain((v) => call(externalImport: v));
  @override
  PostExternalCopyWith<$R, PostExternal, PostExternal>? get external =>
      $value.external?.copyWith.$chain((v) => call(external: v));
  @override
  ModerationMetadataCopyWith<$R, ModerationMetadata, ModerationMetadata>?
  get moderation =>
      $value.moderation?.copyWith.$chain((v) => call(moderation: v));
  @override
  ProjectCopyWith<$R, Project, Project>? get project =>
      $value.project?.copyWith.$chain((v) => call(project: v));
  @override
  ContentRelationshipCopyWith<$R, ContentRelationship, ContentRelationship>?
  get relationship =>
      $value.relationship?.copyWith.$chain((v) => call(relationship: v));
  @override
  $R call({
    String? uri,
    String? cid,
    String? rkey,
    String? text,
    List<String>? tags,
    DateTime? createdAt,
    DateTime? indexedAt,
    PostAuthor? author,
    int? likeCount,
    int? repostCount,
    int? replyCount,
    bool? viewerHasLiked,
    bool? viewerHasReposted,
    bool? viewerHasSaved,
    List<String>? langs,
    int? quoteCount,
    bool? viewerHasReplied,
    Object? viewerSavedFolderId = $none,
    Object? images = $none,
    Object? video = $none,
    Object? facets = $none,
    Object? reply = $none,
    Object? quote = $none,
    Object? quoteView = $none,
    Object? externalImport = $none,
    Object? external = $none,
    Object? moderation = $none,
    Object? project = $none,
    Object? availability = $none,
    Object? relationship = $none,
  }) => $apply(
    FieldCopyWithData({
      if (uri != null) #uri: uri,
      if (cid != null) #cid: cid,
      if (rkey != null) #rkey: rkey,
      if (text != null) #text: text,
      if (tags != null) #tags: tags,
      if (createdAt != null) #createdAt: createdAt,
      if (indexedAt != null) #indexedAt: indexedAt,
      if (author != null) #author: author,
      if (likeCount != null) #likeCount: likeCount,
      if (repostCount != null) #repostCount: repostCount,
      if (replyCount != null) #replyCount: replyCount,
      if (viewerHasLiked != null) #viewerHasLiked: viewerHasLiked,
      if (viewerHasReposted != null) #viewerHasReposted: viewerHasReposted,
      if (viewerHasSaved != null) #viewerHasSaved: viewerHasSaved,
      if (langs != null) #langs: langs,
      if (quoteCount != null) #quoteCount: quoteCount,
      if (viewerHasReplied != null) #viewerHasReplied: viewerHasReplied,
      if (viewerSavedFolderId != $none)
        #viewerSavedFolderId: viewerSavedFolderId,
      if (images != $none) #images: images,
      if (video != $none) #video: video,
      if (facets != $none) #facets: facets,
      if (reply != $none) #reply: reply,
      if (quote != $none) #quote: quote,
      if (quoteView != $none) #quoteView: quoteView,
      if (externalImport != $none) #externalImport: externalImport,
      if (external != $none) #external: external,
      if (moderation != $none) #moderation: moderation,
      if (project != $none) #project: project,
      if (availability != $none) #availability: availability,
      if (relationship != $none) #relationship: relationship,
    }),
  );
  @override
  Post $make(CopyWithData data) => Post(
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    rkey: data.get(#rkey, or: $value.rkey),
    text: data.get(#text, or: $value.text),
    tags: data.get(#tags, or: $value.tags),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    indexedAt: data.get(#indexedAt, or: $value.indexedAt),
    author: data.get(#author, or: $value.author),
    likeCount: data.get(#likeCount, or: $value.likeCount),
    repostCount: data.get(#repostCount, or: $value.repostCount),
    replyCount: data.get(#replyCount, or: $value.replyCount),
    viewerHasLiked: data.get(#viewerHasLiked, or: $value.viewerHasLiked),
    viewerHasReposted: data.get(
      #viewerHasReposted,
      or: $value.viewerHasReposted,
    ),
    viewerHasSaved: data.get(#viewerHasSaved, or: $value.viewerHasSaved),
    langs: data.get(#langs, or: $value.langs),
    quoteCount: data.get(#quoteCount, or: $value.quoteCount),
    viewerHasReplied: data.get(#viewerHasReplied, or: $value.viewerHasReplied),
    viewerSavedFolderId: data.get(
      #viewerSavedFolderId,
      or: $value.viewerSavedFolderId,
    ),
    images: data.get(#images, or: $value.images),
    video: data.get(#video, or: $value.video),
    facets: data.get(#facets, or: $value.facets),
    reply: data.get(#reply, or: $value.reply),
    quote: data.get(#quote, or: $value.quote),
    quoteView: data.get(#quoteView, or: $value.quoteView),
    externalImport: data.get(#externalImport, or: $value.externalImport),
    external: data.get(#external, or: $value.external),
    moderation: data.get(#moderation, or: $value.moderation),
    project: data.get(#project, or: $value.project),
    availability: data.get(#availability, or: $value.availability),
    relationship: data.get(#relationship, or: $value.relationship),
  );

  @override
  PostCopyWith<$R2, Post, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _PostCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostAuthorMapper extends ClassMapperBase<PostAuthor> {
  PostAuthorMapper._();

  static PostAuthorMapper? _instance;
  static PostAuthorMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostAuthorMapper._());
      MapperContainer.globals.useAll([
        DidMapper(),
        HandleMapper(),
        CidMapper(),
        ProfileCustomisationMapper(),
      ]);
    }
    return _instance!;
  }

  @override
  final String id = 'PostAuthor';

  static Did _$did(PostAuthor v) => v.did;
  static dynamic _arg$did(f) => f<Did>();
  static const Field<PostAuthor, String> _f$did = Field(
    'did',
    _$did,
    arg: _arg$did,
  );
  static Handle _$handle(PostAuthor v) => v.handle;
  static dynamic _arg$handle(f) => f<Handle>();
  static const Field<PostAuthor, String> _f$handle = Field(
    'handle',
    _$handle,
    arg: _arg$handle,
  );
  static String? _$displayName(PostAuthor v) => v.displayName;
  static const Field<PostAuthor, String> _f$displayName = Field(
    'displayName',
    _$displayName,
    opt: true,
  );
  static String? _$avatar(PostAuthor v) => v.avatar;
  static const Field<PostAuthor, String> _f$avatar = Field(
    'avatar',
    _$avatar,
    opt: true,
  );
  static Cid? _$avatarCid(PostAuthor v) => v.avatarCid;
  static dynamic _arg$avatarCid(f) => f<Cid>();
  static const Field<PostAuthor, String> _f$avatarCid = Field(
    'avatarCid',
    _$avatarCid,
    opt: true,
    arg: _arg$avatarCid,
  );
  static bool? _$muted(PostAuthor v) => v.muted;
  static const Field<PostAuthor, bool> _f$muted = Field(
    'muted',
    _$muted,
    opt: true,
  );
  static bool? _$blocking(PostAuthor v) => v.blocking;
  static const Field<PostAuthor, bool> _f$blocking = Field(
    'blocking',
    _$blocking,
    opt: true,
  );
  static bool? _$blockedBy(PostAuthor v) => v.blockedBy;
  static const Field<PostAuthor, bool> _f$blockedBy = Field(
    'blockedBy',
    _$blockedBy,
    opt: true,
  );
  static ProfileCustomisation _$customisation(PostAuthor v) => v.customisation;
  static const Field<PostAuthor, ProfileCustomisation> _f$customisation = Field(
    'customisation',
    _$customisation,
    opt: true,
    def: ProfileCustomisation.defaults,
  );

  @override
  final MappableFields<PostAuthor> fields = const {
    #did: _f$did,
    #handle: _f$handle,
    #displayName: _f$displayName,
    #avatar: _f$avatar,
    #avatarCid: _f$avatarCid,
    #muted: _f$muted,
    #blocking: _f$blocking,
    #blockedBy: _f$blockedBy,
    #customisation: _f$customisation,
  };
  @override
  final bool ignoreNull = true;

  static PostAuthor _instantiate(DecodingData data) {
    return PostAuthor(
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      displayName: data.dec(_f$displayName),
      avatar: data.dec(_f$avatar),
      avatarCid: data.dec(_f$avatarCid),
      muted: data.dec(_f$muted),
      blocking: data.dec(_f$blocking),
      blockedBy: data.dec(_f$blockedBy),
      customisation: data.dec(_f$customisation),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostAuthor fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostAuthor>(map);
  }

  static PostAuthor fromJson(String json) {
    return ensureInitialized().decodeJson<PostAuthor>(json);
  }
}

mixin PostAuthorMappable {
  String toJson() {
    return PostAuthorMapper.ensureInitialized().encodeJson<PostAuthor>(
      this as PostAuthor,
    );
  }

  Map<String, dynamic> toMap() {
    return PostAuthorMapper.ensureInitialized().encodeMap<PostAuthor>(
      this as PostAuthor,
    );
  }

  PostAuthorCopyWith<PostAuthor, PostAuthor, PostAuthor> get copyWith =>
      _PostAuthorCopyWithImpl<PostAuthor, PostAuthor>(
        this as PostAuthor,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostAuthorMapper.ensureInitialized().stringifyValue(
      this as PostAuthor,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostAuthorMapper.ensureInitialized().equalsValue(
      this as PostAuthor,
      other,
    );
  }

  @override
  int get hashCode {
    return PostAuthorMapper.ensureInitialized().hashValue(this as PostAuthor);
  }
}

extension PostAuthorValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostAuthor, $Out> {
  PostAuthorCopyWith<$R, PostAuthor, $Out> get $asPostAuthor =>
      $base.as((v, t, t2) => _PostAuthorCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostAuthorCopyWith<$R, $In extends PostAuthor, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? did,
    String? handle,
    String? displayName,
    String? avatar,
    String? avatarCid,
    bool? muted,
    bool? blocking,
    bool? blockedBy,
    ProfileCustomisation? customisation,
  });
  PostAuthorCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostAuthorCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostAuthor, $Out>
    implements PostAuthorCopyWith<$R, PostAuthor, $Out> {
  _PostAuthorCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostAuthor> $mapper =
      PostAuthorMapper.ensureInitialized();
  @override
  $R call({
    String? did,
    String? handle,
    Object? displayName = $none,
    Object? avatar = $none,
    Object? avatarCid = $none,
    Object? muted = $none,
    Object? blocking = $none,
    Object? blockedBy = $none,
    ProfileCustomisation? customisation,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (displayName != $none) #displayName: displayName,
      if (avatar != $none) #avatar: avatar,
      if (avatarCid != $none) #avatarCid: avatarCid,
      if (muted != $none) #muted: muted,
      if (blocking != $none) #blocking: blocking,
      if (blockedBy != $none) #blockedBy: blockedBy,
      if (customisation != null) #customisation: customisation,
    }),
  );
  @override
  PostAuthor $make(CopyWithData data) => PostAuthor(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    displayName: data.get(#displayName, or: $value.displayName),
    avatar: data.get(#avatar, or: $value.avatar),
    avatarCid: data.get(#avatarCid, or: $value.avatarCid),
    muted: data.get(#muted, or: $value.muted),
    blocking: data.get(#blocking, or: $value.blocking),
    blockedBy: data.get(#blockedBy, or: $value.blockedBy),
    customisation: data.get(#customisation, or: $value.customisation),
  );

  @override
  PostAuthorCopyWith<$R2, PostAuthor, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostAuthorCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostImageMapper extends ClassMapperBase<PostImage> {
  PostImageMapper._();

  static PostImageMapper? _instance;
  static PostImageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostImageMapper._());
      MapperContainer.globals.useAll([CidMapper()]);
      PostImageAspectRatioMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostImage';

  static Cid _$cid(PostImage v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<PostImage, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static String _$mime(PostImage v) => v.mime;
  static const Field<PostImage, String> _f$mime = Field('mime', _$mime);
  static int _$size(PostImage v) => v.size;
  static const Field<PostImage, int> _f$size = Field('size', _$size);
  static String _$alt(PostImage v) => v.alt;
  static const Field<PostImage, String> _f$alt = Field('alt', _$alt);
  static PostImageAspectRatio? _$aspectRatio(PostImage v) => v.aspectRatio;
  static const Field<PostImage, PostImageAspectRatio> _f$aspectRatio = Field(
    'aspectRatio',
    _$aspectRatio,
    opt: true,
  );
  static String? _$thumb(PostImage v) => v.thumb;
  static const Field<PostImage, String> _f$thumb = Field(
    'thumb',
    _$thumb,
    opt: true,
  );
  static String? _$fullsize(PostImage v) => v.fullsize;
  static const Field<PostImage, String> _f$fullsize = Field(
    'fullsize',
    _$fullsize,
    opt: true,
  );

  @override
  final MappableFields<PostImage> fields = const {
    #cid: _f$cid,
    #mime: _f$mime,
    #size: _f$size,
    #alt: _f$alt,
    #aspectRatio: _f$aspectRatio,
    #thumb: _f$thumb,
    #fullsize: _f$fullsize,
  };
  @override
  final bool ignoreNull = true;

  static PostImage _instantiate(DecodingData data) {
    return PostImage(
      cid: data.dec(_f$cid),
      mime: data.dec(_f$mime),
      size: data.dec(_f$size),
      alt: data.dec(_f$alt),
      aspectRatio: data.dec(_f$aspectRatio),
      thumb: data.dec(_f$thumb),
      fullsize: data.dec(_f$fullsize),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostImage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostImage>(map);
  }

  static PostImage fromJson(String json) {
    return ensureInitialized().decodeJson<PostImage>(json);
  }
}

mixin PostImageMappable {
  String toJson() {
    return PostImageMapper.ensureInitialized().encodeJson<PostImage>(
      this as PostImage,
    );
  }

  Map<String, dynamic> toMap() {
    return PostImageMapper.ensureInitialized().encodeMap<PostImage>(
      this as PostImage,
    );
  }

  PostImageCopyWith<PostImage, PostImage, PostImage> get copyWith =>
      _PostImageCopyWithImpl<PostImage, PostImage>(
        this as PostImage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostImageMapper.ensureInitialized().stringifyValue(
      this as PostImage,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostImageMapper.ensureInitialized().equalsValue(
      this as PostImage,
      other,
    );
  }

  @override
  int get hashCode {
    return PostImageMapper.ensureInitialized().hashValue(this as PostImage);
  }
}

extension PostImageValueCopy<$R, $Out> on ObjectCopyWith<$R, PostImage, $Out> {
  PostImageCopyWith<$R, PostImage, $Out> get $asPostImage =>
      $base.as((v, t, t2) => _PostImageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostImageCopyWith<$R, $In extends PostImage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostImageAspectRatioCopyWith<$R, PostImageAspectRatio, PostImageAspectRatio>?
  get aspectRatio;
  $R call({
    String? cid,
    String? mime,
    int? size,
    String? alt,
    PostImageAspectRatio? aspectRatio,
    String? thumb,
    String? fullsize,
  });
  PostImageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostImageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostImage, $Out>
    implements PostImageCopyWith<$R, PostImage, $Out> {
  _PostImageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostImage> $mapper =
      PostImageMapper.ensureInitialized();
  @override
  PostImageAspectRatioCopyWith<$R, PostImageAspectRatio, PostImageAspectRatio>?
  get aspectRatio =>
      $value.aspectRatio?.copyWith.$chain((v) => call(aspectRatio: v));
  @override
  $R call({
    String? cid,
    String? mime,
    int? size,
    String? alt,
    Object? aspectRatio = $none,
    Object? thumb = $none,
    Object? fullsize = $none,
  }) => $apply(
    FieldCopyWithData({
      if (cid != null) #cid: cid,
      if (mime != null) #mime: mime,
      if (size != null) #size: size,
      if (alt != null) #alt: alt,
      if (aspectRatio != $none) #aspectRatio: aspectRatio,
      if (thumb != $none) #thumb: thumb,
      if (fullsize != $none) #fullsize: fullsize,
    }),
  );
  @override
  PostImage $make(CopyWithData data) => PostImage(
    cid: data.get(#cid, or: $value.cid),
    mime: data.get(#mime, or: $value.mime),
    size: data.get(#size, or: $value.size),
    alt: data.get(#alt, or: $value.alt),
    aspectRatio: data.get(#aspectRatio, or: $value.aspectRatio),
    thumb: data.get(#thumb, or: $value.thumb),
    fullsize: data.get(#fullsize, or: $value.fullsize),
  );

  @override
  PostImageCopyWith<$R2, PostImage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostImageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostImageAspectRatioMapper extends ClassMapperBase<PostImageAspectRatio> {
  PostImageAspectRatioMapper._();

  static PostImageAspectRatioMapper? _instance;
  static PostImageAspectRatioMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostImageAspectRatioMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'PostImageAspectRatio';

  static int _$width(PostImageAspectRatio v) => v.width;
  static const Field<PostImageAspectRatio, int> _f$width = Field(
    'width',
    _$width,
  );
  static int _$height(PostImageAspectRatio v) => v.height;
  static const Field<PostImageAspectRatio, int> _f$height = Field(
    'height',
    _$height,
  );

  @override
  final MappableFields<PostImageAspectRatio> fields = const {
    #width: _f$width,
    #height: _f$height,
  };

  static PostImageAspectRatio _instantiate(DecodingData data) {
    return PostImageAspectRatio(
      width: data.dec(_f$width),
      height: data.dec(_f$height),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostImageAspectRatio fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostImageAspectRatio>(map);
  }

  static PostImageAspectRatio fromJson(String json) {
    return ensureInitialized().decodeJson<PostImageAspectRatio>(json);
  }
}

mixin PostImageAspectRatioMappable {
  String toJson() {
    return PostImageAspectRatioMapper.ensureInitialized()
        .encodeJson<PostImageAspectRatio>(this as PostImageAspectRatio);
  }

  Map<String, dynamic> toMap() {
    return PostImageAspectRatioMapper.ensureInitialized()
        .encodeMap<PostImageAspectRatio>(this as PostImageAspectRatio);
  }

  PostImageAspectRatioCopyWith<
    PostImageAspectRatio,
    PostImageAspectRatio,
    PostImageAspectRatio
  >
  get copyWith =>
      _PostImageAspectRatioCopyWithImpl<
        PostImageAspectRatio,
        PostImageAspectRatio
      >(this as PostImageAspectRatio, $identity, $identity);
  @override
  String toString() {
    return PostImageAspectRatioMapper.ensureInitialized().stringifyValue(
      this as PostImageAspectRatio,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostImageAspectRatioMapper.ensureInitialized().equalsValue(
      this as PostImageAspectRatio,
      other,
    );
  }

  @override
  int get hashCode {
    return PostImageAspectRatioMapper.ensureInitialized().hashValue(
      this as PostImageAspectRatio,
    );
  }
}

extension PostImageAspectRatioValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostImageAspectRatio, $Out> {
  PostImageAspectRatioCopyWith<$R, PostImageAspectRatio, $Out>
  get $asPostImageAspectRatio => $base.as(
    (v, t, t2) => _PostImageAspectRatioCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class PostImageAspectRatioCopyWith<
  $R,
  $In extends PostImageAspectRatio,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({int? width, int? height});
  PostImageAspectRatioCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PostImageAspectRatioCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostImageAspectRatio, $Out>
    implements PostImageAspectRatioCopyWith<$R, PostImageAspectRatio, $Out> {
  _PostImageAspectRatioCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostImageAspectRatio> $mapper =
      PostImageAspectRatioMapper.ensureInitialized();
  @override
  $R call({int? width, int? height}) => $apply(
    FieldCopyWithData({
      if (width != null) #width: width,
      if (height != null) #height: height,
    }),
  );
  @override
  PostImageAspectRatio $make(CopyWithData data) => PostImageAspectRatio(
    width: data.get(#width, or: $value.width),
    height: data.get(#height, or: $value.height),
  );

  @override
  PostImageAspectRatioCopyWith<$R2, PostImageAspectRatio, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _PostImageAspectRatioCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostVideoMapper extends ClassMapperBase<PostVideo> {
  PostVideoMapper._();

  static PostVideoMapper? _instance;
  static PostVideoMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostVideoMapper._());
      MapperContainer.globals.useAll([CidMapper()]);
      PostImageAspectRatioMapper.ensureInitialized();
      PostVideoCaptionMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostVideo';

  static Cid _$cid(PostVideo v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<PostVideo, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static String _$mime(PostVideo v) => v.mime;
  static const Field<PostVideo, String> _f$mime = Field('mime', _$mime);
  static int _$size(PostVideo v) => v.size;
  static const Field<PostVideo, int> _f$size = Field('size', _$size);
  static String? _$alt(PostVideo v) => v.alt;
  static const Field<PostVideo, String> _f$alt = Field('alt', _$alt, opt: true);
  static PostImageAspectRatio? _$aspectRatio(PostVideo v) => v.aspectRatio;
  static const Field<PostVideo, PostImageAspectRatio> _f$aspectRatio = Field(
    'aspectRatio',
    _$aspectRatio,
    opt: true,
  );
  static String? _$playlist(PostVideo v) => v.playlist;
  static const Field<PostVideo, String> _f$playlist = Field(
    'playlist',
    _$playlist,
    opt: true,
  );
  static String? _$thumbnail(PostVideo v) => v.thumbnail;
  static const Field<PostVideo, String> _f$thumbnail = Field(
    'thumbnail',
    _$thumbnail,
    opt: true,
  );
  static List<PostVideoCaption> _$captions(PostVideo v) => v.captions;
  static const Field<PostVideo, List<PostVideoCaption>> _f$captions = Field(
    'captions',
    _$captions,
    opt: true,
    def: const [],
  );

  @override
  final MappableFields<PostVideo> fields = const {
    #cid: _f$cid,
    #mime: _f$mime,
    #size: _f$size,
    #alt: _f$alt,
    #aspectRatio: _f$aspectRatio,
    #playlist: _f$playlist,
    #thumbnail: _f$thumbnail,
    #captions: _f$captions,
  };
  @override
  final bool ignoreNull = true;

  static PostVideo _instantiate(DecodingData data) {
    return PostVideo(
      cid: data.dec(_f$cid),
      mime: data.dec(_f$mime),
      size: data.dec(_f$size),
      alt: data.dec(_f$alt),
      aspectRatio: data.dec(_f$aspectRatio),
      playlist: data.dec(_f$playlist),
      thumbnail: data.dec(_f$thumbnail),
      captions: data.dec(_f$captions),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostVideo fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostVideo>(map);
  }

  static PostVideo fromJson(String json) {
    return ensureInitialized().decodeJson<PostVideo>(json);
  }
}

mixin PostVideoMappable {
  String toJson() {
    return PostVideoMapper.ensureInitialized().encodeJson<PostVideo>(
      this as PostVideo,
    );
  }

  Map<String, dynamic> toMap() {
    return PostVideoMapper.ensureInitialized().encodeMap<PostVideo>(
      this as PostVideo,
    );
  }

  PostVideoCopyWith<PostVideo, PostVideo, PostVideo> get copyWith =>
      _PostVideoCopyWithImpl<PostVideo, PostVideo>(
        this as PostVideo,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostVideoMapper.ensureInitialized().stringifyValue(
      this as PostVideo,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostVideoMapper.ensureInitialized().equalsValue(
      this as PostVideo,
      other,
    );
  }

  @override
  int get hashCode {
    return PostVideoMapper.ensureInitialized().hashValue(this as PostVideo);
  }
}

extension PostVideoValueCopy<$R, $Out> on ObjectCopyWith<$R, PostVideo, $Out> {
  PostVideoCopyWith<$R, PostVideo, $Out> get $asPostVideo =>
      $base.as((v, t, t2) => _PostVideoCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostVideoCopyWith<$R, $In extends PostVideo, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostImageAspectRatioCopyWith<$R, PostImageAspectRatio, PostImageAspectRatio>?
  get aspectRatio;
  ListCopyWith<
    $R,
    PostVideoCaption,
    PostVideoCaptionCopyWith<$R, PostVideoCaption, PostVideoCaption>
  >
  get captions;
  $R call({
    String? cid,
    String? mime,
    int? size,
    String? alt,
    PostImageAspectRatio? aspectRatio,
    String? playlist,
    String? thumbnail,
    List<PostVideoCaption>? captions,
  });
  PostVideoCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostVideoCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostVideo, $Out>
    implements PostVideoCopyWith<$R, PostVideo, $Out> {
  _PostVideoCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostVideo> $mapper =
      PostVideoMapper.ensureInitialized();
  @override
  PostImageAspectRatioCopyWith<$R, PostImageAspectRatio, PostImageAspectRatio>?
  get aspectRatio =>
      $value.aspectRatio?.copyWith.$chain((v) => call(aspectRatio: v));
  @override
  ListCopyWith<
    $R,
    PostVideoCaption,
    PostVideoCaptionCopyWith<$R, PostVideoCaption, PostVideoCaption>
  >
  get captions => ListCopyWith(
    $value.captions,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(captions: v),
  );
  @override
  $R call({
    String? cid,
    String? mime,
    int? size,
    Object? alt = $none,
    Object? aspectRatio = $none,
    Object? playlist = $none,
    Object? thumbnail = $none,
    List<PostVideoCaption>? captions,
  }) => $apply(
    FieldCopyWithData({
      if (cid != null) #cid: cid,
      if (mime != null) #mime: mime,
      if (size != null) #size: size,
      if (alt != $none) #alt: alt,
      if (aspectRatio != $none) #aspectRatio: aspectRatio,
      if (playlist != $none) #playlist: playlist,
      if (thumbnail != $none) #thumbnail: thumbnail,
      if (captions != null) #captions: captions,
    }),
  );
  @override
  PostVideo $make(CopyWithData data) => PostVideo(
    cid: data.get(#cid, or: $value.cid),
    mime: data.get(#mime, or: $value.mime),
    size: data.get(#size, or: $value.size),
    alt: data.get(#alt, or: $value.alt),
    aspectRatio: data.get(#aspectRatio, or: $value.aspectRatio),
    playlist: data.get(#playlist, or: $value.playlist),
    thumbnail: data.get(#thumbnail, or: $value.thumbnail),
    captions: data.get(#captions, or: $value.captions),
  );

  @override
  PostVideoCopyWith<$R2, PostVideo, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostVideoCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostVideoCaptionMapper extends ClassMapperBase<PostVideoCaption> {
  PostVideoCaptionMapper._();

  static PostVideoCaptionMapper? _instance;
  static PostVideoCaptionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostVideoCaptionMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'PostVideoCaption';

  static String _$lang(PostVideoCaption v) => v.lang;
  static const Field<PostVideoCaption, String> _f$lang = Field('lang', _$lang);
  static String _$name(PostVideoCaption v) => v.name;
  static const Field<PostVideoCaption, String> _f$name = Field('name', _$name);
  static String _$uri(PostVideoCaption v) => v.uri;
  static const Field<PostVideoCaption, String> _f$uri = Field('uri', _$uri);

  @override
  final MappableFields<PostVideoCaption> fields = const {
    #lang: _f$lang,
    #name: _f$name,
    #uri: _f$uri,
  };

  static PostVideoCaption _instantiate(DecodingData data) {
    return PostVideoCaption(
      lang: data.dec(_f$lang),
      name: data.dec(_f$name),
      uri: data.dec(_f$uri),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostVideoCaption fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostVideoCaption>(map);
  }

  static PostVideoCaption fromJson(String json) {
    return ensureInitialized().decodeJson<PostVideoCaption>(json);
  }
}

mixin PostVideoCaptionMappable {
  String toJson() {
    return PostVideoCaptionMapper.ensureInitialized()
        .encodeJson<PostVideoCaption>(this as PostVideoCaption);
  }

  Map<String, dynamic> toMap() {
    return PostVideoCaptionMapper.ensureInitialized()
        .encodeMap<PostVideoCaption>(this as PostVideoCaption);
  }

  PostVideoCaptionCopyWith<PostVideoCaption, PostVideoCaption, PostVideoCaption>
  get copyWith =>
      _PostVideoCaptionCopyWithImpl<PostVideoCaption, PostVideoCaption>(
        this as PostVideoCaption,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostVideoCaptionMapper.ensureInitialized().stringifyValue(
      this as PostVideoCaption,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostVideoCaptionMapper.ensureInitialized().equalsValue(
      this as PostVideoCaption,
      other,
    );
  }

  @override
  int get hashCode {
    return PostVideoCaptionMapper.ensureInitialized().hashValue(
      this as PostVideoCaption,
    );
  }
}

extension PostVideoCaptionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostVideoCaption, $Out> {
  PostVideoCaptionCopyWith<$R, PostVideoCaption, $Out>
  get $asPostVideoCaption =>
      $base.as((v, t, t2) => _PostVideoCaptionCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostVideoCaptionCopyWith<$R, $In extends PostVideoCaption, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? lang, String? name, String? uri});
  PostVideoCaptionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PostVideoCaptionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostVideoCaption, $Out>
    implements PostVideoCaptionCopyWith<$R, PostVideoCaption, $Out> {
  _PostVideoCaptionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostVideoCaption> $mapper =
      PostVideoCaptionMapper.ensureInitialized();
  @override
  $R call({String? lang, String? name, String? uri}) => $apply(
    FieldCopyWithData({
      if (lang != null) #lang: lang,
      if (name != null) #name: name,
      if (uri != null) #uri: uri,
    }),
  );
  @override
  PostVideoCaption $make(CopyWithData data) => PostVideoCaption(
    lang: data.get(#lang, or: $value.lang),
    name: data.get(#name, or: $value.name),
    uri: data.get(#uri, or: $value.uri),
  );

  @override
  PostVideoCaptionCopyWith<$R2, PostVideoCaption, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostVideoCaptionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostReplyMapper extends ClassMapperBase<PostReply> {
  PostReplyMapper._();

  static PostReplyMapper? _instance;
  static PostReplyMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostReplyMapper._());
      PostRefMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostReply';

  static PostRef _$root(PostReply v) => v.root;
  static const Field<PostReply, PostRef> _f$root = Field('root', _$root);
  static PostRef _$parent(PostReply v) => v.parent;
  static const Field<PostReply, PostRef> _f$parent = Field('parent', _$parent);

  @override
  final MappableFields<PostReply> fields = const {
    #root: _f$root,
    #parent: _f$parent,
  };

  static PostReply _instantiate(DecodingData data) {
    return PostReply(root: data.dec(_f$root), parent: data.dec(_f$parent));
  }

  @override
  final Function instantiate = _instantiate;

  static PostReply fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostReply>(map);
  }

  static PostReply fromJson(String json) {
    return ensureInitialized().decodeJson<PostReply>(json);
  }
}

mixin PostReplyMappable {
  String toJson() {
    return PostReplyMapper.ensureInitialized().encodeJson<PostReply>(
      this as PostReply,
    );
  }

  Map<String, dynamic> toMap() {
    return PostReplyMapper.ensureInitialized().encodeMap<PostReply>(
      this as PostReply,
    );
  }

  PostReplyCopyWith<PostReply, PostReply, PostReply> get copyWith =>
      _PostReplyCopyWithImpl<PostReply, PostReply>(
        this as PostReply,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostReplyMapper.ensureInitialized().stringifyValue(
      this as PostReply,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostReplyMapper.ensureInitialized().equalsValue(
      this as PostReply,
      other,
    );
  }

  @override
  int get hashCode {
    return PostReplyMapper.ensureInitialized().hashValue(this as PostReply);
  }
}

extension PostReplyValueCopy<$R, $Out> on ObjectCopyWith<$R, PostReply, $Out> {
  PostReplyCopyWith<$R, PostReply, $Out> get $asPostReply =>
      $base.as((v, t, t2) => _PostReplyCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostReplyCopyWith<$R, $In extends PostReply, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostRefCopyWith<$R, PostRef, PostRef> get root;
  PostRefCopyWith<$R, PostRef, PostRef> get parent;
  $R call({PostRef? root, PostRef? parent});
  PostReplyCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostReplyCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostReply, $Out>
    implements PostReplyCopyWith<$R, PostReply, $Out> {
  _PostReplyCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostReply> $mapper =
      PostReplyMapper.ensureInitialized();
  @override
  PostRefCopyWith<$R, PostRef, PostRef> get root =>
      $value.root.copyWith.$chain((v) => call(root: v));
  @override
  PostRefCopyWith<$R, PostRef, PostRef> get parent =>
      $value.parent.copyWith.$chain((v) => call(parent: v));
  @override
  $R call({PostRef? root, PostRef? parent}) => $apply(
    FieldCopyWithData({
      if (root != null) #root: root,
      if (parent != null) #parent: parent,
    }),
  );
  @override
  PostReply $make(CopyWithData data) => PostReply(
    root: data.get(#root, or: $value.root),
    parent: data.get(#parent, or: $value.parent),
  );

  @override
  PostReplyCopyWith<$R2, PostReply, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostReplyCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostRefMapper extends ClassMapperBase<PostRef> {
  PostRefMapper._();

  static PostRefMapper? _instance;
  static PostRefMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostRefMapper._());
      MapperContainer.globals.useAll([AtUriMapper(), CidMapper()]);
    }
    return _instance!;
  }

  @override
  final String id = 'PostRef';

  static AtUri _$uri(PostRef v) => v.uri;
  static dynamic _arg$uri(f) => f<AtUri>();
  static const Field<PostRef, String> _f$uri = Field(
    'uri',
    _$uri,
    arg: _arg$uri,
  );
  static Cid _$cid(PostRef v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<PostRef, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );

  @override
  final MappableFields<PostRef> fields = const {#uri: _f$uri, #cid: _f$cid};

  static PostRef _instantiate(DecodingData data) {
    return PostRef(uri: data.dec(_f$uri), cid: data.dec(_f$cid));
  }

  @override
  final Function instantiate = _instantiate;

  static PostRef fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostRef>(map);
  }

  static PostRef fromJson(String json) {
    return ensureInitialized().decodeJson<PostRef>(json);
  }
}

mixin PostRefMappable {
  String toJson() {
    return PostRefMapper.ensureInitialized().encodeJson<PostRef>(
      this as PostRef,
    );
  }

  Map<String, dynamic> toMap() {
    return PostRefMapper.ensureInitialized().encodeMap<PostRef>(
      this as PostRef,
    );
  }

  PostRefCopyWith<PostRef, PostRef, PostRef> get copyWith =>
      _PostRefCopyWithImpl<PostRef, PostRef>(
        this as PostRef,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostRefMapper.ensureInitialized().stringifyValue(this as PostRef);
  }

  @override
  bool operator ==(Object other) {
    return PostRefMapper.ensureInitialized().equalsValue(
      this as PostRef,
      other,
    );
  }

  @override
  int get hashCode {
    return PostRefMapper.ensureInitialized().hashValue(this as PostRef);
  }
}

extension PostRefValueCopy<$R, $Out> on ObjectCopyWith<$R, PostRef, $Out> {
  PostRefCopyWith<$R, PostRef, $Out> get $asPostRef =>
      $base.as((v, t, t2) => _PostRefCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostRefCopyWith<$R, $In extends PostRef, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? uri, String? cid});
  PostRefCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostRefCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostRef, $Out>
    implements PostRefCopyWith<$R, PostRef, $Out> {
  _PostRefCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostRef> $mapper =
      PostRefMapper.ensureInitialized();
  @override
  $R call({String? uri, String? cid}) => $apply(
    FieldCopyWithData({if (uri != null) #uri: uri, if (cid != null) #cid: cid}),
  );
  @override
  PostRef $make(CopyWithData data) => PostRef(
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
  );

  @override
  PostRefCopyWith<$R2, PostRef, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _PostRefCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class QuoteViewMapper extends ClassMapperBase<QuoteView> {
  QuoteViewMapper._();

  static QuoteViewMapper? _instance;
  static QuoteViewMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = QuoteViewMapper._());
      QuotePreviewPostMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'QuoteView';

  static String _$state(QuoteView v) => v.state;
  static const Field<QuoteView, String> _f$state = Field('state', _$state);
  static bool? _$revealable(QuoteView v) => v.revealable;
  static const Field<QuoteView, bool> _f$revealable = Field(
    'revealable',
    _$revealable,
    opt: true,
  );
  static QuotePreviewPost? _$post(QuoteView v) => v.post;
  static const Field<QuoteView, QuotePreviewPost> _f$post = Field(
    'post',
    _$post,
    opt: true,
  );

  @override
  final MappableFields<QuoteView> fields = const {
    #state: _f$state,
    #revealable: _f$revealable,
    #post: _f$post,
  };
  @override
  final bool ignoreNull = true;

  static QuoteView _instantiate(DecodingData data) {
    return QuoteView(
      state: data.dec(_f$state),
      revealable: data.dec(_f$revealable),
      post: data.dec(_f$post),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static QuoteView fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<QuoteView>(map);
  }

  static QuoteView fromJson(String json) {
    return ensureInitialized().decodeJson<QuoteView>(json);
  }
}

mixin QuoteViewMappable {
  String toJson() {
    return QuoteViewMapper.ensureInitialized().encodeJson<QuoteView>(
      this as QuoteView,
    );
  }

  Map<String, dynamic> toMap() {
    return QuoteViewMapper.ensureInitialized().encodeMap<QuoteView>(
      this as QuoteView,
    );
  }

  QuoteViewCopyWith<QuoteView, QuoteView, QuoteView> get copyWith =>
      _QuoteViewCopyWithImpl<QuoteView, QuoteView>(
        this as QuoteView,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return QuoteViewMapper.ensureInitialized().stringifyValue(
      this as QuoteView,
    );
  }

  @override
  bool operator ==(Object other) {
    return QuoteViewMapper.ensureInitialized().equalsValue(
      this as QuoteView,
      other,
    );
  }

  @override
  int get hashCode {
    return QuoteViewMapper.ensureInitialized().hashValue(this as QuoteView);
  }
}

extension QuoteViewValueCopy<$R, $Out> on ObjectCopyWith<$R, QuoteView, $Out> {
  QuoteViewCopyWith<$R, QuoteView, $Out> get $asQuoteView =>
      $base.as((v, t, t2) => _QuoteViewCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class QuoteViewCopyWith<$R, $In extends QuoteView, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  QuotePreviewPostCopyWith<$R, QuotePreviewPost, QuotePreviewPost>? get post;
  $R call({String? state, bool? revealable, QuotePreviewPost? post});
  QuoteViewCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _QuoteViewCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, QuoteView, $Out>
    implements QuoteViewCopyWith<$R, QuoteView, $Out> {
  _QuoteViewCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<QuoteView> $mapper =
      QuoteViewMapper.ensureInitialized();
  @override
  QuotePreviewPostCopyWith<$R, QuotePreviewPost, QuotePreviewPost>? get post =>
      $value.post?.copyWith.$chain((v) => call(post: v));
  @override
  $R call({String? state, Object? revealable = $none, Object? post = $none}) =>
      $apply(
        FieldCopyWithData({
          if (state != null) #state: state,
          if (revealable != $none) #revealable: revealable,
          if (post != $none) #post: post,
        }),
      );
  @override
  QuoteView $make(CopyWithData data) => QuoteView(
    state: data.get(#state, or: $value.state),
    revealable: data.get(#revealable, or: $value.revealable),
    post: data.get(#post, or: $value.post),
  );

  @override
  QuoteViewCopyWith<$R2, QuoteView, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _QuoteViewCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class QuotePreviewPostMapper extends ClassMapperBase<QuotePreviewPost> {
  QuotePreviewPostMapper._();

  static QuotePreviewPostMapper? _instance;
  static QuotePreviewPostMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = QuotePreviewPostMapper._());
      MapperContainer.globals.useAll([AtUriMapper(), CidMapper()]);
      PostAuthorMapper.ensureInitialized();
      PostImageMapper.ensureInitialized();
      ProjectMapper.ensureInitialized();
      ExternalImportMapper.ensureInitialized();
      PostExternalMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'QuotePreviewPost';

  static AtUri _$uri(QuotePreviewPost v) => v.uri;
  static dynamic _arg$uri(f) => f<AtUri>();
  static const Field<QuotePreviewPost, String> _f$uri = Field(
    'uri',
    _$uri,
    arg: _arg$uri,
  );
  static Cid _$cid(QuotePreviewPost v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<QuotePreviewPost, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static String _$text(QuotePreviewPost v) => v.text;
  static const Field<QuotePreviewPost, String> _f$text = Field('text', _$text);
  static PostAuthor _$author(QuotePreviewPost v) => v.author;
  static const Field<QuotePreviewPost, PostAuthor> _f$author = Field(
    'author',
    _$author,
  );
  static DateTime _$createdAt(QuotePreviewPost v) => v.createdAt;
  static const Field<QuotePreviewPost, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
  );
  static List<PostImage>? _$images(QuotePreviewPost v) => v.images;
  static const Field<QuotePreviewPost, List<PostImage>> _f$images = Field(
    'images',
    _$images,
    opt: true,
  );
  static Project? _$project(QuotePreviewPost v) => v.project;
  static const Field<QuotePreviewPost, Project> _f$project = Field(
    'project',
    _$project,
    opt: true,
  );
  static ExternalImport? _$externalImport(QuotePreviewPost v) =>
      v.externalImport;
  static const Field<QuotePreviewPost, ExternalImport> _f$externalImport =
      Field('externalImport', _$externalImport, opt: true);
  static PostExternal? _$external(QuotePreviewPost v) => v.external;
  static const Field<QuotePreviewPost, PostExternal> _f$external = Field(
    'external',
    _$external,
    opt: true,
  );

  @override
  final MappableFields<QuotePreviewPost> fields = const {
    #uri: _f$uri,
    #cid: _f$cid,
    #text: _f$text,
    #author: _f$author,
    #createdAt: _f$createdAt,
    #images: _f$images,
    #project: _f$project,
    #externalImport: _f$externalImport,
    #external: _f$external,
  };
  @override
  final bool ignoreNull = true;

  static QuotePreviewPost _instantiate(DecodingData data) {
    return QuotePreviewPost(
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      text: data.dec(_f$text),
      author: data.dec(_f$author),
      createdAt: data.dec(_f$createdAt),
      images: data.dec(_f$images),
      project: data.dec(_f$project),
      externalImport: data.dec(_f$externalImport),
      external: data.dec(_f$external),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static QuotePreviewPost fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<QuotePreviewPost>(map);
  }

  static QuotePreviewPost fromJson(String json) {
    return ensureInitialized().decodeJson<QuotePreviewPost>(json);
  }
}

mixin QuotePreviewPostMappable {
  String toJson() {
    return QuotePreviewPostMapper.ensureInitialized()
        .encodeJson<QuotePreviewPost>(this as QuotePreviewPost);
  }

  Map<String, dynamic> toMap() {
    return QuotePreviewPostMapper.ensureInitialized()
        .encodeMap<QuotePreviewPost>(this as QuotePreviewPost);
  }

  QuotePreviewPostCopyWith<QuotePreviewPost, QuotePreviewPost, QuotePreviewPost>
  get copyWith =>
      _QuotePreviewPostCopyWithImpl<QuotePreviewPost, QuotePreviewPost>(
        this as QuotePreviewPost,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return QuotePreviewPostMapper.ensureInitialized().stringifyValue(
      this as QuotePreviewPost,
    );
  }

  @override
  bool operator ==(Object other) {
    return QuotePreviewPostMapper.ensureInitialized().equalsValue(
      this as QuotePreviewPost,
      other,
    );
  }

  @override
  int get hashCode {
    return QuotePreviewPostMapper.ensureInitialized().hashValue(
      this as QuotePreviewPost,
    );
  }
}

extension QuotePreviewPostValueCopy<$R, $Out>
    on ObjectCopyWith<$R, QuotePreviewPost, $Out> {
  QuotePreviewPostCopyWith<$R, QuotePreviewPost, $Out>
  get $asQuotePreviewPost =>
      $base.as((v, t, t2) => _QuotePreviewPostCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class QuotePreviewPostCopyWith<$R, $In extends QuotePreviewPost, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostAuthorCopyWith<$R, PostAuthor, PostAuthor> get author;
  ListCopyWith<$R, PostImage, PostImageCopyWith<$R, PostImage, PostImage>>?
  get images;
  ProjectCopyWith<$R, Project, Project>? get project;
  ExternalImportCopyWith<$R, ExternalImport, ExternalImport>?
  get externalImport;
  PostExternalCopyWith<$R, PostExternal, PostExternal>? get external;
  $R call({
    String? uri,
    String? cid,
    String? text,
    PostAuthor? author,
    DateTime? createdAt,
    List<PostImage>? images,
    Project? project,
    ExternalImport? externalImport,
    PostExternal? external,
  });
  QuotePreviewPostCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _QuotePreviewPostCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, QuotePreviewPost, $Out>
    implements QuotePreviewPostCopyWith<$R, QuotePreviewPost, $Out> {
  _QuotePreviewPostCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<QuotePreviewPost> $mapper =
      QuotePreviewPostMapper.ensureInitialized();
  @override
  PostAuthorCopyWith<$R, PostAuthor, PostAuthor> get author =>
      $value.author.copyWith.$chain((v) => call(author: v));
  @override
  ListCopyWith<$R, PostImage, PostImageCopyWith<$R, PostImage, PostImage>>?
  get images => $value.images != null
      ? ListCopyWith(
          $value.images!,
          (v, t) => v.copyWith.$chain(t),
          (v) => call(images: v),
        )
      : null;
  @override
  ProjectCopyWith<$R, Project, Project>? get project =>
      $value.project?.copyWith.$chain((v) => call(project: v));
  @override
  ExternalImportCopyWith<$R, ExternalImport, ExternalImport>?
  get externalImport =>
      $value.externalImport?.copyWith.$chain((v) => call(externalImport: v));
  @override
  PostExternalCopyWith<$R, PostExternal, PostExternal>? get external =>
      $value.external?.copyWith.$chain((v) => call(external: v));
  @override
  $R call({
    String? uri,
    String? cid,
    String? text,
    PostAuthor? author,
    DateTime? createdAt,
    Object? images = $none,
    Object? project = $none,
    Object? externalImport = $none,
    Object? external = $none,
  }) => $apply(
    FieldCopyWithData({
      if (uri != null) #uri: uri,
      if (cid != null) #cid: cid,
      if (text != null) #text: text,
      if (author != null) #author: author,
      if (createdAt != null) #createdAt: createdAt,
      if (images != $none) #images: images,
      if (project != $none) #project: project,
      if (externalImport != $none) #externalImport: externalImport,
      if (external != $none) #external: external,
    }),
  );
  @override
  QuotePreviewPost $make(CopyWithData data) => QuotePreviewPost(
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    text: data.get(#text, or: $value.text),
    author: data.get(#author, or: $value.author),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    images: data.get(#images, or: $value.images),
    project: data.get(#project, or: $value.project),
    externalImport: data.get(#externalImport, or: $value.externalImport),
    external: data.get(#external, or: $value.external),
  );

  @override
  QuotePreviewPostCopyWith<$R2, QuotePreviewPost, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _QuotePreviewPostCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ExternalImportMapper extends ClassMapperBase<ExternalImport> {
  ExternalImportMapper._();

  static ExternalImportMapper? _instance;
  static ExternalImportMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ExternalImportMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ExternalImport';

  static String _$source(ExternalImport v) => v.source;
  static const Field<ExternalImport, String> _f$source = Field(
    'source',
    _$source,
  );

  @override
  final MappableFields<ExternalImport> fields = const {#source: _f$source};

  static ExternalImport _instantiate(DecodingData data) {
    return ExternalImport(source: data.dec(_f$source));
  }

  @override
  final Function instantiate = _instantiate;

  static ExternalImport fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ExternalImport>(map);
  }

  static ExternalImport fromJson(String json) {
    return ensureInitialized().decodeJson<ExternalImport>(json);
  }
}

mixin ExternalImportMappable {
  String toJson() {
    return ExternalImportMapper.ensureInitialized().encodeJson<ExternalImport>(
      this as ExternalImport,
    );
  }

  Map<String, dynamic> toMap() {
    return ExternalImportMapper.ensureInitialized().encodeMap<ExternalImport>(
      this as ExternalImport,
    );
  }

  ExternalImportCopyWith<ExternalImport, ExternalImport, ExternalImport>
  get copyWith => _ExternalImportCopyWithImpl<ExternalImport, ExternalImport>(
    this as ExternalImport,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return ExternalImportMapper.ensureInitialized().stringifyValue(
      this as ExternalImport,
    );
  }

  @override
  bool operator ==(Object other) {
    return ExternalImportMapper.ensureInitialized().equalsValue(
      this as ExternalImport,
      other,
    );
  }

  @override
  int get hashCode {
    return ExternalImportMapper.ensureInitialized().hashValue(
      this as ExternalImport,
    );
  }
}

extension ExternalImportValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ExternalImport, $Out> {
  ExternalImportCopyWith<$R, ExternalImport, $Out> get $asExternalImport =>
      $base.as((v, t, t2) => _ExternalImportCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ExternalImportCopyWith<$R, $In extends ExternalImport, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? source});
  ExternalImportCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ExternalImportCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ExternalImport, $Out>
    implements ExternalImportCopyWith<$R, ExternalImport, $Out> {
  _ExternalImportCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ExternalImport> $mapper =
      ExternalImportMapper.ensureInitialized();
  @override
  $R call({String? source}) =>
      $apply(FieldCopyWithData({if (source != null) #source: source}));
  @override
  ExternalImport $make(CopyWithData data) =>
      ExternalImport(source: data.get(#source, or: $value.source));

  @override
  ExternalImportCopyWith<$R2, ExternalImport, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ExternalImportCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostExternalMapper extends ClassMapperBase<PostExternal> {
  PostExternalMapper._();

  static PostExternalMapper? _instance;
  static PostExternalMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostExternalMapper._());
      MapperContainer.globals.useAll([CidMapper()]);
      PostExternalThumbMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostExternal';

  static String _$uri(PostExternal v) => v.uri;
  static const Field<PostExternal, String> _f$uri = Field('uri', _$uri);
  static String _$title(PostExternal v) => v.title;
  static const Field<PostExternal, String> _f$title = Field('title', _$title);
  static String _$description(PostExternal v) => v.description;
  static const Field<PostExternal, String> _f$description = Field(
    'description',
    _$description,
  );
  static PostExternalThumb? _$thumb(PostExternal v) => v.thumb;
  static const Field<PostExternal, PostExternalThumb> _f$thumb = Field(
    'thumb',
    _$thumb,
    opt: true,
  );

  @override
  final MappableFields<PostExternal> fields = const {
    #uri: _f$uri,
    #title: _f$title,
    #description: _f$description,
    #thumb: _f$thumb,
  };
  @override
  final bool ignoreNull = true;

  static PostExternal _instantiate(DecodingData data) {
    return PostExternal(
      uri: data.dec(_f$uri),
      title: data.dec(_f$title),
      description: data.dec(_f$description),
      thumb: data.dec(_f$thumb),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostExternal fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostExternal>(map);
  }

  static PostExternal fromJson(String json) {
    return ensureInitialized().decodeJson<PostExternal>(json);
  }
}

mixin PostExternalMappable {
  String toJson() {
    return PostExternalMapper.ensureInitialized().encodeJson<PostExternal>(
      this as PostExternal,
    );
  }

  Map<String, dynamic> toMap() {
    return PostExternalMapper.ensureInitialized().encodeMap<PostExternal>(
      this as PostExternal,
    );
  }

  PostExternalCopyWith<PostExternal, PostExternal, PostExternal> get copyWith =>
      _PostExternalCopyWithImpl<PostExternal, PostExternal>(
        this as PostExternal,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostExternalMapper.ensureInitialized().stringifyValue(
      this as PostExternal,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostExternalMapper.ensureInitialized().equalsValue(
      this as PostExternal,
      other,
    );
  }

  @override
  int get hashCode {
    return PostExternalMapper.ensureInitialized().hashValue(
      this as PostExternal,
    );
  }
}

extension PostExternalValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostExternal, $Out> {
  PostExternalCopyWith<$R, PostExternal, $Out> get $asPostExternal =>
      $base.as((v, t, t2) => _PostExternalCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PostExternalCopyWith<$R, $In extends PostExternal, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostExternalThumbCopyWith<$R, PostExternalThumb, PostExternalThumb>?
  get thumb;
  $R call({
    String? uri,
    String? title,
    String? description,
    PostExternalThumb? thumb,
  });
  PostExternalCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PostExternalCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostExternal, $Out>
    implements PostExternalCopyWith<$R, PostExternal, $Out> {
  _PostExternalCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostExternal> $mapper =
      PostExternalMapper.ensureInitialized();
  @override
  PostExternalThumbCopyWith<$R, PostExternalThumb, PostExternalThumb>?
  get thumb => $value.thumb?.copyWith.$chain((v) => call(thumb: v));
  @override
  $R call({
    String? uri,
    String? title,
    String? description,
    Object? thumb = $none,
  }) => $apply(
    FieldCopyWithData({
      if (uri != null) #uri: uri,
      if (title != null) #title: title,
      if (description != null) #description: description,
      if (thumb != $none) #thumb: thumb,
    }),
  );
  @override
  PostExternal $make(CopyWithData data) => PostExternal(
    uri: data.get(#uri, or: $value.uri),
    title: data.get(#title, or: $value.title),
    description: data.get(#description, or: $value.description),
    thumb: data.get(#thumb, or: $value.thumb),
  );

  @override
  PostExternalCopyWith<$R2, PostExternal, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostExternalCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class PostExternalThumbMapper extends ClassMapperBase<PostExternalThumb> {
  PostExternalThumbMapper._();

  static PostExternalThumbMapper? _instance;
  static PostExternalThumbMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostExternalThumbMapper._());
      MapperContainer.globals.useAll([CidMapper()]);
    }
    return _instance!;
  }

  @override
  final String id = 'PostExternalThumb';

  static Cid _$cid(PostExternalThumb v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<PostExternalThumb, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static String _$mime(PostExternalThumb v) => v.mime;
  static const Field<PostExternalThumb, String> _f$mime = Field('mime', _$mime);
  static int _$size(PostExternalThumb v) => v.size;
  static const Field<PostExternalThumb, int> _f$size = Field('size', _$size);
  static String _$url(PostExternalThumb v) => v.url;
  static const Field<PostExternalThumb, String> _f$url = Field('url', _$url);

  @override
  final MappableFields<PostExternalThumb> fields = const {
    #cid: _f$cid,
    #mime: _f$mime,
    #size: _f$size,
    #url: _f$url,
  };

  static PostExternalThumb _instantiate(DecodingData data) {
    return PostExternalThumb(
      cid: data.dec(_f$cid),
      mime: data.dec(_f$mime),
      size: data.dec(_f$size),
      url: data.dec(_f$url),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostExternalThumb fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostExternalThumb>(map);
  }

  static PostExternalThumb fromJson(String json) {
    return ensureInitialized().decodeJson<PostExternalThumb>(json);
  }
}

mixin PostExternalThumbMappable {
  String toJson() {
    return PostExternalThumbMapper.ensureInitialized()
        .encodeJson<PostExternalThumb>(this as PostExternalThumb);
  }

  Map<String, dynamic> toMap() {
    return PostExternalThumbMapper.ensureInitialized()
        .encodeMap<PostExternalThumb>(this as PostExternalThumb);
  }

  PostExternalThumbCopyWith<
    PostExternalThumb,
    PostExternalThumb,
    PostExternalThumb
  >
  get copyWith =>
      _PostExternalThumbCopyWithImpl<PostExternalThumb, PostExternalThumb>(
        this as PostExternalThumb,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostExternalThumbMapper.ensureInitialized().stringifyValue(
      this as PostExternalThumb,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostExternalThumbMapper.ensureInitialized().equalsValue(
      this as PostExternalThumb,
      other,
    );
  }

  @override
  int get hashCode {
    return PostExternalThumbMapper.ensureInitialized().hashValue(
      this as PostExternalThumb,
    );
  }
}

extension PostExternalThumbValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostExternalThumb, $Out> {
  PostExternalThumbCopyWith<$R, PostExternalThumb, $Out>
  get $asPostExternalThumb => $base.as(
    (v, t, t2) => _PostExternalThumbCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class PostExternalThumbCopyWith<
  $R,
  $In extends PostExternalThumb,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? cid, String? mime, int? size, String? url});
  PostExternalThumbCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PostExternalThumbCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostExternalThumb, $Out>
    implements PostExternalThumbCopyWith<$R, PostExternalThumb, $Out> {
  _PostExternalThumbCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostExternalThumb> $mapper =
      PostExternalThumbMapper.ensureInitialized();
  @override
  $R call({String? cid, String? mime, int? size, String? url}) => $apply(
    FieldCopyWithData({
      if (cid != null) #cid: cid,
      if (mime != null) #mime: mime,
      if (size != null) #size: size,
      if (url != null) #url: url,
    }),
  );
  @override
  PostExternalThumb $make(CopyWithData data) => PostExternalThumb(
    cid: data.get(#cid, or: $value.cid),
    mime: data.get(#mime, or: $value.mime),
    size: data.get(#size, or: $value.size),
    url: data.get(#url, or: $value.url),
  );

  @override
  PostExternalThumbCopyWith<$R2, PostExternalThumb, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostExternalThumbCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ContentRelationshipMapper extends ClassMapperBase<ContentRelationship> {
  ContentRelationshipMapper._();

  static ContentRelationshipMapper? _instance;
  static ContentRelationshipMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ContentRelationshipMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ContentRelationship';

  static String _$state(ContentRelationship v) => v.state;
  static const Field<ContentRelationship, String> _f$state = Field(
    'state',
    _$state,
  );
  static bool _$revealable(ContentRelationship v) => v.revealable;
  static const Field<ContentRelationship, bool> _f$revealable = Field(
    'revealable',
    _$revealable,
  );

  @override
  final MappableFields<ContentRelationship> fields = const {
    #state: _f$state,
    #revealable: _f$revealable,
  };

  static ContentRelationship _instantiate(DecodingData data) {
    return ContentRelationship(
      state: data.dec(_f$state),
      revealable: data.dec(_f$revealable),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ContentRelationship fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ContentRelationship>(map);
  }

  static ContentRelationship fromJson(String json) {
    return ensureInitialized().decodeJson<ContentRelationship>(json);
  }
}

mixin ContentRelationshipMappable {
  String toJson() {
    return ContentRelationshipMapper.ensureInitialized()
        .encodeJson<ContentRelationship>(this as ContentRelationship);
  }

  Map<String, dynamic> toMap() {
    return ContentRelationshipMapper.ensureInitialized()
        .encodeMap<ContentRelationship>(this as ContentRelationship);
  }

  ContentRelationshipCopyWith<
    ContentRelationship,
    ContentRelationship,
    ContentRelationship
  >
  get copyWith =>
      _ContentRelationshipCopyWithImpl<
        ContentRelationship,
        ContentRelationship
      >(this as ContentRelationship, $identity, $identity);
  @override
  String toString() {
    return ContentRelationshipMapper.ensureInitialized().stringifyValue(
      this as ContentRelationship,
    );
  }

  @override
  bool operator ==(Object other) {
    return ContentRelationshipMapper.ensureInitialized().equalsValue(
      this as ContentRelationship,
      other,
    );
  }

  @override
  int get hashCode {
    return ContentRelationshipMapper.ensureInitialized().hashValue(
      this as ContentRelationship,
    );
  }
}

extension ContentRelationshipValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ContentRelationship, $Out> {
  ContentRelationshipCopyWith<$R, ContentRelationship, $Out>
  get $asContentRelationship => $base.as(
    (v, t, t2) => _ContentRelationshipCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ContentRelationshipCopyWith<
  $R,
  $In extends ContentRelationship,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? state, bool? revealable});
  ContentRelationshipCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ContentRelationshipCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ContentRelationship, $Out>
    implements ContentRelationshipCopyWith<$R, ContentRelationship, $Out> {
  _ContentRelationshipCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ContentRelationship> $mapper =
      ContentRelationshipMapper.ensureInitialized();
  @override
  $R call({String? state, bool? revealable}) => $apply(
    FieldCopyWithData({
      if (state != null) #state: state,
      if (revealable != null) #revealable: revealable,
    }),
  );
  @override
  ContentRelationship $make(CopyWithData data) => ContentRelationship(
    state: data.get(#state, or: $value.state),
    revealable: data.get(#revealable, or: $value.revealable),
  );

  @override
  ContentRelationshipCopyWith<$R2, ContentRelationship, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ContentRelationshipCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

