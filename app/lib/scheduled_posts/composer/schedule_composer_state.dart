import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';

enum ScheduleChoice { now, later }

final class ScheduleComposerState {
  const ScheduleComposerState._({
    required this.choice,
    this.scheduledAt,
    this.missedScheduledAt,
  });

  factory ScheduleComposerState.newPost() {
    return const ScheduleComposerState._(choice: ScheduleChoice.now);
  }

  factory ScheduleComposerState.futureEdit(ScheduledInstant scheduledAt) {
    return ScheduleComposerState._(
      choice: ScheduleChoice.later,
      scheduledAt: scheduledAt,
    );
  }

  factory ScheduleComposerState.needsAttentionEdit(
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
