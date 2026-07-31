// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'account_language_preferences_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(AccountLanguagePreferences)
final accountLanguagePreferencesProvider = AccountLanguagePreferencesFamily._();

final class AccountLanguagePreferencesProvider
    extends
        $AsyncNotifierProvider<
          AccountLanguagePreferences,
          LanguagePreferences
        > {
  AccountLanguagePreferencesProvider._({
    required AccountLanguagePreferencesFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'accountLanguagePreferencesProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$accountLanguagePreferencesHash();

  @override
  String toString() {
    return r'accountLanguagePreferencesProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  AccountLanguagePreferences create() => AccountLanguagePreferences();

  @override
  bool operator ==(Object other) {
    return other is AccountLanguagePreferencesProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$accountLanguagePreferencesHash() =>
    r'8e99dbd10e3a36f98eadabd49c983bdae1d6187a';

final class AccountLanguagePreferencesFamily extends $Family
    with
        $ClassFamilyOverride<
          AccountLanguagePreferences,
          AsyncValue<LanguagePreferences>,
          LanguagePreferences,
          FutureOr<LanguagePreferences>,
          ActiveAccountLease
        > {
  AccountLanguagePreferencesFamily._()
    : super(
        retry: null,
        name: r'accountLanguagePreferencesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountLanguagePreferencesProvider call(ActiveAccountLease lease) =>
      AccountLanguagePreferencesProvider._(argument: lease, from: this);

  @override
  String toString() => r'accountLanguagePreferencesProvider';
}

abstract class _$AccountLanguagePreferences
    extends $AsyncNotifier<LanguagePreferences> {
  late final _$args = ref.$arg as ActiveAccountLease;
  ActiveAccountLease get lease => _$args;

  FutureOr<LanguagePreferences> build(ActiveAccountLease lease);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<LanguagePreferences>, LanguagePreferences>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<LanguagePreferences>, LanguagePreferences>,
              AsyncValue<LanguagePreferences>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
