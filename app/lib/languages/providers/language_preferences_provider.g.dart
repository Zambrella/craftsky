// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'language_preferences_provider.dart';

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
    required AccountKey super.argument,
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
    r'8b18a20d4cdff173d2494f67197e315826d4c5be';

final class AccountLanguagePreferencesFamily extends $Family
    with
        $ClassFamilyOverride<
          AccountLanguagePreferences,
          AsyncValue<LanguagePreferences>,
          LanguagePreferences,
          FutureOr<LanguagePreferences>,
          AccountKey
        > {
  AccountLanguagePreferencesFamily._()
    : super(
        retry: null,
        name: r'accountLanguagePreferencesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  AccountLanguagePreferencesProvider call(AccountKey account) =>
      AccountLanguagePreferencesProvider._(argument: account, from: this);

  @override
  String toString() => r'accountLanguagePreferencesProvider';
}

abstract class _$AccountLanguagePreferences
    extends $AsyncNotifier<LanguagePreferences> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<LanguagePreferences> build(AccountKey account);
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

@ProviderFor(activeLanguagePreferences)
final activeLanguagePreferencesProvider = ActiveLanguagePreferencesProvider._();

final class ActiveLanguagePreferencesProvider
    extends
        $FunctionalProvider<
          AsyncValue<LanguagePreferences>,
          LanguagePreferences,
          FutureOr<LanguagePreferences>
        >
    with
        $FutureModifier<LanguagePreferences>,
        $FutureProvider<LanguagePreferences> {
  ActiveLanguagePreferencesProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'activeLanguagePreferencesProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$activeLanguagePreferencesHash();

  @$internal
  @override
  $FutureProviderElement<LanguagePreferences> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<LanguagePreferences> create(Ref ref) {
    return activeLanguagePreferences(ref);
  }
}

String _$activeLanguagePreferencesHash() =>
    r'6eab2959cea7259eb9ef562e88e1da8f1280d76d';

@ProviderFor(activeContentLanguagePolicy)
final activeContentLanguagePolicyProvider =
    ActiveContentLanguagePolicyProvider._();

final class ActiveContentLanguagePolicyProvider
    extends
        $FunctionalProvider<
          AsyncValue<List<String>>,
          List<String>,
          FutureOr<List<String>>
        >
    with $FutureModifier<List<String>>, $FutureProvider<List<String>> {
  ActiveContentLanguagePolicyProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'activeContentLanguagePolicyProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$activeContentLanguagePolicyHash();

  @$internal
  @override
  $FutureProviderElement<List<String>> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<List<String>> create(Ref ref) {
    return activeContentLanguagePolicy(ref);
  }
}

String _$activeContentLanguagePolicyHash() =>
    r'baeb9f8c9dc246fab5c6eeafd3be10eddcfa5d5a';
