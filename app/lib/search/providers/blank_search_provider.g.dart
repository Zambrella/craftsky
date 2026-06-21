// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'blank_search_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(blankSearch)
final blankSearchProvider = BlankSearchProvider._();

final class BlankSearchProvider
    extends
        $FunctionalProvider<
          AsyncValue<BlankSearchData>,
          BlankSearchData,
          FutureOr<BlankSearchData>
        >
    with $FutureModifier<BlankSearchData>, $FutureProvider<BlankSearchData> {
  BlankSearchProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'blankSearchProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$blankSearchHash();

  @$internal
  @override
  $FutureProviderElement<BlankSearchData> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<BlankSearchData> create(Ref ref) {
    return blankSearch(ref);
  }
}

String _$blankSearchHash() => r'f263c1afcb47d828822a6229618b463f82b8c8ea';
