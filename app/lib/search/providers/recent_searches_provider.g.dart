// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'recent_searches_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(recentSearchPage)
final recentSearchPageProvider = RecentSearchPageProvider._();

final class RecentSearchPageProvider
    extends
        $FunctionalProvider<
          AsyncValue<RecentSearchPage>,
          RecentSearchPage,
          FutureOr<RecentSearchPage>
        >
    with $FutureModifier<RecentSearchPage>, $FutureProvider<RecentSearchPage> {
  RecentSearchPageProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'recentSearchPageProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$recentSearchPageHash();

  @$internal
  @override
  $FutureProviderElement<RecentSearchPage> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<RecentSearchPage> create(Ref ref) {
    return recentSearchPage(ref);
  }
}

String _$recentSearchPageHash() => r'230c8197246c4fa4441e703e6516cc3f24c37abf';

@ProviderFor(SaveRecentSearch)
final saveRecentSearchProvider = SaveRecentSearchProvider._();

final class SaveRecentSearchProvider
    extends $AsyncNotifierProvider<SaveRecentSearch, RecentSearchItem?> {
  SaveRecentSearchProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'saveRecentSearchProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$saveRecentSearchHash();

  @$internal
  @override
  SaveRecentSearch create() => SaveRecentSearch();
}

String _$saveRecentSearchHash() => r'cdbe315834d2b4ce0ff7ebc3a8f67d7ff06aa18d';

abstract class _$SaveRecentSearch extends $AsyncNotifier<RecentSearchItem?> {
  FutureOr<RecentSearchItem?> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<RecentSearchItem?>, RecentSearchItem?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<RecentSearchItem?>, RecentSearchItem?>,
              AsyncValue<RecentSearchItem?>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}

@ProviderFor(DeleteRecentSearch)
final deleteRecentSearchProvider = DeleteRecentSearchProvider._();

final class DeleteRecentSearchProvider
    extends $AsyncNotifierProvider<DeleteRecentSearch, void> {
  DeleteRecentSearchProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'deleteRecentSearchProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$deleteRecentSearchHash();

  @$internal
  @override
  DeleteRecentSearch create() => DeleteRecentSearch();
}

String _$deleteRecentSearchHash() =>
    r'4fe86a878ac2d866147a901b5a6f082f74bbc883';

abstract class _$DeleteRecentSearch extends $AsyncNotifier<void> {
  FutureOr<void> build();
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref = this.ref as $Ref<AsyncValue<void>, void>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<void>, void>,
              AsyncValue<void>,
              Object?,
              Object?
            >;
    return element.handleCreate(ref, build);
  }
}
