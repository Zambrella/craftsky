import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';

const scheduledPostPreviewCharacters = 120;

final class ScheduledPostRowModel {
  const ScheduledPostRowModel({
    required this.id,
    required this.kind,
    required this.status,
    required this.preview,
    required this.time,
    this.projectTitle,
    this.firstMediaId,
  });

  factory ScheduledPostRowModel.fromSummary(
    ScheduledPostSummary summary, {
    required String zoneName,
    required Duration offset,
  }) {
    return ScheduledPostRowModel(
      id: summary.id,
      kind: summary.kind,
      status: summary.status,
      preview: _boundedPreview(summary.text),
      projectTitle: summary.projectTitle,
      firstMediaId: summary.mediaIds.firstOrNull,
      time: summary.scheduledAt.displayIn(
        zoneName: zoneName,
        offset: offset,
      ),
    );
  }

  final String id;
  final ScheduledPostKind kind;
  final ScheduledPostStatus status;
  final String preview;
  final String? projectTitle;
  final String? firstMediaId;
  final ScheduleTimeDisplay time;

  @override
  String toString() => 'ScheduledPostRowModel [REDACTED]';
}

String _boundedPreview(String text) {
  final characters = text.runes.toList(growable: false);
  if (characters.length <= scheduledPostPreviewCharacters) return text;
  final visible = characters.take(scheduledPostPreviewCharacters - 1);
  return '${String.fromCharCodes(visible)}…';
}
