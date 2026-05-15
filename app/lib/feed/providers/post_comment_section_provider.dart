import 'dart:async';

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

  void appendCommentPage(model.CommentPage page) {
    final current = state.requireValue;
    state = AsyncData(current.appendCommentPageDeduplicating(page));
  }

  void setRepliesForComment({
    required String commentUri,
    required List<model.ReplyItem> replies,
    String? cursor,
  }) {
    final current = state.requireValue;
    state = AsyncData(
      current.setCommentReplies(
        commentUri: commentUri,
        replies: replies,
        cursor: cursor,
      ),
    );
  }

  void collapseReplies(String commentUri) {
    final current = state.requireValue;
    state = AsyncData(
      current.collapseCommentReplies(commentUri: commentUri),
    );
  }

  void prependCreatedComment(Post post) {
    final current = state.requireValue;
    state = AsyncData(current.prependCreatedComment(post));
  }

  void insertCreatedReply({
    required String parentUri,
    required Post post,
  }) {
    final current = state.requireValue;
    state = AsyncData(
      current.insertCreatedReplyIntoNearestBranch(
        parentUri: parentUri,
        reply: model.ReplyItem(post: post, flattened: false),
      ),
    );
  }
}

@riverpod
class PostCommentPageLoader extends _$PostCommentPageLoader {
  @override
  FutureOr<void> build(
    String did,
    String rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    String? focus,
  }) {}

  Future<void> load() async {
    if (state.isLoading) return;

    final sectionProvider = postCommentSectionProvider(
      did,
      rkey,
      sort: sort,
      focus: focus,
    );
    final current = ref.read(sectionProvider).value;
    if (current == null || current.comments.cursor == null) return;

    state = const AsyncLoading();
    final result = await AsyncValue.guard(() async {
      final page = await ref
          .read(postRepositoryProvider)
          .commentSection(
            did,
            rkey,
            cursor: current.comments.cursor,
            sort: current.sort,
          );
      ref.read(sectionProvider.notifier).appendCommentPage(page.comments);
    });
    if (!ref.mounted) return;
    state = result;
  }
}

@riverpod
class PostCommentRepliesLoader extends _$PostCommentRepliesLoader {
  @override
  FutureOr<void> build(
    String did,
    String rkey, {
    required String commentUri,
    model.CommentSort sort = model.CommentSort.oldest,
    String? focus,
  }) {}

  Future<void> load() async {
    if (state.isLoading) return;

    final sectionProvider = postCommentSectionProvider(
      did,
      rkey,
      sort: sort,
      focus: focus,
    );
    final current = ref.read(sectionProvider).value;
    if (current == null) return;

    final comment = current.comments.items
        .where((item) => item.post.uri == commentUri)
        .firstOrNull;
    if (comment == null) return;
    if (comment.replies.loaded && comment.replies.cursor == null) return;

    state = const AsyncLoading();
    final result = await AsyncValue.guard(() async {
      final page = await ref
          .read(postRepositoryProvider)
          .listDirectReplies(
            comment.post.author.did,
            comment.post.rkey,
            cursor: comment.replies.cursor,
            limit: 10,
          );
      final replies = [
        ...comment.replies.items,
        for (final post in page.items)
          model.ReplyItem(post: post, flattened: false),
      ];
      ref
          .read(sectionProvider.notifier)
          .setRepliesForComment(
            commentUri: commentUri,
            replies: replies,
            cursor: page.cursor,
          );
    });
    if (!ref.mounted) return;
    state = result;
  }
}
