// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'search_post_page.dart';

class SearchPostPageMapper extends ClassMapperBase<SearchPostPage> {
  SearchPostPageMapper._();

  static SearchPostPageMapper? _instance;
  static SearchPostPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SearchPostPageMapper._());
      PostMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SearchPostPage';

  static List<Post> _$items(SearchPostPage v) => v.items;
  static const Field<SearchPostPage, List<Post>> _f$items = Field(
    'items',
    _$items,
  );
  static String? _$cursor(SearchPostPage v) => v.cursor;
  static const Field<SearchPostPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );
  static String? _$hashtag(SearchPostPage v) => v.hashtag;
  static const Field<SearchPostPage, String> _f$hashtag = Field(
    'hashtag',
    _$hashtag,
    opt: true,
  );

  @override
  final MappableFields<SearchPostPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
    #hashtag: _f$hashtag,
  };
  @override
  final bool ignoreNull = true;

  static SearchPostPage _instantiate(DecodingData data) {
    return SearchPostPage(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
      hashtag: data.dec(_f$hashtag),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SearchPostPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SearchPostPage>(map);
  }

  static SearchPostPage fromJson(String json) {
    return ensureInitialized().decodeJson<SearchPostPage>(json);
  }
}

mixin SearchPostPageMappable {
  String toJson() {
    return SearchPostPageMapper.ensureInitialized().encodeJson<SearchPostPage>(
      this as SearchPostPage,
    );
  }

  Map<String, dynamic> toMap() {
    return SearchPostPageMapper.ensureInitialized().encodeMap<SearchPostPage>(
      this as SearchPostPage,
    );
  }

  SearchPostPageCopyWith<SearchPostPage, SearchPostPage, SearchPostPage>
  get copyWith => _SearchPostPageCopyWithImpl<SearchPostPage, SearchPostPage>(
    this as SearchPostPage,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return SearchPostPageMapper.ensureInitialized().stringifyValue(
      this as SearchPostPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return SearchPostPageMapper.ensureInitialized().equalsValue(
      this as SearchPostPage,
      other,
    );
  }

  @override
  int get hashCode {
    return SearchPostPageMapper.ensureInitialized().hashValue(
      this as SearchPostPage,
    );
  }
}

extension SearchPostPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SearchPostPage, $Out> {
  SearchPostPageCopyWith<$R, SearchPostPage, $Out> get $asSearchPostPage =>
      $base.as((v, t, t2) => _SearchPostPageCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class SearchPostPageCopyWith<$R, $In extends SearchPostPage, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items;
  $R call({List<Post>? items, String? cursor, String? hashtag});
  SearchPostPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SearchPostPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SearchPostPage, $Out>
    implements SearchPostPageCopyWith<$R, SearchPostPage, $Out> {
  _SearchPostPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SearchPostPage> $mapper =
      SearchPostPageMapper.ensureInitialized();
  @override
  ListCopyWith<$R, Post, PostCopyWith<$R, Post, Post>> get items =>
      ListCopyWith(
        $value.items,
        (v, t) => v.copyWith.$chain(t),
        (v) => call(items: v),
      );
  @override
  $R call({
    List<Post>? items,
    Object? cursor = $none,
    Object? hashtag = $none,
  }) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
      if (hashtag != $none) #hashtag: hashtag,
    }),
  );
  @override
  SearchPostPage $make(CopyWithData data) => SearchPostPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
    hashtag: data.get(#hashtag, or: $value.hashtag),
  );

  @override
  SearchPostPageCopyWith<$R2, SearchPostPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SearchPostPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

