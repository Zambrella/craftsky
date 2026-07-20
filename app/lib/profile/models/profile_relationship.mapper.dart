// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'profile_relationship.dart';

class ProfileRelationshipActionMapper
    extends EnumMapper<ProfileRelationshipAction> {
  ProfileRelationshipActionMapper._();

  static ProfileRelationshipActionMapper? _instance;
  static ProfileRelationshipActionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = ProfileRelationshipActionMapper._(),
      );
    }
    return _instance!;
  }

  static ProfileRelationshipAction fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  ProfileRelationshipAction decode(dynamic value) {
    switch (value) {
      case r'mute':
        return ProfileRelationshipAction.mute;
      case r'unmute':
        return ProfileRelationshipAction.unmute;
      case r'block':
        return ProfileRelationshipAction.block;
      case r'unblock':
        return ProfileRelationshipAction.unblock;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(ProfileRelationshipAction self) {
    switch (self) {
      case ProfileRelationshipAction.mute:
        return r'mute';
      case ProfileRelationshipAction.unmute:
        return r'unmute';
      case ProfileRelationshipAction.block:
        return r'block';
      case ProfileRelationshipAction.unblock:
        return r'unblock';
    }
  }
}

extension ProfileRelationshipActionMapperExtension
    on ProfileRelationshipAction {
  String toValue() {
    ProfileRelationshipActionMapper.ensureInitialized();
    return MapperContainer.globals.toValue<ProfileRelationshipAction>(this)
        as String;
  }
}

class ProfileRelationshipMapper extends ClassMapperBase<ProfileRelationship> {
  ProfileRelationshipMapper._();

  static ProfileRelationshipMapper? _instance;
  static ProfileRelationshipMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfileRelationshipMapper._());
      ProfileRelationshipActionMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ProfileRelationship';

  static bool _$muted(ProfileRelationship v) => v.muted;
  static const Field<ProfileRelationship, bool> _f$muted = Field(
    'muted',
    _$muted,
    opt: true,
    def: false,
  );
  static bool _$blocking(ProfileRelationship v) => v.blocking;
  static const Field<ProfileRelationship, bool> _f$blocking = Field(
    'blocking',
    _$blocking,
    opt: true,
    def: false,
  );
  static bool _$blockedBy(ProfileRelationship v) => v.blockedBy;
  static const Field<ProfileRelationship, bool> _f$blockedBy = Field(
    'blockedBy',
    _$blockedBy,
    opt: true,
    def: false,
  );
  static String? _$uri(ProfileRelationship v) => v.uri;
  static const Field<ProfileRelationship, String> _f$uri = Field(
    'uri',
    _$uri,
    opt: true,
  );
  static String? _$cid(ProfileRelationship v) => v.cid;
  static const Field<ProfileRelationship, String> _f$cid = Field(
    'cid',
    _$cid,
    opt: true,
  );
  static String? _$rkey(ProfileRelationship v) => v.rkey;
  static const Field<ProfileRelationship, String> _f$rkey = Field(
    'rkey',
    _$rkey,
    opt: true,
  );
  static ProfileRelationshipAction? _$pendingAction(ProfileRelationship v) =>
      v.pendingAction;
  static const Field<ProfileRelationship, ProfileRelationshipAction>
  _f$pendingAction = Field('pendingAction', _$pendingAction, opt: true);
  static Object? _$lastError(ProfileRelationship v) => v.lastError;
  static const Field<ProfileRelationship, Object> _f$lastError = Field(
    'lastError',
    _$lastError,
    opt: true,
  );
  static bool _$confirmedOverlay(ProfileRelationship v) => v.confirmedOverlay;
  static const Field<ProfileRelationship, bool> _f$confirmedOverlay = Field(
    'confirmedOverlay',
    _$confirmedOverlay,
    opt: true,
    def: false,
  );
  static bool _$initialized(ProfileRelationship v) => v.initialized;
  static const Field<ProfileRelationship, bool> _f$initialized = Field(
    'initialized',
    _$initialized,
    opt: true,
    def: false,
  );

  @override
  final MappableFields<ProfileRelationship> fields = const {
    #muted: _f$muted,
    #blocking: _f$blocking,
    #blockedBy: _f$blockedBy,
    #uri: _f$uri,
    #cid: _f$cid,
    #rkey: _f$rkey,
    #pendingAction: _f$pendingAction,
    #lastError: _f$lastError,
    #confirmedOverlay: _f$confirmedOverlay,
    #initialized: _f$initialized,
  };
  @override
  final bool ignoreNull = true;

  @override
  final MappingHook hook = const ProfileRelationshipWireHook();
  static ProfileRelationship _instantiate(DecodingData data) {
    return ProfileRelationship(
      muted: data.dec(_f$muted),
      blocking: data.dec(_f$blocking),
      blockedBy: data.dec(_f$blockedBy),
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      rkey: data.dec(_f$rkey),
      pendingAction: data.dec(_f$pendingAction),
      lastError: data.dec(_f$lastError),
      confirmedOverlay: data.dec(_f$confirmedOverlay),
      initialized: data.dec(_f$initialized),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProfileRelationship fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfileRelationship>(map);
  }

  static ProfileRelationship fromJson(String json) {
    return ensureInitialized().decodeJson<ProfileRelationship>(json);
  }
}

mixin ProfileRelationshipMappable {
  ProfileRelationshipCopyWith<
    ProfileRelationship,
    ProfileRelationship,
    ProfileRelationship
  >
  get copyWith =>
      _ProfileRelationshipCopyWithImpl<
        ProfileRelationship,
        ProfileRelationship
      >(this as ProfileRelationship, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return ProfileRelationshipMapper.ensureInitialized().equalsValue(
      this as ProfileRelationship,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfileRelationshipMapper.ensureInitialized().hashValue(
      this as ProfileRelationship,
    );
  }
}

extension ProfileRelationshipValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfileRelationship, $Out> {
  ProfileRelationshipCopyWith<$R, ProfileRelationship, $Out>
  get $asProfileRelationship => $base.as(
    (v, t, t2) => _ProfileRelationshipCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ProfileRelationshipCopyWith<
  $R,
  $In extends ProfileRelationship,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    bool? muted,
    bool? blocking,
    bool? blockedBy,
    String? uri,
    String? cid,
    String? rkey,
    ProfileRelationshipAction? pendingAction,
    Object? lastError,
    bool? confirmedOverlay,
    bool? initialized,
  });
  ProfileRelationshipCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfileRelationshipCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfileRelationship, $Out>
    implements ProfileRelationshipCopyWith<$R, ProfileRelationship, $Out> {
  _ProfileRelationshipCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfileRelationship> $mapper =
      ProfileRelationshipMapper.ensureInitialized();
  @override
  $R call({
    bool? muted,
    bool? blocking,
    bool? blockedBy,
    Object? uri = $none,
    Object? cid = $none,
    Object? rkey = $none,
    Object? pendingAction = $none,
    Object? lastError = $none,
    bool? confirmedOverlay,
    bool? initialized,
  }) => $apply(
    FieldCopyWithData({
      if (muted != null) #muted: muted,
      if (blocking != null) #blocking: blocking,
      if (blockedBy != null) #blockedBy: blockedBy,
      if (uri != $none) #uri: uri,
      if (cid != $none) #cid: cid,
      if (rkey != $none) #rkey: rkey,
      if (pendingAction != $none) #pendingAction: pendingAction,
      if (lastError != $none) #lastError: lastError,
      if (confirmedOverlay != null) #confirmedOverlay: confirmedOverlay,
      if (initialized != null) #initialized: initialized,
    }),
  );
  @override
  ProfileRelationship $make(CopyWithData data) => ProfileRelationship(
    muted: data.get(#muted, or: $value.muted),
    blocking: data.get(#blocking, or: $value.blocking),
    blockedBy: data.get(#blockedBy, or: $value.blockedBy),
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    rkey: data.get(#rkey, or: $value.rkey),
    pendingAction: data.get(#pendingAction, or: $value.pendingAction),
    lastError: data.get(#lastError, or: $value.lastError),
    confirmedOverlay: data.get(#confirmedOverlay, or: $value.confirmedOverlay),
    initialized: data.get(#initialized, or: $value.initialized),
  );

  @override
  ProfileRelationshipCopyWith<$R2, ProfileRelationship, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ProfileRelationshipCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

