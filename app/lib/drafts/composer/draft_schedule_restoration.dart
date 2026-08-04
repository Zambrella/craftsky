import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';
import 'package:craftsky_app/scheduled_posts/widgets/schedule_time_picker.dart';

final class RestoredDraftSchedule {
  const RestoredDraftSchedule({
    required this.choice,
    required this.scheduledAtLocal,
    required this.needsExplanation,
  });

  final ScheduleChoice choice;
  final DateTime? scheduledAtLocal;
  final bool needsExplanation;
}

RestoredDraftSchedule restoreDraftSchedule(
  DraftScheduleIntent intent, {
  required DateTime now,
}) {
  if (intent.choice == DraftScheduleChoice.now) {
    return const RestoredDraftSchedule(
      choice: ScheduleChoice.now,
      scheduledAtLocal: null,
      needsExplanation: false,
    );
  }

  final scheduledAt = intent.scheduledAtUtc?.toLocal();
  final isValid =
      scheduledAt != null &&
      ScheduleTimeWindow.fromNow(now.toLocal()).contains(scheduledAt);
  if (!isValid) {
    return const RestoredDraftSchedule(
      choice: ScheduleChoice.now,
      scheduledAtLocal: null,
      needsExplanation: true,
    );
  }

  return RestoredDraftSchedule(
    choice: ScheduleChoice.later,
    scheduledAtLocal: scheduledAt,
    needsExplanation: false,
  );
}
