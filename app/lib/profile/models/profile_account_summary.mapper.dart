// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'profile_account_summary.dart';

class ProfileAccountSummaryMapper
    extends ClassMapperBase<ProfileAccountSummary> {
  ProfileAccountSummaryMapper._();

  static ProfileAccountSummaryMapper? _instance;
  static ProfileAccountSummaryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileAccountSummaryMapper._());
      MapperContainer.globals.useAll([DidMapper(), HandleMapper()]);
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileAccountSummary';

  static Did _$did(ProfileAccountSummary v) => v.did;
  static dynamic _arg$did(f) => f<Did>();
  static const Field<ProfileAccountSummary, String> _f$did = Field(
    'did',
    _$did,
    arg: _arg$did,
  );
  static Handle _$handle(ProfileAccountSummary v) => v.handle;
  static dynamic _arg$handle(f) => f<Handle>();
  static const Field<ProfileAccountSummary, String> _f$handle = Field(
    'handle',
    _$handle,
    arg: _arg$handle,
  );
  static bool _$isCraftskyProfile(ProfileAccountSummary v) =>
      v.isCraftskyProfile;
  static const Field<ProfileAccountSummary, bool> _f$isCraftskyProfile = Field(
    'isCraftskyProfile',
    _$isCraftskyProfile,
  );
  static String? _$displayName(ProfileAccountSummary v) => v.displayName;
  static const Field<ProfileAccountSummary, String> _f$displayName = Field(
    'displayName',
    _$displayName,
    opt: true,
  );
  static String? _$description(ProfileAccountSummary v) => v.description;
  static const Field<ProfileAccountSummary, String> _f$description = Field(
    'description',
    _$description,
    opt: true,
  );
  static String? _$avatar(ProfileAccountSummary v) => v.avatar;
  static const Field<ProfileAccountSummary, String> _f$avatar = Field(
    'avatar',
    _$avatar,
    opt: true,
  );

  @override
  final MappableFields<ProfileAccountSummary> fields = const {
    #did: _f$did,
    #handle: _f$handle,
    #isCraftskyProfile: _f$isCraftskyProfile,
    #displayName: _f$displayName,
    #description: _f$description,
    #avatar: _f$avatar,
  };

  static ProfileAccountSummary _instantiate(DecodingData data) {
    return ProfileAccountSummary(
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      isCraftskyProfile: data.dec(_f$isCraftskyProfile),
      displayName: data.dec(_f$displayName),
      description: data.dec(_f$description),
      avatar: data.dec(_f$avatar),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileAccountSummary fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileAccountSummary>(map);
  }

  static ProfileAccountSummary fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileAccountSummary>(json);
  }
}

mixin ProfileAccountSummaryMappable {
  String toJson() {
    return ProfileAccountSummaryMapper.ensureInitialized()
        .encodeJson<ProfileAccountSummary>(this as ProfileAccountSummary);
  }

  Map<String, dynamic> toMap() {
    return ProfileAccountSummaryMapper.ensureInitialized()
        .encodeMap<ProfileAccountSummary>(this as ProfileAccountSummary);
  }

  ProfileAccountSummaryCopyWith<
    ProfileAccountSummary,
    ProfileAccountSummary,
    ProfileAccountSummary
  >
  get copyWith =>
      _ProfileAccountSummaryCopyWithImpl<
        ProfileAccountSummary,
        ProfileAccountSummary
      >(this as ProfileAccountSummary, $identity, $identity);
  @override
  String toString() {
    return ProfileAccountSummaryMapper.ensureInitialized().stringifyValue(
      this as ProfileAccountSummary,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfileAccountSummaryMapper.ensureInitialized().equalsValue(
      this as ProfileAccountSummary,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileAccountSummaryMapper.ensureInitialized().hashValue(
      this as ProfileAccountSummary,
    );
  }
}

extension ProfileAccountSummaryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileAccountSummary, $Out> {
  ProfileAccountSummaryCopyWith<$R, ProfileAccountSummary, $Out>
  get $asProfileAccountSummary => $base.as(
    (v, t, t2) => _ProfileAccountSummaryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileAccountSummaryCopyWith<
  $R,
  $In extends ProfileAccountSummary,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? did,
    String? handle,
    bool? isCraftskyProfile,
    String? displayName,
    String? description,
    String? avatar,
  });
  ProfileAccountSummaryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileAccountSummaryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileAccountSummary, $Out>
    implements ProfileAccountSummaryCopyWith<$R, ProfileAccountSummary, $Out> {
  _ProfileAccountSummaryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileAccountSummary> $mapper =
      ProfileAccountSummaryMapper.ensureInitialized();
  @override
  $R call({
    String? did,
    String? handle,
    bool? isCraftskyProfile,
    Object? displayName = $none,
    Object? description = $none,
    Object? avatar = $none,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (isCraftskyProfile != null) #isCraftskyProfile: isCraftskyProfile,
      if (displayName != $none) #displayName: displayName,
      if (description != $none) #description: description,
      if (avatar != $none) #avatar: avatar,
    }),
  );
  @override
  ProfileAccountSummary $make(CopyWithData data) => ProfileAccountSummary(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    isCraftskyProfile: data.get(
      #isCraftskyProfile,
      or: $value.isCraftskyProfile,
    ),
    displayName: data.get(#displayName, or: $value.displayName),
    description: data.get(#description, or: $value.description),
    avatar: data.get(#avatar, or: $value.avatar),
  );

  @override
  ProfileAccountSummaryCopyWith<$R2, ProfileAccountSummary, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProfileAccountSummaryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

