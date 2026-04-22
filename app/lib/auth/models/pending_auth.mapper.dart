// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'pending_auth.dart';

class PendingAuthMapper extends ClassMapperBase<PendingAuth> {
  PendingAuthMapper._();

  static PendingAuthMapper? _instance;
  static PendingAuthMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = PendingAuthMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'PendingAuth';

  static String _$handle(PendingAuth v) => v.handle;
  static const Field<PendingAuth, String> _f$handle = Field('handle', _$handle);
  static DateTime _$startedAt(PendingAuth v) => v.startedAt;
  static const Field<PendingAuth, DateTime> _f$startedAt = Field(
    'startedAt',
    _$startedAt,
  );

  @override
  final MappableFields<PendingAuth> fields = const {
    #handle: _f$handle,
    #startedAt: _f$startedAt,
  };

  static PendingAuth _instantiate(DecodingData data) {
    return PendingAuth(
      handle: data.dec(_f$handle),
      startedAt: data.dec(_f$startedAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static PendingAuth fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<PendingAuth>(map);
  }

  static PendingAuth fromJson(String json) {
    return ensureInitialized().decodeJson<PendingAuth>(json);
  }
}

mixin PendingAuthMappable {
  String toJson() {
    return PendingAuthMapper.ensureInitialized().encodeJson<PendingAuth>(
      this as PendingAuth,
    );
  }

  Map<String, dynamic> toMap() {
    return PendingAuthMapper.ensureInitialized().encodeMap<PendingAuth>(
      this as PendingAuth,
    );
  }

  PendingAuthCopyWith<PendingAuth, PendingAuth, PendingAuth> get copyWith =>
      _PendingAuthCopyWithImpl<PendingAuth, PendingAuth>(
        this as PendingAuth,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return PendingAuthMapper.ensureInitialized().stringifyValue(
      this as PendingAuth,
    );
  }

  @override
  bool operator ==(Object other) {
    return PendingAuthMapper.ensureInitialized().equalsValue(
      this as PendingAuth,
      other,
    );
  }

  @override
  int get hashCode {
    return PendingAuthMapper.ensureInitialized().hashValue(this as PendingAuth);
  }
}

extension PendingAuthValueCopy<$R, $Out>
    on ObjectCopyWith<$R, PendingAuth, $Out> {
  PendingAuthCopyWith<$R, PendingAuth, $Out> get $asPendingAuth =>
      $base.as((v, t, t2) => _PendingAuthCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class PendingAuthCopyWith<$R, $In extends PendingAuth, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? handle, DateTime? startedAt});
  PendingAuthCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _PendingAuthCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, PendingAuth, $Out>
    implements PendingAuthCopyWith<$R, PendingAuth, $Out> {
  _PendingAuthCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<PendingAuth> $mapper =
      PendingAuthMapper.ensureInitialized();
  @override
  $R call({String? handle, DateTime? startedAt}) => $apply(
    FieldCopyWithData({
      if (handle != null) #handle: handle,
      if (startedAt != null) #startedAt: startedAt,
    }),
  );
  @override
  PendingAuth $make(CopyWithData data) => PendingAuth(
    handle: data.get(#handle, or: $value.handle),
    startedAt: data.get(#startedAt, or: $value.startedAt),
  );

  @override
  PendingAuthCopyWith<$R2, PendingAuth, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _PendingAuthCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

