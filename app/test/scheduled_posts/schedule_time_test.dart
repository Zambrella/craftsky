import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-003 preserves UTC while local zone and offset display changes', () {
    final instant = ScheduledInstant(
      DateTime.parse('2026-03-29T00:30:00Z'),
    );

    final gmt = instant.displayIn(
      zoneName: 'GMT',
      offset: Duration.zero,
    );
    final bst = instant.displayIn(
      zoneName: 'BST',
      offset: const Duration(hours: 1),
    );

    expect(instant.utc, DateTime.parse('2026-03-29T00:30:00Z'));
    expect(instant.apiValue, '2026-03-29T00:30:00.000Z');
    expect(gmt.wallTime, DateTime.utc(2026, 3, 29, 0, 30));
    expect(gmt.zoneLabel, 'GMT (UTC+00:00)');
    expect(bst.wallTime, DateTime.utc(2026, 3, 29, 1, 30));
    expect(bst.zoneLabel, 'BST (UTC+01:00)');
    expect(gmt.instantUtc, instant.utc);
    expect(bst.instantUtc, instant.utc);
    expect(instant.apiValue, '2026-03-29T00:30:00.000Z');
  });
}
