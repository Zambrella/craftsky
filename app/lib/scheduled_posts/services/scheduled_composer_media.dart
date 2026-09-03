import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/feed/composer/link_preview_controller.dart';
import 'package:craftsky_app/feed/composer/prepared_media_validation.dart';
import 'package:craftsky_app/feed/media/composer_image_media_service.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/models/link_preview.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/scheduled_posts/models/scheduled_post.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image/image.dart' as img;
import 'package:mime/mime.dart';

typedef ScheduledMediaBytesLoader = Future<Uint8List> Function(String id);
typedef ScheduledMediaStager =
    Future<void> Function({
      required String id,
      required List<int> bytes,
      required String mimeType,
      CancelToken? cancelToken,
    });
typedef ScheduledComposerMediaMaterializer =
    Future<List<Map<String, dynamic>>> Function(
      List<ComposerImageDraft> images, {
      required ScheduledMediaStager stageMedia,
      void Function(int)? onStaged,
    });

final scheduledComposerMediaMaterializerProvider =
    Provider<ScheduledComposerMediaMaterializer>(
      (ref) => materializeScheduledComposerMedia,
    );

Future<List<ComposerImageDraft>> hydrateScheduledComposerMedia(
  List<dynamic> payloadMedia, {
  required ScheduledMediaBytesLoader loadBytes,
}) async {
  final drafts = <ComposerImageDraft>[];
  for (final value in payloadMedia) {
    if (value is! Map<dynamic, dynamic>) {
      throw const FormatException('invalid scheduled media');
    }
    final id = value['id'];
    final width = value['width'];
    final height = value['height'];
    if (id is! String || width is! int || height is! int) {
      throw const FormatException('invalid scheduled media');
    }
    final bytes = await loadBytes(id);
    drafts.add(
      ComposerImageDraft(
        id: id,
        fileName: 'scheduled-image',
        mimeType: 'application/octet-stream',
        altText: value['alt'] is String ? value['alt']! as String : '',
        previewBytes: bytes,
        previewAspectRatio: CreatePostImageAspectRatio(
          width: width,
          height: height,
        ),
        phase: ScheduledImageReady(width: width, height: height),
      ),
    );
  }
  return List.unmodifiable(drafts);
}

Future<List<Map<String, dynamic>>> materializeScheduledComposerMedia(
  List<ComposerImageDraft> images, {
  required ScheduledMediaStager stageMedia,
  ComposerImageMediaService mediaService = const ComposerImageMediaService(),
  void Function(int)? onStaged,
  Duration transferBudget = const Duration(minutes: 1),
}) async {
  final media = <Map<String, dynamic>>[];
  for (final draft in images) {
    switch (draft.phase) {
      case ScheduledImageReady(:final width, :final height):
        media.add({
          'id': draft.id,
          'alt': draft.altText.trim(),
          'width': width,
          'height': height,
        });
      case ImageUploaded(:final uploaded):
        final bytes = draft.previewBytes;
        if (bytes == null) {
          throw StateError('scheduled image bytes are unavailable');
        }
        final aspectRatio = uploaded.aspectRatio;
        if (aspectRatio == null) {
          throw StateError('scheduled image dimensions are unavailable');
        }
        verifyPreparedMediaBytes(
          bytes: bytes,
          mimeType: uploaded.mime,
          width: aspectRatio.width,
          height: aspectRatio.height,
          altText: draft.altText,
          mediaService: mediaService,
        );
        await _stageWithBudget(
          stageMedia,
          id: draft.id,
          bytes: bytes,
          mimeType: uploaded.mime,
          transferBudget: transferBudget,
        );
        media.add({
          'id': draft.id,
          'alt': draft.altText.trim(),
          'width': aspectRatio.width,
          'height': aspectRatio.height,
        });
        onStaged?.call(media.length);
      case ImageReady(
        :final bytes,
        :final mimeType,
        :final width,
        :final height,
        :final sha256,
      ):
        verifyPreparedMediaBytes(
          bytes: bytes,
          mimeType: mimeType,
          width: width,
          height: height,
          altText: draft.altText,
          mediaService: mediaService,
          expectedSha256: sha256,
        );
        await _stageWithBudget(
          stageMedia,
          id: draft.id,
          bytes: bytes,
          mimeType: mimeType,
          transferBudget: transferBudget,
        );
        media.add({
          'id': draft.id,
          'alt': draft.altText.trim(),
          'width': width,
          'height': height,
        });
        onStaged?.call(media.length);
      case ImageQueued() ||
          ImageReading() ||
          ImagePreparing() ||
          ImageUploading() ||
          ImageUnavailable() ||
          ImageFailed():
        throw StateError('scheduled image is not ready');
    }
  }
  return media;
}

Future<ScheduledPostExternal> materializeScheduledExternal(
  SelectedLinkPreview selection, {
  required String mediaId,
  required ScheduledMediaStager stageMedia,
  Duration transferBudget = const Duration(minutes: 1),
}) async {
  final thumbnail = selection.preview.thumbnail;
  if (thumbnail != null) {
    await _stageWithBudget(
      stageMedia,
      id: mediaId,
      bytes: thumbnail.bytes,
      mimeType: thumbnail.mimeType,
      transferBudget: transferBudget,
    );
  }
  return ScheduledPostExternal(
    sourceUri: selection.candidate.identity.toString(),
    uri: selection.navigationUri.toString(),
    title: selection.preview.title,
    description: selection.preview.description,
    thumbMediaId: thumbnail == null ? null : mediaId,
  );
}

Future<LinkPreviewThumbnail> hydrateScheduledExternalThumbnail(
  String mediaId, {
  required ScheduledMediaBytesLoader loadBytes,
}) async {
  final bytes = await loadBytes(mediaId);
  if (bytes.isEmpty || bytes.length > maxLinkPreviewThumbnailBytes) {
    throw const FormatException('invalid scheduled external thumbnail size');
  }
  final mimeType = lookupMimeType('', headerBytes: bytes);
  if (mimeType != 'image/jpeg' &&
      mimeType != 'image/png' &&
      mimeType != 'image/webp') {
    throw const FormatException('invalid scheduled external thumbnail type');
  }
  final decoded = img.decodeImage(bytes);
  if (decoded == null || decoded.width <= 0 || decoded.height <= 0) {
    throw const FormatException('invalid scheduled external thumbnail');
  }
  return LinkPreviewThumbnail(
    bytes: bytes,
    mimeType: mimeType!,
    width: decoded.width,
    height: decoded.height,
  );
}

Future<void> _stageWithBudget(
  ScheduledMediaStager stageMedia, {
  required String id,
  required List<int> bytes,
  required String mimeType,
  required Duration transferBudget,
}) async {
  final cancelToken = CancelToken();
  await stageMedia(
    id: id,
    bytes: bytes,
    mimeType: mimeType,
    cancelToken: cancelToken,
  ).timeout(
    transferBudget,
    onTimeout: () {
      cancelToken.cancel('scheduled image transfer timed out');
      throw TimeoutException(
        'scheduled image transfer timed out',
        transferBudget,
      );
    },
  );
}
