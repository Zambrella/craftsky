import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-011 live coordinator clears credentials after success', () async {
    final coordinator = _coordinator(
      upload: () async => const VideoServiceResult(
        outcome: VideoServiceOutcome.completed,
        jobId: 'job-one',
        blob: VideoServiceBlob(
          cid: 'bafyvideo',
          mimeType: 'video/mp4',
          size: 8,
        ),
      ),
    );

    await coordinator.publish(altText: '', aspectRatio: null);

    expect(coordinator.hasEphemeralState, isFalse);
    expect(coordinator, isNot(isA<Map<Object?, Object?>>()));
    expect(coordinator.toString(), 'VideoPublicationCoordinator(<redacted>)');
  });

  test('UT-011 live coordinator clears credentials after failure', () async {
    final coordinator = _coordinator(
      upload: () => throw StateError('upload failed'),
    );

    await expectLater(
      coordinator.publish(altText: '', aspectRatio: null),
      throwsStateError,
    );
    expect(coordinator.hasEphemeralState, isFalse);
  });
}

VideoPublicationCoordinator _coordinator({
  required Future<VideoServiceResult> Function() upload,
  Future<void> Function(Duration)? wait,
}) => VideoPublicationCoordinator(
  checkEligibility: () async => const VideoUploadLimits(canUpload: true),
  authorize: () async => VideoUploadAuthorization.fromMap({
    'token': 'service.jwt.secret',
    'expiresAt': '2030-01-01T00:00:00Z',
  }),
  upload:
      ({
        required authorizationHeader,
        required cancelToken,
        required onProgress,
      }) => upload(),
  poll: (_, _) => throw StateError('unexpected poll'),
  wait: wait ?? (_) async {},
  publish: (_) async {},
  onProgress: (_) {},
  clock: () => DateTime.utc(2026),
);
