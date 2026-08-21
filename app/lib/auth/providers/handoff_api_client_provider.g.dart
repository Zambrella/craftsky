// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'handoff_api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Uses the anonymous client for code redemption. The pending bearer is passed
/// directly to the one confirmation request, never retained in provider
/// identity or diagnostics.

@ProviderFor(handoffApiClient)
final handoffApiClientProvider = HandoffApiClientProvider._();

/// Uses the anonymous client for code redemption. The pending bearer is passed
/// directly to the one confirmation request, never retained in provider
/// identity or diagnostics.

final class HandoffApiClientProvider
    extends
        $FunctionalProvider<
          HandoffApiClient,
          HandoffApiClient,
          HandoffApiClient
        >
    with $Provider<HandoffApiClient> {
  /// Uses the anonymous client for code redemption. The pending bearer is passed
  /// directly to the one confirmation request, never retained in provider
  /// identity or diagnostics.
  HandoffApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'handoffApiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$handoffApiClientHash();

  @$internal
  @override
  $ProviderElement<HandoffApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  HandoffApiClient create(Ref ref) {
    return handoffApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(HandoffApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<HandoffApiClient>(value),
    );
  }
}

String _$handoffApiClientHash() => r'f28efecf8e6aefeea5da98330db179b759ecdc2b';
