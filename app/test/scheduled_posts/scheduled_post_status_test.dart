import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-014 exposes only the four unpublished management statuses', () {
    expect(
      scheduledPostStatusFromWire('scheduled'),
      ScheduledPostStatus.scheduled,
    );
    expect(
      scheduledPostStatusFromWire('publishing'),
      ScheduledPostStatus.publishing,
    );
    expect(
      scheduledPostStatusFromWire('retrying'),
      ScheduledPostStatus.retrying,
    );
    expect(
      scheduledPostStatusFromWire('needs_attention'),
      ScheduledPostStatus.needsAttention,
    );

    for (final hidden in [
      'published',
      'deleted',
      'cleanup_pending',
      'unknown',
    ]) {
      expect(
        scheduledPostStatusFromWire(hidden),
        isNull,
        reason: '$hidden must not become management history',
      );
    }
    expect(ScheduledPostStatus.values, hasLength(4));
  });
}
