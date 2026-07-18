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
    required HashtagSearchQuery super.argument,
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
        '($argument)';
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

String _$hashtagSearchHash() => r'ac67e583619c1c0ff64c22756e6d782ff8a94abe';

final class HashtagSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          HashtagSearch,
          AsyncValue<SearchPostResultsState>,
          SearchPostResultsState,
          FutureOr<SearchPostResultsState>,
          HashtagSearchQuery
        > {
  HashtagSearchFamily._()
    : super(
        retry: null,
        name: r'hashtagSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  HashtagSearchProvider call(HashtagSearchQuery query) =>
      HashtagSearchProvider._(argument: query, from: this);

  @override
  String toString() => r'hashtagSearchProvider';
}

abstract class _$HashtagSearch extends $AsyncNotifier<SearchPostResultsState> {
  late final _$args = ref.$arg as HashtagSearchQuery;
  HashtagSearchQuery get query => _$args;

  FutureOr<SearchPostResultsState> build(HashtagSearchQuery query);
  @$mustCallSuper
  @override
  void runBuild() {
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
    element.handleCreate(ref, () => build(_$args));
  }
}
