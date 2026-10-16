import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/scheduled_posts/data/scheduled_post_repository.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/services/scheduled_composer_media.dart';

typedef ScheduledComposerPayload = Future<Map<String, dynamic>> Function(
  ScheduledPostRepository repository,
  List<Map<String, dynamic>> media,
);

final class ScheduledComposerProgress {
  const new({required this.staged, required this.total, required this.saving});

  final int staged;
  final int total;
  final bool saving;
}

/// Stages media, writes the schedule and refreshes its account-scoped list
/// under one captured operation. Payloads and presentation stay with editors.
final class ScheduledComposerSubmission {
  const new({
    required this.operation,
    required this.repository,
    required this.refresh,
    required this.materializeMedia,
  });

  final ActiveAccountOperation operation;
  final Future<ScheduledPostRepository> Function() repository;
  final Future<void> Function() refresh;
  final ScheduledComposerMediaMaterializer materializeMedia;

  Future<void> save({
    required String operationId,
    required DateTime scheduledAt,
    required ScheduledPostDetail? existing,
    required List<ComposerImageDraft> images,
    required ScheduledComposerPayload payload,
    required void Function(ScheduledComposerProgress) onProgress,
  }) => _submit(
    images: images,
    payload: payload,
    onProgress: onProgress,
    write: (repository, content) async {
      if (existing == null) {
        await repository.create(
          operationId: operationId,
          scheduledAt: scheduledAt.toUtc(),
          payload: content,
        );
      } else {
        await repository.update(
          id: existing.id,
          scheduledAt: scheduledAt.toUtc(),
          payload: content,
        );
      }
    },
  );

  Future<void> publishNow({
    required ScheduledPostDetail existing,
    required List<ComposerImageDraft> images,
    required ScheduledComposerPayload payload,
    required void Function(ScheduledComposerProgress) onProgress,
  }) => _submit(
    images: images,
    payload: payload,
    onProgress: onProgress,
    write: (repository, content) => repository.publishNow(
      id: existing.id,
      payload: {...existing.payload, ...content},
    ),
  );

  Future<void> _submit({
    required List<ComposerImageDraft> images,
    required ScheduledComposerPayload payload,
    required void Function(ScheduledComposerProgress) onProgress,
    required Future<void> Function(
      ScheduledPostRepository repository,
      Map<String, dynamic> payload,
    )
    write,
  }) async {
    var staged = 0;
    final total = images.where((image) => image.phase is ImageUploaded).length;
    void progress({required bool saving}) {
      operation.publish(
        () => onProgress(
          ScheduledComposerProgress(
            staged: staged,
            total: total,
            saving: saving,
          ),
        ),
      );
    }

    progress(saving: total == 0);
    final target = await operation.step(repository);
    final media = await operation.step(
      () => materializeMedia(
        images,
        stageMedia: target.stageMedia,
        onStaged: (_) {
          staged++;
          progress(saving: total == 0);
        },
      ),
    );
    final content = await operation.step(() => payload(target, media));
    progress(saving: true);
    await operation.step(() => write(target, content));
    await operation.step(refresh);
  }

  Future<void> delete(String id) async {
    final target = await operation.step(repository);
    await operation.step(() => target.delete(id));
    await operation.step(refresh);
  }
}
