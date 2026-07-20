// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'craftsky_notification.dart';

class NotificationActorMapper extends ClassMapperBase<NotificationActor> {
  NotificationActorMapper._();

  static NotificationActorMapper? _instance;
  static NotificationActorMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = NotificationActorMapper._());
      MapperContainer.globals.useAll([
        DidMapper(),
        HandleMapper(),
        CidMapper(),
      ]);
    }
    return _instance!;
  }

  @override
  final String id = 'NotificationActor';

  static Did _$did(NotificationActor v) => v.did;
  static const Field<NotificationActor, Did> _f$did = Field('did', _$did);
  static Handle _$handle(NotificationActor v) => v.handle;
  static const Field<NotificationActor, Handle> _f$handle = Field(
    'handle',
    _$handle,
  );
  static String? _$displayName(NotificationActor v) => v.displayName;
  static const Field<NotificationActor, String> _f$displayName = Field(
    'displayName',
    _$displayName,
    opt: true,
  );
  static String? _$avatar(NotificationActor v) => v.avatar;
  static const Field<NotificationActor, String> _f$avatar = Field(
    'avatar',
    _$avatar,
    opt: true,
  );
  static Cid? _$avatarCid(NotificationActor v) => v.avatarCid;
  static const Field<NotificationActor, Cid> _f$avatarCid = Field(
    'avatarCid',
    _$avatarCid,
    opt: true,
  );
  static bool _$viewerIsFollowing(NotificationActor v) => v.viewerIsFollowing;
  static const Field<NotificationActor, bool> _f$viewerIsFollowing = Field(
    'viewerIsFollowing',
    _$viewerIsFollowing,
    opt: true,
    def: false,
  );
  static bool _$available(NotificationActor v) => v.available;
  static const Field<NotificationActor, bool> _f$available = Field(
    'available',
    _$available,
    opt: true,
    def: true,
  );
  static bool? _$muted(NotificationActor v) => v.muted;
  static const Field<NotificationActor, bool> _f$muted = Field(
    'muted',
    _$muted,
    opt: true,
  );
  static bool? _$blocking(NotificationActor v) => v.blocking;
  static const Field<NotificationActor, bool> _f$blocking = Field(
    'blocking',
    _$blocking,
    opt: true,
  );
  static bool? _$blockedBy(NotificationActor v) => v.blockedBy;
  static const Field<NotificationActor, bool> _f$blockedBy = Field(
    'blockedBy',
    _$blockedBy,
    opt: true,
  );

  @override
  final MappableFields<NotificationActor> fields = const {
    #did: _f$did,
    #handle: _f$handle,
    #displayName: _f$displayName,
    #avatar: _f$avatar,
    #avatarCid: _f$avatarCid,
    #viewerIsFollowing: _f$viewerIsFollowing,
    #available: _f$available,
    #muted: _f$muted,
    #blocking: _f$blocking,
    #blockedBy: _f$blockedBy,
  };

  static NotificationActor _instantiate(DecodingData data) {
    return NotificationActor(
      did: data.dec(_f$did),
      handle: data.dec(_f$handle),
      displayName: data.dec(_f$displayName),
      avatar: data.dec(_f$avatar),
      avatarCid: data.dec(_f$avatarCid),
      viewerIsFollowing: data.dec(_f$viewerIsFollowing),
      available: data.dec(_f$available),
      muted: data.dec(_f$muted),
      blocking: data.dec(_f$blocking),
      blockedBy: data.dec(_f$blockedBy),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static NotificationActor fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<NotificationActor>(map);
  }

  static NotificationActor fromJson(String json) {
    return ensureInitialized().decodeJson<NotificationActor>(json);
  }
}

mixin NotificationActorMappable {
  String toJson() {
    return NotificationActorMapper.ensureInitialized()
        .encodeJson<NotificationActor>(this as NotificationActor);
  }

  Map<String, dynamic> toMap() {
    return NotificationActorMapper.ensureInitialized()
        .encodeMap<NotificationActor>(this as NotificationActor);
  }

  NotificationActorCopyWith<
    NotificationActor,
    NotificationActor,
    NotificationActor
  >
  get copyWith =>
      _NotificationActorCopyWithImpl<NotificationActor, NotificationActor>(
        this as NotificationActor,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return NotificationActorMapper.ensureInitialized().equalsValue(
      this as NotificationActor,
      other,
    );
  }

  @override
  int get hashCode {
    return NotificationActorMapper.ensureInitialized().hashValue(
      this as NotificationActor,
    );
  }
}

