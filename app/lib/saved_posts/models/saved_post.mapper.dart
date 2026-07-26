// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'saved_post.dart';

class SavedPostStateMapper extends ClassMapperBase<SavedPostState> {
  SavedPostStateMapper._();

  static SavedPostStateMapper? _instance;
  static SavedPostStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostStateMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostState';

  static DateTime _$savedAt(SavedPostState v) => v.savedAt;
  static const Field<SavedPostState, DateTime> _f$savedAt = Field(
    'savedAt',
    _$savedAt,
  );
  static String? _$folderId(SavedPostState v) => v.folderId;
  static const Field<SavedPostState, String> _f$folderId = Field(
    'folderId',
    _$folderId,
    opt: true,
  );

  @override
  final MappableFields<SavedPostState> fields = const {
    #savedAt: _f$savedAt,
    #folderId: _f$folderId,
  };
  @override
  final bool ignoreNull = true;

  static SavedPostState _instantiate(DecodingData data) {
    return SavedPostState(
      savedAt: data.dec(_f$savedAt),
      folderId: data.dec(_f$folderId),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SavedPostState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SavedPostState>(map);
  }

  static SavedPostState fromJson(String json) {
    return ensureInitialized().decodeJson<SavedPostState>(json);
  }
}

mixin SavedPostStateMappable {
  String toJson() {
    return SavedPostStateMapper.ensureInitialized().encodeJson<SavedPostState>(
      this as SavedPostState,
    );
  }

  Map<String, dynamic> toMap() {
    return SavedPostStateMapper.ensureInitialized().encodeMap<SavedPostState>(
      this as SavedPostState,
    );
  }

  SavedPostStateCopyWith<SavedPostState, SavedPostState, SavedPostState>
  get copyWith => _SavedPostStateCopyWithImpl<SavedPostState, SavedPostState>(
    this as SavedPostState,
    $identity,
    $identity,
  );
  @override
  bool operator ==(Object other) {
    return SavedPostStateMapper.ensureInitialized().equalsValue(
      this as SavedPostState,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostStateMapper.ensureInitialized().hashValue(
      this as SavedPostState,
    );
  }
}

extension SavedPostStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostState, $Out> {
  SavedPostStateCopyWith<$R, SavedPostState, $Out> get $asSavedPostState =>
      $base.as((v, t, t2) => _SavedPostStateCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class SavedPostStateCopyWith<$R, $In extends SavedPostState, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({DateTime? savedAt, String? folderId});
  SavedPostStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SavedPostStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostState, $Out>
    implements SavedPostStateCopyWith<$R, SavedPostState, $Out> {
  _SavedPostStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostState> $mapper =
      SavedPostStateMapper.ensureInitialized();
  @override
  $R call({DateTime? savedAt, Object? folderId = $none}) => $apply(
    FieldCopyWithData({
      if (savedAt != null) #savedAt: savedAt,
      if (folderId != $none) #folderId: folderId,
    }),
  );
  @override
  SavedPostState $make(CopyWithData data) => SavedPostState(
    savedAt: data.get(#savedAt, or: $value.savedAt),
    folderId: data.get(#folderId, or: $value.folderId),
  );

  @override
  SavedPostStateCopyWith<$R2, SavedPostState, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavedPostStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SavedPostItemMapper extends ClassMapperBase<SavedPostItem> {
  SavedPostItemMapper._();

  static SavedPostItemMapper? _instance;
  static SavedPostItemMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostItemMapper._());
      PostMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostItem';

  static Post _$post(SavedPostItem v) => v.post;
  static const Field<SavedPostItem, Post> _f$post = Field('post', _$post);
  static DateTime _$savedAt(SavedPostItem v) => v.savedAt;
  static const Field<SavedPostItem, DateTime> _f$savedAt = Field(
    'savedAt',
    _$savedAt,
  );
  static String? _$folderId(SavedPostItem v) => v.folderId;
  static const Field<SavedPostItem, String> _f$folderId = Field(
    'folderId',
    _$folderId,
    opt: true,
  );

  @override
  final MappableFields<SavedPostItem> fields = const {
    #post: _f$post,
    #savedAt: _f$savedAt,
    #folderId: _f$folderId,
  };
  @override
  final bool ignoreNull = true;

  static SavedPostItem _instantiate(DecodingData data) {
    return SavedPostItem(
      post: data.dec(_f$post),
      savedAt: data.dec(_f$savedAt),
      folderId: data.dec(_f$folderId),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SavedPostItem fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SavedPostItem>(map);
  }

  static SavedPostItem fromJson(String json) {
    return ensureInitialized().decodeJson<SavedPostItem>(json);
  }
}

mixin SavedPostItemMappable {
  String toJson() {
    return SavedPostItemMapper.ensureInitialized().encodeJson<SavedPostItem>(
      this as SavedPostItem,
    );
  }

  Map<String, dynamic> toMap() {
    return SavedPostItemMapper.ensureInitialized().encodeMap<SavedPostItem>(
      this as SavedPostItem,
    );
  }

  SavedPostItemCopyWith<SavedPostItem, SavedPostItem, SavedPostItem>
  get copyWith => _SavedPostItemCopyWithImpl<SavedPostItem, SavedPostItem>(
    this as SavedPostItem,
    $identity,
    $identity,
  );
  @override
  bool operator ==(Object other) {
    return SavedPostItemMapper.ensureInitialized().equalsValue(
      this as SavedPostItem,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostItemMapper.ensureInitialized().hashValue(
      this as SavedPostItem,
    );
  }
}

extension SavedPostItemValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostItem, $Out> {
  SavedPostItemCopyWith<$R, SavedPostItem, $Out> get $asSavedPostItem =>
      $base.as((v, t, t2) => _SavedPostItemCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class SavedPostItemCopyWith<$R, $In extends SavedPostItem, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  PostCopyWith<$R, Post, Post> get post;
  $R call({Post? post, DateTime? savedAt, String? folderId});
  SavedPostItemCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _SavedPostItemCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostItem, $Out>
    implements SavedPostItemCopyWith<$R, SavedPostItem, $Out> {
  _SavedPostItemCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostItem> $mapper =
      SavedPostItemMapper.ensureInitialized();
  @override
  PostCopyWith<$R, Post, Post> get post =>
      $value.post.copyWith.$chain((v) => call(post: v));
  @override
  $R call({Post? post, DateTime? savedAt, Object? folderId = $none}) => $apply(
    FieldCopyWithData({
      if (post != null) #post: post,
      if (savedAt != null) #savedAt: savedAt,
      if (folderId != $none) #folderId: folderId,
    }),
  );
  @override
  SavedPostItem $make(CopyWithData data) => SavedPostItem(
    post: data.get(#post, or: $value.post),
    savedAt: data.get(#savedAt, or: $value.savedAt),
    folderId: data.get(#folderId, or: $value.folderId),
  );

  @override
  SavedPostItemCopyWith<$R2, SavedPostItem, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavedPostItemCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class SavedPostPageMapper extends ClassMapperBase<SavedPostPage> {
  SavedPostPageMapper._();

  static SavedPostPageMapper? _instance;
  static SavedPostPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SavedPostPageMapper._());
      SavedPostItemMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SavedPostPage';

  static List<SavedPostItem> _$items(SavedPostPage v) => v.items;
  static const Field<SavedPostPage, List<SavedPostItem>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(SavedPostPage v) => v.cursor;
  static const Field<SavedPostPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<SavedPostPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };
  @override
  final bool ignoreNull = true;

  static SavedPostPage _instantiate(DecodingData data) {
    return SavedPostPage(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SavedPostPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SavedPostPage>(map);
  }

  static SavedPostPage fromJson(String json) {
    return ensureInitialized().decodeJson<SavedPostPage>(json);
  }
}

mixin SavedPostPageMappable {
  String toJson() {
    return SavedPostPageMapper.ensureInitialized().encodeJson<SavedPostPage>(
      this as SavedPostPage,
    );
  }

  Map<String, dynamic> toMap() {
    return SavedPostPageMapper.ensureInitialized().encodeMap<SavedPostPage>(
      this as SavedPostPage,
    );
  }

  SavedPostPageCopyWith<SavedPostPage, SavedPostPage, SavedPostPage>
  get copyWith => _SavedPostPageCopyWithImpl<SavedPostPage, SavedPostPage>(
    this as SavedPostPage,
    $identity,
    $identity,
  );
  @override
  bool operator ==(Object other) {
    return SavedPostPageMapper.ensureInitialized().equalsValue(
      this as SavedPostPage,
      other,
    );
  }

  @override
  int get hashCode {
    return SavedPostPageMapper.ensureInitialized().hashValue(
      this as SavedPostPage,
    );
  }
}

extension SavedPostPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SavedPostPage, $Out> {
  SavedPostPageCopyWith<$R, SavedPostPage, $Out> get $asSavedPostPage =>
      $base.as((v, t, t2) => _SavedPostPageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class SavedPostPageCopyWith<$R, $In extends SavedPostPage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    SavedPostItem,
    SavedPostItemCopyWith<$R, SavedPostItem, SavedPostItem>
  >
  get items;
  $R call({List<SavedPostItem>? items, String? cursor});
  SavedPostPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _SavedPostPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SavedPostPage, $Out>
    implements SavedPostPageCopyWith<$R, SavedPostPage, $Out> {
  _SavedPostPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SavedPostPage> $mapper =
      SavedPostPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    SavedPostItem,
    SavedPostItemCopyWith<$R, SavedPostItem, SavedPostItem>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<SavedPostItem>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  SavedPostPage $make(CopyWithData data) => SavedPostPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  SavedPostPageCopyWith<$R2, SavedPostPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SavedPostPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

