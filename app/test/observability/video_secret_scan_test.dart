import 'package:craftsky_app/observability/video_diagnostics.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-012 emits only bounded video diagnostics', () {
    const event = VideoDiagnosticEvent(
      operation: VideoOperation.upload,
      outcome: VideoOperationOutcome.failed,
      byteCount: 234567890,
      requestId: 'request-safe',
    );

    final output = event.toString();
    expect(output, contains('100-299mb'));
    expect(output, contains('request-safe'));
    for (final secret in [
      'Bearer secret',
      'service.jwt',
      '/private/video.mp4',
      'did:plc:alice',
      'bafyvideo',
      'job-123',
      'https://video.bsky.app/watch/private',
      'authored alt text',
    ]) {
      expect(output, isNot(contains(secret)));
    }
  });

  test('UT-012 diagnostic API cannot accept authored or credential data', () {
    expect(
      VideoDiagnosticEvent.new,
      isA<
        VideoDiagnosticEvent Function({
          required VideoOperation operation,
          required VideoOperationOutcome outcome,
          int? byteCount,
          String? requestId,
        })
      >(),
    );
  });
}
