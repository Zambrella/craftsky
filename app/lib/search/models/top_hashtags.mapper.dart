// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'top_hashtags.dart';

class TopHashtagsResponseMapper extends ClassMapperBase<TopHashtagsResponse> {
  TopHashtagsResponseMapper._();

  static TopHashtagsResponseMapper? _instance;
  static TopHashtagsResponseMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TopHashtagsResponseMapper._());
      TopHashtagGroupMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'TopHashtagsResponse';

  static List<TopHashtagGroup> _$groups(TopHashtagsResponse v) => v.groups;
  static const Field<TopHashtagsResponse, List<TopHashtagGroup>> _f$groups =
      Field('groups', _$groups);

  @override
  final MappableFields<TopHashtagsResponse> fields = const {#groups: _f$groups};

  static TopHashtagsResponse _instantiate(DecodingData data) {
    return TopHashtagsResponse(groups: data.dec(_f$groups));
  }

  @override
  final Function instantiate = _instantiate;

  static TopHashtagsResponse fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TopHashtagsResponse>(map);
  }

  static TopHashtagsResponse fromJson(String json) {
    return ensureInitialized().decodeJson<TopHashtagsResponse>(json);
  }
}

mixin TopHashtagsResponseMappable {
  String toJson() {
    return TopHashtagsResponseMapper.ensureInitialized()
        .encodeJson<TopHashtagsResponse>(this as TopHashtagsResponse);
  }

  Map<String, dynamic> toMap() {
    return TopHashtagsResponseMapper.ensureInitialized()
        .encodeMap<TopHashtagsResponse>(this as TopHashtagsResponse);
  }

  TopHashtagsResponseCopyWith<
    TopHashtagsResponse,
    TopHashtagsResponse,
    TopHashtagsResponse
  >
  get copyWith =>
      _TopHashtagsResponseCopyWithImpl<
        TopHashtagsResponse,
        TopHashtagsResponse
      >(this as TopHashtagsResponse, $identity, $identity);
  @override
  String toString() {
    return TopHashtagsResponseMapper.ensureInitialized().stringifyValue(
      this as TopHashtagsResponse,
    );
  }

  @override
  bool operator ==(Object other) {
    return TopHashtagsResponseMapper.ensureInitialized().equalsValue(
      this as TopHashtagsResponse,
      other,
    );
  }

  @override
  int get hashCode {
    return TopHashtagsResponseMapper.ensureInitialized().hashValue(
      this as TopHashtagsResponse,
    );
  }
}

