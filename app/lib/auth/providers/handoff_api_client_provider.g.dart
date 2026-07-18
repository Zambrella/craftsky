// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'handoff_api_client_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Family-keyed by a redacted credential wrapper: one instance per in-flight
/// handoff. Not keep-alive — auto-disposes when no one watches it, so
/// the token doesn't linger.
///
/// The server enforces X-Craftsky-Device-Id on every authenticated
/// /v1/* call, so the handoff Dio bakes both Authorization and
/// X-Craftsky-Device-Id into BaseOptions. Callers must pre-resolve
/// `deviceIdProvider.future` and pass the value explicitly — this
/// keeps the provider itself synchronous.

@ProviderFor(handoffApiClient)
final handoffApiClientProvider = HandoffApiClientFamily._();

/// Family-keyed by a redacted credential wrapper: one instance per in-flight
/// handoff. Not keep-alive — auto-disposes when no one watches it, so
/// the token doesn't linger.
///
/// The server enforces X-Craftsky-Device-Id on every authenticated
/// /v1/* call, so the handoff Dio bakes both Authorization and
/// X-Craftsky-Device-Id into BaseOptions. Callers must pre-resolve
/// `deviceIdProvider.future` and pass the value explicitly — this
/// keeps the provider itself synchronous.

final class HandoffApiClientProvider
    extends
        $FunctionalProvider<
          HandoffApiClient,
          HandoffApiClient,
          HandoffApiClient
        >
    with $Provider<HandoffApiClient> {
  /// Family-keyed by a redacted credential wrapper: one instance per in-flight
  /// handoff. Not keep-alive — auto-disposes when no one watches it, so
  /// the token doesn't linger.
  ///
  /// The server enforces X-Craftsky-Device-Id on every authenticated
  /// /v1/* call, so the handoff Dio bakes both Authorization and
  /// X-Craftsky-Device-Id into BaseOptions. Callers must pre-resolve
  /// `deviceIdProvider.future` and pass the value explicitly — this
  /// keeps the provider itself synchronous.
  HandoffApiClientProvider._({
    required HandoffApiClientFamily super.from,
    required HandoffClientKey super.argument,
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
    final argument = this.argument as HandoffClientKey;
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

String _$handoffApiClientHash() => r'cc34af0f3913c026326e22f8d5288903ada6154d';

/// Family-keyed by a redacted credential wrapper: one instance per in-flight
/// handoff. Not keep-alive — auto-disposes when no one watches it, so
/// the token doesn't linger.
///
/// The server enforces X-Craftsky-Device-Id on every authenticated
/// /v1/* call, so the handoff Dio bakes both Authorization and
/// X-Craftsky-Device-Id into BaseOptions. Callers must pre-resolve
/// `deviceIdProvider.future` and pass the value explicitly — this
/// keeps the provider itself synchronous.

final class HandoffApiClientFamily extends $Family
    with $FunctionalFamilyOverride<HandoffApiClient, HandoffClientKey> {
  HandoffApiClientFamily._()
    : super(
        retry: null,
        name: r'handoffApiClientProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Family-keyed by a redacted credential wrapper: one instance per in-flight
  /// handoff. Not keep-alive — auto-disposes when no one watches it, so
  /// the token doesn't linger.
  ///
  /// The server enforces X-Craftsky-Device-Id on every authenticated
  /// /v1/* call, so the handoff Dio bakes both Authorization and
  /// X-Craftsky-Device-Id into BaseOptions. Callers must pre-resolve
  /// `deviceIdProvider.future` and pass the value explicitly — this
  /// keeps the provider itself synchronous.

  HandoffApiClientProvider call(HandoffClientKey key) =>
      HandoffApiClientProvider._(argument: key, from: this);

  @override
  String toString() => r'handoffApiClientProvider';
}
