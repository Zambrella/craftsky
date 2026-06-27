// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'blank_search_data.dart';

class BlankSearchDataMapper extends ClassMapperBase<BlankSearchData> {
  BlankSearchDataMapper._();

  static BlankSearchDataMapper? _instance;
  static BlankSearchDataMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = BlankSearchDataMapper._());
      RecentSearchPageMapper.ensureInitialized();
      TopHashtagsResponseMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'BlankSearchData';

  static RecentSearchPage _$recentSearches(BlankSearchData v) =>
      v.recentSearches;
  static const Field<BlankSearchData, RecentSearchPage> _f$recentSearches =
      Field('recentSearches', _$recentSearches);
  static TopHashtagsResponse _$topHashtags(BlankSearchData v) => v.topHashtags;
  static const Field<BlankSearchData, TopHashtagsResponse> _f$topHashtags =
      Field('topHashtags', _$topHashtags);

  @override
  final MappableFields<BlankSearchData> fields = const {
    #recentSearches: _f$recentSearches,
    #topHashtags: _f$topHashtags,
  };

  static BlankSearchData _instantiate(DecodingData data) {
    return BlankSearchData(
      recentSearches: data.dec(_f$recentSearches),
      topHashtags: data.dec(_f$topHashtags),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static BlankSearchData fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<BlankSearchData>(map);
  }

  static BlankSearchData fromJson(String json) {
    return ensureInitialized().decodeJson<BlankSearchData>(json);
  }
}

mixin BlankSearchDataMappable {
  String toJson() {
    return BlankSearchDataMapper.ensureInitialized()
        .encodeJson<BlankSearchData>(this as BlankSearchData);
  }

  Map<String, dynamic> toMap() {
    return BlankSearchDataMapper.ensureInitialized().encodeMap<BlankSearchData>(
      this as BlankSearchData,
    );
  }

  BlankSearchDataCopyWith<BlankSearchData, BlankSearchData, BlankSearchData>
  get copyWith =>
      _BlankSearchDataCopyWithImpl<BlankSearchData, BlankSearchData>(
        this as BlankSearchData,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return BlankSearchDataMapper.ensureInitialized().stringifyValue(
      this as BlankSearchData,
    );
  }

  @override
  bool operator ==(Object other) {
    return BlankSearchDataMapper.ensureInitialized().equalsValue(
      this as BlankSearchData,
      other,
    );
  }

  @override
  int get hashCode {
    return BlankSearchDataMapper.ensureInitialized().hashValue(
      this as BlankSearchData,
    );
  }
}

extension BlankSearchDataValueCopy<$R, $Out>
    on ObjectCopyWith<$R, BlankSearchData, $Out> {
  BlankSearchDataCopyWith<$R, BlankSearchData, $Out> get $asBlankSearchData =>
      $base.as((v, t, t2) => _BlankSearchDataCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class BlankSearchDataCopyWith<$R, $In extends BlankSearchData, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  RecentSearchPageCopyWith<$R, RecentSearchPage, RecentSearchPage>
  get recentSearches;
  TopHashtagsResponseCopyWith<$R, TopHashtagsResponse, TopHashtagsResponse>
  get topHashtags;
  $R call({RecentSearchPage? recentSearches, TopHashtagsResponse? topHashtags});
  BlankSearchDataCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _BlankSearchDataCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, BlankSearchData, $Out>
    implements BlankSearchDataCopyWith<$R, BlankSearchData, $Out> {
  _BlankSearchDataCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<BlankSearchData> $mapper =
      BlankSearchDataMapper.ensureInitialized();
  @override
  RecentSearchPageCopyWith<$R, RecentSearchPage, RecentSearchPage>
  get recentSearches =>
      $value.recentSearches.copyWith.$chain((v) => call(recentSearches: v));
  @override
  TopHashtagsResponseCopyWith<$R, TopHashtagsResponse, TopHashtagsResponse>
  get topHashtags =>
      $value.topHashtags.copyWith.$chain((v) => call(topHashtags: v));
  @override
  $R call({
    RecentSearchPage? recentSearches,
    TopHashtagsResponse? topHashtags,
  }) => $apply(
    FieldCopyWithData({
      if (recentSearches != null) #recentSearches: recentSearches,
      if (topHashtags != null) #topHashtags: topHashtags,
    }),
  );
  @override
  BlankSearchData $make(CopyWithData data) => BlankSearchData(
    recentSearches: data.get(#recentSearches, or: $value.recentSearches),
    topHashtags: data.get(#topHashtags, or: $value.topHashtags),
  );

  @override
  BlankSearchDataCopyWith<$R2, BlankSearchData, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _BlankSearchDataCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

