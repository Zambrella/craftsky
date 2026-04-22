// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'pending_auth_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Tracks the in-flight sign-in attempt. Lets
/// `AuthController.completeFromDeepLink` reject deep links that
/// arrive without a prior `signIn()` or later than the 10-minute
/// staleness window.
///
/// The notifier class is named `PendingAuth` — same identifier as
/// the data class it holds, imported under the `model` prefix to
/// dodge the collision inside this file. The generated provider is
/// `pendingAuthProvider`.

@ProviderFor(PendingAuth)
final pendingAuthProvider = PendingAuthProvider._();

/// Tracks the in-flight sign-in attempt. Lets
/// `AuthController.completeFromDeepLink` reject deep links that
/// arrive without a prior `signIn()` or later than the 10-minute
/// staleness window.
///
/// The notifier class is named `PendingAuth` — same identifier as
/// the data class it holds, imported under the `model` prefix to
/// dodge the collision inside this file. The generated provider is
/// `pendingAuthProvider`.
final class PendingAuthProvider
    extends $NotifierProvider<PendingAuth, model.PendingAuth?> {
  /// Tracks the in-flight sign-in attempt. Lets
  /// `AuthController.completeFromDeepLink` reject deep links that
  /// arrive without a prior `signIn()` or later than the 10-minute
  /// staleness window.
  ///
  /// The notifier class is named `PendingAuth` — same identifier as
  /// the data class it holds, imported under the `model` prefix to
  /// dodge the collision inside this file. The generated provider is
  /// `pendingAuthProvider`.
  PendingAuthProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'pendingAuthProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$pendingAuthHash();

  @$internal
  @override
  PendingAuth create() => PendingAuth();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(model.PendingAuth? value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<model.PendingAuth?>(value),
    );
  }
}

String _$pendingAuthHash() => r'e8ae0a167e88045e997d6fc13408a4de2b609bfc';

/// Tracks the in-flight sign-in attempt. Lets
/// `AuthController.completeFromDeepLink` reject deep links that
/// arrive without a prior `signIn()` or later than the 10-minute
/// staleness window.
///
/// The notifier class is named `PendingAuth` — same identifier as
/// the data class it holds, imported under the `model` prefix to
/// dodge the collision inside this file. The generated provider is
/// `pendingAuthProvider`.

abstract class _$PendingAuth extends $Notifier<model.PendingAuth?> {
  model.PendingAuth? build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<model.PendingAuth?, model.PendingAuth?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<model.PendingAuth?, model.PendingAuth?>,
              model.PendingAuth?,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
