// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'subscription_access.dart';

class SubscriptionTierMapper extends EnumMapper<SubscriptionTier> {
  SubscriptionTierMapper._();

  static SubscriptionTierMapper? _instance;
  static SubscriptionTierMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SubscriptionTierMapper._());
    }
    return _instance!;
  }

  static SubscriptionTier fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  SubscriptionTier decode(dynamic value) {
    switch (value) {
      case r'free':
        return SubscriptionTier.free;
      case r'plus':
        return SubscriptionTier.plus;
      case r'business':
        return SubscriptionTier.business;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(SubscriptionTier self) {
    switch (self) {
      case SubscriptionTier.free:
        return r'free';
      case SubscriptionTier.plus:
        return r'plus';
      case SubscriptionTier.business:
        return r'business';
    }
  }
}

extension SubscriptionTierMapperExtension on SubscriptionTier {
  String toValue() {
    SubscriptionTierMapper.ensureInitialized();
    return MapperContainer.globals.toValue<SubscriptionTier>(this) as String;
  }
}

class SubscriptionAccessMapper extends ClassMapperBase<SubscriptionAccess> {
  SubscriptionAccessMapper._();

  static SubscriptionAccessMapper? _instance;
  static SubscriptionAccessMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = SubscriptionAccessMapper._());
      MapperContainer.globals.useAll([DidMapper()]);
      SubscriptionTierMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'SubscriptionAccess';

  static Did _$did(SubscriptionAccess v) => v.did;
  static const Field<SubscriptionAccess, Did> _f$did = Field('did', _$did);
  static SubscriptionTier _$effectiveTier(SubscriptionAccess v) =>
      v.effectiveTier;
  static const Field<SubscriptionAccess, SubscriptionTier> _f$effectiveTier =
      Field('effectiveTier', _$effectiveTier);
  static bool _$givesAccess(SubscriptionAccess v) => v.givesAccess;
  static const Field<SubscriptionAccess, bool> _f$givesAccess = Field(
    'givesAccess',
    _$givesAccess,
  );
  static DateTime? _$accessEndsAt(SubscriptionAccess v) => v.accessEndsAt;
  static const Field<SubscriptionAccess, DateTime> _f$accessEndsAt = Field(
    'accessEndsAt',
    _$accessEndsAt,
    opt: true,
  );
  static SubscriptionTier? _$assignedTier(SubscriptionAccess v) =>
      v.assignedTier;
  static const Field<SubscriptionAccess, SubscriptionTier> _f$assignedTier =
      Field('assignedTier', _$assignedTier, opt: true);

  @override
  final MappableFields<SubscriptionAccess> fields = const {
    #did: _f$did,
    #effectiveTier: _f$effectiveTier,
    #givesAccess: _f$givesAccess,
    #accessEndsAt: _f$accessEndsAt,
    #assignedTier: _f$assignedTier,
  };
  @override
  final bool ignoreNull = true;

  static SubscriptionAccess _instantiate(DecodingData data) {
    return SubscriptionAccess(
      did: data.dec(_f$did),
      effectiveTier: data.dec(_f$effectiveTier),
      givesAccess: data.dec(_f$givesAccess),
      accessEndsAt: data.dec(_f$accessEndsAt),
      assignedTier: data.dec(_f$assignedTier),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static SubscriptionAccess fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<SubscriptionAccess>(map);
  }

  static SubscriptionAccess fromJson(String json) {
    return ensureInitialized().decodeJson<SubscriptionAccess>(json);
  }
}

mixin SubscriptionAccessMappable {
  SubscriptionAccessCopyWith<
    SubscriptionAccess,
    SubscriptionAccess,
    SubscriptionAccess
  >
  get copyWith =>
      _SubscriptionAccessCopyWithImpl<SubscriptionAccess, SubscriptionAccess>(
        this as SubscriptionAccess,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return SubscriptionAccessMapper.ensureInitialized().equalsValue(
      this as SubscriptionAccess,
      other,
    );
  }

  @override
  int get hashCode {
    return SubscriptionAccessMapper.ensureInitialized().hashValue(
      this as SubscriptionAccess,
    );
  }
}

extension SubscriptionAccessValueCopy<$R, $Out>
    on ObjectCopyWith<$R, SubscriptionAccess, $Out> {
  SubscriptionAccessCopyWith<$R, SubscriptionAccess, $Out>
  get $asSubscriptionAccess => $base.as(
    (v, t, t2) => _SubscriptionAccessCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class SubscriptionAccessCopyWith<
  $R,
  $In extends SubscriptionAccess,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    Did? did,
    SubscriptionTier? effectiveTier,
    bool? givesAccess,
    DateTime? accessEndsAt,
    SubscriptionTier? assignedTier,
  });
  SubscriptionAccessCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _SubscriptionAccessCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, SubscriptionAccess, $Out>
    implements SubscriptionAccessCopyWith<$R, SubscriptionAccess, $Out> {
  _SubscriptionAccessCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<SubscriptionAccess> $mapper =
      SubscriptionAccessMapper.ensureInitialized();
  @override
  $R call({
    Did? did,
    SubscriptionTier? effectiveTier,
    bool? givesAccess,
    Object? accessEndsAt = $none,
    Object? assignedTier = $none,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (effectiveTier != null) #effectiveTier: effectiveTier,
      if (givesAccess != null) #givesAccess: givesAccess,
      if (accessEndsAt != $none) #accessEndsAt: accessEndsAt,
      if (assignedTier != $none) #assignedTier: assignedTier,
    }),
  );
  @override
  SubscriptionAccess $make(CopyWithData data) => SubscriptionAccess(
    did: data.get(#did, or: $value.did),
    effectiveTier: data.get(#effectiveTier, or: $value.effectiveTier),
    givesAccess: data.get(#givesAccess, or: $value.givesAccess),
    accessEndsAt: data.get(#accessEndsAt, or: $value.accessEndsAt),
    assignedTier: data.get(#assignedTier, or: $value.assignedTier),
  );

  @override
  SubscriptionAccessCopyWith<$R2, SubscriptionAccess, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _SubscriptionAccessCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
