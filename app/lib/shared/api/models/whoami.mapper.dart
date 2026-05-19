// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'whoami.dart';

class WhoAmIMapper extends ClassMapperBase<WhoAmI> {
  WhoAmIMapper._();

  static WhoAmIMapper? _instance;
  static WhoAmIMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = WhoAmIMapper._());
      MapperContainer.globals.useAll([DidMapper(), HandleMapper()]);
    }
    return _instance!;
  }

  @override
  final String id = 'WhoAmI';

  static Did _$did(WhoAmI v) => v.did;
  static dynamic _arg$did(f) => f<Did>();
  static const Field<WhoAmI, String> _f$did = Field(
    'did',
    _$did,
    arg: _arg$did,
  );
  static Handle _$handle(WhoAmI v) => v.handle;
  static dynamic _arg$handle(f) => f<Handle>();
  static const Field<WhoAmI, String> _f$handle = Field(
    'handle',
    _$handle,
    arg: _arg$handle,
  );

  @override
  final MappableFields<WhoAmI> fields = const {
    #did: _f$did,
    #handle: _f$handle,
  };

  static WhoAmI _instantiate(DecodingData data) {
    return WhoAmI(did: data.dec(_f$did), handle: data.dec(_f$handle));
  }

  @override
  final Function instantiate = _instantiate;

  static WhoAmI fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<WhoAmI>(map);
  }

  static WhoAmI fromJson(String json) {
    return ensureInitialized().decodeJson<WhoAmI>(json);
  }
}

mixin WhoAmIMappable {
  String toJson() {
    return WhoAmIMapper.ensureInitialized().encodeJson<WhoAmI>(this as WhoAmI);
  }

  Map<String, dynamic> toMap() {
    return WhoAmIMapper.ensureInitialized().encodeMap<WhoAmI>(this as WhoAmI);
  }

  WhoAmICopyWith<WhoAmI, WhoAmI, WhoAmI> get copyWith =>
      _WhoAmICopyWithImpl<WhoAmI, WhoAmI>(this as WhoAmI, $identity, $identity);
  @override
  String toString() {
    return WhoAmIMapper.ensureInitialized().stringifyValue(this as WhoAmI);
  }

  @override
  bool operator ==(Object other) {
    return WhoAmIMapper.ensureInitialized().equalsValue(this as WhoAmI, other);
  }

  @override
  int get hashCode {
    return WhoAmIMapper.ensureInitialized().hashValue(this as WhoAmI);
  }
}

extension WhoAmIValueCopy<$R, $Out> on ObjectCopyWith<$R, WhoAmI, $Out> {
  WhoAmICopyWith<$R, WhoAmI, $Out> get $asWhoAmI =>
      $base.as((v, t, t2) => _WhoAmICopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class WhoAmICopyWith<$R, $In extends WhoAmI, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({String? did, String? handle});
  WhoAmICopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _WhoAmICopyWithImpl<$R, $Out> extends ClassCopyWithBase<$R, WhoAmI, $Out>
    implements WhoAmICopyWith<$R, WhoAmI, $Out> {
  _WhoAmICopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<WhoAmI> $mapper = WhoAmIMapper.ensureInitialized();
  @override
  $R call({String? did, String? handle}) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
    }),
  );
  @override
  WhoAmI $make(CopyWithData data) => WhoAmI(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
  );

  @override
  WhoAmICopyWith<$R2, WhoAmI, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _WhoAmICopyWithImpl<$R2, $Out2>($value, $cast, t);
}

