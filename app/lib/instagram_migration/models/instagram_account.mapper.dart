// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'instagram_account.dart';

class InstagramAccountLinkStateMapper
    extends EnumMapper<InstagramAccountLinkState> {
  InstagramAccountLinkStateMapper._();

  static InstagramAccountLinkStateMapper? _instance;
  static InstagramAccountLinkStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramAccountLinkStateMapper._(),
      );
    }
    return _instance!;
  }

  static InstagramAccountLinkState fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  InstagramAccountLinkState decode(dynamic value) {
    switch (value) {
      case r'active':
        return InstagramAccountLinkState.active;
      case r'membershipInactive':
        return InstagramAccountLinkState.membershipInactive;
      case r'revoked':
        return InstagramAccountLinkState.revoked;
      case r'superseded':
        return InstagramAccountLinkState.superseded;
      case r'disputed':
        return InstagramAccountLinkState.disputed;
      case r'unknown':
        return InstagramAccountLinkState.unknown;
      default:
        return InstagramAccountLinkState.values[5];
    }
  }

  @override
  dynamic encode(InstagramAccountLinkState self) {
    switch (self) {
      case InstagramAccountLinkState.active:
        return r'active';
      case InstagramAccountLinkState.membershipInactive:
        return r'membershipInactive';
      case InstagramAccountLinkState.revoked:
        return r'revoked';
      case InstagramAccountLinkState.superseded:
        return r'superseded';
      case InstagramAccountLinkState.disputed:
        return r'disputed';
      case InstagramAccountLinkState.unknown:
        return r'unknown';
    }
  }
}

extension InstagramAccountLinkStateMapperExtension
    on InstagramAccountLinkState {
  String toValue() {
    InstagramAccountLinkStateMapper.ensureInitialized();
    return MapperContainer.globals.toValue<InstagramAccountLinkState>(this)
        as String;
  }
}

class InstagramAccountLinkMapper extends ClassMapperBase<InstagramAccountLink> {
  InstagramAccountLinkMapper._();

  static InstagramAccountLinkMapper? _instance;
  static InstagramAccountLinkMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = InstagramAccountLinkMapper._());
      InstagramAccountLinkStateMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramAccountLink';

  static InstagramAccountLinkState _$state(InstagramAccountLink v) => v.state;
  static const Field<InstagramAccountLink, InstagramAccountLinkState> _f$state =
      Field('state', _$state);
  static String _$username(InstagramAccountLink v) => v.username;
  static const Field<InstagramAccountLink, String> _f$username = Field(
    'username',
    _$username,
  );
  static bool _$discoverable(InstagramAccountLink v) => v.discoverable;
  static const Field<InstagramAccountLink, bool> _f$discoverable = Field(
    'discoverable',
    _$discoverable,
  );
  static bool _$conflictPending(InstagramAccountLink v) => v.conflictPending;
  static const Field<InstagramAccountLink, bool> _f$conflictPending = Field(
    'conflictPending',
    _$conflictPending,
  );
  static bool _$reactivationRequired(InstagramAccountLink v) =>
      v.reactivationRequired;
  static const Field<InstagramAccountLink, bool> _f$reactivationRequired =
      Field('reactivationRequired', _$reactivationRequired);
  static DateTime _$verifiedAt(InstagramAccountLink v) => v.verifiedAt;
  static const Field<InstagramAccountLink, DateTime> _f$verifiedAt = Field(
    'verifiedAt',
    _$verifiedAt,
  );

  @override
  final MappableFields<InstagramAccountLink> fields = const {
    #state: _f$state,
    #username: _f$username,
    #discoverable: _f$discoverable,
    #conflictPending: _f$conflictPending,
    #reactivationRequired: _f$reactivationRequired,
    #verifiedAt: _f$verifiedAt,
  };

  static InstagramAccountLink _instantiate(DecodingData data) {
    return InstagramAccountLink(
      state: data.dec(_f$state),
      username: data.dec(_f$username),
      discoverable: data.dec(_f$discoverable),
      conflictPending: data.dec(_f$conflictPending),
      reactivationRequired: data.dec(_f$reactivationRequired),
      verifiedAt: data.dec(_f$verifiedAt),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static InstagramAccountLink fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<InstagramAccountLink>(map);
  }

  static InstagramAccountLink fromJson(String json) {
    return ensureInitialized().decodeJson<InstagramAccountLink>(json);
  }
}

