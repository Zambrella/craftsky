import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/draft_manifest_error.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';

export 'package:craftsky_app/drafts/models/draft_manifest_error.dart';

/// Explicit persistence schema for device-local post drafts.
abstract final class DraftManifestCodec {
  static const schemaVersion = 1;

  static String encode(LocalPostDraft draft) {
    _validateMediaList(draft.media);
    return jsonEncode({
      'schemaVersion': schemaVersion,
      'id': draft.id,
      'owner': draft.owner.did.value,
      'kind': draft.kind.name,
      'createdAt': draft.createdAt.toUtc().toIso8601String(),
      'updatedAt': draft.updatedAt.toUtc().toIso8601String(),
      'revision': draft.revision,
      'content': _encodeContent(draft.content),
      'schedule': {
        'choice': draft.schedule.choice.name,
        if (draft.schedule.scheduledAtUtc case final scheduledAt?)
          'scheduledAtUtc': scheduledAt.toUtc().toIso8601String(),
        'savedOffsetMinutes': ?draft.schedule.savedOffsetMinutes,
      },
      'media': draft.media.map(_encodeMedia).toList(growable: false),
    });
  }

  static LocalPostDraft decode(String source) {
    try {
      final manifest = jsonDecode(source) as Map<String, Object?>;
      if (manifest['schemaVersion'] != schemaVersion) {
        throw const DraftManifestException(
          DraftManifestFailureReason.unsupportedVersion,
        );
      }
      final content = manifest['content']! as Map<String, Object?>;
      final schedule = manifest['schedule']! as Map<String, Object?>;
      final media = manifest['media']! as List<Object?>;

      return LocalPostDraft(
        id: manifest['id']! as String,
        owner: AccountKey(manifest['owner']! as String),
        kind: LocalPostDraftKind.values.byName(manifest['kind']! as String),
        createdAt: DateTime.parse(manifest['createdAt']! as String).toUtc(),
        updatedAt: DateTime.parse(manifest['updatedAt']! as String).toUtc(),
        revision: manifest['revision']! as int,
        content: _decodeContent(content),
        schedule: _decodeSchedule(schedule),
        media: _decodeMediaList(media),
      );
    } on DraftManifestException {
      rethrow;
    } on Object {
      throw const DraftManifestException(DraftManifestFailureReason.corrupt);
    }
  }

  static Map<String, Object?> _encodeContent(LocalDraftContent content) =>
      switch (content) {
        StandardDraftContent(:final text, :final languages) => {
          'type': 'standard',
          'text': text,
          'languages': languages,
        },
        ProjectDraftContent(
          :final body,
          :final languages,
          :final knownProjectFieldValues,
        ) =>
          {
            'type': 'project',
            'body': body,
            'languages': languages,
            'knownProjectFieldValues': knownProjectFieldValues,
          },
      };

  static LocalDraftContent _decodeContent(Map<String, Object?> content) {
    final languages = (content['languages']! as List<Object?>).cast<String>();
    return switch (content['type']) {
      'standard' => StandardDraftContent(
        text: content['text']! as String,
        languages: List.unmodifiable(languages),
      ),
      'project' => ProjectDraftContent(
        body: content['body']! as String,
        languages: List.unmodifiable(languages),
        knownProjectFieldValues: Map.unmodifiable(
          content['knownProjectFieldValues']! as Map<String, Object?>,
        ),
      ),
      _ => throw const FormatException('Draft manifest is unavailable'),
    };
  }

  static DraftScheduleIntent _decodeSchedule(Map<String, Object?> schedule) {
    if (schedule['choice'] == 'now') return const DraftScheduleIntent.now();
    return DraftScheduleIntent.later(
      scheduledAtUtc: DateTime.parse(
        schedule['scheduledAtUtc']! as String,
      ).toUtc(),
      savedOffsetMinutes: schedule['savedOffsetMinutes']! as int,
    );
  }

  static Map<String, Object?> _encodeMedia(DraftMediaDescriptor media) => {
    'mediaId': media.mediaId,
    'storageRevision': media.storageRevision,
    'storageFileName': media.storageFileName,
    'displayFileName': media.displayFileName,
    'mimeType': media.mimeType,
    'byteLength': media.byteLength,
    'sha256': media.sha256,
    'width': media.width,
    'height': media.height,
    'altText': media.altText,
    'order': media.order,
  };

  static DraftMediaDescriptor _decodeMedia(Map<String, Object?> media) {
    final descriptor = DraftMediaDescriptor(
      mediaId: media['mediaId']! as String,
      storageRevision: media['storageRevision']! as String,
      storageFileName: media['storageFileName']! as String,
      displayFileName: media['displayFileName']! as String,
      mimeType: media['mimeType']! as String,
      byteLength: media['byteLength']! as int,
      sha256: media['sha256']! as String,
      width: media['width']! as int,
      height: media['height']! as int,
      altText: media['altText']! as String,
      order: media['order']! as int,
    )..validate();
    return descriptor;
  }

  static List<DraftMediaDescriptor> _decodeMediaList(List<Object?> source) {
    final descriptors = source
        .cast<Map<String, Object?>>()
        .map(_decodeMedia)
        .toList(growable: false);
    _validateMediaList(descriptors);
    return descriptors;
  }

  static void _validateMediaList(List<DraftMediaDescriptor> descriptors) {
    if (descriptors.length > 4) {
      throw const DraftManifestException(
        DraftManifestFailureReason.invalidMedia,
      );
    }
    final mediaIds = <String>{};
    final storageFileNames = <String>{};
    for (var index = 0; index < descriptors.length; index++) {
      final descriptor = descriptors[index]..validate();
      if (descriptor.order != index ||
          !mediaIds.add(descriptor.mediaId) ||
          !storageFileNames.add(descriptor.storageFileName)) {
        throw const DraftManifestException(
          DraftManifestFailureReason.invalidMedia,
        );
      }
    }
  }
}
