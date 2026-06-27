// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'hashtag_search_page.dart';

class HashtagSearchPageMapper extends ClassMapperBase<HashtagSearchPage> {
  HashtagSearchPageMapper._();

  static HashtagSearchPageMapper? _instance;
  static HashtagSearchPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = HashtagSearchPageMapper._());
      HashtagSearchResultMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'HashtagSearchPage';

  static List<HashtagSearchResult> _$items(HashtagSearchPage v) => v.items;
  static const Field<HashtagSearchPage, List<HashtagSearchResult>> _f$items =
      Field('items', _$items);
  static String? _$cursor(HashtagSearchPage v) => v.cursor;
  static const Field<HashtagSearchPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<HashtagSearchPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };
  @override
  final bool ignoreNull = true;

  static HashtagSearchPage _instantiate(DecodingData data) {
    return HashtagSearchPage(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static HashtagSearchPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<HashtagSearchPage>(map);
  }

  static HashtagSearchPage fromJson(String json) {
    return ensureInitialized().decodeJson<HashtagSearchPage>(json);
  }
}

mixin HashtagSearchPageMappable {
  String toJson() {
    return HashtagSearchPageMapper.ensureInitialized()
        .encodeJson<HashtagSearchPage>(this as HashtagSearchPage);
  }

  Map<String, dynamic> toMap() {
    return HashtagSearchPageMapper.ensureInitialized()
        .encodeMap<HashtagSearchPage>(this as HashtagSearchPage);
  }

  HashtagSearchPageCopyWith<
    HashtagSearchPage,
    HashtagSearchPage,
    HashtagSearchPage
  >
  get copyWith =>
      _HashtagSearchPageCopyWithImpl<HashtagSearchPage, HashtagSearchPage>(
        this as HashtagSearchPage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return HashtagSearchPageMapper.ensureInitialized().stringifyValue(
      this as HashtagSearchPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return HashtagSearchPageMapper.ensureInitialized().equalsValue(
      this as HashtagSearchPage,
      other,
    );
  }

  @override
  int get hashCode {
    return HashtagSearchPageMapper.ensureInitialized().hashValue(
      this as HashtagSearchPage,
    );
  }
}

extension HashtagSearchPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, HashtagSearchPage, $Out> {
  HashtagSearchPageCopyWith<$R, HashtagSearchPage, $Out>
  get $asHashtagSearchPage => $base.as(
    (v, t, t2) => _HashtagSearchPageCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class HashtagSearchPageCopyWith<
  $R,
  $In extends HashtagSearchPage,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    HashtagSearchResult,
    HashtagSearchResultCopyWith<$R, HashtagSearchResult, HashtagSearchResult>
  >
  get items;
  $R call({List<HashtagSearchResult>? items, String? cursor});
  HashtagSearchPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _HashtagSearchPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, HashtagSearchPage, $Out>
    implements HashtagSearchPageCopyWith<$R, HashtagSearchPage, $Out> {
  _HashtagSearchPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<HashtagSearchPage> $mapper =
      HashtagSearchPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    HashtagSearchResult,
    HashtagSearchResultCopyWith<$R, HashtagSearchResult, HashtagSearchResult>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<HashtagSearchResult>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  HashtagSearchPage $make(CopyWithData data) => HashtagSearchPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  HashtagSearchPageCopyWith<$R2, HashtagSearchPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _HashtagSearchPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class HashtagSearchResultMapper extends ClassMapperBase<HashtagSearchResult> {
  HashtagSearchResultMapper._();

  static HashtagSearchResultMapper? _instance;
  static HashtagSearchResultMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = HashtagSearchResultMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'HashtagSearchResult';

  static String _$tag(HashtagSearchResult v) => v.tag;
  static const Field<HashtagSearchResult, String> _f$tag = Field('tag', _$tag);
  static int _$postsLast28Days(HashtagSearchResult v) => v.postsLast28Days;
  static const Field<HashtagSearchResult, int> _f$postsLast28Days = Field(
    'postsLast28Days',
    _$postsLast28Days,
  );

  @override
  final MappableFields<HashtagSearchResult> fields = const {
    #tag: _f$tag,
    #postsLast28Days: _f$postsLast28Days,
  };

  static HashtagSearchResult _instantiate(DecodingData data) {
    return HashtagSearchResult(
      tag: data.dec(_f$tag),
      postsLast28Days: data.dec(_f$postsLast28Days),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static HashtagSearchResult fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<HashtagSearchResult>(map);
  }

  static HashtagSearchResult fromJson(String json) {
    return ensureInitialized().decodeJson<HashtagSearchResult>(json);
  }
}

mixin HashtagSearchResultMappable {
  String toJson() {
    return HashtagSearchResultMapper.ensureInitialized()
        .encodeJson<HashtagSearchResult>(this as HashtagSearchResult);
  }

  Map<String, dynamic> toMap() {
    return HashtagSearchResultMapper.ensureInitialized()
        .encodeMap<HashtagSearchResult>(this as HashtagSearchResult);
  }

  HashtagSearchResultCopyWith<
    HashtagSearchResult,
    HashtagSearchResult,
    HashtagSearchResult
  >
  get copyWith =>
      _HashtagSearchResultCopyWithImpl<
        HashtagSearchResult,
        HashtagSearchResult
      >(this as HashtagSearchResult, $identity, $identity);
  @override
  String toString() {
    return HashtagSearchResultMapper.ensureInitialized().stringifyValue(
      this as HashtagSearchResult,
    );
  }

  @override
  bool operator ==(Object other) {
    return HashtagSearchResultMapper.ensureInitialized().equalsValue(
      this as HashtagSearchResult,
      other,
    );
  }

  @override
  int get hashCode {
    return HashtagSearchResultMapper.ensureInitialized().hashValue(
      this as HashtagSearchResult,
    );
  }
}

extension HashtagSearchResultValueCopy<$R, $Out>
    on ObjectCopyWith<$R, HashtagSearchResult, $Out> {
  HashtagSearchResultCopyWith<$R, HashtagSearchResult, $Out>
  get $asHashtagSearchResult => $base.as(
    (v, t, t2) => _HashtagSearchResultCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class HashtagSearchResultCopyWith<
  $R,
  $In extends HashtagSearchResult,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? tag, int? postsLast28Days});
  HashtagSearchResultCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _HashtagSearchResultCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, HashtagSearchResult, $Out>
    implements HashtagSearchResultCopyWith<$R, HashtagSearchResult, $Out> {
  _HashtagSearchResultCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<HashtagSearchResult> $mapper =
      HashtagSearchResultMapper.ensureInitialized();
  @override
  $R call({String? tag, int? postsLast28Days}) => $apply(
    FieldCopyWithData({
      if (tag != null) #tag: tag,
      if (postsLast28Days != null) #postsLast28Days: postsLast28Days,
    }),
  );
  @override
  HashtagSearchResult $make(CopyWithData data) => HashtagSearchResult(
    tag: data.get(#tag, or: $value.tag),
    postsLast28Days: data.get(#postsLast28Days, or: $value.postsLast28Days),
  );

  @override
  HashtagSearchResultCopyWith<$R2, HashtagSearchResult, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _HashtagSearchResultCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

