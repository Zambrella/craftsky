import 'package:craftsky_app/feed/composer/video_publication_coordinator.dart';
import 'package:craftsky_app/feed/data/video_service_client.dart';
import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:craftsky_app/feed/models/video_upload_limits.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-013 a fresh attempt obtains fresh authorization', () async {
    var authorizations = 0;
    var uploads = 0;
    final coordinator = _coordinator(
      authorize: () async {
        authorizations++;
        return _authorization('token-$authorizations');
      },
      upload: () async {
        uploads++;
        if (uploads == 1) {
          throw const VideoTransportException(
            VideoTransportFailure.unavailable,
          );
        }
        return _completed;
      },
      poll: (_, _) async => _completed,
    );

    await expectLater(
      coordinator.publish(altText: '', aspectRatio: null),
      throwsA(isA<VideoTransportException>()),
    );
    await coordinator.publish(altText: '', aspectRatio: null);

    expect(authorizations, 2);
    expect(coordinator.hasEphemeralState, isFalse);
  });

  test('UT-013 transient poll recovery retains the current job', () async {
    var authorizations = 0;
    var polls = 0;
    final coordinator = _coordinator(
      authorize: () async {
        authorizations++;
        return _authorization('token-one');
      },
      upload: () async => const VideoServiceResult(
        outcome: VideoServiceOutcome.processing,
        jobId: 'job-one',
      ),
      poll: (jobId, _) async {
        expect(jobId, 'job-one');
        polls++;
        if (polls == 1) {
          throw const VideoTransportException(
            VideoTransportFailure.unavailable,
          );
        }
        return _completed;
      },
    );

    await coordinator.publish(altText: '', aspectRatio: null);

    expect(polls, 2);
    expect(authorizations, 1);
    expect(coordinator.hasEphemeralState, isFalse);
  });
}

const _completed = VideoServiceResult(
  outcome: VideoServiceOutcome.completed,
  jobId: 'job-one',
  blob: VideoServiceBlob(
    cid: 'bafyvideo',
    mimeType: 'video/mp4',
    size: 8,
  ),
);

VideoUploadAuthorization _authorization(String token) =>
    VideoUploadAuthorization.fromMap({
      'token': token,
      'expiresAt': '2030-01-01T00:00:00Z',
    });

VideoPublicationCoordinator _coordinator({
  required Future<VideoUploadAuthorization> Function() authorize,
  required Future<VideoServiceResult> Function() upload,
  required Future<VideoServiceResult> Function(String, Object) poll,
}) => VideoPublicationCoordinator(
  checkEligibility: () async => const VideoUploadLimits(canUpload: true),
  authorize: authorize,
  upload:
      ({
        required authorizationHeader,
        required cancelToken,
        required onProgress,
      }) => upload(),
  poll: poll,
  wait: (_) async {},
  publish: (_) async {},
  onProgress: (_) {},
  clock: () => DateTime.utc(2026),
);
