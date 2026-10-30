// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'profile_link.dart';

class ProfileLinkMapper extends ClassMapperBase<ProfileLink> {
  ProfileLinkMapper._();

  static ProfileLinkMapper? _instance;
  static ProfileLinkMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileLinkMapper._());
      UploadedBlobMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileLink';

  static String _$url(ProfileLink v) => v.url;
  static const Field<ProfileLink, String> _f$url = Field('url', _$url);
  static String? _$uri(ProfileLink v) => v.uri;
  static const Field<ProfileLink, String> _f$uri = Field(
    'uri',
    _$uri,
    opt: true,
  );
  static String? _$cid(ProfileLink v) => v.cid;
  static const Field<ProfileLink, String> _f$cid = Field(
    'cid',
    _$cid,
    opt: true,
  );
  static String? _$title(ProfileLink v) => v.title;
  static const Field<ProfileLink, String> _f$title = Field(
    'title',
    _$title,
    opt: true,
  );
  static String? _$icon(ProfileLink v) => v.icon;
  static const Field<ProfileLink, String> _f$icon = Field(
    'icon',
    _$icon,
    opt: true,
  );
  static UploadedBlob? _$iconBlob(ProfileLink v) => v.iconBlob;
  static const Field<ProfileLink, UploadedBlob> _f$iconBlob = Field(
    'iconBlob',
    _$iconBlob,
    opt: true,
  );
  static String? _$createdAt(ProfileLink v) => v.createdAt;
  static const Field<ProfileLink, String> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
    opt: true,
  );
  static String? _$rkey(ProfileLink v) => v.rkey;
  static const Field<ProfileLink, String> _f$rkey = Field(
    'rkey',
    _$rkey,
    opt: true,
  );

  @override
  final MappableFields<ProfileLink> fields = const {
    #url: _f$url,
    #uri: _f$uri,
    #cid: _f$cid,
    #title: _f$title,
    #icon: _f$icon,
    #iconBlob: _f$iconBlob,
    #createdAt: _f$createdAt,
    #rkey: _f$rkey,
  };

  static ProfileLink _instantiate(DecodingData data) {
    return ProfileLink(
      url: data.dec(_f$url),
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      title: data.dec(_f$title),
      icon: data.dec(_f$icon),
      iconBlob: data.dec(_f$iconBlob),
      createdAt: data.dec(_f$createdAt),
      rkey: data.dec(_f$rkey),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileLink fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileLink>(map);
  }

  static ProfileLink fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileLink>(json);
  }
}

mixin ProfileLinkMappable {
  String toJson() {
    return ProfileLinkMapper.ensureInitialized().encodeJson<ProfileLink>(
      this as ProfileLink,
    );
  }

  Map<String, dynamic> toMap() {
    return ProfileLinkMapper.ensureInitialized().encodeMap<ProfileLink>(
      this as ProfileLink,
    );
  }

  ProfileLinkCopyWith<ProfileLink, ProfileLink, ProfileLink> get copyWith =>
      _ProfileLinkCopyWithImpl<ProfileLink, ProfileLink>(
        this as ProfileLink,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProfileLinkMapper.ensureInitialized().stringifyValue(
      this as ProfileLink,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfileLinkMapper.ensureInitialized().equalsValue(
      this as ProfileLink,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileLinkMapper.ensureInitialized().hashValue(this as ProfileLink);
  }
}

extension ProfileLinkValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileLink, $Out> {
  ProfileLinkCopyWith<$R, ProfileLink, $Out> get $asProfileLink =>
      $base.as((v, t, t2) => _ProfileLinkCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProfileLinkCopyWith<$R, $In extends ProfileLink, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  UploadedBlobCopyWith<$R, UploadedBlob, UploadedBlob>? get iconBlob;
  $R call({
    String? url,
    String? uri,
    String? cid,
    String? title,
    String? icon,
    UploadedBlob? iconBlob,
    String? createdAt,
    String? rkey,
  });
  ProfileLinkCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _ProfileLinkCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileLink, $Out>
    implements ProfileLinkCopyWith<$R, ProfileLink, $Out> {
  _ProfileLinkCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileLink> $mapper =
      ProfileLinkMapper.ensureInitialized();
  @override
  UploadedBlobCopyWith<$R, UploadedBlob, UploadedBlob>? get iconBlob =>
      $value.iconBlob?.copyWith.$chain((v) => call(iconBlob: v));
  @override
  $R call({
    String? url,
    Object? uri = $none,
    Object? cid = $none,
    Object? title = $none,
    Object? icon = $none,
    Object? iconBlob = $none,
    Object? createdAt = $none,
    Object? rkey = $none,
  }) => $apply(
    FieldCopyWithData({
      if (url != null) #url: url,
      if (uri != $none) #uri: uri,
      if (cid != $none) #cid: cid,
      if (title != $none) #title: title,
      if (icon != $none) #icon: icon,
      if (iconBlob != $none) #iconBlob: iconBlob,
      if (createdAt != $none) #createdAt: createdAt,
      if (rkey != $none) #rkey: rkey,
    }),
  );
  @override
  ProfileLink $make(CopyWithData data) => ProfileLink(
    url: data.get(#url, or: $value.url),
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    title: data.get(#title, or: $value.title),
    icon: data.get(#icon, or: $value.icon),
    iconBlob: data.get(#iconBlob, or: $value.iconBlob),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    rkey: data.get(#rkey, or: $value.rkey),
  );

  @override
  ProfileLinkCopyWith<$R2, ProfileLink, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProfileLinkCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
