// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'instagram_verification_provider.dart';

class InstagramVerificationViewStateMapper
    extends ClassMapperBase<InstagramVerificationViewState> {
  InstagramVerificationViewStateMapper._();

  static InstagramVerificationViewStateMapper? _instance;
  static InstagramVerificationViewStateMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = InstagramVerificationViewStateMapper._(),
      );
      InstagramVerificationAttemptMapper.ensureInitialized();
    }
    return _instance!;
  }

  @override
  final String id = 'InstagramVerificationViewState';

  static InstagramVerificationAttempt? _$attempt(
    InstagramVerificationViewState v,
  ) => v.attempt;
  static const Field<
    InstagramVerificationViewState,
    InstagramVerificationAttempt
  >
  _f$attempt = Field('attempt', _$attempt, opt: true);
  static bool _$isBusy(InstagramVerificationViewState v) => v.isBusy;
  static const Field<InstagramVerificationViewState, bool> _f$isBusy = Field(
    'isBusy',
    _$isBusy,
    opt: true,
    def: false,
  );
  static bool _$hasError(InstagramVerificationViewState v) => v.hasError;
  static const Field<InstagramVerificationViewState, bool> _f$hasError = Field(
    'hasError',
    _$hasError,
    opt: true,
    def: false,
  );

  @override
  final MappableFields<InstagramVerificationViewState> fields = const {
    #attempt: _f$attempt,
    #isBusy: _f$isBusy,
    #hasError: _f$hasError,
  };

  static InstagramVerificationViewState _instantiate(DecodingData data) {
    return InstagramVerificationViewState(
      attempt: data.dec(_f$attempt),
      isBusy: data.dec(_f$isBusy),
      hasError: data.dec(_f$hasError),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin InstagramVerificationViewStateMappable {
  InstagramVerificationViewStateCopyWith<
    InstagramVerificationViewState,
    InstagramVerificationViewState,
    InstagramVerificationViewState
  >
  get copyWith =>
      _InstagramVerificationViewStateCopyWithImpl<
        InstagramVerificationViewState,
        InstagramVerificationViewState
      >(this as InstagramVerificationViewState, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return InstagramVerificationViewStateMapper.ensureInitialized().equalsValue(
      this as InstagramVerificationViewState,
      other,
    );
  }

  @override
  int get hashCode {
    return InstagramVerificationViewStateMapper.ensureInitialized().hashValue(
      this as InstagramVerificationViewState,
    );
  }
}

extension InstagramVerificationViewStateValueCopy<$R, $Out>
    on ObjectCopyWith<$R, InstagramVerificationViewState, $Out> {
  InstagramVerificationViewStateCopyWith<
    $R,
    InstagramVerificationViewState,
    $Out
  >
  get $asInstagramVerificationViewState => $base.as(
    (v, t, t2) =>
        _InstagramVerificationViewStateCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class InstagramVerificationViewStateCopyWith<
  $R,
  $In extends InstagramVerificationViewState,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  InstagramVerificationAttemptCopyWith<
    $R,
    InstagramVerificationAttempt,
    InstagramVerificationAttempt
  >?
  get attempt;
  $R call({
    InstagramVerificationAttempt? attempt,
    bool? isBusy,
    bool? hasError,
  });
  InstagramVerificationViewStateCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _InstagramVerificationViewStateCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, InstagramVerificationViewState, $Out>
    implements
        InstagramVerificationViewStateCopyWith<
          $R,
          InstagramVerificationViewState,
          $Out
        > {
  _InstagramVerificationViewStateCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<InstagramVerificationViewState> $mapper =
      InstagramVerificationViewStateMapper.ensureInitialized();
  @override
  InstagramVerificationAttemptCopyWith<
    $R,
    InstagramVerificationAttempt,
    InstagramVerificationAttempt
  >?
  get attempt => $value.attempt?.copyWith.$chain((v) => call(attempt: v));
  @override
  $R call({Object? attempt = $none, bool? isBusy, bool? hasError}) => $apply(
    FieldCopyWithData({
      if (attempt != $none) #attempt: attempt,
      if (isBusy != null) #isBusy: isBusy,
      if (hasError != null) #hasError: hasError,
    }),
  );
  @override
  InstagramVerificationViewState $make(CopyWithData data) =>
      InstagramVerificationViewState(
        attempt: data.get(#attempt, or: $value.attempt),
        isBusy: data.get(#isBusy, or: $value.isBusy),
        hasError: data.get(#hasError, or: $value.hasError),
      );

  @override
  InstagramVerificationViewStateCopyWith<
    $R2,
    InstagramVerificationViewState,
    $Out2
  >
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _InstagramVerificationViewStateCopyWithImpl<$R2, $Out2>($value, $cast, t);
}
