import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/video_draft_descriptor.dart';

enum LocalPostDraftKind { standard, project }

enum DraftScheduleChoice { now, later }

enum LocalPostDraftAvailability { available, unavailable }

sealed class LocalDraftContent {
  const LocalDraftContent();
}

final class StandardDraftContent extends LocalDraftContent {
  const StandardDraftContent({required this.text, required this.languages});

  final String text;
  final List<String> languages;

  @override
  String toString() => 'StandardDraftContent(<redacted>)';
}

/// A whitelisted snapshot of the project composer's known editable fields.
final class ProjectDraftContent extends LocalDraftContent {
  const ProjectDraftContent({
    required this.body,
    required this.languages,
    required this.knownProjectFieldValues,
  });

  final String body;
  final List<String> languages;
  final Map<String, Object?> knownProjectFieldValues;

  @override
  String toString() => 'ProjectDraftContent(<redacted>)';
}

final class DraftScheduleIntent {
  const DraftScheduleIntent.now()
    : choice = DraftScheduleChoice.now,
      scheduledAtUtc = null,
      savedOffsetMinutes = null;

  const DraftScheduleIntent.later({
    required this.scheduledAtUtc,
    required this.savedOffsetMinutes,
  }) : choice = DraftScheduleChoice.later;

  final DraftScheduleChoice choice;
  final DateTime? scheduledAtUtc;
  final int? savedOffsetMinutes;

  @override
  String toString() => 'DraftScheduleIntent(<redacted>)';
}

final class LocalPostDraft {
  LocalPostDraft({
    required this.id,
    required this.owner,
    required this.kind,
    required this.createdAt,
    required this.updatedAt,
    required this.content,
    required this.schedule,
    required List<DraftMediaDescriptor> media,
    this.video,
    this.availability = LocalPostDraftAvailability.available,
    this.revision = 1,
  }) : media = List.unmodifiable(media);

  factory LocalPostDraft.unavailable({
    required String id,
    required AccountKey owner,
  }) => LocalPostDraft(
    id: id,
    owner: owner,
    kind: LocalPostDraftKind.standard,
    createdAt: DateTime.fromMillisecondsSinceEpoch(0, isUtc: true),
    updatedAt: DateTime.fromMillisecondsSinceEpoch(0, isUtc: true),
    content: const StandardDraftContent(text: '', languages: []),
    schedule: const DraftScheduleIntent.now(),
    media: const [],
    availability: LocalPostDraftAvailability.unavailable,
    revision: 0,
  );

  final String id;
  final AccountKey owner;
  final LocalPostDraftKind kind;
  final DateTime createdAt;
  final DateTime updatedAt;
  final LocalDraftContent content;
  final DraftScheduleIntent schedule;
  final List<DraftMediaDescriptor> media;
  final VideoDraftDescriptor? video;
  final LocalPostDraftAvailability availability;
  final int revision;

  /// Whether the manifest parsed successfully enough to reopen the composer.
  /// Individual unavailable media can still be replaced after opening.
  bool get canEdit => revision > 0;

  LocalPostDraft withStorageState({
    required LocalPostDraftAvailability availability,
    required List<DraftMediaDescriptor> media,
    VideoDraftDescriptor? video,
  }) => LocalPostDraft(
    id: id,
    owner: owner,
    kind: kind,
    createdAt: createdAt,
    updatedAt: updatedAt,
    content: content,
    schedule: schedule,
    media: media,
    video: video ?? this.video,
    availability: availability,
    revision: revision,
  );

  @override
  String toString() => 'LocalPostDraft(<redacted>)';
}
