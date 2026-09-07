import 'dart:async';

import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:logging/logging.dart';

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
              required bypassDeduplication,
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
        publish: (video, {required allowBlobRecovery}) async {
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
              required bypassDeduplication,
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
        publish: (_, {required allowBlobRecovery}) async {},
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
            required bypassDeduplication,
            required onProgress,
          }) async {
            uploadCalls++;
            throw StateError('upload must not run after cancellation');
          },
      poll: (_, _) => throw StateError('poll must not run'),
      wait: (_) async {},
      publish: (_, {required allowBlobRecovery}) =>
          throw StateError('publish must not run'),
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
            required bypassDeduplication,
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
      publish: (_, {required allowBlobRecovery}) async => publishCalls++,
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

  test('logs a bounded diagnostic for the failing video operation', () async {
    final records = <LogRecord>[];
    final subscription = Logger(
      'VideoPublication',
    ).onRecord.listen(records.add);
    addTearDown(subscription.cancel);
    final coordinator = VideoPublicationCoordinator(
      checkEligibility: () async => const VideoUploadLimits(canUpload: true),
      authorize: () async => _authorization(),
      upload:
          ({
            required authorizationHeader,
            required cancelToken,
            required bypassDeduplication,
            required onProgress,
          }) async => throw const VideoTransportException(
            VideoTransportFailure.unavailable,
          ),
      poll: (_, _) => throw StateError('poll must not run'),
      wait: (_) async {},
      publish: (_, {required allowBlobRecovery}) =>
          throw StateError('publish must not run'),
      onProgress: (_) {},
    );

    await expectLater(
      coordinator.publish(altText: '', aspectRatio: null),
      throwsA(isA<VideoTransportException>()),
    );

    expect(
      records.map((record) => record.message),
      contains(
        'VideoDiagnosticEvent(operation: upload, outcome: failed, '
        'bytes: unknown, requestId: none)',
      ),
    );
    expect(records.last.level, Level.SEVERE);
  });

  test('missing PDS blob retries once with deduplication bypass', () async {
    final calls = <String>[];
    var authorizationCalls = 0;
    var uploadCalls = 0;
    var publishCalls = 0;
    final coordinator = VideoPublicationCoordinator(
      checkEligibility: () async => const VideoUploadLimits(canUpload: true),
      authorize: () async {
        authorizationCalls++;
        return _authorization();
      },
      upload:
          ({
            required authorizationHeader,
            required cancelToken,
            required bypassDeduplication,
            required onProgress,
          }) async {
            calls.add('upload:$bypassDeduplication');
            uploadCalls++;
            return VideoServiceResult(
              outcome: VideoServiceOutcome.completed,
              jobId: 'job-$uploadCalls',
              blob: VideoServiceBlob(
                cid: 'bafyvideo$uploadCalls',
                mimeType: 'video/mp4',
                size: 8,
              ),
            );
          },
      poll: (_, _) => throw StateError('poll must not run'),
      wait: (_) async {},
      publish: (_, {required allowBlobRecovery}) async {
        publishCalls++;
        calls.add('publish:$publishCalls:$allowBlobRecovery');
        if (publishCalls == 1) {
          throw const ApiServerError(
            'http_502',
            details: ApiFailureDetails(
              appViewError: 'video_blob_missing',
              requestId: 'request-one',
            ),
          );
        }
      },
      onProgress: (_) {},
    );

    await coordinator.publish(altText: '', aspectRatio: null);

    expect(authorizationCalls, 2);
    expect(uploadCalls, 2);
    expect(publishCalls, 2);
    expect(calls, [
      'upload:false',
      'publish:1:true',
      'upload:true',
      'publish:2:false',
    ]);
    expect(coordinator.hasEphemeralState, isFalse);
  });

  test('missing PDS blob recovery runs at most once', () async {
    var uploadCalls = 0;
    var publishCalls = 0;
    final coordinator = VideoPublicationCoordinator(
      checkEligibility: () async => const VideoUploadLimits(canUpload: true),
      authorize: () async => _authorization(),
      upload:
          ({
            required authorizationHeader,
            required cancelToken,
            required bypassDeduplication,
            required onProgress,
          }) async {
            uploadCalls++;
            return VideoServiceResult(
              outcome: VideoServiceOutcome.completed,
              jobId: 'job-$uploadCalls',
              blob: const VideoServiceBlob(
                cid: 'bafyvideo',
                mimeType: 'video/mp4',
                size: 8,
              ),
            );
          },
      poll: (_, _) => throw StateError('poll must not run'),
      wait: (_) async {},
      publish: (_, {required allowBlobRecovery}) async {
        publishCalls++;
        expect(allowBlobRecovery, publishCalls == 1);
        throw const ApiServerError(
          'http_502',
          details: ApiFailureDetails(appViewError: 'video_blob_missing'),
        );
      },
      onProgress: (_) {},
    );

    await expectLater(
      coordinator.publish(altText: '', aspectRatio: null),
      throwsA(
        isA<ApiServerError>().having(
          (error) => error.details.appViewError,
          'appViewError',
          'video_blob_missing',
        ),
      ),
    );
    expect(uploadCalls, 2);
    expect(publishCalls, 2);
  });
}

VideoUploadAuthorization _authorization() => VideoUploadAuthorization.fromMap({
  'token': 'ephemeral-secret',
  'expiresAt': '2030-01-01T00:00:00Z',
});
