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
    required (String, String, {model.CommentSort sort, String? focus})
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
    r'a3fa7bfd2a23b2d38f044a9068c64251fbce7ad6';

final class PostCommentSectionFamily extends $Family
    with
        $ClassFamilyOverride<
          PostCommentSection,
          AsyncValue<model.PostCommentSection>,
          model.PostCommentSection,
          FutureOr<model.PostCommentSection>,
          (String, String, {model.CommentSort sort, String? focus})
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
    String did,
    String rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    String? focus,
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
      ref.$arg as (String, String, {model.CommentSort sort, String? focus});
  String get did => _$args.$1;
  String get rkey => _$args.$2;
  model.CommentSort get sort => _$args.sort;
  String? get focus => _$args.focus;

  FutureOr<model.PostCommentSection> build(
    String did,
    String rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    String? focus,
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
