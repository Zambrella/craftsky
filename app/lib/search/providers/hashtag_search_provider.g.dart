// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'hashtag_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(HashtagSearch)
final hashtagSearchProvider = HashtagSearchFamily._();

final class HashtagSearchProvider
    extends $AsyncNotifierProvider<HashtagSearch, SearchPostResultsState> {
  HashtagSearchProvider._({
    required HashtagSearchFamily super.from,
    required (HashtagSearchQuery, {BrowsingContext? audience}) super.argument,
  }) : super(
         retry: null,
         name: r'hashtagSearchProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$hashtagSearchHash();

  @override
  String toString() {
    return r'hashtagSearchProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  HashtagSearch create() => HashtagSearch();

  @override
  bool operator ==(Object other) {
    return other is HashtagSearchProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$hashtagSearchHash() => r'cf04f4a4cfbd960447ea1f9c7248128172bbe650';

final class HashtagSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          HashtagSearch,
          AsyncValue<SearchPostResultsState>,
          SearchPostResultsState,
          FutureOr<SearchPostResultsState>,
          (HashtagSearchQuery, {BrowsingContext? audience})
        > {
  HashtagSearchFamily._()
    : super(
        retry: null,
        name: r'hashtagSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  HashtagSearchProvider call(
    HashtagSearchQuery query, {
    BrowsingContext? audience,
  }) => HashtagSearchProvider._(
    argument: (query, audience: audience),
    from: this,
  );

  @override
  String toString() => r'hashtagSearchProvider';
}

abstract class _$HashtagSearch extends $AsyncNotifier<SearchPostResultsState> {
  late final _$args =
      ref.$arg as (HashtagSearchQuery, {BrowsingContext? audience});
  HashtagSearchQuery get query => _$args.$1;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<SearchPostResultsState> build(
    HashtagSearchQuery query, {
    BrowsingContext? audience,
  });
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
    final ref =
        this.ref
            as $Ref<AsyncValue<SearchPostResultsState>, SearchPostResultsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<SearchPostResultsState>,
                SearchPostResultsState
              >,
              AsyncValue<SearchPostResultsState>,
              Object?,
              Object?
            >;
    return element.handleCreate(
      ref,
      () => build(_$args.$1, audience: _$args.audience),
    );
  }
}