extension TopHashtagsResponseValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TopHashtagsResponse, $Out> {
  TopHashtagsResponseCopyWith<$R, TopHashtagsResponse, $Out>
  get $asTopHashtagsResponse => $base.as(
    (v, t, t2) => _TopHashtagsResponseCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class TopHashtagsResponseCopyWith<
  $R,
  $In extends TopHashtagsResponse,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    TopHashtagGroup,
    TopHashtagGroupCopyWith<$R, TopHashtagGroup, TopHashtagGroup>
  >
  get groups;
  $R call({List<TopHashtagGroup>? groups});
  TopHashtagsResponseCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _TopHashtagsResponseCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TopHashtagsResponse, $Out>
    implements TopHashtagsResponseCopyWith<$R, TopHashtagsResponse, $Out> {
  _TopHashtagsResponseCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TopHashtagsResponse> $mapper =
      TopHashtagsResponseMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    TopHashtagGroup,
    TopHashtagGroupCopyWith<$R, TopHashtagGroup, TopHashtagGroup>
  >
  get groups => ListCopyWith(
    $value.groups,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(groups: v),
  );
  @override
  $R call({List<TopHashtagGroup>? groups}) =>
      $apply(FieldCopyWithData({if (groups != null) #groups: groups}));
  @override
  TopHashtagsResponse $make(CopyWithData data) =>
      TopHashtagsResponse(groups: data.get(#groups, or: $value.groups));

  @override
  TopHashtagsResponseCopyWith<$R2, TopHashtagsResponse, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _TopHashtagsResponseCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class TopHashtagGroupMapper extends ClassMapperBase<TopHashtagGroup> {
  TopHashtagGroupMapper._();

  static TopHashtagGroupMapper? _instance;
  static TopHashtagGroupMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TopHashtagGroupMapper._());
      TopHashtagItemMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'TopHashtagGroup';

  static String _$craftType(TopHashtagGroup v) => v.craftType;
  static const Field<TopHashtagGroup, String> _f$craftType = Field(
    'craftType',
    _$craftType,
  );
  static List<TopHashtagItem> _$items(TopHashtagGroup v) => v.items;
  static const Field<TopHashtagGroup, List<TopHashtagItem>> _f$items = Field(
    'items',
    _$items,
  );

  @override
  final MappableFields<TopHashtagGroup> fields = const {
    #craftType: _f$craftType,
    #items: _f$items,
  };

  static TopHashtagGroup _instantiate(DecodingData data) {
    return TopHashtagGroup(
      craftType: data.dec(_f$craftType),
      items: data.dec(_f$items),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static TopHashtagGroup fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TopHashtagGroup>(map);
  }

  static TopHashtagGroup fromJson(String json) {
    return ensureInitialized().decodeJson<TopHashtagGroup>(json);
  }
}

mixin TopHashtagGroupMappable {
  String toJson() {
    return TopHashtagGroupMapper.ensureInitialized()
        .encodeJson<TopHashtagGroup>(this as TopHashtagGroup);
  }

  Map<String, dynamic> toMap() {
    return TopHashtagGroupMapper.ensureInitialized().encodeMap<TopHashtagGroup>(
      this as TopHashtagGroup,
    );
  }

  TopHashtagGroupCopyWith<TopHashtagGroup, TopHashtagGroup, TopHashtagGroup>
  get copyWith =>
      _TopHashtagGroupCopyWithImpl<TopHashtagGroup, TopHashtagGroup>(
        this as TopHashtagGroup,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return TopHashtagGroupMapper.ensureInitialized().stringifyValue(
      this as TopHashtagGroup,
    );
  }

  @override
  bool operator ==(Object other) {
    return TopHashtagGroupMapper.ensureInitialized().equalsValue(
      this as TopHashtagGroup,
      other,
    );
  }

  @override
  int get hashCode {
    return TopHashtagGroupMapper.ensureInitialized().hashValue(
      this as TopHashtagGroup,
    );
  }
}

extension TopHashtagGroupValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TopHashtagGroup, $Out> {
  TopHashtagGroupCopyWith<$R, TopHashtagGroup, $Out> get $asTopHashtagGroup =>
      $base.as((v, t, t2) => _TopHashtagGroupCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class TopHashtagGroupCopyWith<$R, $In extends TopHashtagGroup, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    TopHashtagItem,
    TopHashtagItemCopyWith<$R, TopHashtagItem, TopHashtagItem>
  >
  get items;
  $R call({String? craftType, List<TopHashtagItem>? items});
  TopHashtagGroupCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _TopHashtagGroupCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TopHashtagGroup, $Out>
    implements TopHashtagGroupCopyWith<$R, TopHashtagGroup, $Out> {
  _TopHashtagGroupCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TopHashtagGroup> $mapper =
      TopHashtagGroupMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    TopHashtagItem,
    TopHashtagItemCopyWith<$R, TopHashtagItem, TopHashtagItem>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({String? craftType, List<TopHashtagItem>? items}) => $apply(
    FieldCopyWithData({
      if (craftType != null) #craftType: craftType,
      if (items != null) #items: items,
    }),
  );
  @override
  TopHashtagGroup $make(CopyWithData data) => TopHashtagGroup(
    craftType: data.get(#craftType, or: $value.craftType),
    items: data.get(#items, or: $value.items),
  );

  @override
  TopHashtagGroupCopyWith<$R2, TopHashtagGroup, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _TopHashtagGroupCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class TopHashtagItemMapper extends ClassMapperBase<TopHashtagItem> {
  TopHashtagItemMapper._();

  static TopHashtagItemMapper? _instance;
  static TopHashtagItemMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = TopHashtagItemMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'TopHashtagItem';

  static String _$tag(TopHashtagItem v) => v.tag;
  static const Field<TopHashtagItem, String> _f$tag = Field('tag', _$tag);
  static int _$count(TopHashtagItem v) => v.count;
  static const Field<TopHashtagItem, int> _f$count = Field('count', _$count);

  @override
  final MappableFields<TopHashtagItem> fields = const {
    #tag: _f$tag,
    #count: _f$count,
  };

  static TopHashtagItem _instantiate(DecodingData data) {
    return TopHashtagItem(tag: data.dec(_f$tag), count: data.dec(_f$count));
  }

  @override
  final Function instantiate = _instantiate;

  static TopHashtagItem fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<TopHashtagItem>(map);
  }

  static TopHashtagItem fromJson(String json) {
    return ensureInitialized().decodeJson<TopHashtagItem>(json);
  }
}

mixin TopHashtagItemMappable {
  String toJson() {
    return TopHashtagItemMapper.ensureInitialized().encodeJson<TopHashtagItem>(
      this as TopHashtagItem,
    );
  }

  Map<String, dynamic> toMap() {
    return TopHashtagItemMapper.ensureInitialized().encodeMap<TopHashtagItem>(
      this as TopHashtagItem,
    );
  }

  TopHashtagItemCopyWith<TopHashtagItem, TopHashtagItem, TopHashtagItem>
  get copyWith => _TopHashtagItemCopyWithImpl<TopHashtagItem, TopHashtagItem>(
    this as TopHashtagItem,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return TopHashtagItemMapper.ensureInitialized().stringifyValue(
      this as TopHashtagItem,
    );
  }

  @override
  bool operator ==(Object other) {
    return TopHashtagItemMapper.ensureInitialized().equalsValue(
      this as TopHashtagItem,
      other,
    );
  }

  @override
  int get hashCode {
    return TopHashtagItemMapper.ensureInitialized().hashValue(
      this as TopHashtagItem,
    );
  }
}

extension TopHashtagItemValueCopy<$R, $Out>
    on ObjectCopyWith<$R, TopHashtagItem, $Out> {
  TopHashtagItemCopyWith<$R, TopHashtagItem, $Out> get $asTopHashtagItem =>
      $base.as((v, t, t2) => _TopHashtagItemCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class TopHashtagItemCopyWith<$R, $In extends TopHashtagItem, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? tag, int? count});
  TopHashtagItemCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _TopHashtagItemCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, TopHashtagItem, $Out>
    implements TopHashtagItemCopyWith<$R, TopHashtagItem, $Out> {
  _TopHashtagItemCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<TopHashtagItem> $mapper =
      TopHashtagItemMapper.ensureInitialized();
  @override
  $R call({String? tag, int? count}) => $apply(
    FieldCopyWithData({
      if (tag != null) #tag: tag,
      if (count != null) #count: count,
    }),
  );
  @override
  TopHashtagItem $make(CopyWithData data) => TopHashtagItem(
    tag: data.get(#tag, or: $value.tag),
    count: data.get(#count, or: $value.count),
  );

  @override
  TopHashtagItemCopyWith<$R2, TopHashtagItem, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _TopHashtagItemCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

