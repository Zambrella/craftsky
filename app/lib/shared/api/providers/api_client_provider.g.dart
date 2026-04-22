// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(craftskyApiClient)
final craftskyApiClientProvider = CraftskyApiClientProvider._();

final class CraftskyApiClientProvider
    extends
        $FunctionalProvider<
          CraftskyApiClient,
          CraftskyApiClient,
          CraftskyApiClient
        >
    with $Provider<CraftskyApiClient> {
  CraftskyApiClientProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'craftskyApiClientProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$craftskyApiClientHash();

  @$internal
  @override
  $ProviderElement<CraftskyApiClient> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  CraftskyApiClient create(Ref ref) {
    return craftskyApiClient(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(CraftskyApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<CraftskyApiClient>(value),
    );
  }
}

String _$craftskyApiClientHash() => r'0c24d2ad6f99c0784a0bbed6c68b9cd43a8fe8e0';

/// Family-keyed by token: one instance per in-flight handoff. Not
/// keep-alive — auto-disposes when no one watches it, so the token
/// doesn't linger.

@ProviderFor(handoffApiClient)
final handoffApiClientProvider = HandoffApiClientFamily._();

/// Family-keyed by token: one instance per in-flight handoff. Not
/// keep-alive — auto-disposes when no one watches it, so the token
/// doesn't linger.

final class HandoffApiClientProvider
    extends
        $FunctionalProvider<
          HandoffApiClient,
          HandoffApiClient,
          HandoffApiClient
        >
    with $Provider<HandoffApiClient> {
  /// Family-keyed by token: one instance per in-flight handoff. Not
  /// keep-alive — auto-disposes when no one watches it, so the token
  /// doesn't linger.
  HandoffApiClientProvider._({
    required HandoffApiClientFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'handoffApiClientProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$handoffApiClientHash();

  @override
  String toString() {
    return r'handoffApiClientProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $ProviderElement<HandoffApiClient> $createElement($ProviderPointer pointer) =>
      $ProviderElement(pointer);

  @override
  HandoffApiClient create(Ref ref) {
    final argument = this.argument as String;
    return handoffApiClient(ref, argument);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(HandoffApiClient value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<HandoffApiClient>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is HandoffApiClientProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$handoffApiClientHash() => r'197e37d122f9978021e227262e2823045c66799f';

/// Family-keyed by token: one instance per in-flight handoff. Not
/// keep-alive — auto-disposes when no one watches it, so the token
/// doesn't linger.

final class HandoffApiClientFamily extends $Family
    with $FunctionalFamilyOverride<HandoffApiClient, String> {
  HandoffApiClientFamily._()
    : super(
        retry: null,
        name: r'handoffApiClientProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Family-keyed by token: one instance per in-flight handoff. Not
  /// keep-alive — auto-disposes when no one watches it, so the token
  /// doesn't linger.

  HandoffApiClientProvider call(String token) =>
      HandoffApiClientProvider._(argument: token, from: this);

  @override
  String toString() => r'handoffApiClientProvider';
}
