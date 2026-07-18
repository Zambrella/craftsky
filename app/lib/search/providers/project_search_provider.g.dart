// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'project_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProjectSearch)
final projectSearchProvider = ProjectSearchFamily._();

final class ProjectSearchProvider
    extends $AsyncNotifierProvider<ProjectSearch, SearchPostResultsState> {
  ProjectSearchProvider._({
    required ProjectSearchFamily super.from,
    required ProjectSearchQuery super.argument,
  }) : super(
         retry: null,
         name: r'projectSearchProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$projectSearchHash();

  @override
  String toString() {
    return r'projectSearchProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProjectSearch create() => ProjectSearch();

  @override
  bool operator ==(Object other) {
    return other is ProjectSearchProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$projectSearchHash() => r'f1effc126cb990cf9d31e0efc7d66f9a3eb2a5ef';

final class ProjectSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          ProjectSearch,
          AsyncValue<SearchPostResultsState>,
          SearchPostResultsState,
          FutureOr<SearchPostResultsState>,
          ProjectSearchQuery
        > {
  ProjectSearchFamily._()
    : super(
        retry: null,
        name: r'projectSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProjectSearchProvider call(ProjectSearchQuery query) =>
      ProjectSearchProvider._(argument: query, from: this);

  @override
  String toString() => r'projectSearchProvider';
}

abstract class _$ProjectSearch extends $AsyncNotifier<SearchPostResultsState> {
  late final _$args = ref.$arg as ProjectSearchQuery;
  ProjectSearchQuery get query => _$args;

  FutureOr<SearchPostResultsState> build(ProjectSearchQuery query);
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
