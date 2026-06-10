// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_projects_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(UserProjects)
final userProjectsProvider = UserProjectsFamily._();

final class UserProjectsProvider
    extends $AsyncNotifierProvider<UserProjects, UserProjectsState> {
  UserProjectsProvider._({
    required UserProjectsFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'userProjectsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$userProjectsHash();

  @override
  String toString() {
    return r'userProjectsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  UserProjects create() => UserProjects();

  @override
  bool operator ==(Object other) {
    return other is UserProjectsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$userProjectsHash() => r'0a8a8b24f43e22a4f975f7f0361af80e99f95e65';

final class UserProjectsFamily extends $Family
    with
        $ClassFamilyOverride<
          UserProjects,
          AsyncValue<UserProjectsState>,
          UserProjectsState,
          FutureOr<UserProjectsState>,
          String
        > {
  UserProjectsFamily._()
    : super(
        retry: null,
        name: r'userProjectsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  UserProjectsProvider call(String handleOrDid) =>
      UserProjectsProvider._(argument: handleOrDid, from: this);

  @override
  String toString() => r'userProjectsProvider';
}

abstract class _$UserProjects extends $AsyncNotifier<UserProjectsState> {
  late final _$args = ref.$arg as String;
  String get handleOrDid => _$args;

  FutureOr<UserProjectsState> build(String handleOrDid);
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
