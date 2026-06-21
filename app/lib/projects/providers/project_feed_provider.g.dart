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
    required ({String? craftType, SearchSort sort}) super.argument,
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

String _$projectFeedHash() => r'4a3c1812f1aea9b110d9a2d742c5d9b538841aaa';

final class ProjectFeedFamily extends $Family
    with
        $ClassFamilyOverride<
          ProjectFeed,
          AsyncValue<UserProjectsState>,
          UserProjectsState,
          FutureOr<UserProjectsState>,
          ({String? craftType, SearchSort sort})
        > {
  ProjectFeedFamily._()
    : super(
        retry: null,
        name: r'projectFeedProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProjectFeedProvider call({
    String? craftType,
    SearchSort sort = SearchSort.chronological,
  }) => ProjectFeedProvider._(
    argument: (craftType: craftType, sort: sort),
    from: this,
  );

  @override
  String toString() => r'projectFeedProvider';
}

abstract class _$ProjectFeed extends $AsyncNotifier<UserProjectsState> {
  late final _$args = ref.$arg as ({String? craftType, SearchSort sort});
  String? get craftType => _$args.craftType;
  SearchSort get sort => _$args.sort;

  FutureOr<UserProjectsState> build({
    String? craftType,
    SearchSort sort = SearchSort.chronological,
  });
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
    element.handleCreate(
      ref,
      () => build(craftType: _$args.craftType, sort: _$args.sort),
    );
  }
}
