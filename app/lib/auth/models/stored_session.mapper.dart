// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'stored_session.dart';

class StoredSessionMapper extends ClassMapperBase<StoredSession> {
  StoredSessionMapper._();

  static StoredSessionMapper? _instance;
  static StoredSessionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = StoredSessionMapper._());
      MapperContainer.globals.useAll([DidMapper(), HandleMapper()]);
    }
    return _instance!;
  }

  @override
  final String id = 'StoredSession';

  static String _$token(StoredSession v) => v.token;
  static const Field<StoredSession, String> _f$token = Field('token', _$token);
  static Did _$did(StoredSession v) => v.did;
  static dynamic _arg$did(f) => f<Did>();
  static const Field<StoredSession, String> _f$did = Field(
    'did',
    _$did,
    arg: _arg$did,
  );
  static Handle _$handle(StoredSession v) => v.handle;
  static dynamic _arg$handle(f) => f<Handle>();
  static const Field<StoredSession, String> _f$handle = Field(
    'handle',
    _$handle,
    arg: _arg$handle,
  );
  static int _$sessionGeneration(StoredSession v) => v.sessionGeneration;
  static const Field<StoredSession, int> _f$sessionGeneration = Field(
    'sessionGeneration',
    _$sessionGeneration,
    opt: true,
    def: 0,
  );
  static int _$lastUsedOrdinal(StoredSession v) => v.lastUsedOrdinal;
  static const Field<StoredSession, int> _f$lastUsedOrdinal = Field(
    'lastUsedOrdinal',
    _$lastUsedOrdinal,
    opt: true,
    def: 0,
  );
  static String? _$cachedDisplayName(StoredSession v) => v.cachedDisplayName;
  static const Field<StoredSession, String> _f$cachedDisplayName = Field(
    'cachedDisplayName',
    _$cachedDisplayName,
    opt: true,
  );
  static String? _$cachedAvatarUrl(StoredSession v) => v.cachedAvatarUrl;
  static const Field<StoredSession, String> _f$cachedAvatarUrl = Field(
    'cachedAvatarUrl',
    _$cachedAvatarUrl,
    opt: true,
  );

  @override
  final MappableFields<StoredSession> fields = const {
    #token: _f$token,
    #did: _f$did,
    #handle: _f$handle,
    #sessionGeneration: _f$sessionGeneration,
    #lastUsedOrdinal: _f$lastUsedOrdinal,
    #cachedDisplayName: _f$cachedDisplayName,
    #cachedAvatarUrl: _f$cachedAvatarUrl,
  };

  static StoredSession _instantiate(DecodingData data) {
    return StoredSession(
      token: data.dec(_f$token),
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      sessionGeneration: data.dec(_f$sessionGeneration),
      lastUsedOrdinal: data.dec(_f$lastUsedOrdinal),
      cachedDisplayName: data.dec(_f$cachedDisplayName),
      cachedAvatarUrl: data.dec(_f$cachedAvatarUrl),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static StoredSession fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<StoredSession>(map);
  }

  static StoredSession fromJson(String json) {
    return ensureInitialized().decodeJson<StoredSession>(json);
  }
}

mixin StoredSessionMappable {
  String toJson() {
    return StoredSessionMapper.ensureInitialized().encodeJson<StoredSession>(
      this as StoredSession,
    );
  }

  Map<String, dynamic> toMap() {
    return StoredSessionMapper.ensureInitialized().encodeMap<StoredSession>(
      this as StoredSession,
    );
  }

  StoredSessionCopyWith<StoredSession, StoredSession, StoredSession>
  get copyWith => _StoredSessionCopyWithImpl<StoredSession, StoredSession>(
    this as StoredSession,
    $identity,
    $identity,
  );
  @override
  String toString() {
    return StoredSessionMapper.ensureInitialized().stringifyValue(
      this as StoredSession,
    );
  }

  @override
  bool operator ==(Object other) {
    return StoredSessionMapper.ensureInitialized().equalsValue(
      this as StoredSession,
      other,
    );
  }

  @override
  int get hashCode {
    return StoredSessionMapper.ensureInitialized().hashValue(
      this as StoredSession,
    );
  }
}

extension StoredSessionValueCopy<$R, $Out>
    on ObjectCopyWith<$R, StoredSession, $Out> {
  StoredSessionCopyWith<$R, StoredSession, $Out> get $asStoredSession =>
      $base.as((v, t, t2) => _StoredSessionCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class StoredSessionCopyWith<$R, $In extends StoredSession, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? token,
    String? did,
    String? handle,
    int? sessionGeneration,
    int? lastUsedOrdinal,
    String? cachedDisplayName,
    String? cachedAvatarUrl,
  });
  StoredSessionCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(Then<$Out2, $R2> t);
}

class _StoredSessionCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, StoredSession, $Out>
    implements StoredSessionCopyWith<$R, StoredSession, $Out> {
  _StoredSessionCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<StoredSession> $mapper =
      StoredSessionMapper.ensureInitialized();
  @override
  $R call({
    String? token,
    String? did,
    String? handle,
    int? sessionGeneration,
    int? lastUsedOrdinal,
    Object? cachedDisplayName = $none,
    Object? cachedAvatarUrl = $none,
  }) => $apply(
    FieldCopyWithData({
      if (token != null) #token: token,
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (sessionGeneration != null) #sessionGeneration: sessionGeneration,
      if (lastUsedOrdinal != null) #lastUsedOrdinal: lastUsedOrdinal,
      if (cachedDisplayName != $none) #cachedDisplayName: cachedDisplayName,
      if (cachedAvatarUrl != $none) #cachedAvatarUrl: cachedAvatarUrl,
    }),
  );
  @override
  StoredSession $make(CopyWithData data) => StoredSession(
    token: data.get(#token, or: $value.token),
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    sessionGeneration: data.get(
      #sessionGeneration,
      or: $value.sessionGeneration,
    ),
    lastUsedOrdinal: data.get(#lastUsedOrdinal, or: $value.lastUsedOrdinal),
    cachedDisplayName: data.get(
      #cachedDisplayName,
      or: $value.cachedDisplayName,
    ),
    cachedAvatarUrl: data.get(#cachedAvatarUrl, or: $value.cachedAvatarUrl),
  );

  @override
  StoredSessionCopyWith<$R2, StoredSession, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _StoredSessionCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

