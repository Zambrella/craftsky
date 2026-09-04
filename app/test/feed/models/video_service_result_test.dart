import 'package:craftsky_app/feed/models/video_service_result.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('VideoServiceResult', () {
    const blob = VideoServiceBlob(
      cid: 'bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a',
      mimeType: 'video/mp4',
      size: 123,
    );

    for (final testCase
        in <
          ({
            String name,
            Map<String, Object?> json,
            VideoServiceOutcome outcome,
          })
        >[
          (
            name: 'created is processing',
            json: {'state': 'JOB_STATE_CREATED', 'jobId': 'job-1'},
            outcome: VideoServiceOutcome.processing,
          ),
          (
            name: 'unknown state is processing',
            json: {'state': 'future_processing_state', 'jobId': 'job-1'},
            outcome: VideoServiceOutcome.processing,
          ),
          (
            name: 'completed with blob succeeds',
            json: {
              'state': 'JOB_STATE_COMPLETED',
              'jobId': 'job-1',
              'blob': blob.toJson(),
            },
            outcome: VideoServiceOutcome.completed,
          ),
          (
            name: 'already exists with blob succeeds',
            json: {
              'state': 'JOB_STATE_FAILED',
              'jobId': 'job-1',
              'error': 'already_exists',
              'blob': blob.toJson(),
            },
            outcome: VideoServiceOutcome.completed,
          ),
          (
            name: 'unverified email is actionable',
            json: {
              'state': 'JOB_STATE_FAILED',
              'jobId': 'job-1',
              'error': 'email_unverified',
            },
            outcome: VideoServiceOutcome.emailUnverified,
          ),
          (
            name: 'quota is actionable',
            json: {
              'state': 'JOB_STATE_FAILED',
              'jobId': 'job-1',
              'error': 'quota_exhausted',
            },
            outcome: VideoServiceOutcome.quotaExhausted,
          ),
          (
            name: 'provider is actionable',
            json: {
              'state': 'JOB_STATE_FAILED',
              'jobId': 'job-1',
              'error': 'provider_unsupported',
            },
            outcome: VideoServiceOutcome.providerUnsupported,
          ),
          (
            name: 'validation is actionable',
            json: {
              'state': 'JOB_STATE_FAILED',
              'jobId': 'job-1',
              'error': 'video_too_long',
            },
            outcome: VideoServiceOutcome.validationFailed,
          ),
          (
            name: 'unknown failure is bounded',
            json: {
              'state': 'JOB_STATE_FAILED',
              'jobId': 'job-1',
              'error': 'private upstream detail',
              'message': 'authored or provider text',
            },
            outcome: VideoServiceOutcome.processingFailed,
          ),
          (
            name: 'already exists without blob fails closed',
            json: {
              'state': 'JOB_STATE_FAILED',
              'jobId': 'job-1',
              'error': 'already_exists',
            },
            outcome: VideoServiceOutcome.processingFailed,
          ),
          (
            name: 'completed without blob fails closed',
            json: {'state': 'JOB_STATE_COMPLETED', 'jobId': 'job-1'},
            outcome: VideoServiceOutcome.processingFailed,
          ),
        ]) {
      test(testCase.name, () {
        final result = VideoServiceResult.fromJson(testCase.json);

        expect(result.outcome, testCase.outcome);
        expect(result.toString(), isNot(contains('private upstream detail')));
        expect(result.toString(), isNot(contains('authored or provider text')));
      });
    }
  });
}
