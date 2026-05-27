// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'profile_account_page.dart';

class ProfileAccountPageMapper extends ClassMapperBase<ProfileAccountPage> {
  ProfileAccountPageMapper._();

  static ProfileAccountPageMapper? _instance;
  static ProfileAccountPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileAccountPageMapper._());
      ProfileAccountSummaryMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileAccountPage';

  static List<ProfileAccountSummary> _$items(ProfileAccountPage v) => v.items;
  static const Field<ProfileAccountPage, List<ProfileAccountSummary>> _f$items =
      Field('items', _$items);
  static int _$totalCount(ProfileAccountPage v) => v.totalCount;
  static const Field<ProfileAccountPage, int> _f$totalCount = Field(
    'totalCount',
    _$totalCount,
  );
  static String? _$cursor(ProfileAccountPage v) => v.cursor;
  static const Field<ProfileAccountPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<ProfileAccountPage> fields = const {
    #items: _f$items,
    #totalCount: _f$totalCount,
    #cursor: _f$cursor,
  };

  static ProfileAccountPage _instantiate(DecodingData data) {
    return ProfileAccountPage(
      items: data.dec(_f$items),
      totalCount: data.dec(_f$totalCount),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileAccountPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileAccountPage>(map);
  }

  static ProfileAccountPage fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileAccountPage>(json);
  }
}

mixin ProfileAccountPageMappable {
  String toJson() {
    return ProfileAccountPageMapper.ensureInitialized()
        .encodeJson<ProfileAccountPage>(this as ProfileAccountPage);
  }

  Map<String, dynamic> toMap() {
    return ProfileAccountPageMapper.ensureInitialized()
        .encodeMap<ProfileAccountPage>(this as ProfileAccountPage);
  }

  ProfileAccountPageCopyWith<
    ProfileAccountPage,
    ProfileAccountPage,
    ProfileAccountPage
  >
  get copyWith =>
      _ProfileAccountPageCopyWithImpl<ProfileAccountPage, ProfileAccountPage>(
        this as ProfileAccountPage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProfileAccountPageMapper.ensureInitialized().stringifyValue(
      this as ProfileAccountPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfileAccountPageMapper.ensureInitialized().equalsValue(
      this as ProfileAccountPage,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileAccountPageMapper.ensureInitialized().hashValue(
      this as ProfileAccountPage,
    );
  }
}

extension ProfileAccountPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileAccountPage, $Out> {
  ProfileAccountPageCopyWith<$R, ProfileAccountPage, $Out>
  get $asProfileAccountPage => $base.as(
    (v, t, t2) => _ProfileAccountPageCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileAccountPageCopyWith<
  $R,
  $In extends ProfileAccountPage,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    ProfileAccountSummary,
    ProfileAccountSummaryCopyWith<
      $R,
      ProfileAccountSummary,
      ProfileAccountSummary
    >
  >
  get items;
  $R call({
    List<ProfileAccountSummary>? items,
    int? totalCount,
    String? cursor,
  });
  ProfileAccountPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileAccountPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileAccountPage, $Out>
    implements ProfileAccountPageCopyWith<$R, ProfileAccountPage, $Out> {
  _ProfileAccountPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileAccountPage> $mapper =
      ProfileAccountPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    ProfileAccountSummary,
    ProfileAccountSummaryCopyWith<
      $R,
      ProfileAccountSummary,
      ProfileAccountSummary
    >
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({
    List<ProfileAccountSummary>? items,
    int? totalCount,
    Object? cursor = $none,
  }) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (totalCount != null) #totalCount: totalCount,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  ProfileAccountPage $make(CopyWithData data) => ProfileAccountPage(
    items: data.get(#items, or: $value.items),
    totalCount: data.get(#totalCount, or: $value.totalCount),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  ProfileAccountPageCopyWith<$R2, ProfileAccountPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProfileAccountPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

