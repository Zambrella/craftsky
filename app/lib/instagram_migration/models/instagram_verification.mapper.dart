// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'instagram_verification.dart';

class InstagramVerificationStateMapper
    extends EnumMapper<InstagramVerificationState> {
  InstagramVerificationStateMapper._();

  static InstagramVerificationStateMapper? _instance;
  static InstagramVerificationStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramVerificationStateMapper._(),
      );
    }
    return _instance!;
  }

  static InstagramVerificationState fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  InstagramVerificationState decode(dynamic value) {
    switch (value) {
      case r'pendingDm':
        return InstagramVerificationState.pendingDm;
      case r'processing':
        return InstagramVerificationState.processing;
      case r'pendingConfirmation':
        return InstagramVerificationState.pendingConfirmation;
      case r'confirmed':
        return InstagramVerificationState.confirmed;
      case r'expired':
        return InstagramVerificationState.expired;
      case r'cancelled':
        return InstagramVerificationState.cancelled;
      case r'superseded':
        return InstagramVerificationState.superseded;
      case r'rejected':
        return InstagramVerificationState.rejected;
      case r'conflicted':
        return InstagramVerificationState.conflicted;
      case r'unknown':
        return InstagramVerificationState.unknown;
      default:
        return InstagramVerificationState.values[9];
    }
  }

  @override
  dynamic encode(InstagramVerificationState self) {
    switch (self) {
      case InstagramVerificationState.pendingDm:
        return r'pendingDm';
      case InstagramVerificationState.processing:
        return r'processing';
      case InstagramVerificationState.pendingConfirmation:
        return r'pendingConfirmation';
      case InstagramVerificationState.confirmed:
        return r'confirmed';
      case InstagramVerificationState.expired:
        return r'expired';
      case InstagramVerificationState.cancelled:
        return r'cancelled';
      case InstagramVerificationState.superseded:
        return r'superseded';
      case InstagramVerificationState.rejected:
        return r'rejected';
      case InstagramVerificationState.conflicted:
        return r'conflicted';
      case InstagramVerificationState.unknown:
        return r'unknown';
    }
  }
}

extension InstagramVerificationStateMapperExtension
    on InstagramVerificationState {
  String toValue() {
    InstagramVerificationStateMapper.ensureInitialized();
    return MapperContainer.globals.toValue<InstagramVerificationState>(this)
        as String;
  }
}

class InstagramVerificationRetryCodeMapper
    extends EnumMapper<InstagramVerificationRetryCode> {
  InstagramVerificationRetryCodeMapper._();

  static InstagramVerificationRetryCodeMapper? _instance;
  static InstagramVerificationRetryCodeMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramVerificationRetryCodeMapper._(),
      );
    }
    return _instance!;
  }

  static InstagramVerificationRetryCode fromValue(dynamic value) {
    ensureInitialized();
    return MapperContainer.globals.fromValue(value);
  }

  @override
  InstagramVerificationRetryCode decode(dynamic value) {
    switch (value) {
      case r'profileLookupUnavailable':
        return InstagramVerificationRetryCode.profileLookupUnavailable;
      case r'invalidProfileResponse':
        return InstagramVerificationRetryCode.invalidProfileResponse;
      case r'membershipInactive':
        return InstagramVerificationRetryCode.membershipInactive;
      case r'unknown':
        return InstagramVerificationRetryCode.unknown;
      default:
        return InstagramVerificationRetryCode.values[3];
    }
  }

  @override
  dynamic encode(InstagramVerificationRetryCode self) {
    switch (self) {
      case InstagramVerificationRetryCode.profileLookupUnavailable:
        return r'profileLookupUnavailable';
      case InstagramVerificationRetryCode.invalidProfileResponse:
        return r'invalidProfileResponse';
      case InstagramVerificationRetryCode.membershipInactive:
        return r'membershipInactive';
      case InstagramVerificationRetryCode.unknown:
        return r'unknown';
    }
  }
}

extension InstagramVerificationRetryCodeMapperExtension
    on InstagramVerificationRetryCode {
  String toValue() {
    InstagramVerificationRetryCodeMapper.ensureInitialized();
    return MapperContainer.globals.toValue<InstagramVerificationRetryCode>(this)
        as String;
  }
}

