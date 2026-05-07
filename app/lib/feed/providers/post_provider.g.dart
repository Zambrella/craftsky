// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'post_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Single-post read by `(did, rkey)`. No UI consumer in v1; exists for
/// future routes (deep-link share, thread page).

@ProviderFor(post)
final postProvider = PostFamily._();

/// Single-post read by `(did, rkey)`. No UI consumer in v1; exists for
/// future routes (deep-link share, thread page).

final class PostProvider
    extends $FunctionalProvider<AsyncValue<Post>, Post, FutureOr<Post>>
    with $FutureModifier<Post>, $FutureProvider<Post> {
  /// Single-post read by `(did, rkey)`. No UI consumer in v1; exists for
  /// future routes (deep-link share, thread page).
  PostProvider._({
    required PostFamily super.from,
    required (String, String) super.argument,
  }) : super(
         retry: null,
         name: r'postProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postHash();

  @override
  String toString() {
    return r'postProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<Post> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<Post> create(Ref ref) {
    final argument = this.argument as (String, String);
    return post(ref, argument.$1, argument.$2);
  }

  @override
  bool operator ==(Object other) {
    return other is PostProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postHash() => r'a7f3ea41d0704a39c02d22783e67336a5a545a7c';

/// Single-post read by `(did, rkey)`. No UI consumer in v1; exists for
/// future routes (deep-link share, thread page).

final class PostFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<Post>, (String, String)> {
  PostFamily._()
    : super(
        retry: null,
        name: r'postProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  /// Single-post read by `(did, rkey)`. No UI consumer in v1; exists for
  /// future routes (deep-link share, thread page).

  PostProvider call(String did, String rkey) =>
      PostProvider._(argument: (did, rkey), from: this);

  @override
  String toString() => r'postProvider';
}
