// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'project_feed_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProjectFeed)
final projectFeedProvider = ProjectFeedFamily._();

final class ProjectFeedProvider
    extends $AsyncNotifierProvider<ProjectFeed, UserProjectsState> {
  ProjectFeedProvider._({
    required ProjectFeedFamily super.from,
    required ProjectBrowseQuery super.argument,
  }) : super(
         retry: null,
         name: r'projectFeedProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$projectFeedHash();

  @override
  String toString() {
    return r'projectFeedProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProjectFeed create() => ProjectFeed();

  @override
  bool operator ==(Object other) {
    return other is ProjectFeedProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$projectFeedHash() => r'9ea05c9f069e173aea765c7c568abebe8a6f55c2';

final class ProjectFeedFamily extends $Family
    with
        $ClassFamilyOverride<
          ProjectFeed,
          AsyncValue<UserProjectsState>,
          UserProjectsState,
          FutureOr<UserProjectsState>,
          ProjectBrowseQuery
        > {
  ProjectFeedFamily._()
    : super(
        retry: null,
        name: r'projectFeedProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProjectFeedProvider call(ProjectBrowseQuery query) =>
      ProjectFeedProvider._(argument: query, from: this);

  @override
  String toString() => r'projectFeedProvider';
}

abstract class _$ProjectFeed extends $AsyncNotifier<UserProjectsState> {
  late final _$args = ref.$arg as ProjectBrowseQuery;
  ProjectBrowseQuery get query => _$args;

  FutureOr<UserProjectsState> build(ProjectBrowseQuery query);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<UserProjectsState>, UserProjectsState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<UserProjectsState>, UserProjectsState>,
              AsyncValue<UserProjectsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
