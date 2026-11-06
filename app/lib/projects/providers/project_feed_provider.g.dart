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
    required (ProjectBrowseQuery, {BrowsingContext? audience}) super.argument,
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
        '$argument';
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

String _$projectFeedHash() => r'ef89dbdaed60e477b3203cd63b3e054700e3edc7';

final class ProjectFeedFamily extends $Family
    with
        $ClassFamilyOverride<
          ProjectFeed,
          AsyncValue<UserProjectsState>,
          UserProjectsState,
          FutureOr<UserProjectsState>,
          (ProjectBrowseQuery, {BrowsingContext? audience})
        > {
  ProjectFeedFamily._()
    : super(
        retry: null,
        name: r'projectFeedProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProjectFeedProvider call(
    ProjectBrowseQuery query, {
    BrowsingContext? audience,
  }) =>
      ProjectFeedProvider._(argument: (query, audience: audience), from: this);

  @override
  String toString() => r'projectFeedProvider';
}

abstract class _$ProjectFeed extends $AsyncNotifier<UserProjectsState> {
  late final _$args =
      ref.$arg as (ProjectBrowseQuery, {BrowsingContext? audience});
  ProjectBrowseQuery get query => _$args.$1;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<UserProjectsState> build(
    ProjectBrowseQuery query, {
    BrowsingContext? audience,
  });
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
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
    return element.handleCreate(
      ref,
      () => build(_$args.$1, audience: _$args.audience),
    );
  }
}
