// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'language_preferences_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(activeLanguagePreferences)
final activeLanguagePreferencesProvider = ActiveLanguagePreferencesProvider._();

final class ActiveLanguagePreferencesProvider
    extends
        $FunctionalProvider<
          LanguagePreferences,
          LanguagePreferences,
          LanguagePreferences
        >
    with $Provider<LanguagePreferences> {
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
  $ProviderElement<LanguagePreferences> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  LanguagePreferences create(Ref ref) {
    return activeLanguagePreferences(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(LanguagePreferences value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<LanguagePreferences>(value),
    );
  }
}

String _$activeLanguagePreferencesHash() =>
    r'f27a506baccd4e50fc20ba23f7e1dd4f88cc2516';

@ProviderFor(ActiveContentLanguagePolicy)
final activeContentLanguagePolicyProvider =
    ActiveContentLanguagePolicyProvider._();

final class ActiveContentLanguagePolicyProvider
    extends $NotifierProvider<ActiveContentLanguagePolicy, List<String>> {
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
  ActiveContentLanguagePolicy create() => ActiveContentLanguagePolicy();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(List<String> value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<List<String>>(value),
    );
  }
}

String _$activeContentLanguagePolicyHash() =>
    r'2a9897b447aeebcb6efeea33af8299f524354675';

abstract class _$ActiveContentLanguagePolicy extends $Notifier<List<String>> {
  List<String> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<List<String>, List<String>>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<List<String>, List<String>>,
              List<String>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
