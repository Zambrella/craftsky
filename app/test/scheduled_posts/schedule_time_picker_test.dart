import 'package:craftsky_app/scheduled_posts/widgets/schedule_time_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('AT-003 permits only the exact whole-minute schedule window', (
    tester,
  ) async {
    late BuildContext context;
    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (value) {
            context = value;
            return const SizedBox();
          },
        ),
      ),
    );

    final now = DateTime(2026, 8, 1, 12, 0, 30);
    final rejected = <DateTime>[];

    Future<DateTime?> pick(DateTime value) => pickScheduledLocalTime(
      context: context,
      now: now,
      selectDate:
          ({
            required initialDate,
            required firstDate,
            required lastDate,
          }) async => value,
      selectTime: ({required initialTime}) async =>
          TimeOfDay.fromDateTime(value),
      onOutOfRange: rejected.add,
    );

    expect(await pick(DateTime(2026, 8, 1, 12, 5)), isNull);
    expect(
      await pick(DateTime(2026, 8, 1, 12, 6)),
      DateTime(2026, 8, 1, 12, 6),
    );
    expect(
      await pick(DateTime(2026, 8, 29, 12)),
      DateTime(2026, 8, 29, 12),
    );
    expect(await pick(DateTime(2026, 8, 29, 12, 1)), isNull);
    expect(rejected, [
      DateTime(2026, 8, 1, 12, 5),
      DateTime(2026, 8, 29, 12, 1),
    ]);
  });
}
