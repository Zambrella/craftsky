import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/create_post_external.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'create_post_provider.g.dart';

/// Standalone create-a-post mutation notifier. Idle until [create] runs,
/// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
/// `AsyncError` on failure.
///
/// On success, prepends the synthetic post into any live
/// `userPostsProvider` family entries keyed by either the author's
/// handle or DID — sidestepping the AppView's read-after-write window
/// (where a refetch could miss the just-created row until the firehose
/// indexer catches up). `ref.exists` guards against accidentally
/// instantiating a non-live family entry, which would race a fresh
/// `build` against our prepend.
///
/// Callers should bind via `ref.listen(createPostProvider, ...)` and
/// call [reset] after consuming a transition so a re-entry to the
/// compose page doesn't see the previous result.
@riverpod
class CreatePost extends _$CreatePost {
  @override
  FutureOr<Post?> build() => null;

  Future<Post?> create({
    required String text,
    required List<String> langs,
    required bool sponsored,
    PostReply? reply,
    PostRef? quote,
    Project? project,
    List<CreatePostImage>? images,
    CreatePostExternal? external,
    CreatePostVideo? video,
    List<Map<String, dynamic>>? facets,
    ActiveAccountLease? ownership,
    bool allowVideoBlobRecovery = false,
  }) async {
    final operationOwnership = ownership ?? captureActiveAccountOperation(ref);
    if (!isActiveAccountOperationCurrent(ref, operationOwnership)) return null;
    state = const AsyncLoading();
    final result = await AsyncValue.guard(() async {
      final repo = ref.read(postRepositoryProvider);
      assert(
        project == null || reply == null,
        'Project posts cannot be replies',
      );
      assert(
        quote == null || reply == null,
        'Quote posts cannot be replies',
      );
      assert(
        quote == null || project == null,
        'Project posts cannot be quote posts',
      );
      final created = await repo.create(
        text: text,
        langs: langs,
        sponsored: sponsored,
        reply: reply,
        quote: quote,
        project: project,
        images: images,
        external: external,
        video: video,
        facets: facets,
      );
      var post = created;
      if (reply != null && post.reply == null) {
        post = post.copyWith(reply: reply);
      }
      if (quote != null && post.quote == null) {
        post = post.copyWith(quote: quote);
      }
      if (project != null && post.project == null) {
        post = post.copyWith(project: project);
      }
      if (post.langs.isEmpty) {
        post = post.copyWith(langs: langs);
      }
      if (!isActiveAccountOperationCurrent(ref, operationOwnership)) {
        return null;
      }

      if (reply == null) {
        prependLiveTimelineCache(ref, post);
        if (post.project == null) {
          final provider = userPostsProvider(post.author.did);
          if (ref.exists(provider)) {
            ref.read(provider.notifier).prepend(post);
          }
        } else {
          prependLiveUserProjectCaches(ref, post);
        }
      } else {
        updateLiveUserCommentCaches(ref, post);
      }

      return post;
    });
    if (!isActiveAccountOperationCurrent(ref, operationOwnership)) return null;
    final error = result.error;
    if (allowVideoBlobRecovery &&
        error is ApiException &&
        error.details.appViewError == 'video_blob_missing') {
      Error.throwWithStackTrace(error, result.stackTrace ?? StackTrace.current);
    }
    state = result;
    return result.value;
  }

  /// Resets the notifier to its idle state. Call after consuming a
  /// success/failure transition so a re-entry doesn't see prior result.
  void reset() => state = const AsyncData(null);
}
