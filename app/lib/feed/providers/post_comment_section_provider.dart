import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_comment_section.dart' as model;
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'post_comment_section_provider.g.dart';

@riverpod
class PostCommentSection extends _$PostCommentSection {
  static String formatLogValue(Object? value) => value.toString();

  @override
  Future<model.PostCommentSection> build(
    Did did,
    RecordKey rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  }) async {
    final repo = ref.watch(postRepositoryProvider);
    return repo.commentSection(did, rkey, sort: sort, focus: focus);
  }

  void appendCommentPage(model.CommentPage page) {
    final current = state.requireValue;
    state = AsyncData(current.appendCommentPageDeduplicating(page));
  }

  void setRepliesForComment({
    required AtUri commentUri,
    required List<model.ReplyItem> replies,
    String? cursor,
    bool incrementRootReplyCount = false,
  }) {
    final current = state.requireValue;
    state = AsyncData(
      current.setCommentReplies(
        commentUri: commentUri,
        replies: replies,
        cursor: cursor,
        incrementRootReplyCount: incrementRootReplyCount,
      ),
    );
  }

  void collapseReplies(AtUri commentUri) {
    final current = state.requireValue;
    state = AsyncData(current.collapseCommentReplies(commentUri: commentUri));
  }

  void prependCreatedComment(Post post) {
    final current = state.requireValue;
    state = AsyncData(current.prependCreatedComment(post));
  }

  void replacePost(Post post) {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(current.replacePost(post));
  }

  void insertCreatedReply({required AtUri parentUri, required Post post}) {
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
    Did did,
    RecordKey rkey, {
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  }) {}

  Future<void> load() async {
    if (state.isLoading) return;

    final current = ref
        .read(postCommentSectionProvider(did, rkey, sort: sort, focus: focus))
        .value;
    if (current == null || current.comments.cursor == null) return;
    final ownership = captureActiveAccountOperation(ref);

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
      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      ref
          .read(
            postCommentSectionProvider(
              did,
              rkey,
              sort: sort,
              focus: focus,
            ).notifier,
          )
          .appendCommentPage(page.comments);
    });
    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = result;
  }
}

@riverpod
class PostCommentRepliesLoader extends _$PostCommentRepliesLoader {
  @override
  FutureOr<void> build(
    Did did,
    RecordKey rkey, {
    required AtUri commentUri,
    model.CommentSort sort = model.CommentSort.oldest,
    AtUri? focus,
  }) {}

  Future<void> load() async {
    await _load(revealParent: false);
  }

  Future<void> revealMutedBranch() async {
    await _load(revealParent: true);
  }

  Future<void> _load({required bool revealParent}) async {
    if (state.isLoading) return;

    final current = ref
        .read(postCommentSectionProvider(did, rkey, sort: sort, focus: focus))
        .value;
    if (current == null) return;

    final comment = current.comments.items
        .where((item) => item.post.uri == commentUri)
        .firstOrNull;
    if (comment == null) return;
    if (comment.replies.loaded && comment.replies.cursor == null) return;
    final ownership = captureActiveAccountOperation(ref);

    state = const AsyncLoading();
    final result = await AsyncValue.guard(() async {
      final parts = parseCraftskyPostUri(comment.post.uri);
      final authorDid = parts?.did ?? comment.post.author.did;
      final postRkey = parts?.rkey ?? comment.post.rkey;
      final revealed = revealParent
          ? await ref.read(postRepositoryProvider).fetch(authorDid, postRkey)
          : null;
      final page = await ref
          .read(postRepositoryProvider)
          .listCommentBranchReplies(
            authorDid,
            postRkey,
            cursor: comment.replies.cursor,
            limit: 10,
          );
      final replies = [...comment.replies.items, ...page.items];
      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      if (revealed != null) {
        ref
            .read(
              postCommentSectionProvider(
                did,
                rkey,
                sort: sort,
                focus: focus,
              ).notifier,
            )
            .replacePost(revealed);
      }
      ref
          .read(
            postCommentSectionProvider(
              did,
              rkey,
              sort: sort,
              focus: focus,
            ).notifier,
          )
          .setRepliesForComment(
            commentUri: commentUri,
            replies: replies,
            cursor: page.cursor,
          );
    });
    if (!isActiveAccountOperationCurrent(ref, ownership)) return;
    state = result;
  }
}
