import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';

enum ScheduleChoice { now, later }

final class ScheduleComposerState {
  const new _({
    required this.choice,
    this.scheduledAt,
    this.missedScheduledAt,
  });

  factory newPost() {
    return const ScheduleComposerState._(choice: ScheduleChoice.now);
  }

  factory futureEdit(ScheduledInstant scheduledAt) {
    return ScheduleComposerState._(
      choice: ScheduleChoice.later,
      scheduledAt: scheduledAt,
    );
  }

  factory needsAttentionEdit(
    ScheduledInstant missedScheduledAt,
  ) {
    return ScheduleComposerState._(
      choice: ScheduleChoice.now,
      missedScheduledAt: missedScheduledAt,
    );
  }

  final ScheduleChoice choice;
  final ScheduledInstant? scheduledAt;
  final ScheduledInstant? missedScheduledAt;
}
