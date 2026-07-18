// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'session_registry_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// The sole mutable source for retained CraftSky account sessions.

@ProviderFor(SessionRegistry)
final sessionRegistryProvider = SessionRegistryProvider._();

/// The sole mutable source for retained CraftSky account sessions.
final class SessionRegistryProvider
    extends $AsyncNotifierProvider<SessionRegistry, registry.SessionRegistry> {
  /// The sole mutable source for retained CraftSky account sessions.
  SessionRegistryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'sessionRegistryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$sessionRegistryHash();

  @$internal
  @override
  SessionRegistry create() => SessionRegistry();
}

String _$sessionRegistryHash() => r'f01951a080b5bd516e791f055a99d7b0a48755c5';

/// The sole mutable source for retained CraftSky account sessions.

abstract class _$SessionRegistry
    extends $AsyncNotifier<registry.SessionRegistry> {
  FutureOr<registry.SessionRegistry> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<registry.SessionRegistry>,
              registry.SessionRegistry
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<registry.SessionRegistry>,
                registry.SessionRegistry
              >,
              AsyncValue<registry.SessionRegistry>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}
