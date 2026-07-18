// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'profile_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ProfileSearch)
final profileSearchProvider = ProfileSearchFamily._();

final class ProfileSearchProvider
    extends $AsyncNotifierProvider<ProfileSearch, ProfileSearchResultsState> {
  ProfileSearchProvider._({
    required ProfileSearchFamily super.from,
    required ProfileSearchQuery super.argument,
  }) : super(
         retry: null,
         name: r'profileSearchProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$profileSearchHash();

  @override
  String toString() {
    return r'profileSearchProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ProfileSearch create() => ProfileSearch();

  @override
  bool operator ==(Object other) {
    return other is ProfileSearchProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$profileSearchHash() => r'a263cddc8a5e4890b71a8f198f3f70f100ac299c';

final class ProfileSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileSearch,
          AsyncValue<ProfileSearchResultsState>,
          ProfileSearchResultsState,
          FutureOr<ProfileSearchResultsState>,
          ProfileSearchQuery
        > {
  ProfileSearchFamily._()
    : super(
        retry: null,
        name: r'profileSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileSearchProvider call(ProfileSearchQuery query) =>
      ProfileSearchProvider._(argument: query, from: this);

  @override
  String toString() => r'profileSearchProvider';
}

abstract class _$ProfileSearch
    extends $AsyncNotifier<ProfileSearchResultsState> {
  late final _$args = ref.$arg as ProfileSearchQuery;
  ProfileSearchQuery get query => _$args;

  FutureOr<ProfileSearchResultsState> build(ProfileSearchQuery query);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<ProfileSearchResultsState>,
              ProfileSearchResultsState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<ProfileSearchResultsState>,
                ProfileSearchResultsState
              >,
              AsyncValue<ProfileSearchResultsState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
