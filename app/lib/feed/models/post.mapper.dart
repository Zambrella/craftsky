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
      PostReplyMapper.ensureInitialized();
      PostRefMapper.ensureInitialized();
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
  static bool _$viewerHasReplied(Post v) => v.viewerHasReplied;
  static const Field<Post, bool> _f$viewerHasReplied = Field(
    'viewerHasReplied',
    _$viewerHasReplied,
    opt: true,
    def: false,
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
    #viewerHasReplied: _f$viewerHasReplied,
    #facets: _f$facets,
    #reply: _f$reply,
    #quote: _f$quote,
  };
  @override
  final bool ignoreNull = true;

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
      viewerHasReplied: data.dec(_f$viewerHasReplied),
      facets: data.dec(_f$facets),
      reply: data.dec(_f$reply),
      quote: data.dec(_f$quote),
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
  ListCopyWith<
    $R,
    Map<String, dynamic>,
    ObjectCopyWith<$R, Map<String, dynamic>, Map<String, dynamic>>
  >?
  get facets;
  PostReplyCopyWith<$R, PostReply, PostReply>? get reply;
  PostRefCopyWith<$R, PostRef, PostRef>? get quote;
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
    bool? viewerHasReplied,
    List<Map<String, dynamic>>? facets,
    PostReply? reply,
    PostRef? quote,
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
    bool? viewerHasReplied,
    Object? facets = $none,
    Object? reply = $none,
    Object? quote = $none,
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
      if (viewerHasReplied != null) #viewerHasReplied: viewerHasReplied,
      if (facets != $none) #facets: facets,
      if (reply != $none) #reply: reply,
      if (quote != $none) #quote: quote,
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
    viewerHasReplied: data.get(#viewerHasReplied, or: $value.viewerHasReplied),
    facets: data.get(#facets, or: $value.facets),
    reply: data.get(#reply, or: $value.reply),
    quote: data.get(#quote, or: $value.quote),
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
  static Cid? _$avatarCid(PostAuthor v) => v.avatarCid;
  static dynamic _arg$avatarCid(f) => f<Cid>();
  static const Field<PostAuthor, String> _f$avatarCid = Field(
    'avatarCid',
    _$avatarCid,
    opt: true,
    arg: _arg$avatarCid,
  );

  @override
  final MappableFields<PostAuthor> fields = const {
    #did: _f$did,
    #handle: _f$handle,
    #displayName: _f$displayName,
    #avatarCid: _f$avatarCid,
  };
  @override
  final bool ignoreNull = true;

  static PostAuthor _instantiate(DecodingData data) {
    return PostAuthor(
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      displayName: data.dec(_f$displayName),
      avatarCid: data.dec(_f$avatarCid),
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
    String? avatarCid,
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
    Object? avatarCid = $none,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (displayName != $none) #displayName: displayName,
      if (avatarCid != $none) #avatarCid: avatarCid,
    }),
  );
  @override
  PostAuthor $make(CopyWithData data) => PostAuthor(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    displayName: data.get(#displayName, or: $value.displayName),
    avatarCid: data.get(#avatarCid, or: $value.avatarCid),
  );

  @override
  PostAuthorCopyWith<$R2, PostAuthor, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostAuthorCopyWithImpl<$R2, $Out2>($value, $cast, t);
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

