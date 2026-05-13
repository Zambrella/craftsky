import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart' as model;
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_comment_section_provider.g.dart';

@riverpod
class PostCommentSection extends _$PostCommentSection {
  @override
  Future<model.PostCommentSection> build(
    String did,
    String rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    String? focus,
  }) async {
    final repo = ref.watch(postRepositoryProvider);
    return repo.commentSection(did, rkey, sort: sort, focus: focus);
  }

  Future<void> loadMoreComments() async {
    final current = state.value;
    if (current == null || current.comments.cursor == null || state.isLoading) {
      return;
    }

    // ignore: invalid_use_of_internal_member
    state = const AsyncLoading<model.PostCommentSection>().copyWithPrevious(
      state,
    );

    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(postRepositoryProvider)
          .commentSection(
            did,
            rkey,
            cursor: current.comments.cursor,
            sort: current.sort,
          );
      return model.appendCommentPageDeduplicating(current, page.comments);
    });

    if (!ref.mounted) return;
    // ignore: invalid_use_of_internal_member
    state = next.copyWithPrevious(state);
  }

  Future<void> loadMoreReplies(String commentUri) async {
    final current = state.value;
    if (current == null || state.isLoading) return;

    final comment = current.comments.items
        .where((item) => item.post.uri == commentUri)
        .firstOrNull;
    if (comment == null) return;
    if (comment.replies.loaded && comment.replies.cursor == null) return;

    // ignore: invalid_use_of_internal_member
    state = const AsyncLoading<model.PostCommentSection>().copyWithPrevious(
      state,
    );

    final next = await AsyncValue.guard(() async {
      final page = await ref
          .read(postRepositoryProvider)
          .listDirectReplies(
            comment.post.author.did,
            comment.post.rkey,
            cursor: comment.replies.cursor,
            limit: 10,
          );
      final newReplies = [
        ...comment.replies.items,
        for (final post in page.items)
          model.ReplyItem(post: post, flattened: false),
      ];
      return _replaceComment(
        current,
        commentUri,
        model.CommentItem(
          post: comment.post,
          placement: comment.placement,
          replies: model.ReplyPage(
            loaded: true,
            items: newReplies,
            cursor: page.cursor,
          ),
        ),
      );
    });

    if (!ref.mounted) return;
    // ignore: invalid_use_of_internal_member
    state = next.copyWithPrevious(state);
  }

  void collapseReplies(String commentUri) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      model.collapseCommentReplies(current, commentUri: commentUri),
    );
  }

  void prependCreatedComment(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(model.prependCreatedComment(current, post));
  }

  void insertCreatedReply({
    required String parentUri,
    required Post post,
  }) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(
      model.insertCreatedReplyIntoNearestBranch(
        current,
        parentUri: parentUri,
        reply: model.ReplyItem(post: post, flattened: false),
      ),
    );
  }

  model.PostCommentSection _replaceComment(
    model.PostCommentSection section,
    String commentUri,
    model.CommentItem replacement,
  ) => model.PostCommentSection(
    post: section.post,
    sort: section.sort,
    focus: section.focus,
    comments: model.CommentPage(
      cursor: section.comments.cursor,
      items: [
        for (final item in section.comments.items)
          if (item.post.uri == commentUri) replacement else item,
      ],
    ),
  );
}
