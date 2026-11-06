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
    required (HashtagResultSearchQuery, {BrowsingContext? audience})
    super.argument,
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
        '$argument';
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
    r'887b2031e82b0c93f9b6a9e0bfcf840bbf178cf2';

final class HashtagResultSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          HashtagResultSearch,
          AsyncValue<HashtagSearchResultsState>,
          HashtagSearchResultsState,
          FutureOr<HashtagSearchResultsState>,
          (HashtagResultSearchQuery, {BrowsingContext? audience})
        > {
  HashtagResultSearchFamily._()
    : super(
        retry: null,
        name: r'hashtagResultSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  HashtagResultSearchProvider call(
    HashtagResultSearchQuery query, {
    BrowsingContext? audience,
  }) => HashtagResultSearchProvider._(
    argument: (query, audience: audience),
    from: this,
  );

  @override
  String toString() => r'hashtagResultSearchProvider';
}

abstract class _$HashtagResultSearch
    extends $AsyncNotifier<HashtagSearchResultsState> {
  late final _$args =
      ref.$arg as (HashtagResultSearchQuery, {BrowsingContext? audience});
  HashtagResultSearchQuery get query => _$args.$1;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<HashtagSearchResultsState> build(
    HashtagResultSearchQuery query, {
    BrowsingContext? audience,
  });
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
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
    return element.handleCreate(
      ref,
      () => build(_$args.$1, audience: _$args.audience),
    );
  }
}
