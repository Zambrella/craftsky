import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';

final class ScheduleCapacityState {
  const ScheduleCapacityState._({
    required this.selectedChoice,
    required this.scheduleVisible,
    required this.scheduleEnabled,
    required this.postNowEnabled,
    required this.showManageLink,
    required this.capacityLabel,
  });

  factory ScheduleCapacityState.derive({
    required int scheduledCount,
    required ScheduleChoice choice,
    bool ownsExistingSlot = false,
  }) {
    if (scheduledCount < 0 || scheduledCount > 3) {
      throw RangeError.range(scheduledCount, 0, 3, 'scheduledCount');
    }
    final full = scheduledCount == 3;
    return ScheduleCapacityState._(
      selectedChoice: choice,
      scheduleVisible: true,
      scheduleEnabled: !full || ownsExistingSlot,
      postNowEnabled: true,
      showManageLink: full,
      capacityLabel: '$scheduledCount of 3 scheduled',
    );
  }

  final ScheduleChoice selectedChoice;
  final bool scheduleVisible;
  final bool scheduleEnabled;
  final bool postNowEnabled;
  final bool showManageLink;
  final String capacityLabel;
}
