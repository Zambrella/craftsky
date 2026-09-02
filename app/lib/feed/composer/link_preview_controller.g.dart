// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'link_preview_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(linkPreviewRepository)
final linkPreviewRepositoryProvider = LinkPreviewRepositoryProvider._();

final class LinkPreviewRepositoryProvider
    extends
        $FunctionalProvider<
          LinkPreviewRepository,
          LinkPreviewRepository,
          LinkPreviewRepository
        >
    with $Provider<LinkPreviewRepository> {
  LinkPreviewRepositoryProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'linkPreviewRepositoryProvider',
        isAutoDispose: false,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$linkPreviewRepositoryHash();

  @$internal
  @override
  $ProviderElement<LinkPreviewRepository> $createElement(
    $ProviderPointer pointer,
  ) => $ProviderElement(pointer);

  @override
  LinkPreviewRepository create(Ref ref) {
    return linkPreviewRepository(ref);
  }

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(LinkPreviewRepository value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<LinkPreviewRepository>(value),
    );
  }
}

String _$linkPreviewRepositoryHash() =>
    r'138b82ce69af5ec6a132bd0a5fa29b72a0737351';

@ProviderFor(LinkPreviewController)
final linkPreviewControllerProvider = LinkPreviewControllerFamily._();

final class LinkPreviewControllerProvider
    extends $NotifierProvider<LinkPreviewController, LinkPreviewSessionState> {
  LinkPreviewControllerProvider._({
    required LinkPreviewControllerFamily super.from,
    required (String, AccountKey) super.argument,
  }) : super(
         retry: null,
         name: r'linkPreviewControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$linkPreviewControllerHash();

  @override
  String toString() {
    return r'linkPreviewControllerProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  LinkPreviewController create() => LinkPreviewController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(LinkPreviewSessionState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<LinkPreviewSessionState>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is LinkPreviewControllerProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$linkPreviewControllerHash() =>
    r'c75ed9d25ec601d7a426ef2371dbed6b816352ef';

final class LinkPreviewControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          LinkPreviewController,
          LinkPreviewSessionState,
          LinkPreviewSessionState,
          LinkPreviewSessionState,
          (String, AccountKey)
        > {
  LinkPreviewControllerFamily._()
    : super(
        retry: null,
        name: r'linkPreviewControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  LinkPreviewControllerProvider call(
    String composerId,
    AccountKey accountKey,
  ) => LinkPreviewControllerProvider._(
    argument: (composerId, accountKey),
    from: this,
  );

  @override
  String toString() => r'linkPreviewControllerProvider';
}

abstract class _$LinkPreviewController
    extends $Notifier<LinkPreviewSessionState> {
  late final _$args = ref.$arg as (String, AccountKey);
  String get composerId => _$args.$1;
  AccountKey get accountKey => _$args.$2;

  LinkPreviewSessionState build(String composerId, AccountKey accountKey);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<LinkPreviewSessionState, LinkPreviewSessionState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<LinkPreviewSessionState, LinkPreviewSessionState>,
              LinkPreviewSessionState,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args.$1, _$args.$2));
  }
}
