// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'post_thread_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(directReplies)
final directRepliesProvider = DirectRepliesFamily._();

final class DirectRepliesProvider
    extends
        $FunctionalProvider<AsyncValue<PostPage>, PostPage, FutureOr<PostPage>>
    with $FutureModifier<PostPage>, $FutureProvider<PostPage> {
  DirectRepliesProvider._({
    required DirectRepliesFamily super.from,
    required (String, String, {String? cursor, int? limit}) super.argument,
  }) : super(
         retry: null,
         name: r'directRepliesProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$directRepliesHash();

  @override
  String toString() {
    return r'directRepliesProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<PostPage> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<PostPage> create(Ref ref) {
    final argument =
        this.argument as (String, String, {String? cursor, int? limit});
    return directReplies(
      ref,
      argument.$1,
      argument.$2,
      cursor: argument.cursor,
      limit: argument.limit,
    );
  }

  @override
  bool operator ==(Object other) {
    return other is DirectRepliesProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$directRepliesHash() => r'a519a75341d56589f2ba82b835e80a2f7a95cf34';

final class DirectRepliesFamily extends $Family
    with
        $FunctionalFamilyOverride<
          FutureOr<PostPage>,
          (String, String, {String? cursor, int? limit})
        > {
  DirectRepliesFamily._()
    : super(
        retry: null,
        name: r'directRepliesProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  DirectRepliesProvider call(
    String did,
    String rkey, {
    String? cursor,
    int? limit,
  }) => DirectRepliesProvider._(
    argument: (did, rkey, cursor: cursor, limit: limit),
    from: this,
  );

  @override
  String toString() => r'directRepliesProvider';
}

@ProviderFor(postThread)
final postThreadProvider = PostThreadFamily._();

final class PostThreadProvider
    extends
        $FunctionalProvider<
          AsyncValue<PostThread>,
          PostThread,
          FutureOr<PostThread>
        >
    with $FutureModifier<PostThread>, $FutureProvider<PostThread> {
  PostThreadProvider._({
    required PostThreadFamily super.from,
    required (String, String) super.argument,
  }) : super(
         retry: null,
         name: r'postThreadProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postThreadHash();

  @override
  String toString() {
    return r'postThreadProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  $FutureProviderElement<PostThread> $createElement($ProviderPointer pointer) =>
      $FutureProviderElement(pointer);

  @override
  FutureOr<PostThread> create(Ref ref) {
    final argument = this.argument as (String, String);
    return postThread(ref, argument.$1, argument.$2);
  }

  @override
  bool operator ==(Object other) {
    return other is PostThreadProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postThreadHash() => r'a55752d54116dd41558bf6c420703eb75db7d5f6';

final class PostThreadFamily extends $Family
    with $FunctionalFamilyOverride<FutureOr<PostThread>, (String, String)> {
  PostThreadFamily._()
    : super(
        retry: null,
        name: r'postThreadProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PostThreadProvider call(String did, String rkey) =>
      PostThreadProvider._(argument: (did, rkey), from: this);

  @override
  String toString() => r'postThreadProvider';
}
