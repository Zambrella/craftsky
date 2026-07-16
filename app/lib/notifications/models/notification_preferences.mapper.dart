// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'notification_preferences.dart';

class NotificationPreferenceScopeMapper
    extends EnumMapper<NotificationPreferenceScope> {
  NotificationPreferenceScopeMapper._();

  static NotificationPreferenceScopeMapper? _instance;
  static NotificationPreferenceScopeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = NotificationPreferenceScopeMapper._(),
      );
    }
    return _instance!;
  }

  static NotificationPreferenceScope fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  NotificationPreferenceScope decode(dynamic value) {
    switch (value) {
      case r'everyone':
        return NotificationPreferenceScope.everyone;
      case r'peopleIFollow':
        return NotificationPreferenceScope.peopleIFollow;
      default:
        throw MapperException.unknownEnumValue(value);
    }
  }

  @override
  dynamic encode(NotificationPreferenceScope self) {
    switch (self) {
      case NotificationPreferenceScope.everyone:
        return r'everyone';
      case NotificationPreferenceScope.peopleIFollow:
        return r'peopleIFollow';
    }
  }
}

extension NotificationPreferenceScopeMapperExtension
    on NotificationPreferenceScope {
  String toValue() {
    NotificationPreferenceScopeMapper.ensureInitialized();
    return MapperContainer.globals.toValue<NotificationPreferenceScope>(this)
        as String;
  }
}

class NotificationPreferenceMapper
    extends ClassMapperBase<NotificationPreference> {
  NotificationPreferenceMapper._();

  static NotificationPreferenceMapper? _instance;
  static NotificationPreferenceMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = NotificationPreferenceMapper._());
      NotificationPreferenceScopeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'NotificationPreference';

  static NotificationPreferenceScope _$scope(NotificationPreference v) =>
      v.scope;
  static const Field<NotificationPreference, NotificationPreferenceScope>
  _f$scope = Field('scope', _$scope);
  static bool _$pushEnabled(NotificationPreference v) => v.pushEnabled;
  static const Field<NotificationPreference, bool> _f$pushEnabled = Field(
    'pushEnabled',
    _$pushEnabled,
  );

  @override
  final MappableFields<NotificationPreference> fields = const {
    #scope: _f$scope,
    #pushEnabled: _f$pushEnabled,
  };

  static NotificationPreference _instantiate(DecodingData data) {
    return NotificationPreference(
      scope: data.dec(_f$scope),
      pushEnabled: data.dec(_f$pushEnabled),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static NotificationPreference fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<NotificationPreference>(map);
  }

  static NotificationPreference fromJson(String json) {
    return ensureInitialized().decodeJson<NotificationPreference>(json);
  }
}

mixin NotificationPreferenceMappable {
  String toJson() {
    return NotificationPreferenceMapper.ensureInitialized()
        .encodeJson<NotificationPreference>(this as NotificationPreference);
  }

  Map<String, dynamic> toMap() {
    return NotificationPreferenceMapper.ensureInitialized()
        .encodeMap<NotificationPreference>(this as NotificationPreference);
  }

  NotificationPreferenceCopyWith<
    NotificationPreference,
    NotificationPreference,
    NotificationPreference
  >
  get copyWith =>
      _NotificationPreferenceCopyWithImpl<
        NotificationPreference,
        NotificationPreference
      >(this as NotificationPreference, $identity, $identity);
  @override
  String toString() {
    return NotificationPreferenceMapper.ensureInitialized().stringifyValue(
      this as NotificationPreference,
    );
  }

  @override
  bool operator ==(Object other) {
    return NotificationPreferenceMapper.ensureInitialized().equalsValue(
      this as NotificationPreference,
      other,
    );
  }

  @override
  int get hashCode {
    return NotificationPreferenceMapper.ensureInitialized().hashValue(
      this as NotificationPreference,
    );
  }
}

extension NotificationPreferenceValueCopy<$R, $Out>
    on ObjectCopyWith<$R, NotificationPreference, $Out> {
  NotificationPreferenceCopyWith<$R, NotificationPreference, $Out>
  get $asNotificationPreference => $base.as(
    (v, t, t2) => _NotificationPreferenceCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class NotificationPreferenceCopyWith<
  $R,
  $In extends NotificationPreference,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({NotificationPreferenceScope? scope, bool? pushEnabled});
  NotificationPreferenceCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _NotificationPreferenceCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, NotificationPreference, $Out>
    implements
        NotificationPreferenceCopyWith<$R, NotificationPreference, $Out> {
  _NotificationPreferenceCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<NotificationPreference> $mapper =
      NotificationPreferenceMapper.ensureInitialized();
  @override
  $R call({NotificationPreferenceScope? scope, bool? pushEnabled}) => $apply(
    FieldCopyWithData({
      if (scope != null) #scope: scope,
      if (pushEnabled != null) #pushEnabled: pushEnabled,
    }),
  );
  @override
  NotificationPreference $make(CopyWithData data) => NotificationPreference(
    scope: data.get(#scope, or: $value.scope),
    pushEnabled: data.get(#pushEnabled, or: $value.pushEnabled),
  );

  @override
  NotificationPreferenceCopyWith<$R2, NotificationPreference, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _NotificationPreferenceCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

