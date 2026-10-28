// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'post_edit_seed_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Editor state belongs to one activation, including a same-account re-login.

@ProviderFor(postEditSeed)
final postEditSeedProvider = PostEditSeedFamily._();

/// Editor state belongs to one activation, including a same-account re-login.

final class PostEditSeedProvider
    extends
        $FunctionalProvider<
          AsyncValue<PostEditSeed>,
          PostEditSeed,
          FutureOr<PostEditSeed>
        >
    with $FutureModifier<PostEditSeed>, $FutureProvider<PostEditSeed> {
  /// Editor state belongs to one activation, including a same-account re-login.
  PostEditSeedProvider._({
    required PostEditSeedFamily super.from,
    required (ActiveAccountLease, Did, RecordKey, {String? requestId})
    super.argument,
  }) : super(
         retry: null,
         name: r'postEditSeedProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postEditSeedHash();

  @override
  String toString() {
    return r'postEditSeedProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<PostEditSeed> $createElement(
    $ProviderPointer pointer,
  ) => $FutureProviderElement(pointer);

  @override
  FutureOr<PostEditSeed> create(Ref ref) {
    final argument =
        this.argument
            as (ActiveAccountLease, Did, RecordKey, {String? requestId});
    return postEditSeed(
      ref,
      argument.$1,
      argument.$2,
      argument.$3,
      requestId: argument.requestId,
    );
  }

  @override
  bool operator ==(Object other) {
    return other is PostEditSeedProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postEditSeedHash() => r'634a50e6ef6752115d4c2b5bc831501385b24d23';

/// Editor state belongs to one activation, including a same-account re-login.

final class PostEditSeedFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<PostEditSeed>,
          (ActiveAccountLease, Did, RecordKey, {String? requestId})
        > {
  PostEditSeedFamily._()
    : super(
        retry: null,
        name: r'postEditSeedProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Editor state belongs to one activation, including a same-account re-login.

  PostEditSeedProvider call(
    ActiveAccountLease ownership,
    Did did,
    RecordKey key, {
    String? requestId,
  }) => PostEditSeedProvider._(
    argument: (ownership, did, key, requestId: requestId),
    from: this,
  );

  @override
  String toString() => r'postEditSeedProvider';
}
