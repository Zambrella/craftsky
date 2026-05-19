// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'interaction_write_response.dart';

class InteractionWriteResponseMapper
    extends ClassMapperBase<InteractionWriteResponse> {
  InteractionWriteResponseMapper._();

  static InteractionWriteResponseMapper? _instance;
  static InteractionWriteResponseMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InteractionWriteResponseMapper._(),
      );
      MapperContainer.globals.useAll([
        AtUriMapper(),
        CidMapper(),
        RecordKeyMapper(),
      ]);
      PostRefMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InteractionWriteResponse';

  static AtUri _$uri(InteractionWriteResponse v) => v.uri;
  static dynamic _arg$uri(f) => f<AtUri>();
  static const Field<InteractionWriteResponse, String> _f$uri = Field(
    'uri',
    _$uri,
    arg: _arg$uri,
  );
  static Cid _$cid(InteractionWriteResponse v) => v.cid;
  static dynamic _arg$cid(f) => f<Cid>();
  static const Field<InteractionWriteResponse, String> _f$cid = Field(
    'cid',
    _$cid,
    arg: _arg$cid,
  );
  static RecordKey _$rkey(InteractionWriteResponse v) => v.rkey;
  static dynamic _arg$rkey(f) => f<RecordKey>();
  static const Field<InteractionWriteResponse, String> _f$rkey = Field(
    'rkey',
    _$rkey,
    arg: _arg$rkey,
  );
  static PostRef _$subject(InteractionWriteResponse v) => v.subject;
  static const Field<InteractionWriteResponse, PostRef> _f$subject = Field(
    'subject',
    _$subject,
  );
  static DateTime _$createdAt(InteractionWriteResponse v) => v.createdAt;
  static const Field<InteractionWriteResponse, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
  );

  @override
  final MappableFields<InteractionWriteResponse> fields = const {
    #uri: _f$uri,
    #cid: _f$cid,
    #rkey: _f$rkey,
    #subject: _f$subject,
    #createdAt: _f$createdAt,
  };

  static InteractionWriteResponse _instantiate(DecodingData data) {
    return InteractionWriteResponse(
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      rkey: data.dec(_f$rkey),
      subject: data.dec(_f$subject),
      createdAt: data.dec(_f$createdAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static InteractionWriteResponse fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<InteractionWriteResponse>(map);
  }

  static InteractionWriteResponse fromJson(String json) {
    return ensureInitialized().decodeJson<InteractionWriteResponse>(json);
  }
}

mixin InteractionWriteResponseMappable {
  String toJson() {
    return InteractionWriteResponseMapper.ensureInitialized()
        .encodeJson<InteractionWriteResponse>(this as InteractionWriteResponse);
  }

  Map<String, dynamic> toMap() {
    return InteractionWriteResponseMapper.ensureInitialized()
        .encodeMap<InteractionWriteResponse>(this as InteractionWriteResponse);
  }

  InteractionWriteResponseCopyWith<
    InteractionWriteResponse,
    InteractionWriteResponse,
    InteractionWriteResponse
  >
  get copyWith =>
      _InteractionWriteResponseCopyWithImpl<
        InteractionWriteResponse,
        InteractionWriteResponse
      >(this as InteractionWriteResponse, $identity, $identity);
  @override
  String toString() {
    return InteractionWriteResponseMapper.ensureInitialized().stringifyValue(
      this as InteractionWriteResponse,
    );
  }

  @override
  bool operator ==(Object other) {
    return InteractionWriteResponseMapper.ensureInitialized().equalsValue(
      this as InteractionWriteResponse,
      other,
    );
  }

  @override
  int get hashCode {
    return InteractionWriteResponseMapper.ensureInitialized().hashValue(
      this as InteractionWriteResponse,
    );
  }
}

extension InteractionWriteResponseValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InteractionWriteResponse, $Out> {
  InteractionWriteResponseCopyWith<$R, InteractionWriteResponse, $Out>
  get $asInteractionWriteResponse => $base.as(
    (v, t, t2) => _InteractionWriteResponseCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InteractionWriteResponseCopyWith<
  $R,
  $In extends InteractionWriteResponse,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  PostRefCopyWith<$R, PostRef, PostRef> get subject;
  $R call({
    String? uri,
    String? cid,
    String? rkey,
    PostRef? subject,
    DateTime? createdAt,
  });
  InteractionWriteResponseCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InteractionWriteResponseCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InteractionWriteResponse, $Out>
    implements
        InteractionWriteResponseCopyWith<$R, InteractionWriteResponse, $Out> {
  _InteractionWriteResponseCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InteractionWriteResponse> $mapper =
      InteractionWriteResponseMapper.ensureInitialized();
  @override
  PostRefCopyWith<$R, PostRef, PostRef> get subject =>
      $value.subject.copyWith.$chain((v) => call(subject: v));
  @override
  $R call({
    String? uri,
    String? cid,
    String? rkey,
    PostRef? subject,
    DateTime? createdAt,
  }) => $apply(
    FieldCopyWithData({
      if (uri != null) #uri: uri,
      if (cid != null) #cid: cid,
      if (rkey != null) #rkey: rkey,
      if (subject != null) #subject: subject,
      if (createdAt != null) #createdAt: createdAt,
    }),
  );
  @override
  InteractionWriteResponse $make(CopyWithData data) => InteractionWriteResponse(
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    rkey: data.get(#rkey, or: $value.rkey),
    subject: data.get(#subject, or: $value.subject),
    createdAt: data.get(#createdAt, or: $value.createdAt),
  );

  @override
  InteractionWriteResponseCopyWith<$R2, InteractionWriteResponse, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InteractionWriteResponseCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

