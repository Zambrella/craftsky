// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'account_moderation.dart';

class ModerationReasonMapper extends EnumMapper<ModerationReason> {
  ModerationReasonMapper._();

  static ModerationReasonMapper? _instance;
  static ModerationReasonMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ModerationReasonMapper._());
    }
    return _instance!;
  }

  static ModerationReason fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  ModerationReason decode(dynamic value) {
    switch (value) {
      case r'harassment':
        return ModerationReason.harassment;
      case r'hate':
        return ModerationReason.hate;
      case r'spam':
        return ModerationReason.spam;
      case r'misleading':
        return ModerationReason.misleading;
      case 'suspected_ai_generated':
        return ModerationReason.suspectedAiGenerated;
      case 'adult_or_graphic':
        return ModerationReason.adultOrGraphic;
      case r'impersonation':
        return ModerationReason.impersonation;
      case 'off_topic':
        return ModerationReason.offTopic;
      case 'intellectual_property':
        return ModerationReason.intellectualProperty;
      case r'other':
        return ModerationReason.other;
      case r'unknown':
        return ModerationReason.unknown;
      default:
        return ModerationReason.values[10];
    }
  }

  @override
  dynamic encode(ModerationReason self) {
    switch (self) {
      case ModerationReason.harassment:
        return r'harassment';
      case ModerationReason.hate:
        return r'hate';
      case ModerationReason.spam:
        return r'spam';
      case ModerationReason.misleading:
        return r'misleading';
      case ModerationReason.suspectedAiGenerated:
        return 'suspected_ai_generated';
      case ModerationReason.adultOrGraphic:
        return 'adult_or_graphic';
      case ModerationReason.impersonation:
        return r'impersonation';
      case ModerationReason.offTopic:
        return 'off_topic';
      case ModerationReason.intellectualProperty:
        return 'intellectual_property';
      case ModerationReason.other:
        return r'other';
      case ModerationReason.unknown:
        return r'unknown';
    }
  }
}
extension ModerationReasonMapperExtension on ModerationReason {
  dynamic toValue() {
    ModerationReasonMapper.ensureInitialized();
    return MapperContainer.globals.toValue<ModerationReason>(this);
  }
}

class ModerationEffectTypeMapper extends EnumMapper<ModerationEffectType> {
  ModerationEffectTypeMapper._();

  static ModerationEffectTypeMapper? _instance;
  static ModerationEffectTypeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ModerationEffectTypeMapper._());
    }
    return _instance!;
  }

  static ModerationEffectType fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  ModerationEffectType decode(dynamic value) {
    switch (value) {
      case r'formalWarning':
        return ModerationEffectType.formalWarning;
      case r'visibilityWarn':
        return ModerationEffectType.visibilityWarn;
      case r'visibilityHide':
        return ModerationEffectType.visibilityHide;
      case r'visibilityTakedown':
        return ModerationEffectType.visibilityTakedown;
      case r'strike':
        return ModerationEffectType.strike;
      case r'severeSuspension':
        return ModerationEffectType.severeSuspension;
      case r'unknown':
        return ModerationEffectType.unknown;
      default:
        return ModerationEffectType.values[6];
    }
  }

  @override
  dynamic encode(ModerationEffectType self) {
    switch (self) {
      case ModerationEffectType.formalWarning:
        return r'formalWarning';
      case ModerationEffectType.visibilityWarn:
        return r'visibilityWarn';
      case ModerationEffectType.visibilityHide:
        return r'visibilityHide';
      case ModerationEffectType.visibilityTakedown:
        return r'visibilityTakedown';
      case ModerationEffectType.strike:
        return r'strike';
      case ModerationEffectType.severeSuspension:
        return r'severeSuspension';
      case ModerationEffectType.unknown:
        return r'unknown';
    }
  }
}

extension ModerationEffectTypeMapperExtension on ModerationEffectType {
  String toValue() {
    ModerationEffectTypeMapper.ensureInitialized();
    return MapperContainer.globals.toValue<ModerationEffectType>(this)
        as String;
  }
}

class ModerationEffectActionMapper extends EnumMapper<ModerationEffectAction> {
  ModerationEffectActionMapper._();

