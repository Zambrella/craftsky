// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'instagram_suggestions_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(InstagramSuggestions)
final instagramSuggestionsProvider = InstagramSuggestionsFamily._();

final class InstagramSuggestionsProvider
    extends
        $AsyncNotifierProvider<
          InstagramSuggestions,
          InstagramSuggestionReviewState
        > {
  InstagramSuggestionsProvider._({
    required InstagramSuggestionsFamily super.from,
    required ActiveAccountLease super.argument,
  }) : super(
         retry: null,
         name: r'instagramSuggestionsProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$instagramSuggestionsHash();

  @override
  String toString() {
    return r'instagramSuggestionsProvider'
        ''
        '($argument)';
  }

  @$internal
  @override
  InstagramSuggestions create() => InstagramSuggestions();

  @override
  bool operator ==(Object other) {
    return other is InstagramSuggestionsProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$instagramSuggestionsHash() =>
    r'384b529926d16af30bd333b120fc2f1c69ddf101';

final class InstagramSuggestionsFamily extends $Family
    with
        $ClassFamilyOverride<
          InstagramSuggestions,
          AsyncValue<InstagramSuggestionReviewState>,
          InstagramSuggestionReviewState,
          FutureOr<InstagramSuggestionReviewState>,
          ActiveAccountLease
        > {
  InstagramSuggestionsFamily._()
    : super(
        retry: null,
        name: r'instagramSuggestionsProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  InstagramSuggestionsProvider call(ActiveAccountLease lease) =>
      InstagramSuggestionsProvider._(argument: lease, from: this);

  @override
  String toString() => r'instagramSuggestionsProvider';
}

abstract class _$InstagramSuggestions
    extends $AsyncNotifier<InstagramSuggestionReviewState> {
  late final _$args = ref.$arg as ActiveAccountLease;
  ActiveAccountLease get lease => _$args;

  FutureOr<InstagramSuggestionReviewState> build(ActiveAccountLease lease);
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<InstagramSuggestionReviewState>,
              InstagramSuggestionReviewState
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<InstagramSuggestionReviewState>,
                InstagramSuggestionReviewState
              >,
              AsyncValue<InstagramSuggestionReviewState>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, () => build(_$args));
  }
}
