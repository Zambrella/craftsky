import 'dart:async';

import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/create_post_video.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_emitter.dart';
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
    const tokenCanary = 'PRIVATE_VIDEO_SERVICE_TOKEN';
    const altTextCanary = 'PRIVATE_AUTHORED_ALT_TEXT';
    final records = <LogRecord>[];
    final subscription = Logger(
      'VideoPublication',
    ).onRecord.listen(records.add);
    addTearDown(subscription.cancel);
    final coordinator = VideoPublicationCoordinator(
      checkEligibility: () async => const VideoUploadLimits(canUpload: true),
      authorize: () async => _authorization(token: tokenCanary),
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
      coordinator.publish(altText: altTextCanary, aspectRatio: null),
      throwsA(isA<VideoTransportException>()),
    );

    final selected = selectDiagnosticRecord(records.last);
    expect(selected['operation'], 'video.upload');
    expect(selected['failureStage'], 'upload');
    expect(selected['outcome'], 'failed');
    expect(selected.toString(), contains('VideoTransportException'));
    expect(selected['stack'], isNotEmpty);
    expect(records.last.level, Level.SEVERE);
    final diagnosticOutput = records.map((record) => record.message).join('\n');
    expect(diagnosticOutput, isNot(contains(tokenCanary)));
    expect(diagnosticOutput, isNot(contains(altTextCanary)));
  });

  test(
    'IT-010 late video failure remains attributed to initiating account',
    () async {
      var activeAccount = 'did:plc:alice';
      final records = <LogRecord>[];
      final sub = Logger('VideoPublication').onRecord.listen(records.add);
      addTearDown(sub.cancel);
      final uploadDone = Completer<VideoServiceResult>();
      final coordinator = VideoPublicationCoordinator(
        operationAccountDid: activeAccount,
        checkEligibility: () async => const VideoUploadLimits(canUpload: true),
        authorize: () async => _authorization(),
        upload:
            ({
              required authorizationHeader,
              required cancelToken,
              required bypassDeduplication,
              required onProgress,
            }) => uploadDone.future,
        poll: (_, _) => throw StateError('poll must not run'),
        wait: (_) async {},
        publish: (_, {required allowBlobRecovery}) async {},
        onProgress: (_) {},
      );
      final pending = coordinator.publish(
        altText: 'private media text',
        aspectRatio: null,
      );
      final failure = expectLater(pending, throwsStateError);
      await Future<void>.delayed(Duration.zero);
      activeAccount = 'did:plc:bob';
      uploadDone.completeError(
        StateError('private source /Users/secret/video.mp4'),
      );
      await failure;
      final selected = selectDiagnosticRecord(records.last);
      expect(selected['diagnostic'], containsPair('actorDid', 'did:plc:alice'));
      expect(selected.toString(), isNot(contains(activeAccount)));
      expect(selected.toString(), isNot(contains('private source')));
      expect(coordinator.hasEphemeralState, isFalse);
    },
  );

  for (final phase in [
    'limits',
    'authorization',
    'upload',
    'polling',
    'publication',
  ]) {
    test(
      'IT-010 video $phase failure keeps initiating actor cause and stage',
      () async {
        final failure = StateError('private video $phase payload JWT canary');
        final stack = StackTrace.fromString(
          '#0 videoPhase (package:craftsky_app/feed/video.dart:25:3)',
        );
        final records = <LogRecord>[];
        final logs = Logger.root.onRecord.listen(records.add);
        addTearDown(logs.cancel);
        Future<T> failed<T>() async =>
            Error.throwWithStackTrace(failure, stack);
        const completed = VideoServiceResult(
          outcome: VideoServiceOutcome.completed,
          jobId: 'private job canary',
          blob: VideoServiceBlob(
            cid: 'bafyvideo',
            mimeType: 'video/mp4',
            size: 8,
          ),
        );
        final coordinator = VideoPublicationCoordinator(
          operationAccountDid: 'did:plc:alice',
          checkEligibility: () => phase == 'limits'
              ? failed()
              : Future.value(const VideoUploadLimits(canUpload: true)),
          authorize: () => phase == 'authorization'
              ? failed()
              : Future.value(_authorization()),
          upload:
              ({
                required authorizationHeader,
                required cancelToken,
                required bypassDeduplication,
                required onProgress,
              }) => phase == 'upload'
              ? failed()
              : Future.value(
                  phase == 'polling'
                      ? const VideoServiceResult(
                          outcome: VideoServiceOutcome.processing,
                          jobId: 'private job canary',
                        )
                      : completed,
                ),
          poll: (_, _) =>
              phase == 'polling' ? failed() : Future.value(completed),
          wait: (_) async {},
          publish: (_, {required allowBlobRecovery}) =>
              phase == 'publication' ? failed<void>() : Future.value(),
          onProgress: (_) {},
        );
        await expectLater(
          coordinator.publish(altText: 'private alt canary', aspectRatio: null),
          throwsA(same(failure)),
        );
        final record = records.singleWhere(
          (r) => r.level == Level.SEVERE && r.error == failure,
        );
        final diagnostic = selectDiagnosticRecord(record);
        expect(diagnostic['failureStage'], phase);
        expect(
          diagnostic['diagnostic'],
          containsPair('actorDid', 'did:plc:alice'),
        );
        expect(diagnostic['cause'].toString(), contains('StateError'));
        expect(diagnostic['stack'].toString(), contains('videoPhase'));
        for (final canary in [
          'private video',
          'private job canary',
          'private alt canary',
        ]) {
          expect(diagnostic.toString(), isNot(contains(canary)));
        }
        expect(coordinator.hasEphemeralState, isFalse);
      },
    );
  }

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

VideoUploadAuthorization _authorization({String token = 'ephemeral-secret'}) =>
    VideoUploadAuthorization.fromMap({
      'token': token,
      'expiresAt': '2030-01-01T00:00:00Z',
    });
