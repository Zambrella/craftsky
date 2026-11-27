import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/drafts/models/draft_manifest_error.dart';
import 'package:craftsky_app/drafts/models/draft_media_descriptor.dart';
import 'package:craftsky_app/drafts/models/local_post_draft.dart';
import 'package:craftsky_app/drafts/models/video_draft_descriptor.dart';
import 'package:craftsky_app/feed/media/media_config.dart';
import 'package:craftsky_app/feed/models/instagram_import_publication.dart';
import 'package:craftsky_app/feed/models/instagram_post_import_source.dart';
import 'package:craftsky_app/feed/models/video_thumbnail_frame.dart';

export 'package:craftsky_app/drafts/models/draft_manifest_error.dart';

/// Explicit persistence schema for device-local post drafts.
abstract final class DraftManifestCodec {
  static const schemaVersion = 1;

  static String encode(LocalPostDraft draft) {
    _validateMediaList(draft.media);
    draft.video?.validate();
    _validateImportSchedule(draft.instagramImport, draft.schedule);
    _validateInstagramPublication(
      draft.instagramImport,
      draft.instagramPublication,
    );
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
      'instagramImport': ?draft.instagramImport?.toMap(),
      'instagramPublication': ?draft.instagramPublication?.toMap(),
      'media': draft.media.map(_encodeMedia).toList(growable: false),
      if (draft.video case final video?) 'video': _encodeVideo(video),
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
      final schedule = _decodeSchedule(
        manifest['schedule']! as Map<String, Object?>,
      );
      final rawImport = manifest['instagramImport'];
      final instagramImport = rawImport == null
          ? null
          : InstagramPostImportSource.fromMap(
              rawImport as Map<String, Object?>,
            );
      _validateImportSchedule(instagramImport, schedule);
      final rawPublication = manifest['instagramPublication'];
      final publication = rawPublication == null
          ? null
          : InstagramImportPublication.fromMap(
              rawPublication as Map<String, Object?>,
            );
      _validateInstagramPublication(instagramImport, publication);
      final media = manifest['media']! as List<Object?>;

      return LocalPostDraft(
        id: manifest['id']! as String,
        owner: AccountKey(manifest['owner']! as String),
        kind: LocalPostDraftKind.values.byName(manifest['kind']! as String),
        createdAt: DateTime.parse(manifest['createdAt']! as String).toUtc(),
        updatedAt: DateTime.parse(manifest['updatedAt']! as String).toUtc(),
        revision: manifest['revision']! as int,
        content: _decodeContent(content),
        schedule: schedule,
        instagramImport: instagramImport,
        instagramPublication: publication,
        media: _decodeMediaList(media),
        video: _decodeVideo(manifest['video']),
      );
    } on DraftManifestException {
      rethrow;
    } on Object catch (error, stackTrace) {
      Error.throwWithStackTrace(
        DraftManifestException(
          DraftManifestFailureReason.corrupt,
          diagnosticCause: error,
          diagnosticStack: stackTrace,
        ),
        stackTrace,
      );
    }
  }

  static void _validateInstagramPublication(
    InstagramPostImportSource? source,
    InstagramImportPublication? publication,
  ) {
    if (publication != null && !publication.matchesSource(source)) {
      throw const DraftManifestException(DraftManifestFailureReason.corrupt);
    }
  }

  static void _validateImportSchedule(
    InstagramPostImportSource? instagramImport,
    DraftScheduleIntent schedule,
  ) {
    if (instagramImport != null && schedule.choice != DraftScheduleChoice.now) {
      throw const DraftManifestException(DraftManifestFailureReason.corrupt);
    }
  }

  static Map<String, Object?> _encodeContent(LocalDraftContent content) =>
      switch (content) {
        StandardDraftContent(:final text, :final languages, :final sponsored) =>
          {
            'type': 'standard',
            'text': text,
            'languages': languages,
            'sponsored': sponsored,
          },
        ProjectDraftContent(
          :final body,
          :final languages,
          :final knownProjectFieldValues,
          :final sponsored,
        ) =>
          {
            'type': 'project',
            'body': body,
            'languages': languages,
            'knownProjectFieldValues': knownProjectFieldValues,
            'sponsored': sponsored,
          },
      };

  static LocalDraftContent _decodeContent(Map<String, Object?> content) {
    final languages = (content['languages']! as List<Object?>).cast<String>();
    return switch (content['type']) {
      'standard' => StandardDraftContent(
        text: content['text']! as String,
        languages: List.unmodifiable(languages),
        sponsored: content['sponsored'] == true,
      ),
      'project' => ProjectDraftContent(
        body: content['body']! as String,
        languages: List.unmodifiable(languages),
        knownProjectFieldValues: Map.unmodifiable(
          content['knownProjectFieldValues']! as Map<String, Object?>,
        ),
        sponsored: content['sponsored'] == true,
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

  static Map<String, Object?> _encodeVideo(VideoDraftDescriptor video) => {
    'storageRevision': video.storageRevision,
    'sourceStorageFileName': video.sourceStorageFileName,
    'posterStorageFileName': video.posterStorageFileName,
    'displayFileName': video.displayFileName,
    'mimeType': video.mimeType,
    'sourceByteLength': video.sourceByteLength,
    'sourceSha256': video.sourceSha256,
    'posterByteLength': video.posterByteLength,
    'posterSha256': video.posterSha256,
    'thumbnailFramePercent': video.thumbnailFrame.percent,
    if (video.thumbnailStorageFileName != null) ...{
      'thumbnailStorageFileName': video.thumbnailStorageFileName,
      'thumbnailByteLength': video.thumbnailByteLength,
      'thumbnailSha256': video.thumbnailSha256,
    },
    'posterMimeType': video.posterMimeType,
    'width': video.width,
    'height': video.height,
    if (video.duration case final duration?)
      'durationMilliseconds': duration.inMilliseconds,
    'altText': video.altText,
  };

  static VideoDraftDescriptor? _decodeVideo(Object? source) {
    if (source == null) return null;
    try {
      final video = source as Map<String, Object?>;
      return VideoDraftDescriptor(
        storageRevision: video['storageRevision']! as String,
        sourceStorageFileName: video['sourceStorageFileName']! as String,
        posterStorageFileName: video['posterStorageFileName']! as String,
        displayFileName: video['displayFileName']! as String,
        mimeType: video['mimeType']! as String,
        sourceByteLength: video['sourceByteLength']! as int,
        sourceSha256: video['sourceSha256']! as String,
        posterByteLength: video['posterByteLength']! as int,
        posterSha256: video['posterSha256']! as String,
        thumbnailFrame: VideoThumbnailFrame.fromPercent(
          video['thumbnailFramePercent'],
        ),
        thumbnailStorageFileName: video['thumbnailStorageFileName'] as String?,
        thumbnailByteLength: video['thumbnailByteLength'] as int?,
        thumbnailSha256: video['thumbnailSha256'] as String?,
        posterMimeType: video['posterMimeType']! as String,
        width: video['width']! as int,
        height: video['height']! as int,
        duration: switch (video['durationMilliseconds']) {
          final int milliseconds => Duration(milliseconds: milliseconds),
          null => null,
          _ => throw const DraftManifestException(
            DraftManifestFailureReason.invalidMedia,
          ),
        },
        altText: video['altText']! as String,
      )..validate();
    } on DraftManifestException {
      rethrow;
    } on Object catch (error, stackTrace) {
      Error.throwWithStackTrace(
        DraftManifestException(
          DraftManifestFailureReason.invalidMedia,
          diagnosticCause: error,
          diagnosticStack: stackTrace,
        ),
        stackTrace,
      );
    }
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
    if (descriptors.length > defaultMaxPostImages) {
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
