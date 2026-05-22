// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'composer_images_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(ComposerImages)
final composerImagesProvider = ComposerImagesFamily._();

final class ComposerImagesProvider
    extends $NotifierProvider<ComposerImages, ComposerImagesState> {
  ComposerImagesProvider._({
    required ComposerImagesFamily super.from,
    required String super.argument,
  }) : super(
         retry: null,
         name: r'composerImagesProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$composerImagesHash();

  @override
  String toString() {
    return r'composerImagesProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  ComposerImages create() => ComposerImages();

  /// {@macro riverpod.override_with_value}
  Override overrideWithValue(ComposerImagesState value) {
    return $ProviderOverride(
      origin: this,
      providerOverride: $SyncValueProvider<ComposerImagesState>(value),
    );
  }

  @override
  bool operator ==(Object other) {
    return other is ComposerImagesProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$composerImagesHash() => r'eca1b40697d312b528a6e4ea7455010ed6b63788';

final class ComposerImagesFamily extends $Family
    with
        $ClassFamilyOverride<
          ComposerImages,
          ComposerImagesState,
          ComposerImagesState,
          ComposerImagesState,
          String
        > {
  ComposerImagesFamily._()
    : super(
        retry: null,
        name: r'composerImagesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  ComposerImagesProvider call(String composerId) =>
      ComposerImagesProvider._(argument: composerId, from: this);

  @override
  String toString() => r'composerImagesProvider';
}

abstract class _$ComposerImages extends $Notifier<ComposerImagesState> {
  late final _$args = ref.$arg as String;
  String get composerId => _$args;

  ComposerImagesState build(String composerId);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<ComposerImagesState, ComposerImagesState>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<ComposerImagesState, ComposerImagesState>,
              ComposerImagesState,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