  static ModerationEffectActionMapper? _instance;
  static ModerationEffectActionMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ModerationEffectActionMapper._());
    }
    return _instance!;
  }

  static ModerationEffectAction fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  ModerationEffectAction decode(dynamic value) {
    switch (value) {
      case r'apply':
        return ModerationEffectAction.apply;
      case r'negate':
        return ModerationEffectAction.negate;
      case r'expire':
        return ModerationEffectAction.expire;
      case r'restore':
        return ModerationEffectAction.restore;
      case r'unknown':
        return ModerationEffectAction.unknown;
      default:
        return ModerationEffectAction.values[4];
    }
  }

  @override
  dynamic encode(ModerationEffectAction self) {
    switch (self) {
      case ModerationEffectAction.apply:
        return r'apply';
      case ModerationEffectAction.negate:
        return r'negate';
      case ModerationEffectAction.expire:
        return r'expire';
      case ModerationEffectAction.restore:
        return r'restore';
      case ModerationEffectAction.unknown:
        return r'unknown';
    }
  }
}

extension ModerationEffectActionMapperExtension on ModerationEffectAction {
  String toValue() {
    ModerationEffectActionMapper.ensureInitialized();
    return MapperContainer.globals.toValue<ModerationEffectAction>(this)
        as String;
  }
}

class ModerationAppealStateMapper extends EnumMapper<ModerationAppealState> {
  ModerationAppealStateMapper._();

  static ModerationAppealStateMapper? _instance;
  static ModerationAppealStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ModerationAppealStateMapper._());
    }
    return _instance!;
  }

  static ModerationAppealState fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  ModerationAppealState decode(dynamic value) {
    switch (value) {
      case r'none':
        return ModerationAppealState.none;
      case r'pending':
        return ModerationAppealState.pending;
      case r'upheld':
        return ModerationAppealState.upheld;
      case r'changed':
        return ModerationAppealState.changed;
      case r'unknown':
        return ModerationAppealState.unknown;
      default:
        return ModerationAppealState.values[4];
    }
  }

  @override
  dynamic encode(ModerationAppealState self) {
    switch (self) {
      case ModerationAppealState.none:
        return r'none';
      case ModerationAppealState.pending:
        return r'pending';
      case ModerationAppealState.upheld:
        return r'upheld';
      case ModerationAppealState.changed:
        return r'changed';
      case ModerationAppealState.unknown:
        return r'unknown';
    }
  }
}

extension ModerationAppealStateMapperExtension on ModerationAppealState {
  String toValue() {
    ModerationAppealStateMapper.ensureInitialized();
    return MapperContainer.globals.toValue<ModerationAppealState>(this)
        as String;
  }
}

class AccountStandingMapper extends ClassMapperBase<AccountStanding> {
  AccountStandingMapper._();

  static AccountStandingMapper? _instance;
  static AccountStandingMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = AccountStandingMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'AccountStanding';

  static int _$activeStrikeCount(AccountStanding v) => v.activeStrikeCount;
  static const Field<AccountStanding, int> _f$activeStrikeCount = Field(
    'activeStrikeCount',
    _$activeStrikeCount,
  );
  static int _$strikeThreshold(AccountStanding v) => v.strikeThreshold;
  static const Field<AccountStanding, int> _f$strikeThreshold = Field(
    'strikeThreshold',
    _$strikeThreshold,
  );
  static bool _$thresholdSuspended(AccountStanding v) => v.thresholdSuspended;
  static const Field<AccountStanding, bool> _f$thresholdSuspended = Field(
    'thresholdSuspended',
    _$thresholdSuspended,
  );
  static bool _$severeSuspended(AccountStanding v) => v.severeSuspended;
  static const Field<AccountStanding, bool> _f$severeSuspended = Field(
    'severeSuspended',
    _$severeSuspended,
  );
  static bool _$suspended(AccountStanding v) => v.suspended;
  static const Field<AccountStanding, bool> _f$suspended = Field(
    'suspended',
    _$suspended,
  );

  @override
  final MappableFields<AccountStanding> fields = const {
    #activeStrikeCount: _f$activeStrikeCount,
    #strikeThreshold: _f$strikeThreshold,
    #thresholdSuspended: _f$thresholdSuspended,
    #severeSuspended: _f$severeSuspended,
    #suspended: _f$suspended,
  };

