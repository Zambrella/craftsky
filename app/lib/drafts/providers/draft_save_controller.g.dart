// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'draft_save_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(DraftSaveController)
final draftSaveControllerProvider = DraftSaveControllerFamily._();

final class DraftSaveControllerProvider
    extends $AsyncNotifierProvider<DraftSaveController, LocalPostDraft?> {
  DraftSaveControllerProvider._({
    required DraftSaveControllerFamily super.from,
    required AccountKey super.argument,
  }) : super(
         retry: null,
         name: r'draftSaveControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$draftSaveControllerHash();

  @override
  String toString() {
    return r'draftSaveControllerProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  DraftSaveController create() => DraftSaveController();

  @override
  bool operator ==(Object other) {
    return other is DraftSaveControllerProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$draftSaveControllerHash() =>
    r'489f88152e4f4a59e84f241628d06ed1b09df87f';

final class DraftSaveControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          DraftSaveController,
          AsyncValue<LocalPostDraft?>,
          LocalPostDraft?,
          FutureOr<LocalPostDraft?>,
          AccountKey
        > {
  DraftSaveControllerFamily._()
    : super(
        retry: null,
        name: r'draftSaveControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  DraftSaveControllerProvider call(AccountKey account) =>
      DraftSaveControllerProvider._(argument: account, from: this);

  @override
  String toString() => r'draftSaveControllerProvider';
}

abstract class _$DraftSaveController extends $AsyncNotifier<LocalPostDraft?> {
  late final _$args = ref.$arg as AccountKey;
  AccountKey get account => _$args;

  FutureOr<LocalPostDraft?> build(AccountKey account);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<LocalPostDraft?>, LocalPostDraft?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<LocalPostDraft?>, LocalPostDraft?>,
              AsyncValue<LocalPostDraft?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