class InstagramVerificationAttemptMapper
    extends ClassMapperBase<InstagramVerificationAttempt> {
  InstagramVerificationAttemptMapper._();

  static InstagramVerificationAttemptMapper? _instance;
  static InstagramVerificationAttemptMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramVerificationAttemptMapper._(),
      );
      InstagramVerificationStateMapper.ensureInitialized();
      InstagramVerificationRetryCodeMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramVerificationAttempt';

  static String _$verificationId(InstagramVerificationAttempt v) =>
      v.verificationId;
  static const Field<InstagramVerificationAttempt, String> _f$verificationId =
      Field('verificationId', _$verificationId);
  static InstagramVerificationState _$state(InstagramVerificationAttempt v) =>
      v.state;
  static const Field<InstagramVerificationAttempt, InstagramVerificationState>
  _f$state = Field('state', _$state);
  static DateTime _$expiresAt(InstagramVerificationAttempt v) => v.expiresAt;
  static const Field<InstagramVerificationAttempt, DateTime> _f$expiresAt =
      Field('expiresAt', _$expiresAt);
  static String? _$challenge(InstagramVerificationAttempt v) => v.challenge;
  static const Field<InstagramVerificationAttempt, String> _f$challenge = Field(
    'challenge',
    _$challenge,
    opt: true,
  );
  static Uri? _$dmUrl(InstagramVerificationAttempt v) => v.dmUrl;
  static const Field<InstagramVerificationAttempt, Uri> _f$dmUrl = Field(
    'dmUrl',
    _$dmUrl,
    opt: true,
  );
  static String? _$candidateUsername(InstagramVerificationAttempt v) =>
      v.candidateUsername;
  static const Field<InstagramVerificationAttempt, String>
  _f$candidateUsername = Field(
    'candidateUsername',
    _$candidateUsername,
    opt: true,
  );
  static InstagramVerificationRetryCode? _$retryCode(
    InstagramVerificationAttempt v,
  ) => v.retryCode;
  static const Field<
    InstagramVerificationAttempt,
    InstagramVerificationRetryCode
  >
  _f$retryCode = Field('retryCode', _$retryCode, opt: true);

  @override
  final MappableFields<InstagramVerificationAttempt> fields = const {
    #verificationId: _f$verificationId,
    #state: _f$state,
    #expiresAt: _f$expiresAt,
    #challenge: _f$challenge,
    #dmUrl: _f$dmUrl,
    #candidateUsername: _f$candidateUsername,
    #retryCode: _f$retryCode,
  };

  static InstagramVerificationAttempt _instantiate(DecodingData data) {
    return InstagramVerificationAttempt(
      verificationId: data.dec(_f$verificationId),
      state: data.dec(_f$state),
      expiresAt: data.dec(_f$expiresAt),
      challenge: data.dec(_f$challenge),
      dmUrl: data.dec(_f$dmUrl),
      candidateUsername: data.dec(_f$candidateUsername),
      retryCode: data.dec(_f$retryCode),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramVerificationAttemptMappable {
  InstagramVerificationAttemptCopyWith<
    InstagramVerificationAttempt,
    InstagramVerificationAttempt,
    InstagramVerificationAttempt
  >
  get copyWith =>
      _InstagramVerificationAttemptCopyWithImpl<
        InstagramVerificationAttempt,
        InstagramVerificationAttempt
      >(this as InstagramVerificationAttempt, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramVerificationAttemptMapper.ensureInitialized().equalsValue(
      this as InstagramVerificationAttempt,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramVerificationAttemptMapper.ensureInitialized().hashValue(
      this as InstagramVerificationAttempt,
    );
  }
}

extension InstagramVerificationAttemptValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramVerificationAttempt, $Out> {
  InstagramVerificationAttemptCopyWith<$R, InstagramVerificationAttempt, $Out>
  get $asInstagramVerificationAttempt => $base.as(
    (v, t, t2) => _InstagramVerificationAttemptCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramVerificationAttemptCopyWith<
  $R,
  $In extends InstagramVerificationAttempt,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    String? verificationId,
    InstagramVerificationState? state,
    DateTime? expiresAt,
    String? challenge,
    Uri? dmUrl,
    String? candidateUsername,
    InstagramVerificationRetryCode? retryCode,
  });
  InstagramVerificationAttemptCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramVerificationAttemptCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramVerificationAttempt, $Out>
    implements
        InstagramVerificationAttemptCopyWith<
          $R,
          InstagramVerificationAttempt,
          $Out
        > {
  _InstagramVerificationAttemptCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<InstagramVerificationAttempt> $mapper =
      InstagramVerificationAttemptMapper.ensureInitialized();
  @override
  $R call({
    String? verificationId,
    InstagramVerificationState? state,
    DateTime? expiresAt,
    Object? challenge = $none,
    Object? dmUrl = $none,
    Object? candidateUsername = $none,
    Object? retryCode = $none,
  }) => $apply(
    FieldCopyWithData({
      if (verificationId != null) #verificationId: verificationId,
      if (state != null) #state: state,
      if (expiresAt != null) #expiresAt: expiresAt,
      if (challenge != $none) #challenge: challenge,
      if (dmUrl != $none) #dmUrl: dmUrl,
      if (candidateUsername != $none) #candidateUsername: candidateUsername,
      if (retryCode != $none) #retryCode: retryCode,
    }),
  );
  @override
  InstagramVerificationAttempt $make(CopyWithData data) =>
      InstagramVerificationAttempt(
        verificationId: data.get(#verificationId, or: $value.verificationId),
        state: data.get(#state, or: $value.state),
        expiresAt: data.get(#expiresAt, or: $value.expiresAt),
        challenge: data.get(#challenge, or: $value.challenge),
        dmUrl: data.get(#dmUrl, or: $value.dmUrl),
        candidateUsername: data.get(
          #candidateUsername,
          or: $value.candidateUsername,
        ),
        retryCode: data.get(#retryCode, or: $value.retryCode),
      );

  @override
  InstagramVerificationAttemptCopyWith<$R2, InstagramVerificationAttempt, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramVerificationAttemptCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
