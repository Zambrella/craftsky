// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'project_api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(projectApiClient)
final projectApiClientProvider = ProjectApiClientProvider._();

final class ProjectApiClientProvider
    extends
        $FunctionalProvider<
          ProjectApiClient,
          ProjectApiClient,
          ProjectApiClient
        >
    with $Provider<ProjectApiClient> {
  ProjectApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'projectApiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$projectApiClientHash();

  @$internal
  @override
  $ProviderElement<ProjectApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  ProjectApiClient create(Ref ref) {
    return projectApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ProjectApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ProjectApiClient>(value),
    );
  }
}

String _$projectApiClientHash() => r'7cc7e536e9d0680c45ca2476b7283946622f6ec5';
