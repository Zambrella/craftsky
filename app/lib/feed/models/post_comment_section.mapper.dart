// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'post_comment_section.dart';

class CommentSortMapper extends EnumMapper<CommentSort> {
  CommentSortMapper._();

  static CommentSortMapper? _instance;
  static CommentSortMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CommentSortMapper._());
    }
    return _instance!;
  }

  static CommentSort fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  CommentSort decode(dynamic value) {
    switch (value) {
      case r'oldest':
        return CommentSort.oldest;
      case r'newest':
        return CommentSort.newest;
      case r'follows':
        return CommentSort.follows;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(CommentSort self) {
    switch (self) {
      case CommentSort.oldest:
        return r'oldest';
      case CommentSort.newest:
        return r'newest';
      case CommentSort.follows:
        return r'follows';
    }
  }
}

extension CommentSortMapperExtension on CommentSort {
  String toValue() {
    CommentSortMapper.ensureInitialized();
    return MapperContainer.globals.toValue<CommentSort>(this) as String;
  }
}

class CommentPlacementMapper extends EnumMapper<CommentPlacement> {
  CommentPlacementMapper._();

  static CommentPlacementMapper? _instance;
  static CommentPlacementMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CommentPlacementMapper._());
    }
    return _instance!;
  }

  static CommentPlacement fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  CommentPlacement decode(dynamic value) {
    switch (value) {
      case r'focused':
        return CommentPlacement.focused;
      case r'viewerAuthored':
        return CommentPlacement.viewerAuthored;
      case r'normal':
        return CommentPlacement.normal;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(CommentPlacement self) {
    switch (self) {
      case CommentPlacement.focused:
        return r'focused';
      case CommentPlacement.viewerAuthored:
        return r'viewerAuthored';
      case CommentPlacement.normal:
        return r'normal';
    }
  }
}

extension CommentPlacementMapperExtension on CommentPlacement {
  String toValue() {
    CommentPlacementMapper.ensureInitialized();
    return MapperContainer.globals.toValue<CommentPlacement>(this) as String;
  }
}

class FocusStatusMapper extends EnumMapper<FocusStatus> {
  FocusStatusMapper._();

  static FocusStatusMapper? _instance;
  static FocusStatusMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = FocusStatusMapper._());
    }
    return _instance!;
  }

  static FocusStatus fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  FocusStatus decode(dynamic value) {
    switch (value) {
      case r'included':
        return FocusStatus.included;
      case r'notFound':
        return FocusStatus.notFound;
      case r'mismatchedRoot':
        return FocusStatus.mismatchedRoot;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(FocusStatus self) {
    switch (self) {
      case FocusStatus.included:
        return r'included';
      case FocusStatus.notFound:
        return r'notFound';
      case FocusStatus.mismatchedRoot:
        return r'mismatchedRoot';
    }
  }
}

extension FocusStatusMapperExtension on FocusStatus {
  String toValue() {
    FocusStatusMapper.ensureInitialized();
    return MapperContainer.globals.toValue<FocusStatus>(this) as String;
  }
}

class FocusKindMapper extends EnumMapper<FocusKind> {
  FocusKindMapper._();

  static FocusKindMapper? _instance;
  static FocusKindMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = FocusKindMapper._());
    }
    return _instance!;
  }

  static FocusKind fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  FocusKind decode(dynamic value) {
    switch (value) {
      case r'comment':
        return FocusKind.comment;
      case r'reply':
        return FocusKind.reply;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(FocusKind self) {
    switch (self) {
      case FocusKind.comment:
        return r'comment';
      case FocusKind.reply:
        return r'reply';
    }
  }
}

extension FocusKindMapperExtension on FocusKind {
  String toValue() {
    FocusKindMapper.ensureInitialized();
    return MapperContainer.globals.toValue<FocusKind>(this) as String;
  }
}

class PostCommentSectionMapper extends ClassMapperBase<PostCommentSection> {
  PostCommentSectionMapper._();

  static PostCommentSectionMapper? _instance;
  static PostCommentSectionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PostCommentSectionMapper._());
      PostMapper.ensureInitialized();
      CommentPageMapper.ensureInitialized();
      CommentSortMapper.ensureInitialized();
      FocusContextMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'PostCommentSection';

  static Post _$post(PostCommentSection v) => v.post;
  static const Field<PostCommentSection, Post> _f$post = Field('post', _$post);
  static CommentPage _$comments(PostCommentSection v) => v.comments;
  static const Field<PostCommentSection, CommentPage> _f$comments = Field(
    'comments',
    _$comments,
  );
  static CommentSort _$sort(PostCommentSection v) => v.sort;
  static const Field<PostCommentSection, CommentSort> _f$sort = Field(
    'sort',
    _$sort,
  );
  static FocusContext? _$focus(PostCommentSection v) => v.focus;
  static const Field<PostCommentSection, FocusContext> _f$focus = Field(
    'focus',
    _$focus,
    opt: true,
  );

  @override
  final MappableFields<PostCommentSection> fields = const {
    #post: _f$post,
    #comments: _f$comments,
    #sort: _f$sort,
    #focus: _f$focus,
  };

  static PostCommentSection _instantiate(DecodingData data) {
    return PostCommentSection(
      post: data.dec(_f$post),
      comments: data.dec(_f$comments),
      sort: data.dec(_f$sort),
      focus: data.dec(_f$focus),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PostCommentSection fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PostCommentSection>(map);
  }

  static PostCommentSection fromJson(String json) {
    return ensureInitialized().decodeJson<PostCommentSection>(json);
  }
}

mixin PostCommentSectionMappable {
  String toJson() {
    return PostCommentSectionMapper.ensureInitialized()
        .encodeJson<PostCommentSection>(this as PostCommentSection);
  }

  Map<String, dynamic> toMap() {
    return PostCommentSectionMapper.ensureInitialized()
        .encodeMap<PostCommentSection>(this as PostCommentSection);
  }

  PostCommentSectionCopyWith<
    PostCommentSection,
    PostCommentSection,
    PostCommentSection
  >
  get copyWith =>
      _PostCommentSectionCopyWithImpl<PostCommentSection, PostCommentSection>(
        this as PostCommentSection,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PostCommentSectionMapper.ensureInitialized().stringifyValue(
      this as PostCommentSection,
    );
  }

  @override
  bool operator ==(Object other) {
    return PostCommentSectionMapper.ensureInitialized().equalsValue(
      this as PostCommentSection,
      other,
    );
  }

  @override
  int get hashCode {
    return PostCommentSectionMapper.ensureInitialized().hashValue(
      this as PostCommentSection,
    );
  }
}

extension PostCommentSectionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PostCommentSection, $Out> {
  PostCommentSectionCopyWith<$R, PostCommentSection, $Out>
  get $asPostCommentSection => $base.as(
    (v, t, t2) => _PostCommentSectionCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class PostCommentSectionCopyWith<
  $R,
  $In extends PostCommentSection,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  PostCopyWith<$R, Post, Post> get post;
  CommentPageCopyWith<$R, CommentPage, CommentPage> get comments;
  FocusContextCopyWith<$R, FocusContext, FocusContext>? get focus;
  $R call({
    Post? post,
    CommentPage? comments,
    CommentSort? sort,
    FocusContext? focus,
  });
  PostCommentSectionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PostCommentSectionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PostCommentSection, $Out>
    implements PostCommentSectionCopyWith<$R, PostCommentSection, $Out> {
  _PostCommentSectionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PostCommentSection> $mapper =
      PostCommentSectionMapper.ensureInitialized();
  @override
  PostCopyWith<$R, Post, Post> get post =>
      $value.post.copyWith.$chain((v) => call(post: v));
  @override
  CommentPageCopyWith<$R, CommentPage, CommentPage> get comments =>
      $value.comments.copyWith.$chain((v) => call(comments: v));
  @override
  FocusContextCopyWith<$R, FocusContext, FocusContext>? get focus =>
      $value.focus?.copyWith.$chain((v) => call(focus: v));
  @override
  $R call({
    Post? post,
    CommentPage? comments,
    CommentSort? sort,
    Object? focus = $none,
  }) => $apply(
    FieldCopyWithData({
      if (post != null) #post: post,
      if (comments != null) #comments: comments,
      if (sort != null) #sort: sort,
      if (focus != $none) #focus: focus,
    }),
  );
  @override
  PostCommentSection $make(CopyWithData data) => PostCommentSection(
    post: data.get(#post, or: $value.post),
    comments: data.get(#comments, or: $value.comments),
    sort: data.get(#sort, or: $value.sort),
    focus: data.get(#focus, or: $value.focus),
  );

  @override
  PostCommentSectionCopyWith<$R2, PostCommentSection, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PostCommentSectionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class CommentPageMapper extends ClassMapperBase<CommentPage> {
  CommentPageMapper._();

  static CommentPageMapper? _instance;
  static CommentPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CommentPageMapper._());
      CommentItemMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'CommentPage';

  static List<CommentItem> _$items(CommentPage v) => v.items;
  static const Field<CommentPage, List<CommentItem>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(CommentPage v) => v.cursor;
  static const Field<CommentPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<CommentPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };

  static CommentPage _instantiate(DecodingData data) {
    return CommentPage(items: data.dec(_f$items), cursor: data.dec(_f$cursor));
  }

  @override
  final Function instantiate = _instantiate;

  static CommentPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<CommentPage>(map);
  }

  static CommentPage fromJson(String json) {
    return ensureInitialized().decodeJson<CommentPage>(json);
  }
}

mixin CommentPageMappable {
  String toJson() {
    return CommentPageMapper.ensureInitialized().encodeJson<CommentPage>(
      this as CommentPage,
    );
  }

  Map<String, dynamic> toMap() {
    return CommentPageMapper.ensureInitialized().encodeMap<CommentPage>(
      this as CommentPage,
    );
  }

  CommentPageCopyWith<CommentPage, CommentPage, CommentPage> get copyWith =>
      _CommentPageCopyWithImpl<CommentPage, CommentPage>(
        this as CommentPage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return CommentPageMapper.ensureInitialized().stringifyValue(
      this as CommentPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return CommentPageMapper.ensureInitialized().equalsValue(
      this as CommentPage,
      other,
    );
  }

  @override
  int get hashCode {
    return CommentPageMapper.ensureInitialized().hashValue(this as CommentPage);
  }
}

extension CommentPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, CommentPage, $Out> {
  CommentPageCopyWith<$R, CommentPage, $Out> get $asCommentPage =>
      $base.as((v, t, t2) => _CommentPageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class CommentPageCopyWith<$R, $In extends CommentPage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    CommentItem,
    CommentItemCopyWith<$R, CommentItem, CommentItem>
  >
  get items;
  $R call({List<CommentItem>? items, String? cursor});
  CommentPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _CommentPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, CommentPage, $Out>
    implements CommentPageCopyWith<$R, CommentPage, $Out> {
  _CommentPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<CommentPage> $mapper =
      CommentPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    CommentItem,
    CommentItemCopyWith<$R, CommentItem, CommentItem>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<CommentItem>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  CommentPage $make(CopyWithData data) => CommentPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  CommentPageCopyWith<$R2, CommentPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _CommentPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class CommentItemMapper extends ClassMapperBase<CommentItem> {
  CommentItemMapper._();

  static CommentItemMapper? _instance;
  static CommentItemMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = CommentItemMapper._());
      PostMapper.ensureInitialized();
      CommentPlacementMapper.ensureInitialized();
      ReplyPageMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'CommentItem';

  static Post _$post(CommentItem v) => v.post;
  static const Field<CommentItem, Post> _f$post = Field('post', _$post);
  static CommentPlacement _$placement(CommentItem v) => v.placement;
  static const Field<CommentItem, CommentPlacement> _f$placement = Field(
    'placement',
    _$placement,
  );
  static ReplyPage _$replies(CommentItem v) => v.replies;
  static const Field<CommentItem, ReplyPage> _f$replies = Field(
    'replies',
    _$replies,
  );

  @override
  final MappableFields<CommentItem> fields = const {
    #post: _f$post,
    #placement: _f$placement,
    #replies: _f$replies,
  };

  static CommentItem _instantiate(DecodingData data) {
    return CommentItem(
      post: data.dec(_f$post),
      placement: data.dec(_f$placement),
      replies: data.dec(_f$replies),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static CommentItem fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<CommentItem>(map);
  }

  static CommentItem fromJson(String json) {
    return ensureInitialized().decodeJson<CommentItem>(json);
  }
}

mixin CommentItemMappable {
  String toJson() {
    return CommentItemMapper.ensureInitialized().encodeJson<CommentItem>(
      this as CommentItem,
    );
  }

  Map<String, dynamic> toMap() {
    return CommentItemMapper.ensureInitialized().encodeMap<CommentItem>(
      this as CommentItem,
    );
  }

  CommentItemCopyWith<CommentItem, CommentItem, CommentItem> get copyWith =>
      _CommentItemCopyWithImpl<CommentItem, CommentItem>(
        this as CommentItem,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return CommentItemMapper.ensureInitialized().stringifyValue(
      this as CommentItem,
    );
  }

  @override
  bool operator ==(Object other) {
    return CommentItemMapper.ensureInitialized().equalsValue(
      this as CommentItem,
      other,
    );
  }

  @override
  int get hashCode {
    return CommentItemMapper.ensureInitialized().hashValue(this as CommentItem);
  }
}

extension CommentItemValueCopy<$R, $Out>
    on ObjectCopyWith<$R, CommentItem, $Out> {
  CommentItemCopyWith<$R, CommentItem, $Out> get $asCommentItem =>
      $base.as((v, t, t2) => _CommentItemCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class CommentItemCopyWith<$R, $In extends CommentItem, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostCopyWith<$R, Post, Post> get post;
  ReplyPageCopyWith<$R, ReplyPage, ReplyPage> get replies;
  $R call({Post? post, CommentPlacement? placement, ReplyPage? replies});
  CommentItemCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _CommentItemCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, CommentItem, $Out>
    implements CommentItemCopyWith<$R, CommentItem, $Out> {
  _CommentItemCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<CommentItem> $mapper =
      CommentItemMapper.ensureInitialized();
  @override
  PostCopyWith<$R, Post, Post> get post =>
      $value.post.copyWith.$chain((v) => call(post: v));
  @override
  ReplyPageCopyWith<$R, ReplyPage, ReplyPage> get replies =>
      $value.replies.copyWith.$chain((v) => call(replies: v));
  @override
  $R call({Post? post, CommentPlacement? placement, ReplyPage? replies}) =>
      $apply(
        FieldCopyWithData({
          if (post != null) #post: post,
          if (placement != null) #placement: placement,
          if (replies != null) #replies: replies,
        }),
      );
  @override
  CommentItem $make(CopyWithData data) => CommentItem(
    post: data.get(#post, or: $value.post),
    placement: data.get(#placement, or: $value.placement),
    replies: data.get(#replies, or: $value.replies),
  );

  @override
  CommentItemCopyWith<$R2, CommentItem, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _CommentItemCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ReplyPageMapper extends ClassMapperBase<ReplyPage> {
  ReplyPageMapper._();

  static ReplyPageMapper? _instance;
  static ReplyPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ReplyPageMapper._());
      ReplyItemMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ReplyPage';

  static bool _$loaded(ReplyPage v) => v.loaded;
  static const Field<ReplyPage, bool> _f$loaded = Field('loaded', _$loaded);
  static List<ReplyItem> _$items(ReplyPage v) => v.items;
  static const Field<ReplyPage, List<ReplyItem>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(ReplyPage v) => v.cursor;
  static const Field<ReplyPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<ReplyPage> fields = const {
    #loaded: _f$loaded,
    #items: _f$items,
    #cursor: _f$cursor,
  };

  static ReplyPage _instantiate(DecodingData data) {
    return ReplyPage(
      loaded: data.dec(_f$loaded),
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ReplyPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ReplyPage>(map);
  }

  static ReplyPage fromJson(String json) {
    return ensureInitialized().decodeJson<ReplyPage>(json);
  }
}

mixin ReplyPageMappable {
  String toJson() {
    return ReplyPageMapper.ensureInitialized().encodeJson<ReplyPage>(
      this as ReplyPage,
    );
  }

  Map<String, dynamic> toMap() {
    return ReplyPageMapper.ensureInitialized().encodeMap<ReplyPage>(
      this as ReplyPage,
    );
  }

  ReplyPageCopyWith<ReplyPage, ReplyPage, ReplyPage> get copyWith =>
      _ReplyPageCopyWithImpl<ReplyPage, ReplyPage>(
        this as ReplyPage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ReplyPageMapper.ensureInitialized().stringifyValue(
      this as ReplyPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return ReplyPageMapper.ensureInitialized().equalsValue(
      this as ReplyPage,
      other,
    );
  }

  @override
  int get hashCode {
    return ReplyPageMapper.ensureInitialized().hashValue(this as ReplyPage);
  }
}

extension ReplyPageValueCopy<$R, $Out> on ObjectCopyWith<$R, ReplyPage, $Out> {
  ReplyPageCopyWith<$R, ReplyPage, $Out> get $asReplyPage =>
      $base.as((v, t, t2) => _ReplyPageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ReplyPageCopyWith<$R, $In extends ReplyPage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, ReplyItem, ReplyItemCopyWith<$R, ReplyItem, ReplyItem>>
  get items;
  $R call({bool? loaded, List<ReplyItem>? items, String? cursor});
  ReplyPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ReplyPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ReplyPage, $Out>
    implements ReplyPageCopyWith<$R, ReplyPage, $Out> {
  _ReplyPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ReplyPage> $mapper =
      ReplyPageMapper.ensureInitialized();
  @override
  ListCopyWith<$R, ReplyItem, ReplyItemCopyWith<$R, ReplyItem, ReplyItem>>
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({bool? loaded, List<ReplyItem>? items, Object? cursor = $none}) =>
      $apply(
        FieldCopyWithData({
          if (loaded != null) #loaded: loaded,
          if (items != null) #items: items,
          if (cursor != $none) #cursor: cursor,
        }),
      );
  @override
  ReplyPage $make(CopyWithData data) => ReplyPage(
    loaded: data.get(#loaded, or: $value.loaded),
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  ReplyPageCopyWith<$R2, ReplyPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ReplyPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ReplyItemMapper extends ClassMapperBase<ReplyItem> {
  ReplyItemMapper._();

  static ReplyItemMapper? _instance;
  static ReplyItemMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ReplyItemMapper._());
      PostMapper.ensureInitialized();
      ReplyingToAuthorMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ReplyItem';

  static Post _$post(ReplyItem v) => v.post;
  static const Field<ReplyItem, Post> _f$post = Field('post', _$post);
  static bool _$flattened(ReplyItem v) => v.flattened;
  static const Field<ReplyItem, bool> _f$flattened = Field(
    'flattened',
    _$flattened,
  );
  static ReplyingToAuthor? _$replyingTo(ReplyItem v) => v.replyingTo;
  static const Field<ReplyItem, ReplyingToAuthor> _f$replyingTo = Field(
    'replyingTo',
    _$replyingTo,
    opt: true,
  );

  @override
  final MappableFields<ReplyItem> fields = const {
    #post: _f$post,
    #flattened: _f$flattened,
    #replyingTo: _f$replyingTo,
  };

  static ReplyItem _instantiate(DecodingData data) {
    return ReplyItem(
      post: data.dec(_f$post),
      flattened: data.dec(_f$flattened),
      replyingTo: data.dec(_f$replyingTo),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ReplyItem fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ReplyItem>(map);
  }

  static ReplyItem fromJson(String json) {
    return ensureInitialized().decodeJson<ReplyItem>(json);
  }
}

mixin ReplyItemMappable {
  String toJson() {
    return ReplyItemMapper.ensureInitialized().encodeJson<ReplyItem>(
      this as ReplyItem,
    );
  }

  Map<String, dynamic> toMap() {
    return ReplyItemMapper.ensureInitialized().encodeMap<ReplyItem>(
      this as ReplyItem,
    );
  }

  ReplyItemCopyWith<ReplyItem, ReplyItem, ReplyItem> get copyWith =>
      _ReplyItemCopyWithImpl<ReplyItem, ReplyItem>(
        this as ReplyItem,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ReplyItemMapper.ensureInitialized().stringifyValue(
      this as ReplyItem,
    );
  }

  @override
  bool operator ==(Object other) {
    return ReplyItemMapper.ensureInitialized().equalsValue(
      this as ReplyItem,
      other,
    );
  }

  @override
  int get hashCode {
    return ReplyItemMapper.ensureInitialized().hashValue(this as ReplyItem);
  }
}

extension ReplyItemValueCopy<$R, $Out> on ObjectCopyWith<$R, ReplyItem, $Out> {
  ReplyItemCopyWith<$R, ReplyItem, $Out> get $asReplyItem =>
      $base.as((v, t, t2) => _ReplyItemCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ReplyItemCopyWith<$R, $In extends ReplyItem, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostCopyWith<$R, Post, Post> get post;
  ReplyingToAuthorCopyWith<$R, ReplyingToAuthor, ReplyingToAuthor>?
  get replyingTo;
  $R call({Post? post, bool? flattened, ReplyingToAuthor? replyingTo});
  ReplyItemCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ReplyItemCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ReplyItem, $Out>
    implements ReplyItemCopyWith<$R, ReplyItem, $Out> {
  _ReplyItemCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ReplyItem> $mapper =
      ReplyItemMapper.ensureInitialized();
  @override
  PostCopyWith<$R, Post, Post> get post =>
      $value.post.copyWith.$chain((v) => call(post: v));
  @override
  ReplyingToAuthorCopyWith<$R, ReplyingToAuthor, ReplyingToAuthor>?
  get replyingTo =>
      $value.replyingTo?.copyWith.$chain((v) => call(replyingTo: v));
  @override
  $R call({Post? post, bool? flattened, Object? replyingTo = $none}) => $apply(
    FieldCopyWithData({
      if (post != null) #post: post,
      if (flattened != null) #flattened: flattened,
      if (replyingTo != $none) #replyingTo: replyingTo,
    }),
  );
  @override
  ReplyItem $make(CopyWithData data) => ReplyItem(
    post: data.get(#post, or: $value.post),
    flattened: data.get(#flattened, or: $value.flattened),
    replyingTo: data.get(#replyingTo, or: $value.replyingTo),
  );

  @override
  ReplyItemCopyWith<$R2, ReplyItem, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ReplyItemCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ReplyingToAuthorMapper extends ClassMapperBase<ReplyingToAuthor> {
  ReplyingToAuthorMapper._();

  static ReplyingToAuthorMapper? _instance;
  static ReplyingToAuthorMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ReplyingToAuthorMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ReplyingToAuthor';

  static String _$uri(ReplyingToAuthor v) => v.uri;
  static const Field<ReplyingToAuthor, String> _f$uri = Field('uri', _$uri);
  static String _$did(ReplyingToAuthor v) => v.did;
  static const Field<ReplyingToAuthor, String> _f$did = Field('did', _$did);
  static String _$handle(ReplyingToAuthor v) => v.handle;
  static const Field<ReplyingToAuthor, String> _f$handle = Field(
    'handle',
    _$handle,
  );
  static String? _$displayName(ReplyingToAuthor v) => v.displayName;
  static const Field<ReplyingToAuthor, String> _f$displayName = Field(
    'displayName',
    _$displayName,
    opt: true,
  );

  @override
  final MappableFields<ReplyingToAuthor> fields = const {
    #uri: _f$uri,
    #did: _f$did,
    #handle: _f$handle,
    #displayName: _f$displayName,
  };

  static ReplyingToAuthor _instantiate(DecodingData data) {
    return ReplyingToAuthor(
      uri: data.dec(_f$uri),
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      displayName: data.dec(_f$displayName),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ReplyingToAuthor fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ReplyingToAuthor>(map);
  }

  static ReplyingToAuthor fromJson(String json) {
    return ensureInitialized().decodeJson<ReplyingToAuthor>(json);
  }
}

mixin ReplyingToAuthorMappable {
  String toJson() {
    return ReplyingToAuthorMapper.ensureInitialized()
        .encodeJson<ReplyingToAuthor>(this as ReplyingToAuthor);
  }

  Map<String, dynamic> toMap() {
    return ReplyingToAuthorMapper.ensureInitialized()
        .encodeMap<ReplyingToAuthor>(this as ReplyingToAuthor);
  }

  ReplyingToAuthorCopyWith<ReplyingToAuthor, ReplyingToAuthor, ReplyingToAuthor>
  get copyWith =>
      _ReplyingToAuthorCopyWithImpl<ReplyingToAuthor, ReplyingToAuthor>(
        this as ReplyingToAuthor,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ReplyingToAuthorMapper.ensureInitialized().stringifyValue(
      this as ReplyingToAuthor,
    );
  }

  @override
  bool operator ==(Object other) {
    return ReplyingToAuthorMapper.ensureInitialized().equalsValue(
      this as ReplyingToAuthor,
      other,
    );
  }

  @override
  int get hashCode {
    return ReplyingToAuthorMapper.ensureInitialized().hashValue(
      this as ReplyingToAuthor,
    );
  }
}

extension ReplyingToAuthorValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ReplyingToAuthor, $Out> {
  ReplyingToAuthorCopyWith<$R, ReplyingToAuthor, $Out>
  get $asReplyingToAuthor =>
      $base.as((v, t, t2) => _ReplyingToAuthorCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ReplyingToAuthorCopyWith<$R, $In extends ReplyingToAuthor, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? uri, String? did, String? handle, String? displayName});
  ReplyingToAuthorCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ReplyingToAuthorCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ReplyingToAuthor, $Out>
    implements ReplyingToAuthorCopyWith<$R, ReplyingToAuthor, $Out> {
  _ReplyingToAuthorCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ReplyingToAuthor> $mapper =
      ReplyingToAuthorMapper.ensureInitialized();
  @override
  $R call({
    String? uri,
    String? did,
    String? handle,
    Object? displayName = $none,
  }) => $apply(
    FieldCopyWithData({
      if (uri != null) #uri: uri,
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (displayName != $none) #displayName: displayName,
    }),
  );
  @override
  ReplyingToAuthor $make(CopyWithData data) => ReplyingToAuthor(
    uri: data.get(#uri, or: $value.uri),
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    displayName: data.get(#displayName, or: $value.displayName),
  );

  @override
  ReplyingToAuthorCopyWith<$R2, ReplyingToAuthor, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ReplyingToAuthorCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class FocusContextMapper extends ClassMapperBase<FocusContext> {
  FocusContextMapper._();

  static FocusContextMapper? _instance;
  static FocusContextMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = FocusContextMapper._());
      FocusStatusMapper.ensureInitialized();
      FocusKindMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'FocusContext';

  static String _$uri(FocusContext v) => v.uri;
  static const Field<FocusContext, String> _f$uri = Field('uri', _$uri);
  static FocusStatus _$status(FocusContext v) => v.status;
  static const Field<FocusContext, FocusStatus> _f$status = Field(
    'status',
    _$status,
  );
  static FocusKind? _$kind(FocusContext v) => v.kind;
  static const Field<FocusContext, FocusKind> _f$kind = Field(
    'kind',
    _$kind,
    opt: true,
  );
  static String? _$commentUri(FocusContext v) => v.commentUri;
  static const Field<FocusContext, String> _f$commentUri = Field(
    'commentUri',
    _$commentUri,
    opt: true,
  );

  @override
  final MappableFields<FocusContext> fields = const {
    #uri: _f$uri,
    #status: _f$status,
    #kind: _f$kind,
    #commentUri: _f$commentUri,
  };

  static FocusContext _instantiate(DecodingData data) {
    return FocusContext(
      uri: data.dec(_f$uri),
      status: data.dec(_f$status),
      kind: data.dec(_f$kind),
      commentUri: data.dec(_f$commentUri),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static FocusContext fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<FocusContext>(map);
  }

  static FocusContext fromJson(String json) {
    return ensureInitialized().decodeJson<FocusContext>(json);
  }
}

mixin FocusContextMappable {
  String toJson() {
    return FocusContextMapper.ensureInitialized().encodeJson<FocusContext>(
      this as FocusContext,
    );
  }

  Map<String, dynamic> toMap() {
    return FocusContextMapper.ensureInitialized().encodeMap<FocusContext>(
      this as FocusContext,
    );
  }

  FocusContextCopyWith<FocusContext, FocusContext, FocusContext> get copyWith =>
      _FocusContextCopyWithImpl<FocusContext, FocusContext>(
        this as FocusContext,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return FocusContextMapper.ensureInitialized().stringifyValue(
      this as FocusContext,
    );
  }

  @override
  bool operator ==(Object other) {
    return FocusContextMapper.ensureInitialized().equalsValue(
      this as FocusContext,
      other,
    );
  }

  @override
  int get hashCode {
    return FocusContextMapper.ensureInitialized().hashValue(
      this as FocusContext,
    );
  }
}

extension FocusContextValueCopy<$R, $Out>
    on ObjectCopyWith<$R, FocusContext, $Out> {
  FocusContextCopyWith<$R, FocusContext, $Out> get $asFocusContext =>
      $base.as((v, t, t2) => _FocusContextCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class FocusContextCopyWith<$R, $In extends FocusContext, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? uri,
    FocusStatus? status,
    FocusKind? kind,
    String? commentUri,
  });
  FocusContextCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _FocusContextCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, FocusContext, $Out>
    implements FocusContextCopyWith<$R, FocusContext, $Out> {
  _FocusContextCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<FocusContext> $mapper =
      FocusContextMapper.ensureInitialized();
  @override
  $R call({
    String? uri,
    FocusStatus? status,
    Object? kind = $none,
    Object? commentUri = $none,
  }) => $apply(
    FieldCopyWithData({
      if (uri != null) #uri: uri,
      if (status != null) #status: status,
      if (kind != $none) #kind: kind,
      if (commentUri != $none) #commentUri: commentUri,
    }),
  );
  @override
  FocusContext $make(CopyWithData data) => FocusContext(
    uri: data.get(#uri, or: $value.uri),
    status: data.get(#status, or: $value.status),
    kind: data.get(#kind, or: $value.kind),
    commentUri: data.get(#commentUri, or: $value.commentUri),
  );

  @override
  FocusContextCopyWith<$R2, FocusContext, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _FocusContextCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

