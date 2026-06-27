// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'search_suggestions_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(searchSuggestions)
final searchSuggestionsProvider = SearchSuggestionsFamily._();

final class SearchSuggestionsProvider
    extends
        $FunctionalProvider<
          AsyncValue<SearchSuggestions>,
          SearchSuggestions,
          FutureOr<SearchSuggestions>
        >
    with
        $FutureModifier<SearchSuggestions>,
        $FutureProvider<SearchSuggestions> {
  SearchSuggestionsProvider._({
    required SearchSuggestionsFamily super.from,
    required SearchSuggestionQuery super.argument,
  }) : super(
         retry: null,
         name: r'searchSuggestionsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$searchSuggestionsHash();

  @override
  String toString() {
    return r'searchSuggestionsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  $FutureProviderElement<SearchSuggestions> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<SearchSuggestions> create(Ref ref) {
    final argument = this.argument as SearchSuggestionQuery;
    return searchSuggestions(ref, argument);
  }

  @override
  bool operator ==(Object other) {
    return other is SearchSuggestionsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$searchSuggestionsHash() => r'f47e6fad6b0b6f0ed72adbf669d2f660997c7320';

final class SearchSuggestionsFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<SearchSuggestions>,
          SearchSuggestionQuery
        > {
  SearchSuggestionsFamily._()
    : super(
        retry: null,
        name: r'searchSuggestionsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  SearchSuggestionsProvider call(SearchSuggestionQuery query) =>
      SearchSuggestionsProvider._(argument: query, from: this);

  @override
  String toString() => r'searchSuggestionsProvider';
}
