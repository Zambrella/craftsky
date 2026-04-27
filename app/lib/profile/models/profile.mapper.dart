// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'profile.dart';

class ProfileMapper extends ClassMapperBase<Profile> {
  ProfileMapper._();

  static ProfileMapper? _instance;
  static ProfileMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'Profile';

  static String _$did(Profile v) => v.did;
  static const Field<Profile, String> _f$did = Field('did', _$did);
  static String _$handle(Profile v) => v.handle;
  static const Field<Profile, String> _f$handle = Field('handle', _$handle);
  static List<String> _$crafts(Profile v) => v.crafts;
  static const Field<Profile, List<String>> _f$crafts = Field(
    'crafts',
    _$crafts,
  );
  static String? _$displayName(Profile v) => v.displayName;
  static const Field<Profile, String> _f$displayName = Field(
    'displayName',
    _$displayName,
    opt: true,
  );
  static String? _$description(Profile v) => v.description;
  static const Field<Profile, String> _f$description = Field(
    'description',
    _$description,
    opt: true,
  );
  static String? _$avatar(Profile v) => v.avatar;
  static const Field<Profile, String> _f$avatar = Field(
    'avatar',
    _$avatar,
    opt: true,
  );
  static String? _$banner(Profile v) => v.banner;
  static const Field<Profile, String> _f$banner = Field(
    'banner',
    _$banner,
    opt: true,
  );
  static DateTime? _$createdAt(Profile v) => v.createdAt;
  static const Field<Profile, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
    opt: true,
  );

  @override
  final MappableFields<Profile> fields = const {
    #did: _f$did,
    #handle: _f$handle,
    #crafts: _f$crafts,
    #displayName: _f$displayName,
    #description: _f$description,
    #avatar: _f$avatar,
    #banner: _f$banner,
    #createdAt: _f$createdAt,
  };

  static Profile _instantiate(DecodingData data) {
    return Profile(
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      crafts: data.dec(_f$crafts),
      displayName: data.dec(_f$displayName),
      description: data.dec(_f$description),
      avatar: data.dec(_f$avatar),
      banner: data.dec(_f$banner),
      createdAt: data.dec(_f$createdAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static Profile fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<Profile>(map);
  }

  static Profile fromJson(String json) {
    return ensureInitialized().decodeJson<Profile>(json);
  }
}

mixin ProfileMappable {
  String toJson() {
    return ProfileMapper.ensureInitialized().encodeJson<Profile>(
      this as Profile,
    );
  }

  Map<String, dynamic> toMap() {
    return ProfileMapper.ensureInitialized().encodeMap<Profile>(
      this as Profile,
    );
  }

  ProfileCopyWith<Profile, Profile, Profile> get copyWith =>
      _ProfileCopyWithImpl<Profile, Profile>(
        this as Profile,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProfileMapper.ensureInitialized().stringifyValue(this as Profile);
  }

  @override
  bool operator ==(Object other) {
    return ProfileMapper.ensureInitialized().equalsValue(
      this as Profile,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileMapper.ensureInitialized().hashValue(this as Profile);
  }
}

extension ProfileValueCopy<$R, $Out> on ObjectCopyWith<$R, Profile, $Out> {
  ProfileCopyWith<$R, Profile, $Out> get $asProfile =>
      $base.as((v, t, t2) => _ProfileCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProfileCopyWith<$R, $In extends Profile, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get crafts;
  $R call({
    String? did,
    String? handle,
    List<String>? crafts,
    String? displayName,
    String? description,
    String? avatar,
    String? banner,
    DateTime? createdAt,
  });
  ProfileCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ProfileCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, Profile, $Out>
    implements ProfileCopyWith<$R, Profile, $Out> {
  _ProfileCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<Profile> $mapper =
      ProfileMapper.ensureInitialized();
  @override
  ListCopyWith<$R, String, ObjectCopyWith<$R, String, String>> get crafts =>
      ListCopyWith(
        $value.crafts,
        (v, t) => ObjectCopyWith(v, $identity, t),
        (v) => call(crafts: v),
      );
  @override
  $R call({
    String? did,
    String? handle,
    List<String>? crafts,
    Object? displayName = $none,
    Object? description = $none,
    Object? avatar = $none,
    Object? banner = $none,
    Object? createdAt = $none,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (crafts != null) #crafts: crafts,
      if (displayName != $none) #displayName: displayName,
      if (description != $none) #description: description,
      if (avatar != $none) #avatar: avatar,
      if (banner != $none) #banner: banner,
      if (createdAt != $none) #createdAt: createdAt,
    }),
  );
  @override
  Profile $make(CopyWithData data) => Profile(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    crafts: data.get(#crafts, or: $value.crafts),
    displayName: data.get(#displayName, or: $value.displayName),
    description: data.get(#description, or: $value.description),
    avatar: data.get(#avatar, or: $value.avatar),
    banner: data.get(#banner, or: $value.banner),
    createdAt: data.get(#createdAt, or: $value.createdAt),
  );

  @override
  ProfileCopyWith<$R2, Profile, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProfileCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

