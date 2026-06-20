// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'profile_search_page.dart';

class ProfileSearchResultMapper extends ClassMapperBase<ProfileSearchResult> {
  ProfileSearchResultMapper._();

  static ProfileSearchResultMapper? _instance;
  static ProfileSearchResultMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileSearchResultMapper._());
      MapperContainer.globals.useAll([DidMapper(), HandleMapper()]);
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileSearchResult';

  static Did _$did(ProfileSearchResult v) => v.did;
  static dynamic _arg$did(f) => f<Did>();
  static const Field<ProfileSearchResult, String> _f$did = Field(
    'did',
    _$did,
    arg: _arg$did,
  );
  static Handle _$handle(ProfileSearchResult v) => v.handle;
  static dynamic _arg$handle(f) => f<Handle>();
  static const Field<ProfileSearchResult, String> _f$handle = Field(
    'handle',
    _$handle,
    arg: _arg$handle,
  );
  static bool _$isCraftskyProfile(ProfileSearchResult v) => v.isCraftskyProfile;
  static const Field<ProfileSearchResult, bool> _f$isCraftskyProfile = Field(
    'isCraftskyProfile',
    _$isCraftskyProfile,
  );
  static bool _$viewerIsFollowing(ProfileSearchResult v) => v.viewerIsFollowing;
  static const Field<ProfileSearchResult, bool> _f$viewerIsFollowing = Field(
    'viewerIsFollowing',
    _$viewerIsFollowing,
  );
  static String? _$displayName(ProfileSearchResult v) => v.displayName;
  static const Field<ProfileSearchResult, String> _f$displayName = Field(
    'displayName',
    _$displayName,
    opt: true,
  );
  static String? _$description(ProfileSearchResult v) => v.description;
  static const Field<ProfileSearchResult, String> _f$description = Field(
    'description',
    _$description,
    opt: true,
  );
  static String? _$avatar(ProfileSearchResult v) => v.avatar;
  static const Field<ProfileSearchResult, String> _f$avatar = Field(
    'avatar',
    _$avatar,
    opt: true,
  );

  @override
  final MappableFields<ProfileSearchResult> fields = const {
    #did: _f$did,
    #handle: _f$handle,
    #isCraftskyProfile: _f$isCraftskyProfile,
    #viewerIsFollowing: _f$viewerIsFollowing,
    #displayName: _f$displayName,
    #description: _f$description,
    #avatar: _f$avatar,
  };

  static ProfileSearchResult _instantiate(DecodingData data) {
    return ProfileSearchResult(
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      isCraftskyProfile: data.dec(_f$isCraftskyProfile),
      viewerIsFollowing: data.dec(_f$viewerIsFollowing),
      displayName: data.dec(_f$displayName),
      description: data.dec(_f$description),
      avatar: data.dec(_f$avatar),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileSearchResult fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileSearchResult>(map);
  }

  static ProfileSearchResult fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileSearchResult>(json);
  }
}

