// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// dart format off
// ignore_for_file: type=lint
// ignore_for_file: invalid_use_of_protected_member
// ignore_for_file: unused_element, unnecessary_cast, override_on_non_overriding_member
// ignore_for_file: strict_raw_type, inference_failure_on_untyped_parameter

part of 'active_account_initialization.dart';

class ActiveAccountInitializationMapper
    extends ClassMapperBase<ActiveAccountInitialization> {
  ActiveAccountInitializationMapper._();

  static ActiveAccountInitializationMapper? _instance;
  static ActiveAccountInitializationMapper ensureInitialized() {
    if (_instance == null) {
      MapperContainer.globals.use(
        _instance = ActiveAccountInitializationMapper._(),
      );
    }
    return _instance!;
  }

  @override
  final String id = 'ActiveAccountInitialization';

  static ActiveAccountLease _$lease(ActiveAccountInitialization v) => v.lease;
  static const Field<ActiveAccountInitialization, ActiveAccountLease> _f$lease =
      Field('lease', _$lease);
  static LanguagePreferences _$languagePreferences(
    ActiveAccountInitialization v,
  ) => v.languagePreferences;
  static const Field<ActiveAccountInitialization, LanguagePreferences>
  _f$languagePreferences = Field('languagePreferences', _$languagePreferences);
  static bool _$onboardingComplete(ActiveAccountInitialization v) =>
      v.onboardingComplete;
  static const Field<ActiveAccountInitialization, bool> _f$onboardingComplete =
      Field('onboardingComplete', _$onboardingComplete);

  @override
  final MappableFields<ActiveAccountInitialization> fields = const {
    #lease: _f$lease,
    #languagePreferences: _f$languagePreferences,
    #onboardingComplete: _f$onboardingComplete,
  };

  static ActiveAccountInitialization _instantiate(DecodingData data) {
    return ActiveAccountInitialization(
      lease: data.dec(_f$lease),
      languagePreferences: data.dec(_f$languagePreferences),
      onboardingComplete: data.dec(_f$onboardingComplete),
    );
  }

  @override
  final Function instantiate = _instantiate;
}

mixin ActiveAccountInitializationMappable {
  ActiveAccountInitializationCopyWith<
    ActiveAccountInitialization,
    ActiveAccountInitialization,
    ActiveAccountInitialization
  >
  get copyWith =>
      _ActiveAccountInitializationCopyWithImpl<
        ActiveAccountInitialization,
        ActiveAccountInitialization
      >(this as ActiveAccountInitialization, $identity, $identity);
  @override
  bool operator ==(Object other) {
    return ActiveAccountInitializationMapper.ensureInitialized().equalsValue(
      this as ActiveAccountInitialization,
      other,
    );
  }

  @override
  int get hashCode {
    return ActiveAccountInitializationMapper.ensureInitialized().hashValue(
      this as ActiveAccountInitialization,
    );
  }
}

extension ActiveAccountInitializationValueCopy<$R, $Out>
    on ObjectCopyWith<$R, ActiveAccountInitialization, $Out> {
  ActiveAccountInitializationCopyWith<$R, ActiveAccountInitialization, $Out>
  get $asActiveAccountInitialization => $base.as(
    (v, t, t2) => _ActiveAccountInitializationCopyWithImpl<$R, $Out>(v, t, t2),
  );
}

abstract class ActiveAccountInitializationCopyWith<
  $R,
  $In extends ActiveAccountInitialization,
  $Out
>
    implements ClassCopyWith<$R, $In, $Out> {
  $R call({
    ActiveAccountLease? lease,
    LanguagePreferences? languagePreferences,
    bool? onboardingComplete,
  });
  ActiveAccountInitializationCopyWith<$R2, $In, $Out2> $chain<$R2, $Out2>(
    Then<$Out2, $R2> t,
  );
}

class _ActiveAccountInitializationCopyWithImpl<$R, $Out>
    extends ClassCopyWithBase<$R, ActiveAccountInitialization, $Out>
    implements
        ActiveAccountInitializationCopyWith<
          $R,
          ActiveAccountInitialization,
          $Out
        > {
  _ActiveAccountInitializationCopyWithImpl(
    super.value,
    super.then,
    super.then2,
  );

  @override
  late final ClassMapperBase<ActiveAccountInitialization> $mapper =
      ActiveAccountInitializationMapper.ensureInitialized();
  @override
  $R call({
    ActiveAccountLease? lease,
    LanguagePreferences? languagePreferences,
    bool? onboardingComplete,
  }) => $apply(
    FieldCopyWithData({
      if (lease != null) #lease: lease,
      if (languagePreferences != null)
        #languagePreferences: languagePreferences,
      if (onboardingComplete != null) #onboardingComplete: onboardingComplete,
    }),
  );
  @override
  ActiveAccountInitialization $make(CopyWithData data) =>
      ActiveAccountInitialization(
        lease: data.get(#lease, or: $value.lease),
        languagePreferences: data.get(
          #languagePreferences,
          or: $value.languagePreferences,
        ),
        onboardingComplete: data.get(
          #onboardingComplete,
          or: $value.onboardingComplete,
        ),
      );

  @override
  ActiveAccountInitializationCopyWith<$R2, ActiveAccountInitialization, $Out2>
  $chain<$R2, $Out2>(Then<$Out2, $R2> t) =>
      _ActiveAccountInitializationCopyWithImpl<$R2, $Out2>($value, $cast, t);
}

