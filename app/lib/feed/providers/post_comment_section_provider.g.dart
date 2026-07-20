// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'post_comment_section_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning

@ProviderFor(PostCommentSection)
final postCommentSectionProvider = PostCommentSectionFamily._();

final class PostCommentSectionProvider
    extends
        $AsyncNotifierProvider<PostCommentSection, model.PostCommentSection> {
  PostCommentSectionProvider._({
    required PostCommentSectionFamily super.from,
    required (Did, RecordKey, {model.CommentSort sort, AtUri? focus})
    super.argument,
  }) : super(
         retry: null,
         name: r'postCommentSectionProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postCommentSectionHash();

  @override
  String toString() {
    return r'postCommentSectionProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  PostCommentSection create() => PostCommentSection();

  @override
  bool operator ==(Object other) {
    return other is PostCommentSectionProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postCommentSectionHash() =>
    r'5577893dc03edc8751f70d02f51701037c6df82c';

final class PostCommentSectionFamily extends $Family
    with
        $ClassFamilyOverride<
          PostCommentSection,
          AsyncValue<model.PostCommentSection>,
          model.PostCommentSection,
          FutureOr<model.PostCommentSection>,
          (Did, RecordKey, {model.CommentSort sort, AtUri? focus})
        > {
  PostCommentSectionFamily._()
    : super(
        retry: null,
        name: r'postCommentSectionProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PostCommentSectionProvider call(
    Did did,
    RecordKey rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  }) => PostCommentSectionProvider._(
    argument: (did, rkey, sort: sort, focus: focus),
    from: this,
  );

  @override
  String toString() => r'postCommentSectionProvider';
}

abstract class _$PostCommentSection
    extends $AsyncNotifier<model.PostCommentSection> {
  late final _$args =
      ref.$arg as (Did, RecordKey, {model.CommentSort sort, AtUri? focus});
  Did get did => _$args.$1;
  RecordKey get rkey => _$args.$2;
  model.CommentSort get sort => _$args.sort;
  AtUri? get focus => _$args.focus;

  FutureOr<model.PostCommentSection> build(
    Did did,
    RecordKey rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  });
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<model.PostCommentSection>,
              model.PostCommentSection
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<model.PostCommentSection>,
                model.PostCommentSection
              >,
              AsyncValue<model.PostCommentSection>,
              Object?,
              Object?
            >;
    element.handleCreate(
      ref,
      () => build(_$args.$1, _$args.$2, sort: _$args.sort, focus: _$args.focus),
    );
  }
}

@ProviderFor(PostCommentPageLoader)
final postCommentPageLoaderProvider = PostCommentPageLoaderFamily._();

final class PostCommentPageLoaderProvider
    extends $AsyncNotifierProvider<PostCommentPageLoader, void> {
  PostCommentPageLoaderProvider._({
    required PostCommentPageLoaderFamily super.from,
    required (Did, RecordKey, {model.CommentSort sort, AtUri? focus})
    super.argument,
  }) : super(
         retry: null,
         name: r'postCommentPageLoaderProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postCommentPageLoaderHash();

  @override
  String toString() {
    return r'postCommentPageLoaderProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  PostCommentPageLoader create() => PostCommentPageLoader();

  @override
  bool operator ==(Object other) {
    return other is PostCommentPageLoaderProvider && other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postCommentPageLoaderHash() =>
    r'ae9d1aede145dbd8adf019236e2287553cdd86ce';

final class PostCommentPageLoaderFamily extends $Family
    with
        $ClassFamilyOverride<
          PostCommentPageLoader,
          AsyncValue<void>,
          void,
          FutureOr<void>,
          (Did, RecordKey, {model.CommentSort sort, AtUri? focus})
        > {
  PostCommentPageLoaderFamily._()
    : super(
        retry: null,
        name: r'postCommentPageLoaderProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PostCommentPageLoaderProvider call(
    Did did,
    RecordKey rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  }) => PostCommentPageLoaderProvider._(
    argument: (did, rkey, sort: sort, focus: focus),
    from: this,
  );

  @override
  String toString() => r'postCommentPageLoaderProvider';
}

abstract class _$PostCommentPageLoader extends $AsyncNotifier<void> {
  late final _$args =
      ref.$arg as (Did, RecordKey, {model.CommentSort sort, AtUri? focus});
  Did get did => _$args.$1;
  RecordKey get rkey => _$args.$2;
  model.CommentSort get sort => _$args.sort;
  AtUri? get focus => _$args.focus;

  FutureOr<void> build(
    Did did,
    RecordKey rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  });
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<void>, void>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<void>, void>,
              AsyncValue<void>,
              Object?,
              Object?
            >;
    element.handleCreate(
      ref,
      () => build(_$args.$1, _$args.$2, sort: _$args.sort, focus: _$args.focus),
    );
  }
}

@ProviderFor(PostCommentRepliesLoader)
final postCommentRepliesLoaderProvider = PostCommentRepliesLoaderFamily._();

final class PostCommentRepliesLoaderProvider
    extends $AsyncNotifierProvider<PostCommentRepliesLoader, void> {
  PostCommentRepliesLoaderProvider._({
    required PostCommentRepliesLoaderFamily super.from,
    required (
      Did,
      RecordKey, {
      AtUri commentUri,
      model.CommentSort sort,
      AtUri? focus,
    })
    super.argument,
  }) : super(
         retry: null,
         name: r'postCommentRepliesLoaderProvider',
         isAutoDispose: true,
         dependencies: null,
         $allTransitiveDependencies: null,
       );

  @override
  String debugGetCreateSourceHash() => _$postCommentRepliesLoaderHash();

  @override
  String toString() {
    return r'postCommentRepliesLoaderProvider'
        ''
        '$argument';
  }

  @$internal
  @override
  PostCommentRepliesLoader create() => PostCommentRepliesLoader();

  @override
  bool operator ==(Object other) {
    return other is PostCommentRepliesLoaderProvider &&
        other.argument == argument;
  }

  @override
  int get hashCode {
    return argument.hashCode;
  }
}

String _$postCommentRepliesLoaderHash() =>
    r'd1ad1993cdd4abf5b90021775e5be31cdd572260';

final class PostCommentRepliesLoaderFamily extends $Family
    with
        $ClassFamilyOverride<
          PostCommentRepliesLoader,
          AsyncValue<void>,
          void,
          FutureOr<void>,
          (
            Did,
            RecordKey, {
            AtUri commentUri,
            model.CommentSort sort,
            AtUri? focus,
          })
        > {
  PostCommentRepliesLoaderFamily._()
    : super(
        retry: null,
        name: r'postCommentRepliesLoaderProvider',
        dependencies: null,
        $allTransitiveDependencies: null,
        isAutoDispose: true,
      );

  PostCommentRepliesLoaderProvider call(
    Did did,
    RecordKey rkey, {
    required AtUri commentUri,
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  }) => PostCommentRepliesLoaderProvider._(
    argument: (did, rkey, commentUri: commentUri, sort: sort, focus: focus),
    from: this,
  );

  @override
  String toString() => r'postCommentRepliesLoaderProvider';
}

abstract class _$PostCommentRepliesLoader extends $AsyncNotifier<void> {
  late final _$args =
      ref.$arg
          as (
            Did,
            RecordKey, {
            AtUri commentUri,
            model.CommentSort sort,
            AtUri? focus,
          });
  Did get did => _$args.$1;
  RecordKey get rkey => _$args.$2;
  AtUri get commentUri => _$args.commentUri;
  model.CommentSort get sort => _$args.sort;
  AtUri? get focus => _$args.focus;

  FutureOr<void> build(
    Did did,
    RecordKey rkey, {
    required AtUri commentUri,
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  });
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<void>, void>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<void>, void>,
              AsyncValue<void>,
              Object?,
              Object?
            >;
    element.handleCreate(
      ref,
      () => build(
        _$args.$1,
        _$args.$2,
        commentUri: _$args.commentUri,
        sort: _$args.sort,
        focus: _$args.focus,
      ),
    );
  }
}
