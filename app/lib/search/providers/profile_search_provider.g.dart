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
    required (ProfileSearchQuery, {BrowsingContext? audience}) super.argument,
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
        '$argument';
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

String _$profileSearchHash() => r'1f46c3cbc1000fd5c8dd45f3badff21ee917ef85';

final class ProfileSearchFamily extends $Family
    with
        $ClassFamilyOverride<
          ProfileSearch,
          AsyncValue<ProfileSearchResultsState>,
          ProfileSearchResultsState,
          FutureOr<ProfileSearchResultsState>,
          (ProfileSearchQuery, {BrowsingContext? audience})
        > {
  ProfileSearchFamily._()
    : super(
        retry: null,
        name: r'profileSearchProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ProfileSearchProvider call(
    ProfileSearchQuery query, {
    BrowsingContext? audience,
  }) => ProfileSearchProvider._(
    argument: (query, audience: audience),
    from: this,
  );

  @override
  String toString() => r'profileSearchProvider';
}

abstract class _$ProfileSearch
    extends $AsyncNotifier<ProfileSearchResultsState> {
  late final _$args =
      ref.$arg as (ProfileSearchQuery, {BrowsingContext? audience});
  ProfileSearchQuery get query => _$args.$1;
  BrowsingContext? get audience => _$args.audience;

  FutureOr<ProfileSearchResultsState> build(
    ProfileSearchQuery query, {
    BrowsingContext? audience,
  });
  @$mustCallSuper
  @override
  WhenComplete runBuild() {
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
    return element.handleCreate(
      ref,
      () => build(_$args.$1, audience: _$args.audience),
    );
  }
}
