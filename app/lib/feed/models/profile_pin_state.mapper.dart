// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'profile_pin_state.dart';

class ProfilePinSlotMapper extends EnumMapper<ProfilePinSlot> {
  ProfilePinSlotMapper._();

  static ProfilePinSlotMapper? _instance;
  static ProfilePinSlotMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfilePinSlotMapper._());
    }
    return _instance!;
  }

  static ProfilePinSlot fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  ProfilePinSlot decode(dynamic value) {
    switch (value) {
      case r'standard':
        return ProfilePinSlot.standard;
      case r'project':
        return ProfilePinSlot.project;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(ProfilePinSlot self) {
    switch (self) {
      case ProfilePinSlot.standard:
        return r'standard';
      case ProfilePinSlot.project:
        return r'project';
    }
  }
}

extension ProfilePinSlotMapperExtension on ProfilePinSlot {
  String toValue() {
    ProfilePinSlotMapper.ensureInitialized();
    return MapperContainer.globals.toValue<ProfilePinSlot>(this) as String;
  }
}

class ProfilePinStateMapper extends ClassMapperBase<ProfilePinState> {
  ProfilePinStateMapper._();

  static ProfilePinStateMapper? _instance;
  static ProfilePinStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ProfilePinStateMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ProfilePinState';

  static String? _$standardPostUri(ProfilePinState v) => v.standardPostUri;
  static const Field<ProfilePinState, String> _f$standardPostUri = Field(
    'standardPostUri',
    _$standardPostUri,
    opt: true,
  );
  static String? _$projectPostUri(ProfilePinState v) => v.projectPostUri;
  static const Field<ProfilePinState, String> _f$projectPostUri = Field(
    'projectPostUri',
    _$projectPostUri,
    opt: true,
  );

  @override
  final MappableFields<ProfilePinState> fields = const {
    #standardPostUri: _f$standardPostUri,
    #projectPostUri: _f$projectPostUri,
  };

  static ProfilePinState _instantiate(DecodingData data) {
    return ProfilePinState(
      standardPostUri: data.dec(_f$standardPostUri),
      projectPostUri: data.dec(_f$projectPostUri),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ProfilePinState fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ProfilePinState>(map);
  }

  static ProfilePinState fromJson(String json) {
    return ensureInitialized().decodeJson<ProfilePinState>(json);
  }
}

mixin ProfilePinStateMappable {
  String toJson() {
    return ProfilePinStateMapper.ensureInitialized()
        .encodeJson<ProfilePinState>(this as ProfilePinState);
  }

  Map<String, dynamic> toMap() {
    return ProfilePinStateMapper.ensureInitialized().encodeMap<ProfilePinState>(
      this as ProfilePinState,
    );
  }

  ProfilePinStateCopyWith<ProfilePinState, ProfilePinState, ProfilePinState>
  get copyWith =>
      _ProfilePinStateCopyWithImpl<ProfilePinState, ProfilePinState>(
        this as ProfilePinState,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return ProfilePinStateMapper.ensureInitialized().stringifyValue(
      this as ProfilePinState,
    );
  }

  @override
  bool operator ==(Object other) {
    return ProfilePinStateMapper.ensureInitialized().equalsValue(
      this as ProfilePinState,
      other,
    );
  }

  @override
  int get hashCode {
    return ProfilePinStateMapper.ensureInitialized().hashValue(
      this as ProfilePinState,
    );
  }
}

extension ProfilePinStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ProfilePinState, $Out> {
  ProfilePinStateCopyWith<$R, ProfilePinState, $Out> get $asProfilePinState =>
      $base.as((v, t, t2) => _ProfilePinStateCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class ProfilePinStateCopyWith<$R, $In extends ProfilePinState, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? standardPostUri, String? projectPostUri});
  ProfilePinStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ProfilePinStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ProfilePinState, $Out>
    implements ProfilePinStateCopyWith<$R, ProfilePinState, $Out> {
  _ProfilePinStateCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ProfilePinState> $mapper =
      ProfilePinStateMapper.ensureInitialized();
  @override
  $R call({Object? standardPostUri = $none, Object? projectPostUri = $none}) =>
      $apply(
        FieldCopyWithData({
          if (standardPostUri != $none) #standardPostUri: standardPostUri,
          if (projectPostUri != $none) #projectPostUri: projectPostUri,
        }),
      );
  @override
  ProfilePinState $make(CopyWithData data) => ProfilePinState(
    standardPostUri: data.get(#standardPostUri, or: $value.standardPostUri),
    projectPostUri: data.get(#projectPostUri, or: $value.projectPostUri),
  );

  @override
  ProfilePinStateCopyWith<$R2, ProfilePinState, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _ProfilePinStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

