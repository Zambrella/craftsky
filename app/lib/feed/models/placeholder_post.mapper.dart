// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'placeholder_post.dart';

class PlaceholderPostMapper extends ClassMapperBase<PlaceholderPost> {
  PlaceholderPostMapper._();

  static PlaceholderPostMapper? _instance;
  static PlaceholderPostMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PlaceholderPostMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'PlaceholderPost';

  static String _$id(PlaceholderPost v) => v.id;
  static const Field<PlaceholderPost, String> _f$id = Field('id', _$id);
  static String _$authorHandle(PlaceholderPost v) => v.authorHandle;
  static const Field<PlaceholderPost, String> _f$authorHandle = Field(
    'authorHandle',
    _$authorHandle,
  );
  static String _$authorDisplayName(PlaceholderPost v) => v.authorDisplayName;
  static const Field<PlaceholderPost, String> _f$authorDisplayName = Field(
    'authorDisplayName',
    _$authorDisplayName,
  );
  static String _$body(PlaceholderPost v) => v.body;
  static const Field<PlaceholderPost, String> _f$body = Field('body', _$body);
  static DateTime _$postedAt(PlaceholderPost v) => v.postedAt;
  static const Field<PlaceholderPost, DateTime> _f$postedAt = Field(
    'postedAt',
    _$postedAt,
  );
  static int _$replyCount(PlaceholderPost v) => v.replyCount;
  static const Field<PlaceholderPost, int> _f$replyCount = Field(
    'replyCount',
    _$replyCount,
  );
  static int _$repostCount(PlaceholderPost v) => v.repostCount;
  static const Field<PlaceholderPost, int> _f$repostCount = Field(
    'repostCount',
    _$repostCount,
  );
  static int _$likeCount(PlaceholderPost v) => v.likeCount;
  static const Field<PlaceholderPost, int> _f$likeCount = Field(
    'likeCount',
    _$likeCount,
  );
  static String? _$craftLabel(PlaceholderPost v) => v.craftLabel;
  static const Field<PlaceholderPost, String> _f$craftLabel = Field(
    'craftLabel',
    _$craftLabel,
    opt: true,
  );

  @override
  final MappableFields<PlaceholderPost> fields = const {
    #id: _f$id,
    #authorHandle: _f$authorHandle,
    #authorDisplayName: _f$authorDisplayName,
    #body: _f$body,
    #postedAt: _f$postedAt,
    #replyCount: _f$replyCount,
    #repostCount: _f$repostCount,
    #likeCount: _f$likeCount,
    #craftLabel: _f$craftLabel,
  };

  static PlaceholderPost _instantiate(DecodingData data) {
    return PlaceholderPost(
      id: data.dec(_f$id),
      authorHandle: data.dec(_f$authorHandle),
      authorDisplayName: data.dec(_f$authorDisplayName),
      body: data.dec(_f$body),
      postedAt: data.dec(_f$postedAt),
      replyCount: data.dec(_f$replyCount),
      repostCount: data.dec(_f$repostCount),
      likeCount: data.dec(_f$likeCount),
      craftLabel: data.dec(_f$craftLabel),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PlaceholderPost fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PlaceholderPost>(map);
  }

  static PlaceholderPost fromJson(String json) {
    return ensureInitialized().decodeJson<PlaceholderPost>(json);
  }
}

mixin PlaceholderPostMappable {
  String toJson() {
    return PlaceholderPostMapper.ensureInitialized()
        .encodeJson<PlaceholderPost>(this as PlaceholderPost);
  }

  Map<String, dynamic> toMap() {
    return PlaceholderPostMapper.ensureInitialized().encodeMap<PlaceholderPost>(
      this as PlaceholderPost,
    );
  }

  PlaceholderPostCopyWith<PlaceholderPost, PlaceholderPost, PlaceholderPost>
  get copyWith =>
      _PlaceholderPostCopyWithImpl<PlaceholderPost, PlaceholderPost>(
        this as PlaceholderPost,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PlaceholderPostMapper.ensureInitialized().stringifyValue(
      this as PlaceholderPost,
    );
  }

  @override
  bool operator ==(Object other) {
    return PlaceholderPostMapper.ensureInitialized().equalsValue(
      this as PlaceholderPost,
      other,
    );
  }

  @override
  int get hashCode {
    return PlaceholderPostMapper.ensureInitialized().hashValue(
      this as PlaceholderPost,
    );
  }
}

extension PlaceholderPostValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PlaceholderPost, $Out> {
  PlaceholderPostCopyWith<$R, PlaceholderPost, $Out> get $asPlaceholderPost =>
      $base.as((v, t, t2) => _PlaceholderPostCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PlaceholderPostCopyWith<$R, $In extends PlaceholderPost, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? id,
    String? authorHandle,
    String? authorDisplayName,
    String? body,
    DateTime? postedAt,
    int? replyCount,
    int? repostCount,
    int? likeCount,
    String? craftLabel,
  });
  PlaceholderPostCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _PlaceholderPostCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PlaceholderPost, $Out>
    implements PlaceholderPostCopyWith<$R, PlaceholderPost, $Out> {
  _PlaceholderPostCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PlaceholderPost> $mapper =
      PlaceholderPostMapper.ensureInitialized();
  @override
  $R call({
    String? id,
    String? authorHandle,
    String? authorDisplayName,
    String? body,
    DateTime? postedAt,
    int? replyCount,
    int? repostCount,
    int? likeCount,
    Object? craftLabel = $none,
  }) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (authorHandle != null) #authorHandle: authorHandle,
      if (authorDisplayName != null) #authorDisplayName: authorDisplayName,
      if (body != null) #body: body,
      if (postedAt != null) #postedAt: postedAt,
      if (replyCount != null) #replyCount: replyCount,
      if (repostCount != null) #repostCount: repostCount,
      if (likeCount != null) #likeCount: likeCount,
      if (craftLabel != $none) #craftLabel: craftLabel,
    }),
  );
  @override
  PlaceholderPost $make(CopyWithData data) => PlaceholderPost(
    id: data.get(#id, or: $value.id),
    authorHandle: data.get(#authorHandle, or: $value.authorHandle),
    authorDisplayName: data.get(
      #authorDisplayName,
      or: $value.authorDisplayName,
    ),
    body: data.get(#body, or: $value.body),
    postedAt: data.get(#postedAt, or: $value.postedAt),
    replyCount: data.get(#replyCount, or: $value.replyCount),
    repostCount: data.get(#repostCount, or: $value.repostCount),
    likeCount: data.get(#likeCount, or: $value.likeCount),
    craftLabel: data.get(#craftLabel, or: $value.craftLabel),
  );

  @override
  PlaceholderPostCopyWith<$R2, PlaceholderPost, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PlaceholderPostCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

