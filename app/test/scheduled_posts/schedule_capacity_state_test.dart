import 'package:craftsky_app/scheduled_posts/composer/schedule_capacity_state.dart';
import 'package:craftsky_app/scheduled_posts/composer/schedule_composer_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-012 derives actions for available and full capacity', () {
    for (var count = 0; count < 3; count++) {
      final state = ScheduleCapacityState.derive(
        scheduledCount: count,
        choice: ScheduleChoice.later,
      );
      expect(state.scheduleVisible, isTrue);
      expect(state.scheduleEnabled, isTrue);
      expect(state.showManageLink, isFalse);
      expect(state.showCapacityWarning, isFalse);
      expect(state.postNowEnabled, isTrue);
    }

    for (final choice in ScheduleChoice.values) {
      final full = ScheduleCapacityState.derive(
        scheduledCount: 3,
        choice: choice,
      );
      expect(full.scheduleVisible, isTrue);
      expect(full.scheduleEnabled, isFalse);
      expect(full.showManageLink, isTrue);
      expect(full.showCapacityWarning, isTrue);
      expect(full.postNowEnabled, isTrue);
      expect(full.selectedChoice, choice);
    }

    final retainedSlot = ScheduleCapacityState.derive(
      scheduledCount: 3,
      choice: ScheduleChoice.later,
      ownsExistingSlot: true,
    );
    expect(retainedSlot.scheduleEnabled, isTrue);
    expect(retainedSlot.showManageLink, isFalse);
    expect(retainedSlot.showCapacityWarning, isFalse);
  });
}
