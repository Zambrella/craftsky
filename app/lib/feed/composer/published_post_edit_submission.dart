import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/composer/composer_submission_coordinator.dart';
import 'package:craftsky_app/feed/composer/composer_video_publication.dart';
import 'package:craftsky_app/feed/composer/post_edit_origin.dart';
import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/post_edit.dart';
import 'package:craftsky_app/feed/providers/composer_video_controller.dart';
import 'package:craftsky_app/feed/providers/edit_post_provider.dart';
import 'package:craftsky_app/feed/providers/post_api_client_provider.dart';
import 'package:craftsky_app/feed/providers/video_service_client_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Shared published-edit lifecycle. Composers prepare their own field changes;
/// this boundary freezes requests and never rebuilds an uncertain retry.
Future<void> submitPublishedPostEdit({
  required WidgetRef ref,
  required PostEditOrigin edit,
  required Map<String, Object?> changes,
  required LocalVideoSelection? selectedVideo,
  required Future<CreatePostVideo> Function(LocalVideoSelection)? prepareVideo,
  required void Function(Future<void> Function()) onRetry,
  required void Function(PostEditResult) onAccepted,
  required void Function(VideoPublicationProgress) onProgress,
  required void Function(VideoPublicationCoordinator, {required bool active})
  onVideo,
}) async {
  final operation = ActiveAccountOperation.fromWidget(ref, edit.ownership)
    ..requireCurrent();
  Future<void> submit(
    CreatePostVideo? video, {
    bool allowBlobRecovery = false,
  }) async {
    final fields = <String, Object?>{
      ...changes,
      if (video != null)
        'embed': {
          r'$type': 'app.bsky.embed.video',
          'video': video.blob.toMap(),
          'jobId': video.jobId,
          'alt': video.alt,
          if (video.aspectRatio != null)
            'aspectRatio': video.aspectRatio!.toMap(),
        },
      if (video != null && (edit.seed.post.images?.isNotEmpty ?? false))
        'images': null,
    };
    final frozen = PostEditChanges(fields);
    Future<void> send({bool recoverBlob = false}) async {
      operation.requireCurrent();
      final result = await ref
          .read(editPostProvider.notifier)
          .save(
            seed: edit.seed,
            changes: frozen,
            ownership: edit.ownership,
            allowVideoBlobRecovery: recoverBlob,
          );
      operation.requireCurrent();
      if (result == null) throw const ComposerSubmissionFailure();
      onAccepted(result);
    }

    onRetry(send);
    await send(recoverBlob: allowBlobRecovery);
  }

  if (selectedVideo == null) {
    await submit(null);
  } else if (prepareVideo != null) {
    final video = await prepareVideo(selectedVideo);
    operation.requireCurrent();
    await submit(video);
  } else {
    final coordinator = composerVideoPublication(
      owner: edit.ownership,
      selection: selectedVideo,
      api: ref.read(postApiClientProvider),
      service: ref.read(videoServiceClientProvider),
      publish: submit,
      onProgress: onProgress,
    );
    onVideo(coordinator, active: true);
    try {
      await coordinator.publish(
        altText: selectedVideo.altText,
        aspectRatio: (selectedVideo.width, selectedVideo.height),
      );
    } finally {
      onVideo(coordinator, active: false);
    }
  }
}
