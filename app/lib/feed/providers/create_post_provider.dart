import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/create_post_external.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/providers/post_comment_section_provider.dart';
import 'package:craftsky_app/feed/providers/post_interaction_lists_provider.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/feed/providers/post_record_overlay.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/feed/providers/user_comments_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/projects/models/project.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/errors/app_error_mapper.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'create_post_provider.g.dart';

final _diagnosticLog = Logger('CreatePostMutation');

/// Standalone create-a-post mutation notifier. Idle until [create] runs,
/// then transitions `AsyncLoading` -> `AsyncData(post)` on success, or
/// `AsyncError` on failure.
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
    final reporter = GuardedErrorReporter(ref.read(errorReporterProvider));
    final operationOwnership = ownership ?? captureActiveAccountOperation(ref);
    final accountOperation = ActiveAccountOperation.fromLease(
      ref,
      operationOwnership,
    );
    if (!accountOperation.isCurrent) return null;
    final frozenLangs = List<String>.unmodifiable(langs);
    final frozenImages = images == null
        ? null
        : List<CreatePostImage>.unmodifiable(images);
    final frozenFacets = facets == null
        ? null
        : (jsonDecode(jsonEncode(facets)) as List)
              .map((item) => Map<String, dynamic>.from(item as Map))
              .toList(growable: false);
    const endpoint = '/v1/posts';
    final immutableBody = jsonEncode({
      'text': text,
      'langs': frozenLangs,
      'sponsored': sponsored,
      'reply': reply?.toMap(),
      'quote': quote?.toMap(),
      'project': project?.toCreateMap(),
      'images': frozenImages?.map((image) => image.toMap()).toList(),
      'external': external?.toMap(),
      'video': video?.toMap(),
      'facets': frozenFacets,
    });
    final controller = ref.read(pdsRecordOperationControllerProvider);
    final token = controller.beginOrRetry(
      scope: postCreateOperationScope(operationOwnership),
      endpoint: endpoint,
      immutableBody: immutableBody,
      newOperationKey: newPdsMutationOperationKey,
    );
    final operationKey = token.operationKey;
    final retry = PdsMutationRetry(
      ref: ref,
      controller: controller,
      token: token,
      isCurrent: () => accountOperation.isCurrent,
      requireCurrentToken: false,
    );

    state = const AsyncLoading<Post?>();
    late final repository = ref.read(postRepositoryProvider);
    while (true) {
      late Post created;
      try {
        final repo = repository;
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
        created = await repo.create(
          operationKey: operationKey,
          text: text,
          langs: frozenLangs,
          sponsored: sponsored,
          reply: reply,
          quote: quote,
          project: project,
          images: frozenImages,
          external: external,
          video: video,
          facets: frozenFacets,
        );
      } on PdsMutationAmbiguousException catch (error, stackTrace) {
        final decision = await retry.wait(error);
        if (!accountOperation.isCurrent) return null;
        switch (decision) {
          case PdsMutationRetryDecision.obsolete:
            return null;
          case PdsMutationRetryDecision.exhausted:
            state = AsyncError<Post?>(
              PdsMutationUnresolvedException(
                diagnosticCause: error,
                diagnosticStack: stackTrace,
              ),
              stackTrace,
            );
            return null;
          case PdsMutationRetryDecision.retry:
            continue;
        }
      } on Object catch (error, stackTrace) {
        final current = accountOperation.isCurrent;
        final mapped = AppErrorMapper.map(
          error,
          source: 'post_mutation',
          fallbackClassification: 'post.create.failed',
        );
        final diagnostic = ReportContext(
          feature: 'CreatePostMutation',
          operation: 'post.create',
          classification: mapped.sentryClassification,
          safeDiagnostics: {
            ...mapped.safeDiagnostics,
            'failureStage': 'pds_write',
            'attempt': retry.attempt,
          },
          workflow: operationOwnership == null
              ? null
              : PublicRecordContext(
                  actorDid: operationOwnership.session.account.did.value,
                ),
        );
        final message = DiagnosticMessage(
          'post mutation failed',
          context: diagnostic,
        );
        if (current) {
          _diagnosticLog.warning(message, error, stackTrace);
        } else {
          _diagnosticLog.severe(message, error, stackTrace);
        }
        if (!current) {
          // No provider state is published for this consumed late failure.
          if (mapped.reportable) {
            unawaited(
              reporter.captureException(
                error,
                stackTrace: stackTrace,
                context: diagnostic,
              ),
            );
          }
          return null;
        }
        controller.markFailed(token);
        if (allowVideoBlobRecovery &&
            error is ApiException &&
            error.details.appViewError == 'video_blob_missing') {
          Error.throwWithStackTrace(error, stackTrace);
        }
        withFailureDiagnosticContext(
          error,
          diagnostic,
          () => state = AsyncError<Post?>(error, stackTrace),
        );
        return null;
      }

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
        post = post.copyWith(langs: frozenLangs);
      }
      if (!accountOperation.isCurrent) {
        return null;
      }

      final accepted = reply == null
          ? controller.markAccepted(
              token,
              overlayScope: postRecordMutationScope(ref, post),
              optimisticValue: post,
              agrees: (value) =>
                  value is PdsRecordProjection &&
                  PdsAppendReconciliation(
                    selectedUri: post.uri.toString(),
                    acceptedCid: post.cid.toString(),
                    controlledContent: postRecordProjection(post).content,
                  ).agrees(value),
              refresh: () => _invalidatePostMutationReads(ref, post),
            )
          : controller.markAcceptedWithoutOverlay(token);
      if (!accepted) {
        return null;
      }
      state = AsyncData<Post?>(post);
      return post;
    }
  }

  /// Resets the notifier to its idle state. Call after consuming a
  /// success/failure transition so a re-entry doesn't see prior result.
  void reset() => state = const AsyncData(null);
}

void _invalidatePostMutationReads(Ref ref, Post post) {
  if (!ref.mounted) return;
  ref
    ..invalidate(postProvider(post.author.did, post.rkey))
    ..invalidate(postCommentSectionProvider)
    ..invalidate(postQuotesProvider)
    ..invalidate(timelineProvider)
    ..invalidate(userPostsProvider)
    ..invalidate(userCommentsProvider)
    ..invalidate(userProjectsProvider);
}
