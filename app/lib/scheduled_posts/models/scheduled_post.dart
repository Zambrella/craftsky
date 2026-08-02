import 'package:craftsky_app/scheduled_posts/models/schedule_time.dart';

enum ScheduledPostKind { standard, project }

enum ScheduledPostStatus { scheduled, publishing, retrying, needsAttention }

ScheduledPostStatus? scheduledPostStatusFromWire(String value) {
  return switch (value) {
    'scheduled' => ScheduledPostStatus.scheduled,
    'publishing' => ScheduledPostStatus.publishing,
    'retrying' => ScheduledPostStatus.retrying,
    'needs_attention' => ScheduledPostStatus.needsAttention,
    _ => null,
  };
}

final class ScheduledPostSummary {
  const ScheduledPostSummary({
    required this.id,
    required this.kind,
    required this.status,
    required this.text,
    required this.scheduledAt,
    this.projectTitle,
    this.mediaIds = const [],
    this.needsAttentionExpiresAt,
  });

  final String id;
  final ScheduledPostKind kind;
  final ScheduledPostStatus status;
  final String text;
  final String? projectTitle;
  final ScheduledInstant scheduledAt;
  final List<String> mediaIds;
  final DateTime? needsAttentionExpiresAt;

  @override
  String toString() => 'ScheduledPostSummary [REDACTED]';
}

final class ScheduledPostDetail {
  const ScheduledPostDetail({
    required this.id,
    required this.operationId,
    required this.status,
    required this.scheduledAt,
    required this.payload,
  });

  final String id;
  final String operationId;
  final ScheduledPostStatus status;
  final ScheduledInstant scheduledAt;
  final Map<String, dynamic> payload;

  @override
  String toString() => 'ScheduledPostDetail [REDACTED]';
}
