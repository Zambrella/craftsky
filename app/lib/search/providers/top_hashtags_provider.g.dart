// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'top_hashtags_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(topHashtags)
final topHashtagsProvider = TopHashtagsFamily._();

final class TopHashtagsProvider
    extends
        $FunctionalProvider<
          AsyncValue<TopHashtagsResponse>,
          TopHashtagsResponse,
          FutureOr<TopHashtagsResponse>
        >
    with
        $FutureModifier<TopHashtagsResponse>,
        $FutureProvider<TopHashtagsResponse> {
  TopHashtagsProvider._({
    required TopHashtagsFamily super.from,
    required TopHashtagsQuery super.argument,
  }) : super(
         retry: null,
         name: r'topHashtagsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$topHashtagsHash();

  @override
  String toString() {
    return r'topHashtagsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<TopHashtagsResponse> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<TopHashtagsResponse> create(Ref ref) {
    final argument = this.argument as TopHashtagsQuery;
    return topHashtags(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is TopHashtagsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$topHashtagsHash() => r'c70ffea587ee237f37629afa1aa7bdeb5d0cb324';

final class TopHashtagsFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<TopHashtagsResponse>,
          TopHashtagsQuery
        > {
  TopHashtagsFamily._()
    : super(
        retry: null,
        name: r'topHashtagsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  TopHashtagsProvider call(TopHashtagsQuery query) =>
      TopHashtagsProvider._(argument: query, from: this);

  @override
  String toString() => r'topHashtagsProvider';
}