extension NotificationActorValueCopy<$R, $Out>
    on ObjectCopyWith<$R, NotificationActor, $Out> {
  NotificationActorCopyWith<$R, NotificationActor, $Out>
  get $asNotificationActor => $base.as(
    (v, t, t2) => _NotificationActorCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class NotificationActorCopyWith<
  $R,
  $In extends NotificationActor,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    Did? did,
    Handle? handle,
    String? displayName,
    String? avatar,
    Cid? avatarCid,
    bool? viewerIsFollowing,
    bool? available,
    bool? muted,
    bool? blocking,
    bool? blockedBy,
  });
  NotificationActorCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _NotificationActorCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, NotificationActor, $Out>
    implements NotificationActorCopyWith<$R, NotificationActor, $Out> {
  _NotificationActorCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<NotificationActor> $mapper =
      NotificationActorMapper.ensureInitialized();
  @override
  $R call({
    Did? did,
    Handle? handle,
    Object? displayName = $none,
    Object? avatar = $none,
    Object? avatarCid = $none,
    bool? viewerIsFollowing,
    bool? available,
    Object? muted = $none,
    Object? blocking = $none,
    Object? blockedBy = $none,
  }) => $apply(
    FieldCopyWithData({
      if (did != null) #did: did,
      if (handle != null) #handle: handle,
      if (displayName != $none) #displayName: displayName,
      if (avatar != $none) #avatar: avatar,
      if (avatarCid != $none) #avatarCid: avatarCid,
      if (viewerIsFollowing != null) #viewerIsFollowing: viewerIsFollowing,
      if (available != null) #available: available,
      if (muted != $none) #muted: muted,
      if (blocking != $none) #blocking: blocking,
      if (blockedBy != $none) #blockedBy: blockedBy,
    }),
  );
  @override
  NotificationActor $make(CopyWithData data) => NotificationActor(
    did: data.get(#did, or: $value.did),
    handle: data.get(#handle, or: $value.handle),
    displayName: data.get(#displayName, or: $value.displayName),
    avatar: data.get(#avatar, or: $value.avatar),
    avatarCid: data.get(#avatarCid, or: $value.avatarCid),
    viewerIsFollowing: data.get(
      #viewerIsFollowing,
      or: $value.viewerIsFollowing,
    ),
    available: data.get(#available, or: $value.available),
    muted: data.get(#muted, or: $value.muted),
    blocking: data.get(#blocking, or: $value.blocking),
    blockedBy: data.get(#blockedBy, or: $value.blockedBy),
  );

  @override
  NotificationActorCopyWith<$R2, NotificationActor, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _NotificationActorCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class NotificationReplyRefMapper extends ClassMapperBase<NotificationReplyRef> {
  NotificationReplyRefMapper._();

  static NotificationReplyRefMapper? _instance;
  static NotificationReplyRefMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = NotificationReplyRefMapper._());
      MapperContainer.globals.useAll([
        AtUriMapper(),
        CidMapper(),
        RecordKeyMapper(),
      ]);
    }
    return _instance!;
  }

  @override
  final String id = 'NotificationReplyRef';

  static AtUri _$uri(NotificationReplyRef v) => v.uri;
  static const Field<NotificationReplyRef, AtUri> _f$uri = Field('uri', _$uri);
  static Cid _$cid(NotificationReplyRef v) => v.cid;
  static const Field<NotificationReplyRef, Cid> _f$cid = Field('cid', _$cid);
  static RecordKey _$rkey(NotificationReplyRef v) => v.rkey;
  static const Field<NotificationReplyRef, RecordKey> _f$rkey = Field(
    'rkey',
    _$rkey,
  );

  @override
  final MappableFields<NotificationReplyRef> fields = const {
    #uri: _f$uri,
    #cid: _f$cid,
    #rkey: _f$rkey,
  };

  static NotificationReplyRef _instantiate(DecodingData data) {
    return NotificationReplyRef(
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      rkey: data.dec(_f$rkey),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static NotificationReplyRef fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<NotificationReplyRef>(map);
  }

  static NotificationReplyRef fromJson(String json) {
    return ensureInitialized().decodeJson<NotificationReplyRef>(json);
  }
}

mixin NotificationReplyRefMappable {
  String toJson() {
    return NotificationReplyRefMapper.ensureInitialized()
        .encodeJson<NotificationReplyRef>(this as NotificationReplyRef);
  }

  Map<String, dynamic> toMap() {
    return NotificationReplyRefMapper.ensureInitialized()
        .encodeMap<NotificationReplyRef>(this as NotificationReplyRef);
  }

  NotificationReplyRefCopyWith<
    NotificationReplyRef,
    NotificationReplyRef,
    NotificationReplyRef
  >
  get copyWith =>
      _NotificationReplyRefCopyWithImpl<
        NotificationReplyRef,
        NotificationReplyRef
      >(this as NotificationReplyRef, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return NotificationReplyRefMapper.ensureInitialized().equalsValue(
      this as NotificationReplyRef,
      other,
    );
  }

  @override
  int get hashCode {
    return NotificationReplyRefMapper.ensureInitialized().hashValue(
      this as NotificationReplyRef,
    );
  }
}

extension NotificationReplyRefValueCopy<$R, $Out>
    on ObjectCopyWith<$R, NotificationReplyRef, $Out> {
  NotificationReplyRefCopyWith<$R, NotificationReplyRef, $Out>
  get $asNotificationReplyRef => $base.as(
    (v, t, t2) => _NotificationReplyRefCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class NotificationReplyRefCopyWith<
  $R,
  $In extends NotificationReplyRef,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({AtUri? uri, Cid? cid, RecordKey? rkey});
  NotificationReplyRefCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _NotificationReplyRefCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, NotificationReplyRef, $Out>
    implements NotificationReplyRefCopyWith<$R, NotificationReplyRef, $Out> {
  _NotificationReplyRefCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<NotificationReplyRef> $mapper =
      NotificationReplyRefMapper.ensureInitialized();
  @override
  $R call({AtUri? uri, Cid? cid, RecordKey? rkey}) => $apply(
    FieldCopyWithData({
      if (uri != null) #uri: uri,
      if (cid != null) #cid: cid,
      if (rkey != null) #rkey: rkey,
    }),
  );
  @override
  NotificationReplyRef $make(CopyWithData data) => NotificationReplyRef(
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    rkey: data.get(#rkey, or: $value.rkey),
  );

  @override
  NotificationReplyRefCopyWith<$R2, NotificationReplyRef, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _NotificationReplyRefCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class NotificationCommonMapper extends ClassMapperBase<NotificationCommon> {
  NotificationCommonMapper._();

  static NotificationCommonMapper? _instance;
  static NotificationCommonMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = NotificationCommonMapper._());
      MapperContainer.globals.useAll([
        AtUriMapper(),
        CidMapper(),
        RecordKeyMapper(),
      ]);
      NotificationActorMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'NotificationCommon';

  static String _$id(NotificationCommon v) => v.id;
  static const Field<NotificationCommon, String> _f$id = Field('id', _$id);
  static AtUri _$uri(NotificationCommon v) => v.uri;
  static const Field<NotificationCommon, AtUri> _f$uri = Field('uri', _$uri);
  static Cid _$cid(NotificationCommon v) => v.cid;
  static const Field<NotificationCommon, Cid> _f$cid = Field('cid', _$cid);
  static RecordKey _$rkey(NotificationCommon v) => v.rkey;
  static const Field<NotificationCommon, RecordKey> _f$rkey = Field(
    'rkey',
    _$rkey,
  );
  static NotificationActor _$actor(NotificationCommon v) => v.actor;
  static const Field<NotificationCommon, NotificationActor> _f$actor = Field(
    'actor',
    _$actor,
  );
  static DateTime _$createdAt(NotificationCommon v) => v.createdAt;
  static const Field<NotificationCommon, DateTime> _f$createdAt = Field(
    'createdAt',
    _$createdAt,
  );
  static DateTime _$indexedAt(NotificationCommon v) => v.indexedAt;
  static const Field<NotificationCommon, DateTime> _f$indexedAt = Field(
    'indexedAt',
    _$indexedAt,
  );

  @override
  final MappableFields<NotificationCommon> fields = const {
    #id: _f$id,
    #uri: _f$uri,
    #cid: _f$cid,
    #rkey: _f$rkey,
    #actor: _f$actor,
    #createdAt: _f$createdAt,
    #indexedAt: _f$indexedAt,
  };

  static NotificationCommon _instantiate(DecodingData data) {
    return NotificationCommon(
      id: data.dec(_f$id),
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      rkey: data.dec(_f$rkey),
      actor: data.dec(_f$actor),
      createdAt: data.dec(_f$createdAt),
      indexedAt: data.dec(_f$indexedAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static NotificationCommon fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<NotificationCommon>(map);
  }

  static NotificationCommon fromJson(String json) {
    return ensureInitialized().decodeJson<NotificationCommon>(json);
  }
}

mixin NotificationCommonMappable {
  String toJson() {
    return NotificationCommonMapper.ensureInitialized()
        .encodeJson<NotificationCommon>(this as NotificationCommon);
  }

  Map<String, dynamic> toMap() {
    return NotificationCommonMapper.ensureInitialized()
        .encodeMap<NotificationCommon>(this as NotificationCommon);
  }

  NotificationCommonCopyWith<
    NotificationCommon,
    NotificationCommon,
    NotificationCommon
  >
  get copyWith =>
      _NotificationCommonCopyWithImpl<NotificationCommon, NotificationCommon>(
        this as NotificationCommon,
        $identity,
        $identity,
      );
  @override
  bool operator ==(Object other) {
    return NotificationCommonMapper.ensureInitialized().equalsValue(
      this as NotificationCommon,
      other,
    );
  }

  @override
  int get hashCode {
    return NotificationCommonMapper.ensureInitialized().hashValue(
      this as NotificationCommon,
    );
  }
}

extension NotificationCommonValueCopy<$R, $Out>
    on ObjectCopyWith<$R, NotificationCommon, $Out> {
  NotificationCommonCopyWith<$R, NotificationCommon, $Out>
  get $asNotificationCommon => $base.as(
    (v, t, t2) => _NotificationCommonCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class NotificationCommonCopyWith<
  $R,
  $In extends NotificationCommon,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  NotificationActorCopyWith<$R, NotificationActor, NotificationActor> get actor;
  $R call({
    String? id,
    AtUri? uri,
    Cid? cid,
    RecordKey? rkey,
    NotificationActor? actor,
    DateTime? createdAt,
    DateTime? indexedAt,
  });
  NotificationCommonCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _NotificationCommonCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, NotificationCommon, $Out>
    implements NotificationCommonCopyWith<$R, NotificationCommon, $Out> {
  _NotificationCommonCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<NotificationCommon> $mapper =
      NotificationCommonMapper.ensureInitialized();
  @override
  NotificationActorCopyWith<$R, NotificationActor, NotificationActor>
  get actor => $value.actor.copyWith.$chain((v) => call(actor: v));
  @override
  $R call({
    String? id,
    AtUri? uri,
    Cid? cid,
    RecordKey? rkey,
    NotificationActor? actor,
    DateTime? createdAt,
    DateTime? indexedAt,
  }) => $apply(
    FieldCopyWithData({
      if (id != null) #id: id,
      if (uri != null) #uri: uri,
      if (cid != null) #cid: cid,
      if (rkey != null) #rkey: rkey,
      if (actor != null) #actor: actor,
      if (createdAt != null) #createdAt: createdAt,
      if (indexedAt != null) #indexedAt: indexedAt,
    }),
  );
  @override
  NotificationCommon $make(CopyWithData data) => NotificationCommon(
    id: data.get(#id, or: $value.id),
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    rkey: data.get(#rkey, or: $value.rkey),
    actor: data.get(#actor, or: $value.actor),
    createdAt: data.get(#createdAt, or: $value.createdAt),
    indexedAt: data.get(#indexedAt, or: $value.indexedAt),
  );

  @override
  NotificationCommonCopyWith<$R2, NotificationCommon, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _NotificationCommonCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

