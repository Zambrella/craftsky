import 'dart:async';

import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'AT-001 authorizes, uploads, polls, then publishes verified proof',
    () async {
      final calls = <String>[];
      final stages = <VideoPublicationProgress>[];
      final waits = <Duration>[];
      CreatePostVideo? proof;
      var polls = 0;
      final coordinator = VideoPublicationCoordinator(
        checkEligibility: () async {
          calls.add('limits');
          return const VideoUploadLimits(canUpload: true);
        },
        authorize: () async {
          calls.add('authorize');
          return VideoUploadAuthorization.fromMap({
            'token': 'ephemeral-secret',
            'expiresAt': '2030-01-01T00:00:00Z',
          });
        },
        upload:
            ({
              required authorizationHeader,
              required cancelToken,
              required onProgress,
            }) async {
              expect(authorizationHeader, 'Bearer ephemeral-secret');
              calls.add('upload');
              onProgress(4, 8);
              return const VideoServiceResult(
                outcome: VideoServiceOutcome.processing,
                jobId: 'job-one',
              );
            },
        poll: (jobId, cancelToken) async {
          calls.add('poll');
          polls++;
          return polls == 1
              ? const VideoServiceResult(
                  outcome: VideoServiceOutcome.processing,
                  jobId: 'job-one',
                  progress: 75,
                  retryAfter: Duration(seconds: 12),
                )
              : const VideoServiceResult(
                  outcome: VideoServiceOutcome.completed,
                  jobId: 'job-one',
                  blob: VideoServiceBlob(
                    cid: 'bafyvideo',
                    mimeType: 'video/mp4',
                    size: 8,
                  ),
                );
        },
        wait: (duration) async => waits.add(duration),
        publish: (video) async {
          calls.add('publish');
          proof = video;
        },
        onProgress: stages.add,
      );

      await coordinator.publish(altText: 'A loom', aspectRatio: (16, 9));

      expect(calls, [
        'limits',
        'authorize',
        'upload',
        'poll',
        'poll',
        'publish',
      ]);
      expect(proof?.jobId, 'job-one');
      expect(proof?.blob.cid, 'bafyvideo');
      expect(waits, const [Duration(seconds: 1), Duration(seconds: 12)]);
      expect(
        stages.map((item) => item.stage),
        containsAllInOrder([
          VideoPublicationStage.validating,
          VideoPublicationStage.uploading,
          VideoPublicationStage.processing,
          VideoPublicationStage.publishing,
          VideoPublicationStage.complete,
        ]),
      );
      expect(coordinator.hasEphemeralState, isFalse);
      expect(coordinator.toString(), isNot(contains('ephemeral-secret')));
    },
  );

  test(
    'AT-005 cancellation stops processing and clears remote state',
    () async {
      final waitStarted = Completer<void>();
      var polls = 0;
      final stages = <VideoPublicationStage>[];
      final coordinator = VideoPublicationCoordinator(
        checkEligibility: () async => const VideoUploadLimits(canUpload: true),
        authorize: () async => VideoUploadAuthorization.fromMap({
          'token': 'ephemeral-secret',
          'expiresAt': '2030-01-01T00:00:00Z',
        }),
        upload:
            ({
              required authorizationHeader,
              required cancelToken,
              required onProgress,
            }) async => const VideoServiceResult(
              outcome: VideoServiceOutcome.processing,
              jobId: 'job-one',
            ),
        poll: (jobId, cancelToken) async {
          polls++;
          throw StateError('poll must not run after cancellation');
        },
        wait: (_) => waitStarted.future,
        publish: (_) async {},
        onProgress: (progress) => stages.add(progress.stage),
      );

      final publication = coordinator.publish(altText: '', aspectRatio: null);
      await Future<void>.delayed(Duration.zero);
      coordinator.cancel();

      await expectLater(publication, throwsA(isA<DioException>()));
      expect(polls, 0);
      expect(stages.last, VideoPublicationStage.canceled);
      expect(coordinator.hasEphemeralState, isFalse);
    },
  );

  test('cancellation after limits prevents authorization', () async {
    final limits = Completer<VideoUploadLimits>();
    var authorizationCalls = 0;
    var uploadCalls = 0;
    final coordinator = VideoPublicationCoordinator(
      checkEligibility: () => limits.future,
      authorize: () async {
        authorizationCalls++;
        return _authorization();
      },
      upload:
          ({
            required authorizationHeader,
            required cancelToken,
            required onProgress,
          }) async {
            uploadCalls++;
            throw StateError('upload must not run after cancellation');
          },
      poll: (_, _) => throw StateError('poll must not run'),
      wait: (_) async {},
      publish: (_) => throw StateError('publish must not run'),
      onProgress: (_) {},
    );

    final publication = coordinator.publish(altText: '', aspectRatio: null);
    await Future<void>.delayed(Duration.zero);
    coordinator.cancel();
    limits.complete(const VideoUploadLimits(canUpload: true));

    await expectLater(publication, throwsA(isA<DioException>()));
    expect(authorizationCalls, 0);
    expect(uploadCalls, 0);
    expect(coordinator.hasEphemeralState, isFalse);
  });

  test('cancellation after authorization clears token before upload', () async {
    final authorization = Completer<VideoUploadAuthorization>();
    var uploadCalls = 0;
    var pollCalls = 0;
    var publishCalls = 0;
    final coordinator = VideoPublicationCoordinator(
      checkEligibility: () async => const VideoUploadLimits(canUpload: true),
      authorize: () => authorization.future,
      upload:
          ({
            required authorizationHeader,
            required cancelToken,
            required onProgress,
          }) async {
            uploadCalls++;
            throw StateError('upload must not run after cancellation');
          },
      poll: (_, _) async {
        pollCalls++;
        throw StateError('poll must not run');
      },
      wait: (_) async {},
      publish: (_) async => publishCalls++,
      onProgress: (_) {},
    );

    final publication = coordinator.publish(altText: '', aspectRatio: null);
    await Future<void>.delayed(Duration.zero);
    coordinator.cancel();
    authorization.complete(_authorization());

    await expectLater(publication, throwsA(isA<DioException>()));
    expect(uploadCalls, 0);
    expect(pollCalls, 0);
    expect(publishCalls, 0);
    expect(coordinator.hasEphemeralState, isFalse);
  });
}

VideoUploadAuthorization _authorization() => VideoUploadAuthorization.fromMap({
  'token': 'ephemeral-secret',
  'expiresAt': '2030-01-01T00:00:00Z',
});