mixin InstagramAccountLinkMappable {
  InstagramAccountLinkCopyWith<
    InstagramAccountLink,
    InstagramAccountLink,
    InstagramAccountLink
  >
  get copyWith =>
      _InstagramAccountLinkCopyWithImpl<
        InstagramAccountLink,
        InstagramAccountLink
      >(this as InstagramAccountLink, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramAccountLinkMapper.ensureInitialized().equalsValue(
      this as InstagramAccountLink,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramAccountLinkMapper.ensureInitialized().hashValue(
      this as InstagramAccountLink,
    );
  }
}

extension InstagramAccountLinkValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramAccountLink, $Out> {
  InstagramAccountLinkCopyWith<$R, InstagramAccountLink, $Out>
  get $asInstagramAccountLink => $base.as(
    (v, t, t2) => _InstagramAccountLinkCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramAccountLinkCopyWith<
  $R,
  $In extends InstagramAccountLink,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    InstagramAccountLinkState? state,
    String? username,
    bool? discoverable,
    bool? conflictPending,
    bool? reactivationRequired,
    DateTime? verifiedAt,
  });
  InstagramAccountLinkCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramAccountLinkCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramAccountLink, $Out>
    implements InstagramAccountLinkCopyWith<$R, InstagramAccountLink, $Out> {
  _InstagramAccountLinkCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InstagramAccountLink> $mapper =
      InstagramAccountLinkMapper.ensureInitialized();
  @override
  $R call({
    InstagramAccountLinkState? state,
    String? username,
    bool? discoverable,
    bool? conflictPending,
    bool? reactivationRequired,
    DateTime? verifiedAt,
  }) => $apply(
    FieldCopyWithData({
      if (state != null) #state: state,
      if (username != null) #username: username,
      if (discoverable != null) #discoverable: discoverable,
      if (conflictPending != null) #conflictPending: conflictPending,
      if (reactivationRequired != null)
        #reactivationRequired: reactivationRequired,
      if (verifiedAt != null) #verifiedAt: verifiedAt,
    }),
  );
  @override
  InstagramAccountLink $make(CopyWithData data) => InstagramAccountLink(
    state: data.get(#state, or: $value.state),
    username: data.get(#username, or: $value.username),
    discoverable: data.get(#discoverable, or: $value.discoverable),
    conflictPending: data.get(#conflictPending, or: $value.conflictPending),
    reactivationRequired: data.get(
      #reactivationRequired,
      or: $value.reactivationRequired,
    ),
    verifiedAt: data.get(#verifiedAt, or: $value.verifiedAt),
  );

  @override
  InstagramAccountLinkCopyWith<$R2, InstagramAccountLink, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramAccountLinkCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramAccountStatusMapper
    extends ClassMapperBase<InstagramAccountStatus> {
  InstagramAccountStatusMapper._();

  static InstagramAccountStatusMapper? _instance;
  static InstagramAccountStatusMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(_instance = InstagramAccountStatusMapper._());
      InstagramAccountLinkMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramAccountStatus';

  static bool _$integrationAvailable(InstagramAccountStatus v) =>
      v.integrationAvailable;
  static const Field<InstagramAccountStatus, bool> _f$integrationAvailable =
      Field('integrationAvailable', _$integrationAvailable);
  static InstagramAccountLink? _$account(InstagramAccountStatus v) => v.account;
  static const Field<InstagramAccountStatus, InstagramAccountLink> _f$account =
      Field('account', _$account);

  @override
  final MappableFields<InstagramAccountStatus> fields = const {
    #integrationAvailable: _f$integrationAvailable,
    #account: _f$account,
  };

  static InstagramAccountStatus _instantiate(DecodingData data) {
    return InstagramAccountStatus(
      integrationAvailable: data.dec(_f$integrationAvailable),
      account: data.dec(_f$account),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static InstagramAccountStatus fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<InstagramAccountStatus>(map);
  }

  static InstagramAccountStatus fromJson(String json) {
    return ensureInitialized().decodeJson<InstagramAccountStatus>(json);
  }
}

mixin InstagramAccountStatusMappable {
  InstagramAccountStatusCopyWith<
    InstagramAccountStatus,
    InstagramAccountStatus,
    InstagramAccountStatus
  >
  get copyWith =>
      _InstagramAccountStatusCopyWithImpl<
        InstagramAccountStatus,
        InstagramAccountStatus
      >(this as InstagramAccountStatus, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramAccountStatusMapper.ensureInitialized().equalsValue(
      this as InstagramAccountStatus,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramAccountStatusMapper.ensureInitialized().hashValue(
      this as InstagramAccountStatus,
    );
  }
}

extension InstagramAccountStatusValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramAccountStatus, $Out> {
  InstagramAccountStatusCopyWith<$R, InstagramAccountStatus, $Out>
  get $asInstagramAccountStatus => $base.as(
    (v, t, t2) => _InstagramAccountStatusCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramAccountStatusCopyWith<
  $R,
  $In extends InstagramAccountStatus,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  InstagramAccountLinkCopyWith<$R, InstagramAccountLink, InstagramAccountLink>?
  get account;
  $R call({bool? integrationAvailable, InstagramAccountLink? account});
  InstagramAccountStatusCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramAccountStatusCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramAccountStatus, $Out>
    implements
        InstagramAccountStatusCopyWith<$R, InstagramAccountStatus, $Out> {
  _InstagramAccountStatusCopyWithImpl(super.value, super.then, super.then2);

  @override
  late final ClassMapperBase<InstagramAccountStatus> $mapper =
      InstagramAccountStatusMapper.ensureInitialized();
  @override
  InstagramAccountLinkCopyWith<$R, InstagramAccountLink, InstagramAccountLink>?
  get account => $value.account?.copyWith.$chain((v) => call(account: v));
  @override
  $R call({bool? integrationAvailable, Object? account = $none}) => $apply(
    FieldCopyWithData({
      if (integrationAvailable != null)
        #integrationAvailable: integrationAvailable,
      if (account != $none) #account: account,
    }),
  );
  @override
  InstagramAccountStatus $make(CopyWithData data) => InstagramAccountStatus(
    integrationAvailable: data.get(
      #integrationAvailable,
      or: $value.integrationAvailable,
    ),
    account: data.get(#account, or: $value.account),
  );

  @override
  InstagramAccountStatusCopyWith<$R2, InstagramAccountStatus, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramAccountStatusCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

class InstagramVerificationConfirmationMapper
    extends ClassMapperBase<InstagramVerificationConfirmation> {
  InstagramVerificationConfirmationMapper._();

  static InstagramVerificationConfirmationMapper? _instance;
  static InstagramVerificationConfirmationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramVerificationConfirmationMapper._(),
      );
      InstagramVerificationStateMapper.ensureInitialized();
      InstagramAccountLinkMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramVerificationConfirmation';

  static InstagramVerificationState _$state(
    InstagramVerificationConfirmation v,
  ) => v.state;
  static const Field<
    InstagramVerificationConfirmation,
    InstagramVerificationState
  >
  _f$state = Field('state', _$state);
  static InstagramAccountLink _$account(InstagramVerificationConfirmation v) =>
      v.account;
  static const Field<InstagramVerificationConfirmation, InstagramAccountLink>
  _f$account = Field('account', _$account);

  @override
  final MappableFields<InstagramVerificationConfirmation> fields = const {
    #state: _f$state,
    #account: _f$account,
  };

  static InstagramVerificationConfirmation _instantiate(DecodingData data) {
    return InstagramVerificationConfirmation(
      state: data.dec(_f$state),
      account: data.dec(_f$account),
    );
  }

  @override
  final Function instantiate = _instantiate;

  static InstagramVerificationConfirmation fromMap(Map<String, dynamic> map) {
    return ensureInitialized().decodeMap<InstagramVerificationConfirmation>(
      map,
    );
  }

  static InstagramVerificationConfirmation fromJson(String json) {
    return ensureInitialized().decodeJson<InstagramVerificationConfirmation>(
      json,
    );
  }
}

mixin InstagramVerificationConfirmationMappable {
  InstagramVerificationConfirmationCopyWith<
    InstagramVerificationConfirmation,
    InstagramVerificationConfirmation,
    InstagramVerificationConfirmation
  >
  get copyWith =>
      _InstagramVerificationConfirmationCopyWithImpl<
        InstagramVerificationConfirmation,
        InstagramVerificationConfirmation
      >(this as InstagramVerificationConfirmation, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramVerificationConfirmationMapper.ensureInitialized()
        .equalsValue(this as InstagramVerificationConfirmation, other);
  }

  @override
  int get hashCode {
    return InstagramVerificationConfirmationMapper.ensureInitialized()
        .hashValue(this as InstagramVerificationConfirmation);
  }
}

extension InstagramVerificationConfirmationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramVerificationConfirmation, $Out> {
  InstagramVerificationConfirmationCopyWith<
    $R,
    InstagramVerificationConfirmation,
    $Out
  >
  get $asInstagramVerificationConfirmation => $base.as(
    (v, t, t2) =>
        _InstagramVerificationConfirmationCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramVerificationConfirmationCopyWith<
  $R,
  $In extends InstagramVerificationConfirmation,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  InstagramAccountLinkCopyWith<$R, InstagramAccountLink, InstagramAccountLink>
  get account;
  $R call({InstagramVerificationState? state, InstagramAccountLink? account});
  InstagramVerificationConfirmationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramVerificationConfirmationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramVerificationConfirmation, $Out>
    implements
        InstagramVerificationConfirmationCopyWith<
          $R,
          InstagramVerificationConfirmation,
          $Out
        > {
  _InstagramVerificationConfirmationCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<InstagramVerificationConfirmation> $mapper =
      InstagramVerificationConfirmationMapper.ensureInitialized();
  @override
  InstagramAccountLinkCopyWith<$R, InstagramAccountLink, InstagramAccountLink>
  get account => $value.account.copyWith.$chain((v) => call(account: v));
  @override
  $R call({InstagramVerificationState? state, InstagramAccountLink? account}) =>
      $apply(
        FieldCopyWithData({
          if (state != null) #state: state,
          if (account != null) #account: account,
        }),
      );
  @override
  InstagramVerificationConfirmation $make(CopyWithData data) =>
      InstagramVerificationConfirmation(
        state: data.get(#state, or: $value.state),
        account: data.get(#account, or: $value.account),
      );

  @override
  InstagramVerificationConfirmationCopyWith<
    $R2,
    InstagramVerificationConfirmation,
    $Out2
  >
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramVerificationConfirmationCopyWithImpl<$R2, $Out2>(
        $value,
        $cast,
        t,
      );
}

class InstagramAccountSettingsPatchMapper
    extends ClassMapperBase<InstagramAccountSettingsPatch> {
  InstagramAccountSettingsPatchMapper._();

  static InstagramAccountSettingsPatchMapper? _instance;
  static InstagramAccountSettingsPatchMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramAccountSettingsPatchMapper._(),
      );
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramAccountSettingsPatch';

  static bool? _$discoverable(InstagramAccountSettingsPatch v) =>
      v.discoverable;
  static const Field<InstagramAccountSettingsPatch, bool> _f$discoverable =
      Field('discoverable', _$discoverable, opt: true);
  static bool? _$reactivate(InstagramAccountSettingsPatch v) => v.reactivate;
  static const Field<InstagramAccountSettingsPatch, bool> _f$reactivate = Field(
    'reactivate',
    _$reactivate,
    opt: true,
  );

  @override
  final MappableFields<InstagramAccountSettingsPatch> fields = const {
    #discoverable: _f$discoverable,
    #reactivate: _f$reactivate,
  };
  @override
  final bool ignoreNull = true;

  static InstagramAccountSettingsPatch _instantiate(DecodingData data) {
    return InstagramAccountSettingsPatch(
      discoverable: data.dec(_f$discoverable),
      reactivate: data.dec(_f$reactivate),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramAccountSettingsPatchMappable {
  String toJson() {
    return InstagramAccountSettingsPatchMapper.ensureInitialized()
        .encodeJson<InstagramAccountSettingsPatch>(
          this as InstagramAccountSettingsPatch,
        );
  }

  Map<String, dynamic> toMap() {
    return InstagramAccountSettingsPatchMapper.ensureInitialized()
        .encodeMap<InstagramAccountSettingsPatch>(
          this as InstagramAccountSettingsPatch,
        );
  }

  InstagramAccountSettingsPatchCopyWith<
    InstagramAccountSettingsPatch,
    InstagramAccountSettingsPatch,
    InstagramAccountSettingsPatch
  >
  get copyWith =>
      _InstagramAccountSettingsPatchCopyWithImpl<
        InstagramAccountSettingsPatch,
        InstagramAccountSettingsPatch
      >(this as InstagramAccountSettingsPatch, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramAccountSettingsPatchMapper.ensureInitialized().equalsValue(
      this as InstagramAccountSettingsPatch,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramAccountSettingsPatchMapper.ensureInitialized().hashValue(
      this as InstagramAccountSettingsPatch,
    );
  }
}

extension InstagramAccountSettingsPatchValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramAccountSettingsPatch, $Out> {
  InstagramAccountSettingsPatchCopyWith<$R, InstagramAccountSettingsPatch, $Out>
  get $asInstagramAccountSettingsPatch => $base.as(
    (v, t, t2) =>
        _InstagramAccountSettingsPatchCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramAccountSettingsPatchCopyWith<
  $R,
  $In extends InstagramAccountSettingsPatch,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({bool? discoverable, bool? reactivate});
  InstagramAccountSettingsPatchCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramAccountSettingsPatchCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramAccountSettingsPatch, $Out>
    implements
        InstagramAccountSettingsPatchCopyWith<
          $R,
          InstagramAccountSettingsPatch,
          $Out
        > {
  _InstagramAccountSettingsPatchCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<InstagramAccountSettingsPatch> $mapper =
      InstagramAccountSettingsPatchMapper.ensureInitialized();
  @override
  $R call({Object? discoverable = $none, Object? reactivate = $none}) => $apply(
    FieldCopyWithData({
      if (discoverable != $none) #discoverable: discoverable,
      if (reactivate != $none) #reactivate: reactivate,
    }),
  );
  @override
  InstagramAccountSettingsPatch $make(CopyWithData data) =>
      InstagramAccountSettingsPatch(
        discoverable: data.get(#discoverable, or: $value.discoverable),
        reactivate: data.get(#reactivate, or: $value.reactivate),
      );

  @override
  InstagramAccountSettingsPatchCopyWith<
    $R2,
    InstagramAccountSettingsPatch,
    $Out2
  >
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramAccountSettingsPatchCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