mixin ProfileSearchResultMappable {
  String toJson() {
    return ProfileSearchResultMapper.ensureInitialized()
        .encodeJson<ProfileSearchResult>(this as ProfileSearchResult);
  }

  Map<String, dynamic> toMap() {
    return ProfileSearchResultMapper.ensureInitialized()
        .encodeMap<ProfileSearchResult>(this as ProfileSearchResult);
  }

  ProfileSearchResultCopyWith<
    ProfileSearchResult,
    ProfileSearchResult,
    ProfileSearchResult
  >
  get copyWith =>
      _ProfileSearchResultCopyWithImpl<
        ProfileSearchResult,
        ProfileSearchResult
      >(this as ProfileSearchResult, $identity, $identity);
  @override
  String toString() {
    return ProfileSearchResultMapper.ensureInitialized().stringifyValue(
      this as ProfileSearchResult,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfileSearchResultMapper.ensureInitialized().equalsValue(
      this as ProfileSearchResult,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileSearchResultMapper.ensureInitialized().hashValue(
      this as ProfileSearchResult,
    );
  }
}

extension ProfileSearchResultValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileSearchResult, $Out> {
  ProfileSearchResultCopyWith<$R, ProfileSearchResult, $Out>
  get $asProfileSearchResult => $base.as(
    (v, t, t2) => _ProfileSearchResultCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileSearchResultCopyWith<
  $R,
  $In extends ProfileSearchResult,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? did,
    String? handle,
    bool? isCraftskyProfile,
    bool? viewerIsFollowing,
    String? displayName,
    String? description,
    String? avatar,
  });
  ProfileSearchResultCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileSearchResultCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileSearchResult, $Out>
    implements ProfileSearchResultCopyWith<$R, ProfileSearchResult, $Out> {
  _ProfileSearchResultCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileSearchResult> $mapper =
      ProfileSearchResultMapper.ensureInitialized();
  @override
  $R call({
    String? did,
    String? handle,
    bool? isCraftskyProfile,
    bool? viewerIsFollowing,
    Object? displayName = $none,
    Object? description = $none,
    Object? avatar = $none,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (isCraftskyProfile != null) #isCraftskyProfile: isCraftskyProfile,
      if (viewerIsFollowing != null) #viewerIsFollowing: viewerIsFollowing,
      if (displayName != $none) #displayName: displayName,
      if (description != $none) #description: description,
      if (avatar != $none) #avatar: avatar,
    }),
  );
  @override
  ProfileSearchResult $make(CopyWithData data) => ProfileSearchResult(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    isCraftskyProfile: data.get(
      #isCraftskyProfile,
      or: $value.isCraftskyProfile,
    ),
    viewerIsFollowing: data.get(
      #viewerIsFollowing,
      or: $value.viewerIsFollowing,
    ),
    displayName: data.get(#displayName, or: $value.displayName),
    description: data.get(#description, or: $value.description),
    avatar: data.get(#avatar, or: $value.avatar),
  );

  @override
  ProfileSearchResultCopyWith<$R2, ProfileSearchResult, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProfileSearchResultCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ProfileSearchPageMapper extends ClassMapperBase<ProfileSearchPage> {
  ProfileSearchPageMapper._();

  static ProfileSearchPageMapper? _instance;
  static ProfileSearchPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileSearchPageMapper._());
      ProfileSearchResultMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileSearchPage';

  static List<ProfileSearchResult> _$items(ProfileSearchPage v) => v.items;
  static const Field<ProfileSearchPage, List<ProfileSearchResult>> _f$items =
      Field('items', _$items);
  static String? _$cursor(ProfileSearchPage v) => v.cursor;
  static const Field<ProfileSearchPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<ProfileSearchPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };
  @override
  final bool ignoreNull = true;

  static ProfileSearchPage _instantiate(DecodingData data) {
    return ProfileSearchPage(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileSearchPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileSearchPage>(map);
  }

  static ProfileSearchPage fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileSearchPage>(json);
  }
}

mixin ProfileSearchPageMappable {
  String toJson() {
    return ProfileSearchPageMapper.ensureInitialized()
        .encodeJson<ProfileSearchPage>(this as ProfileSearchPage);
  }

  Map<String, dynamic> toMap() {
    return ProfileSearchPageMapper.ensureInitialized()
        .encodeMap<ProfileSearchPage>(this as ProfileSearchPage);
  }

  ProfileSearchPageCopyWith<
    ProfileSearchPage,
    ProfileSearchPage,
    ProfileSearchPage
  >
  get copyWith =>
      _ProfileSearchPageCopyWithImpl<ProfileSearchPage, ProfileSearchPage>(
        this as ProfileSearchPage,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProfileSearchPageMapper.ensureInitialized().stringifyValue(
      this as ProfileSearchPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfileSearchPageMapper.ensureInitialized().equalsValue(
      this as ProfileSearchPage,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileSearchPageMapper.ensureInitialized().hashValue(
      this as ProfileSearchPage,
    );
  }
}

extension ProfileSearchPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileSearchPage, $Out> {
  ProfileSearchPageCopyWith<$R, ProfileSearchPage, $Out>
  get $asProfileSearchPage => $base.as(
    (v, t, t2) => _ProfileSearchPageCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileSearchPageCopyWith<
  $R,
  $In extends ProfileSearchPage,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    ProfileSearchResult,
    ProfileSearchResultCopyWith<$R, ProfileSearchResult, ProfileSearchResult>
  >
  get items;
  $R call({List<ProfileSearchResult>? items, String? cursor});
  ProfileSearchPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileSearchPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileSearchPage, $Out>
    implements ProfileSearchPageCopyWith<$R, ProfileSearchPage, $Out> {
  _ProfileSearchPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileSearchPage> $mapper =
      ProfileSearchPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    ProfileSearchResult,
    ProfileSearchResultCopyWith<$R, ProfileSearchResult, ProfileSearchResult>
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<ProfileSearchResult>? items, Object? cursor = $none}) => $apply(
    FieldCopyWithData({
      if (items != null) #items: items,
      if (cursor != $none) #cursor: cursor,
    }),
  );
  @override
  ProfileSearchPage $make(CopyWithData data) => ProfileSearchPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  ProfileSearchPageCopyWith<$R2, ProfileSearchPage, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProfileSearchPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

