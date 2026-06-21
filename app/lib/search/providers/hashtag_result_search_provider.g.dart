// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'hashtag_result_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(HashtagResultSearch)
final hashtagResultSearchProvider = HashtagResultSearchFamily._();

final class HashtagResultSearchProvider
    extends
        $AsyncNotifierProvider<HashtagResultSearch, HashtagSearchResultsState> {
  HashtagResultSearchProvider._({
    required HashtagResultSearchFamily super.from,
    required HashtagResultSearchQuery super.argument,
  }) : super(
         retry: null,
         name: r'hashtagResultSearchProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$hashtagResultSearchHash();

  @override
  String toString() {
    return r'hashtagResultSearchProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  HashtagResultSearch create() => HashtagResultSearch();

  @override
  bool operator ==(Object other) {
    return other is HashtagResultSearchProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$hashtagResultSearchHash() =>
    r'9ec9f4098ec88f4d347d801eba007e4755756baf';

final class HashtagResultSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          HashtagResultSearch,
          AsyncValue<HashtagSearchResultsState>,
          HashtagSearchResultsState,
          FutureOr<HashtagSearchResultsState>,
          HashtagResultSearchQuery
        > {
  HashtagResultSearchFamily._()
    : super(
        retry: null,
        name: r'hashtagResultSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  HashtagResultSearchProvider call(HashtagResultSearchQuery query) =>
      HashtagResultSearchProvider._(argument: query, from: this);

  @override
  String toString() => r'hashtagResultSearchProvider';
}

abstract class _$HashtagResultSearch
    extends $AsyncNotifier<HashtagSearchResultsState> {
  late final _$args = ref.$arg as HashtagResultSearchQuery;
  HashtagResultSearchQuery get query => _$args;

  FutureOr<HashtagSearchResultsState> build(HashtagResultSearchQuery query);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<HashtagSearchResultsState>,
              HashtagSearchResultsState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<HashtagSearchResultsState>,
                HashtagSearchResultsState
              >,
              AsyncValue<HashtagSearchResultsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
