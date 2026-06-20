// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'recent_searches_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(RecentSearches)
final recentSearchesProvider = RecentSearchesProvider._();

final class RecentSearchesProvider
    extends $AsyncNotifierProvider<RecentSearches, RecentSearchPage> {
  RecentSearchesProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'recentSearchesProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$recentSearchesHash();

  @$internal
  @override
  RecentSearches create() => RecentSearches();
}

String _$recentSearchesHash() => r'db13a3e82f362031583f497e0a23a45932cac434';

abstract class _$RecentSearches extends $AsyncNotifier<RecentSearchPage> {
  FutureOr<RecentSearchPage> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<RecentSearchPage>, RecentSearchPage>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<RecentSearchPage>, RecentSearchPage>,
              AsyncValue<RecentSearchPage>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
