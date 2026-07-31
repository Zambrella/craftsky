// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'active_account_initialization_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Resolves the account-critical state for the exact active session lease.
///
/// A signed-out registry is a successful initialization with no account.
/// Loading and failures from either dependency remain visible to the gate.

@ProviderFor(activeAccountInitialization)
final activeAccountInitializationProvider =
    ActiveAccountInitializationProvider._();

/// Resolves the account-critical state for the exact active session lease.
///
/// A signed-out registry is a successful initialization with no account.
/// Loading and failures from either dependency remain visible to the gate.

final class ActiveAccountInitializationProvider
    extends
        $FunctionalProvider<
          AsyncValue<ActiveAccountInitialization?>,
          ActiveAccountInitialization?,
          FutureOr<ActiveAccountInitialization?>
        >
    with
        $FutureModifier<ActiveAccountInitialization?>,
        $FutureProvider<ActiveAccountInitialization?> {
  /// Resolves the account-critical state for the exact active session lease.
  ///
  /// A signed-out registry is a successful initialization with no account.
  /// Loading and failures from either dependency remain visible to the gate.
  ActiveAccountInitializationProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'activeAccountInitializationProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$activeAccountInitializationHash();

  @$internal
  @override
  $FutureProviderElement<ActiveAccountInitialization?> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<ActiveAccountInitialization?> create(Ref ref) {
    return activeAccountInitialization(ref);
  }
}

String _$activeAccountInitializationHash() =>
    r'b7194735b85a2cc2198b4e1f81e5c731e1bfd21e';
