import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post_row_model.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('UT-008 maps summaries to bounded management rows', () {
    final longText = List.filled(140, 'x').join();
    final scheduledAt = ScheduledInstant(DateTime.utc(2026, 8, 1, 12));

    final standard = ScheduledPostRowModel.fromSummary(
      ScheduledPostSummary(
        id: 'standard-id',
        kind: ScheduledPostKind.standard,
        status: ScheduledPostStatus.scheduled,
        text: longText,
        scheduledAt: scheduledAt,
      ),
      zoneName: 'BST',
      offset: const Duration(hours: 1),
    );
    expect(standard.firstMediaId, isNull);
    expect(standard.kind, ScheduledPostKind.standard);
    expect(standard.projectTitle, isNull);
    expect(standard.preview, '${List.filled(119, 'x').join()}…');
    expect(standard.preview.length, 120);
    expect(standard.time.wallTime, DateTime.utc(2026, 8, 1, 13));
    expect(standard.time.zoneLabel, 'BST (UTC+01:00)');
    expect(standard.status, ScheduledPostStatus.scheduled);

    final project = ScheduledPostRowModel.fromSummary(
      ScheduledPostSummary(
        id: 'project-id',
        kind: ScheduledPostKind.project,
        status: ScheduledPostStatus.retrying,
        text: 'Project body',
        projectTitle: 'Cardigan',
        scheduledAt: scheduledAt,
        mediaIds: const ['first-private-media', 'second-private-media'],
      ),
      zoneName: 'BST',
      offset: const Duration(hours: 1),
    );
    expect(project.firstMediaId, 'first-private-media');
    expect(project.kind, ScheduledPostKind.project);
    expect(project.projectTitle, 'Cardigan');
    expect(project.preview, 'Project body');
    expect(project.status, ScheduledPostStatus.retrying);
  });
}