  static AccountStanding _instantiate(DecodingData data) {
    return AccountStanding(
      activeStrikeCount: data.dec(_f$activeStrikeCount),
      strikeThreshold: data.dec(_f$strikeThreshold),
      thresholdSuspended: data.dec(_f$thresholdSuspended),
      severeSuspended: data.dec(_f$severeSuspended),
      suspended: data.dec(_f$suspended),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static AccountStanding fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<AccountStanding>(map);
  }

  static AccountStanding fromJson(String json) {
    return ensureInitialized().decodeJson<AccountStanding>(json);
  }
}

mixin AccountStandingMappable {
  String toJson() {
    return AccountStandingMapper.ensureInitialized()
        .encodeJson<AccountStanding>(this as AccountStanding);
  }

  Map<String, dynamic> toMap() {
    return AccountStandingMapper.ensureInitialized().encodeMap<AccountStanding>(
      this as AccountStanding,
    );
  }

  AccountStandingCopyWith<AccountStanding, AccountStanding, AccountStanding>
  get copyWith =>
      _AccountStandingCopyWithImpl<AccountStanding, AccountStanding>(
        this as AccountStanding,
        $identity,
        $identity,
      );
  @override
  String toString() {
    return AccountStandingMapper.ensureInitialized().stringifyValue(
      this as AccountStanding,
    );
  }

  @override
  bool operator ==(Object other) {
    return AccountStandingMapper.ensureInitialized().equalsValue(
      this as AccountStanding,
      other,
    );
  }

  @override
  int get hashCode {
    return AccountStandingMapper.ensureInitialized().hashValue(
      this as AccountStanding,
    );
  }
}

extension AccountStandingValueCopy<$R, $Out>
    on ObjectCopyWith<$R, AccountStanding, $Out> {
  AccountStandingCopyWith<$R, AccountStanding, $Out> get $asAccountStanding =>
      $base.as((v, t, t2) => _AccountStandingCopyWithImpl<$R, $Out>(v, t, t2));
}

abstract class AccountStandingCopyWith<$R, $In extends AccountStanding, $Out>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    int? activeStrikeCount,
    int? strikeThreshold,
    bool? thresholdSuspended,
    bool? severeSuspended,
    bool? suspended,
  });
  AccountStandingCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _AccountStandingCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, AccountStanding, $Out>
    implements AccountStandingCopyWith<$R, AccountStanding, $Out> {
  _AccountStandingCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<AccountStanding> $mapper =
      AccountStandingMapper.ensureInitialized();
  @override
  $R call({
    int? activeStrikeCount,
    int? strikeThreshold,
    bool? thresholdSuspended,
    bool? severeSuspended,
    bool? suspended,
  }) => $apply(
    FieldCopyWithData({
      if (activeStrikeCount != null) #activeStrikeCount: activeStrikeCount,
      if (strikeThreshold != null) #strikeThreshold: strikeThreshold,
      if (thresholdSuspended != null) #thresholdSuspended: thresholdSuspended,
      if (severeSuspended != null) #severeSuspended: severeSuspended,
      if (suspended != null) #suspended: suspended,
    }),
  );
  @override
  AccountStanding $make(CopyWithData data) => AccountStanding(
    activeStrikeCount: data.get(
      #activeStrikeCount,
      or: $value.activeStrikeCount,
    ),
    strikeThreshold: data.get(#strikeThreshold, or: $value.strikeThreshold),
    thresholdSuspended: data.get(
      #thresholdSuspended,
      or: $value.thresholdSuspended,
    ),
    severeSuspended: data.get(#severeSuspended, or: $value.severeSuspended),
    suspended: data.get(#suspended, or: $value.suspended),
  );

  @override
  AccountStandingCopyWith<$R2, AccountStanding, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  ) => _AccountStandingCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ModerationSafeSnapshotMapper
    extends ClassMapperBase<ModerationSafeSnapshot> {
  ModerationSafeSnapshotMapper._();

  static ModerationSafeSnapshotMapper? _instance;
  static ModerationSafeSnapshotMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ModerationSafeSnapshotMapper._());
    }
    return _instance!;
  }

  @override
  final String id = 'ModerationSafeSnapshot';

  static String _$type(ModerationSafeSnapshot v) => v.type;
  static const Field<ModerationSafeSnapshot, String> _f$type = Field(
    'type',
    _$type,
  );
  static String _$did(ModerationSafeSnapshot v) => v.did;
  static const Field<ModerationSafeSnapshot, String> _f$did = Field(
    'did',
    _$did,
  );
  static String? _$collection(ModerationSafeSnapshot v) => v.collection;
  static const Field<ModerationSafeSnapshot, String> _f$collection = Field(
    'collection',
    _$collection,
    opt: true,
  );
  static String? _$rkey(ModerationSafeSnapshot v) => v.rkey;
  static const Field<ModerationSafeSnapshot, String> _f$rkey = Field(
    'rkey',
    _$rkey,
    opt: true,
  );
  static String? _$uri(ModerationSafeSnapshot v) => v.uri;
  static const Field<ModerationSafeSnapshot, String> _f$uri = Field(
    'uri',
    _$uri,
    opt: true,
  );
  static String? _$cid(ModerationSafeSnapshot v) => v.cid;
  static const Field<ModerationSafeSnapshot, String> _f$cid = Field(
    'cid',
    _$cid,
    opt: true,
  );
  static String? _$submittedHandle(ModerationSafeSnapshot v) =>
      v.submittedHandle;
  static const Field<ModerationSafeSnapshot, String> _f$submittedHandle = Field(
    'submittedHandle',
    _$submittedHandle,
    opt: true,
  );

  @override
  final MappableFields<ModerationSafeSnapshot> fields = const {
    #type: _f$type,
    #did: _f$did,
    #collection: _f$collection,
    #rkey: _f$rkey,
    #uri: _f$uri,
    #cid: _f$cid,
    #submittedHandle: _f$submittedHandle,
  };

  static ModerationSafeSnapshot _instantiate(DecodingData data) {
    return ModerationSafeSnapshot(
      type: data.dec(_f$type),
      did: data.dec(_f$did),
      collection: data.dec(_f$collection),
      rkey: data.dec(_f$rkey),
      uri: data.dec(_f$uri),
      cid: data.dec(_f$cid),
      submittedHandle: data.dec(_f$submittedHandle),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ModerationSafeSnapshot fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ModerationSafeSnapshot>(map);
  }

  static ModerationSafeSnapshot fromJson(String json) {
    return ensureInitialized().decodeJson<ModerationSafeSnapshot>(json);
  }
}

mixin ModerationSafeSnapshotMappable {
  String toJson() {
    return ModerationSafeSnapshotMapper.ensureInitialized()
        .encodeJson<ModerationSafeSnapshot>(this as ModerationSafeSnapshot);
  }

  Map<String, dynamic> toMap() {
    return ModerationSafeSnapshotMapper.ensureInitialized()
        .encodeMap<ModerationSafeSnapshot>(this as ModerationSafeSnapshot);
  }

  ModerationSafeSnapshotCopyWith<
    ModerationSafeSnapshot,
    ModerationSafeSnapshot,
    ModerationSafeSnapshot
  >
  get copyWith =>
      _ModerationSafeSnapshotCopyWithImpl<
        ModerationSafeSnapshot,
        ModerationSafeSnapshot
      >(this as ModerationSafeSnapshot, $identity, $identity);
  @override
  String toString() {
    return ModerationSafeSnapshotMapper.ensureInitialized().stringifyValue(
      this as ModerationSafeSnapshot,
    );
  }

  @override
  bool operator ==(Object other) {
    return ModerationSafeSnapshotMapper.ensureInitialized().equalsValue(
      this as ModerationSafeSnapshot,
      other,
    );
  }

  @override
  int get hashCode {
    return ModerationSafeSnapshotMapper.ensureInitialized().hashValue(
      this as ModerationSafeSnapshot,
    );
  }
}

extension ModerationSafeSnapshotValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ModerationSafeSnapshot, $Out> {
  ModerationSafeSnapshotCopyWith<$R, ModerationSafeSnapshot, $Out>
  get $asModerationSafeSnapshot => $base.as(
    (v, t, t2) => _ModerationSafeSnapshotCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ModerationSafeSnapshotCopyWith<
  $R,
  $In extends ModerationSafeSnapshot,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? type,
    String? did,
    String? collection,
    String? rkey,
    String? uri,
    String? cid,
    String? submittedHandle,
  });
  ModerationSafeSnapshotCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ModerationSafeSnapshotCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ModerationSafeSnapshot, $Out>
    implements
        ModerationSafeSnapshotCopyWith<$R, ModerationSafeSnapshot, $Out> {
  _ModerationSafeSnapshotCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ModerationSafeSnapshot> $mapper =
      ModerationSafeSnapshotMapper.ensureInitialized();
  @override
  $R call({
    String? type,
    String? did,
    Object? collection = $none,
    Object? rkey = $none,
    Object? uri = $none,
    Object? cid = $none,
    Object? submittedHandle = $none,
  }) => $apply(
    FieldCopyWithData({
      if (type != null) #type: type,
      if (did != null) #did: did,
      if (collection != $none) #collection: collection,
      if (rkey != $none) #rkey: rkey,
      if (uri != $none) #uri: uri,
      if (cid != $none) #cid: cid,
      if (submittedHandle != $none) #submittedHandle: submittedHandle,
    }),
  );
  @override
  ModerationSafeSnapshot $make(CopyWithData data) => ModerationSafeSnapshot(
    type: data.get(#type, or: $value.type),
    did: data.get(#did, or: $value.did),
    collection: data.get(#collection, or: $value.collection),
    rkey: data.get(#rkey, or: $value.rkey),
    uri: data.get(#uri, or: $value.uri),
    cid: data.get(#cid, or: $value.cid),
    submittedHandle: data.get(#submittedHandle, or: $value.submittedHandle),
  );

  @override
  ModerationSafeSnapshotCopyWith<$R2, ModerationSafeSnapshot, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ModerationSafeSnapshotCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ModerationHistoryEffectMapper
    extends ClassMapperBase<ModerationHistoryEffect> {
  ModerationHistoryEffectMapper._();

  static ModerationHistoryEffectMapper? _instance;
  static ModerationHistoryEffectMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = ModerationHistoryEffectMapper._(),
      );
      ModerationEffectTypeMapper.ensureInitialized();
      ModerationEffectActionMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ModerationHistoryEffect';

  static ModerationEffectType _$type(ModerationHistoryEffect v) => v.type;
  static const Field<ModerationHistoryEffect, ModerationEffectType> _f$type =
      Field('type', _$type);
  static ModerationEffectAction _$action(ModerationHistoryEffect v) => v.action;
  static const Field<ModerationHistoryEffect, ModerationEffectAction>
  _f$action = Field('action', _$action);
  static DateTime? _$dueAt(ModerationHistoryEffect v) => v.dueAt;
  static const Field<ModerationHistoryEffect, DateTime> _f$dueAt = Field(
    'dueAt',
    _$dueAt,
    opt: true,
  );

  @override
  final MappableFields<ModerationHistoryEffect> fields = const {
    #type: _f$type,
    #action: _f$action,
    #dueAt: _f$dueAt,
  };

  static ModerationHistoryEffect _instantiate(DecodingData data) {
    return ModerationHistoryEffect(
      type: data.dec(_f$type),
      action: data.dec(_f$action),
      dueAt: data.dec(_f$dueAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ModerationHistoryEffect fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ModerationHistoryEffect>(map);
  }

  static ModerationHistoryEffect fromJson(String json) {
    return ensureInitialized().decodeJson<ModerationHistoryEffect>(json);
  }
}

mixin ModerationHistoryEffectMappable {
  String toJson() {
    return ModerationHistoryEffectMapper.ensureInitialized()
        .encodeJson<ModerationHistoryEffect>(this as ModerationHistoryEffect);
  }

  Map<String, dynamic> toMap() {
    return ModerationHistoryEffectMapper.ensureInitialized()
        .encodeMap<ModerationHistoryEffect>(this as ModerationHistoryEffect);
  }

  ModerationHistoryEffectCopyWith<
    ModerationHistoryEffect,
    ModerationHistoryEffect,
    ModerationHistoryEffect
  >
  get copyWith =>
      _ModerationHistoryEffectCopyWithImpl<
        ModerationHistoryEffect,
        ModerationHistoryEffect
      >(this as ModerationHistoryEffect, $identity, $identity);
  @override
  String toString() {
    return ModerationHistoryEffectMapper.ensureInitialized().stringifyValue(
      this as ModerationHistoryEffect,
    );
  }

  @override
  bool operator ==(Object other) {
    return ModerationHistoryEffectMapper.ensureInitialized().equalsValue(
      this as ModerationHistoryEffect,
      other,
    );
  }

  @override
  int get hashCode {
    return ModerationHistoryEffectMapper.ensureInitialized().hashValue(
      this as ModerationHistoryEffect,
    );
  }
}

extension ModerationHistoryEffectValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ModerationHistoryEffect, $Out> {
  ModerationHistoryEffectCopyWith<$R, ModerationHistoryEffect, $Out>
  get $asModerationHistoryEffect => $base.as(
    (v, t, t2) => _ModerationHistoryEffectCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ModerationHistoryEffectCopyWith<
  $R,
  $In extends ModerationHistoryEffect,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    ModerationEffectType? type,
    ModerationEffectAction? action,
    DateTime? dueAt,
  });
  ModerationHistoryEffectCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ModerationHistoryEffectCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ModerationHistoryEffect, $Out>
    implements
        ModerationHistoryEffectCopyWith<$R, ModerationHistoryEffect, $Out> {
  _ModerationHistoryEffectCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ModerationHistoryEffect> $mapper =
      ModerationHistoryEffectMapper.ensureInitialized();
  @override
  $R call({
    ModerationEffectType? type,
    ModerationEffectAction? action,
    Object? dueAt = $none,
  }) => $apply(
    FieldCopyWithData({
      if (type != null) #type: type,
      if (action != null) #action: action,
      if (dueAt != $none) #dueAt: dueAt,
    }),
  );
  @override
  ModerationHistoryEffect $make(CopyWithData data) => ModerationHistoryEffect(
    type: data.get(#type, or: $value.type),
    action: data.get(#action, or: $value.action),
    dueAt: data.get(#dueAt, or: $value.dueAt),
  );

  @override
  ModerationHistoryEffectCopyWith<$R2, ModerationHistoryEffect, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ModerationHistoryEffectCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ModerationHistoryEntryMapper
    extends ClassMapperBase<ModerationHistoryEntry> {
  ModerationHistoryEntryMapper._();

  static ModerationHistoryEntryMapper? _instance;
  static ModerationHistoryEntryMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ModerationHistoryEntryMapper._());
      ModerationHistoryEffectMapper.ensureInitialized();
      ModerationSafeSnapshotMapper.ensureInitialized();
      ModerationReasonMapper.ensureInitialized();
      ModerationAppealStateMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ModerationHistoryEntry';

  static String _$caseReference(ModerationHistoryEntry v) => v.caseReference;
  static const Field<ModerationHistoryEntry, String> _f$caseReference = Field(
    'caseReference',
    _$caseReference,
  );
  static String _$eventType(ModerationHistoryEntry v) => v.eventType;
  static const Field<ModerationHistoryEntry, String> _f$eventType = Field(
    'eventType',
    _$eventType,
  );
  static List<ModerationHistoryEffect> _$effects(ModerationHistoryEntry v) =>
      v.effects;
  static const Field<ModerationHistoryEntry, List<ModerationHistoryEffect>>
  _f$effects = Field('effects', _$effects);
  static ModerationSafeSnapshot _$safeSnapshot(ModerationHistoryEntry v) =>
      v.safeSnapshot;
  static const Field<ModerationHistoryEntry, ModerationSafeSnapshot>
  _f$safeSnapshot = Field('safeSnapshot', _$safeSnapshot);
  static DateTime _$occurredAt(ModerationHistoryEntry v) => v.occurredAt;
  static const Field<ModerationHistoryEntry, DateTime> _f$occurredAt = Field(
    'occurredAt',
    _$occurredAt,
  );
  static ModerationReason _$reason(ModerationHistoryEntry v) => v.reason;
  static const Field<ModerationHistoryEntry, ModerationReason> _f$reason =
      Field('reason', _$reason, opt: true, def: ModerationReason.unknown);
  static String _$userSafeDetail(ModerationHistoryEntry v) => v.userSafeDetail;
  static const Field<ModerationHistoryEntry, String> _f$userSafeDetail = Field(
    'userSafeDetail',
    _$userSafeDetail,
    opt: true,
    def: '',
  );
  static ModerationAppealState _$appealStatus(ModerationHistoryEntry v) =>
      v.appealStatus;
  static const Field<ModerationHistoryEntry, ModerationAppealState>
  _f$appealStatus = Field(
    'appealStatus',
    _$appealStatus,
    opt: true,
    def: ModerationAppealState.none,
  );

  @override
  final MappableFields<ModerationHistoryEntry> fields = const {
    #caseReference: _f$caseReference,
    #eventType: _f$eventType,
    #effects: _f$effects,
    #safeSnapshot: _f$safeSnapshot,
    #occurredAt: _f$occurredAt,
    #reason: _f$reason,
    #userSafeDetail: _f$userSafeDetail,
    #appealStatus: _f$appealStatus,
  };

  static ModerationHistoryEntry _instantiate(DecodingData data) {
    return ModerationHistoryEntry(
      caseReference: data.dec(_f$caseReference),
      eventType: data.dec(_f$eventType),
      effects: data.dec(_f$effects),
      safeSnapshot: data.dec(_f$safeSnapshot),
      occurredAt: data.dec(_f$occurredAt),
      reason: data.dec(_f$reason),
      userSafeDetail: data.dec(_f$userSafeDetail),
      appealStatus: data.dec(_f$appealStatus),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ModerationHistoryEntry fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ModerationHistoryEntry>(map);
  }

  static ModerationHistoryEntry fromJson(String json) {
    return ensureInitialized().decodeJson<ModerationHistoryEntry>(json);
  }
}

mixin ModerationHistoryEntryMappable {
  String toJson() {
    return ModerationHistoryEntryMapper.ensureInitialized()
        .encodeJson<ModerationHistoryEntry>(this as ModerationHistoryEntry);
  }

  Map<String, dynamic> toMap() {
    return ModerationHistoryEntryMapper.ensureInitialized()
        .encodeMap<ModerationHistoryEntry>(this as ModerationHistoryEntry);
  }

  ModerationHistoryEntryCopyWith<
    ModerationHistoryEntry,
    ModerationHistoryEntry,
    ModerationHistoryEntry
  >
  get copyWith =>
      _ModerationHistoryEntryCopyWithImpl<
        ModerationHistoryEntry,
        ModerationHistoryEntry
      >(this as ModerationHistoryEntry, $identity, $identity);
  @override
  String toString() {
    return ModerationHistoryEntryMapper.ensureInitialized().stringifyValue(
      this as ModerationHistoryEntry,
    );
  }

  @override
  bool operator ==(Object other) {
    return ModerationHistoryEntryMapper.ensureInitialized().equalsValue(
      this as ModerationHistoryEntry,
      other,
    );
  }

  @override
  int get hashCode {
    return ModerationHistoryEntryMapper.ensureInitialized().hashValue(
      this as ModerationHistoryEntry,
    );
  }
}

extension ModerationHistoryEntryValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ModerationHistoryEntry, $Out> {
  ModerationHistoryEntryCopyWith<$R, ModerationHistoryEntry, $Out>
  get $asModerationHistoryEntry => $base.as(
    (v, t, t2) => _ModerationHistoryEntryCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ModerationHistoryEntryCopyWith<
  $R,
  $In extends ModerationHistoryEntry,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    ModerationHistoryEffect,
    ModerationHistoryEffectCopyWith<
      $R,
      ModerationHistoryEffect,
      ModerationHistoryEffect
    >
  >
  get effects;
  ModerationSafeSnapshotCopyWith<
    $R,
    ModerationSafeSnapshot,
    ModerationSafeSnapshot
  >
  get safeSnapshot;
  $R call({
    String? caseReference,
    String? eventType,
    List<ModerationHistoryEffect>? effects,
    ModerationSafeSnapshot? safeSnapshot,
    DateTime? occurredAt,
    ModerationReason? reason,
    String? userSafeDetail,
    ModerationAppealState? appealStatus,
  });
  ModerationHistoryEntryCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ModerationHistoryEntryCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ModerationHistoryEntry, $Out>
    implements
        ModerationHistoryEntryCopyWith<$R, ModerationHistoryEntry, $Out> {
  _ModerationHistoryEntryCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ModerationHistoryEntry> $mapper =
      ModerationHistoryEntryMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    ModerationHistoryEffect,
    ModerationHistoryEffectCopyWith<
      $R,
      ModerationHistoryEffect,
      ModerationHistoryEffect
    >
  >
  get effects => ListCopyWith(
    $value.effects,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(effects: v),
  );
  @override
  ModerationSafeSnapshotCopyWith<
    $R,
    ModerationSafeSnapshot,
    ModerationSafeSnapshot
  >
  get safeSnapshot =>
      $value.safeSnapshot.copyWith.$chain((v) => call(safeSnapshot: v));
  @override
  $R call({
    String? caseReference,
    String? eventType,
    List<ModerationHistoryEffect>? effects,
    ModerationSafeSnapshot? safeSnapshot,
    DateTime? occurredAt,
    ModerationReason? reason,
    String? userSafeDetail,
    ModerationAppealState? appealStatus,
  }) => $apply(
    FieldCopyWithData({
      if (caseReference != null) #caseReference: caseReference,
      if (eventType != null) #eventType: eventType,
      if (effects != null) #effects: effects,
      if (safeSnapshot != null) #safeSnapshot: safeSnapshot,
      if (occurredAt != null) #occurredAt: occurredAt,
      if (reason != null) #reason: reason,
      if (userSafeDetail != null) #userSafeDetail: userSafeDetail,
      if (appealStatus != null) #appealStatus: appealStatus,
    }),
  );
  @override
  ModerationHistoryEntry $make(CopyWithData data) => ModerationHistoryEntry(
    caseReference: data.get(#caseReference, or: $value.caseReference),
    eventType: data.get(#eventType, or: $value.eventType),
    effects: data.get(#effects, or: $value.effects),
    safeSnapshot: data.get(#safeSnapshot, or: $value.safeSnapshot),
    occurredAt: data.get(#occurredAt, or: $value.occurredAt),
    reason: data.get(#reason, or: $value.reason),
    userSafeDetail: data.get(#userSafeDetail, or: $value.userSafeDetail),
    appealStatus: data.get(#appealStatus, or: $value.appealStatus),
  );

  @override
  ModerationHistoryEntryCopyWith<$R2, ModerationHistoryEntry, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ModerationHistoryEntryCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class ModerationHistoryPageMapper
    extends ClassMapperBase<ModerationHistoryPage> {
  ModerationHistoryPageMapper._();

  static ModerationHistoryPageMapper? _instance;
  static ModerationHistoryPageMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = ModerationHistoryPageMapper._());
      ModerationHistoryEntryMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'ModerationHistoryPage';

  static List<ModerationHistoryEntry> _$items(ModerationHistoryPage v) =>
      v.items;
  static const Field<ModerationHistoryPage, List<ModerationHistoryEntry>>
  _f$items = Field('items', _$items);
  static String? _$cursor(ModerationHistoryPage v) => v.cursor;
  static const Field<ModerationHistoryPage, String> _f$cursor = Field(
    'cursor',
    _$cursor,
    opt: true,
  );

  @override
  final MappableFields<ModerationHistoryPage> fields = const {
    #items: _f$items,
    #cursor: _f$cursor,
  };

  static ModerationHistoryPage _instantiate(DecodingData data) {
    return ModerationHistoryPage(
      items: data.dec(_f$items),
      cursor: data.dec(_f$cursor),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static ModerationHistoryPage fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<ModerationHistoryPage>(map);
  }

  static ModerationHistoryPage fromJson(String json) {
    return ensureInitialized().decodeJson<ModerationHistoryPage>(json);
  }
}

mixin ModerationHistoryPageMappable {
  String toJson() {
    return ModerationHistoryPageMapper.ensureInitialized()
        .encodeJson<ModerationHistoryPage>(this as ModerationHistoryPage);
  }

  Map<String, dynamic> toMap() {
    return ModerationHistoryPageMapper.ensureInitialized()
        .encodeMap<ModerationHistoryPage>(this as ModerationHistoryPage);
  }

  ModerationHistoryPageCopyWith<
    ModerationHistoryPage,
    ModerationHistoryPage,
    ModerationHistoryPage
  >
  get copyWith =>
      _ModerationHistoryPageCopyWithImpl<
        ModerationHistoryPage,
        ModerationHistoryPage
      >(this as ModerationHistoryPage, $identity, $identity);
  @override
  String toString() {
    return ModerationHistoryPageMapper.ensureInitialized().stringifyValue(
      this as ModerationHistoryPage,
    );
  }

  @override
  bool operator ==(Object other) {
    return ModerationHistoryPageMapper.ensureInitialized().equalsValue(
      this as ModerationHistoryPage,
      other,
    );
  }

  @override
  int get hashCode {
    return ModerationHistoryPageMapper.ensureInitialized().hashValue(
      this as ModerationHistoryPage,
    );
  }
}

extension ModerationHistoryPageValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ModerationHistoryPage, $Out> {
  ModerationHistoryPageCopyWith<$R, ModerationHistoryPage, $Out>
  get $asModerationHistoryPage => $base.as(
    (v, t, t2) => _ModerationHistoryPageCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ModerationHistoryPageCopyWith<
  $R,
  $In extends ModerationHistoryPage,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  ListCopyWith<
    $R,
    ModerationHistoryEntry,
    ModerationHistoryEntryCopyWith<
      $R,
      ModerationHistoryEntry,
      ModerationHistoryEntry
    >
  >
  get items;
  $R call({List<ModerationHistoryEntry>? items, String? cursor});
  ModerationHistoryPageCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ModerationHistoryPageCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ModerationHistoryPage, $Out>
    implements ModerationHistoryPageCopyWith<$R, ModerationHistoryPage, $Out> {
  _ModerationHistoryPageCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<ModerationHistoryPage> $mapper =
      ModerationHistoryPageMapper.ensureInitialized();
  @override
  ListCopyWith<
    $R,
    ModerationHistoryEntry,
    ModerationHistoryEntryCopyWith<
      $R,
      ModerationHistoryEntry,
      ModerationHistoryEntry
    >
  >
  get items => ListCopyWith(
    $value.items,
    (v, t) => v.copyWith.$chain(t),
    (v) => call(items: v),
  );
  @override
  $R call({List<ModerationHistoryEntry>? items, Object? cursor = $none}) =>
      $apply(
        FieldCopyWithData({
          if (items != null) #items: items,
          if (cursor != $none) #cursor: cursor,
        }),
      );
  @override
  ModerationHistoryPage $make(CopyWithData data) => ModerationHistoryPage(
    items: data.get(#items, or: $value.items),
    cursor: data.get(#cursor, or: $value.cursor),
  );

  @override
  ModerationHistoryPageCopyWith<$R2, ModerationHistoryPage, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ModerationHistoryPageCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
