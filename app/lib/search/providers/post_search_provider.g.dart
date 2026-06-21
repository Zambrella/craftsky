// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'post_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(PostSearch)
final postSearchProvider = PostSearchFamily._();

final class PostSearchProvider
    extends $AsyncNotifierProvider<PostSearch, SearchPostResultsState> {
  PostSearchProvider._({
    required PostSearchFamily super.from,
    required PostSearchQuery super.argument,
  }) : super(
         retry: null,
         name: r'postSearchProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postSearchHash();

  @override
  String toString() {
    return r'postSearchProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  PostSearch create() => PostSearch();

  @override
  bool operator ==(Object other) {
    return other is PostSearchProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postSearchHash() => r'c285c8af6ae38f9cf90963b5dbbe3ac1b6beb892';

final class PostSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          PostSearch,
          AsyncValue<SearchPostResultsState>,
          SearchPostResultsState,
          FutureOr<SearchPostResultsState>,
          PostSearchQuery
        > {
  PostSearchFamily._()
    : super(
        retry: null,
        name: r'postSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PostSearchProvider call(PostSearchQuery query) =>
      PostSearchProvider._(argument: query, from: this);

  @override
  String toString() => r'postSearchProvider';
}

abstract class _$PostSearch extends $AsyncNotifier<SearchPostResultsState> {
  late final _$args = ref.$arg as PostSearchQuery;
  PostSearchQuery get query => _$args;

  FutureOr<SearchPostResultsState> build(PostSearchQuery query);
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
