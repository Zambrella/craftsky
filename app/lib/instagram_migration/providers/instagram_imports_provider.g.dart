// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'instagram_imports_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(InstagramImports)
final instagramImportsProvider = InstagramImportsFamily._();

final class InstagramImportsProvider
    extends $AsyncNotifierProvider<InstagramImports, InstagramImportPage> {
  InstagramImportsProvider._({
    required InstagramImportsFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'instagramImportsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$instagramImportsHash();

  @override
  String toString() {
    return r'instagramImportsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  InstagramImports create() => InstagramImports();

  @override
  bool operator ==(Object other) {
    return other is InstagramImportsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$instagramImportsHash() => r'abc5dcd46dd3f6acf629fef0dd145bbffd982b91';

final class InstagramImportsFamily extends $Family
    with
        $ClassFamilyOverride<
          InstagramImports,
          AsyncValue<InstagramImportPage>,
          InstagramImportPage,
          FutureOr<InstagramImportPage>,
          ActiveAccountLease
        > {
  InstagramImportsFamily._()
    : super(
        retry: null,
        name: r'instagramImportsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  InstagramImportsProvider call(ActiveAccountLease lease) =>
      InstagramImportsProvider._(argument: lease, from: this);

  @override
  String toString() => r'instagramImportsProvider';
}

abstract class _$InstagramImports extends $AsyncNotifier<InstagramImportPage> {
  late final _$args = ref.$arg as ActiveAccountLease;
  ActiveAccountLease get lease => _$args;

  FutureOr<InstagramImportPage> build(ActiveAccountLease lease);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref as $Ref<AsyncValue<InstagramImportPage>, InstagramImportPage>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<InstagramImportPage>, InstagramImportPage>,
              AsyncValue<InstagramImportPage>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
