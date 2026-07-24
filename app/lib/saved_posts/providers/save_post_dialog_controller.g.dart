// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'save_post_dialog_controller.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(SavePostDialogController)
final savePostDialogControllerProvider = SavePostDialogControllerFamily._();

final class SavePostDialogControllerProvider
    extends $NotifierProvider<SavePostDialogController, SavePostDialogState> {
  SavePostDialogControllerProvider._({
    required SavePostDialogControllerFamily super.from,
    required SavePostDialogKey super.argument,
  }) : super(
         retry: null,
         name: r'savePostDialogControllerProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$savePostDialogControllerHash();

  @override
  String toString() {
    return r'savePostDialogControllerProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  SavePostDialogController create() => SavePostDialogController();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(SavePostDialogState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<SavePostDialogState>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is SavePostDialogControllerProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$savePostDialogControllerHash() =>
    r'a6f96236333cb7c85337c1b77e8e095111483801';

final class SavePostDialogControllerFamily extends $Family
    with
        $ClassFamilyOverride<
          SavePostDialogController,
          SavePostDialogState,
          SavePostDialogState,
          SavePostDialogState,
          SavePostDialogKey
        > {
  SavePostDialogControllerFamily._()
    : super(
        retry: null,
        name: r'savePostDialogControllerProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  SavePostDialogControllerProvider call(SavePostDialogKey key) =>
      SavePostDialogControllerProvider._(argument: key, from: this);

  @override
  String toString() => r'savePostDialogControllerProvider';
}

abstract class _$SavePostDialogController
    extends $Notifier<SavePostDialogState> {
  late final _$args = ref.$arg as SavePostDialogKey;
  SavePostDialogKey get key => _$args;

  SavePostDialogState build(SavePostDialogKey key);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<SavePostDialogState, SavePostDialogState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<SavePostDialogState, SavePostDialogState>,
              SavePostDialogState,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
