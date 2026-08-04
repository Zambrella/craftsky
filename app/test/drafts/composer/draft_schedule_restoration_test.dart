import 'package:craftsky_app/drafts/composer/draft_schedule_restoration.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final now = DateTime.utc(2026, 8, 3, 10, 0, 30);

  test('preserves a saved Later instant that still meets current rules', () {
    final instant = DateTime.utc(2026, 8, 3, 10, 30);

    final restored = restoreDraftSchedule(
      DraftScheduleIntent.later(
        scheduledAtUtc: instant,
        savedOffsetMinutes: 0,
      ),
      now: now,
    );

    expect(restored.choice, ScheduleChoice.later);
    expect(restored.scheduledAtLocal, instant.toLocal());
    expect(restored.needsExplanation, isFalse);
  });

  test('resets expired or newly invalid Later intent to Now', () {
    for (final invalid in [
      DateTime.utc(2026, 8, 3, 10, 4),
      DateTime.utc(2026, 9),
      DateTime.utc(2026, 8, 3, 10, 30, 10),
    ]) {
      final restored = restoreDraftSchedule(
        DraftScheduleIntent.later(
          scheduledAtUtc: invalid,
          savedOffsetMinutes: 0,
        ),
        now: now,
      );

      expect(restored.choice, ScheduleChoice.now);
      expect(restored.scheduledAtLocal, isNull);
      expect(restored.needsExplanation, isTrue);
    }
  });

  test('restores an explicit Now intent without an explanation', () {
    final restored = restoreDraftSchedule(
      const DraftScheduleIntent.now(),
      now: now,
    );

    expect(restored.choice, ScheduleChoice.now);
    expect(restored.needsExplanation, isFalse);
  });
}
