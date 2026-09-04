import 'package:craftsky_app/feed/composer/video_job_poller.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-009 polling starts at one second and backs off to five', () {
    expect(
      List.generate(7, videoPollingDelay),
      const [
        Duration(seconds: 1),
        Duration(seconds: 2),
        Duration(seconds: 3),
        Duration(seconds: 4),
        Duration(seconds: 5),
        Duration(seconds: 5),
        Duration(seconds: 5),
      ],
    );
  });

  test('UT-009 honors longer Retry-After and all stop boundaries', () {
    expect(
      videoPollingDelay(0, retryAfter: const Duration(seconds: 12)),
      const Duration(seconds: 12),
    );
    for (final boundary in VideoPollingStop.values) {
      expect(shouldStopVideoPolling(boundary), isTrue);
    }
    expect(
      isVideoJobExpired(
        startedAt: DateTime.utc(2030),
        now: DateTime.utc(2030, 1, 1, 1),
      ),
      isTrue,
    );
  });
}
