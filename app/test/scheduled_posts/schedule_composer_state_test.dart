import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';
import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-007 initializes timing state for new, future and failed posts', () {
    final future = ScheduledInstant(DateTime.utc(2026, 8, 1, 12));
    final missed = ScheduledInstant(DateTime.utc(2026, 7, 31, 11));

    final newPost = ScheduleComposerState.newPost();
    expect(newPost.choice, ScheduleChoice.now);
    expect(newPost.scheduledAt, isNull);
    expect(newPost.missedScheduledAt, isNull);

    final futureEdit = ScheduleComposerState.futureEdit(future);
    expect(futureEdit.choice, ScheduleChoice.later);
    expect(futureEdit.scheduledAt?.utc, future.utc);
    expect(futureEdit.missedScheduledAt, isNull);

    final needsAttention = ScheduleComposerState.needsAttentionEdit(missed);
    expect(needsAttention.choice, ScheduleChoice.now);
    expect(needsAttention.scheduledAt, isNull);
    expect(needsAttention.missedScheduledAt?.utc, missed.utc);
  });
}
